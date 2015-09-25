package imagecomp

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/qiniu/api.v6/auth/digest"
	"github.com/qiniu/rs"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
	"ufop"
	"ufop/utils"
)

const (
	IMAGECOMP_MAX_URL_COUNT = 1000

	IMAGECOMP_ORDER_BY_ROW = 0
	IMAGECOMP_ORDER_BY_COL = 1
)

const (
	H_ALIGN_LEFT   = "left"
	H_ALIGN_RIGHT  = "right"
	H_ALIGN_CENTER = "center"
	V_ALIGN_TOP    = "top"
	V_ALIGN_BOTTOM = "bottom"
	V_ALIGN_MIDDLE = "middle"
)

type ImageComposer struct {
	mac *digest.Mac
}

type ImageComposerConfig struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

func (this *ImageComposer) Name() string {
	return "imagecomp"
}

func (this *ImageComposer) InitConfig(jobConf string) (err error) {
	confFp, openErr := os.Open(jobConf)
	if openErr != nil {
		err = errors.New(fmt.Sprintf("Open imagecomp config failed, %s", openErr.Error()))
		return
	}

	config := ImageComposerConfig{}

	decoder := json.NewDecoder(confFp)
	decodeErr := decoder.Decode(&config)
	if decodeErr != nil {
		err = errors.New(fmt.Sprintf("Parse mkzip config failed, %s", decodeErr.Error()))
		return
	}

	this.mac = &digest.Mac{config.AccessKey, []byte(config.SecretKey)}
	return
}

/*

imagecomp
/bucket/<string>
/format/<string> 	optional, default jpg
/halign/<string> 	optional, default center
/valign/<string> 	optional, default middle
/rows/<int>			optional, default 1
/cols/<int>			optional, default 1
/order/<int>		optional, default 1
/alpha/<int> 		optional, default 0
/bgcolor/<string>	optional, default gray
/url/<string>
/url/<string>

*/
func (this *ImageComposer) parse(cmd string) (bucket, format, halign, valign string,
	rows, cols, order int, bgColor *color.Color, urls []map[string]string, err error) {
	pattern = `^imagecomp/bucket/[0-9a-zA-Z-_=]+(/format/(png|jpg|jpeg)|/halign/(left|right|center)|/valign/(top|bottom|middle)|/rows/\d+|/cols/\d+|/order/(0|1)|/alpha/\d+|/bgcolor/[0-9a-zA-Z-_=]+){0,8}(/url/[0-9a-zA-Z-_=]+)+$`

	matched, _ := regexp.Match(pattern, []byte(cmd))
	if !matched {
		err = errors.New("invalid imagecomp command format")
		return
	}

	var decodeErr error

	//bucket
	bucket, decodeErr = utils.GetParamDecoded(cmd, "bucket/[0-9a-zA-Z-_=]+", "bucket")
	if decodeErr != nil {
		err = errors.New("invalid imagecomp parameter 'bucket'")
		return
	}

	//format
	format = "jpg"
	if v := utils.GetParam(cmd, "format/(png|jpg|jpeg)", "format"); v != "" {
		format = v
	}

	//halign
	halign = "center"
	if v := utils.GetParam(cmd, "halign/(left|right|center)", "halign"); v != "" {
		halign = v
	}

	//valign
	valign = "middle"
	if v := utils.GetParam(cmd, "valign/(top|bottom|middle)", "valign"); v != "" {
		valign = v
	}

	//check later by url count
	//row
	if rowsStr := utils.GetParam(cmd, `row/\d+`, "row"); rowStr != "" {
		rows, _ = strconv.Atoi(rowsStr)
	}

	//col
	if colsStr := utils.GetParam(cmd, `col/\d+`, "col"); colStr != "" {
		cols, _ = strconv.Atoi(colsStr)
	}

	//order
	order = 0
	if orderStr := utils.GetParam(cmd, "order/(0|1)", "order"); orderStr != "" {
		order, _ = strconv.Atoi(orderStr)
	}

	//alpha
	alpha := 0
	if alphaStr := utils.GetParam(cmd, "alpha/(0|1)", "alpha"); alphaStr != "" {
		alpha, _ = strconv.Atoi(alphaStr)
	}

	if alpha < 0 || alpha > 100 {
		err = errors.New("invalid imagecomp parameter 'alhpa', should between [0,100]")
	}

	//bgcolor
	bgColor = color.Gray

	var bgColorStr string
	bgColorStr, decodeErr = utils.GetParamDecoded(cmd, "bgcolor/[0-9a-zA-Z-_=]+", "bgcolor")
	if decodeErr != nil {
		err = errors.New("invalid imagecomp parameter 'bgcolor'")
		return
	} else {
		colorPattern := `^#[a-fA-F0-9]{6}$`
		if matched, _ := regexp.Match(colorPattern, []byte(bgColorStr)); !matched {
			err = errors.New("invalid imagecomp parameter 'bgcolor', should in format '#FFFFFF'")
			return
		}

		bgColorStr = bgColorStr[1:]

		redPart := bgColorStr[0:2]
		greenPart := bgColorStr[2:4]
		bluePart := bgColorStr[4:6]

		redInt, _ := strconv.ParseInt(redPart, 16, 64)
		greenInt, _ := strconv.ParseInt(greenPart, 16, 64)
		blueInt, _ := strconv.ParseInt(bluePart, 16, 64)

		bgColor = color.NRGBA(uint32(redInt), uint32(greenInt), uint32(blueInt), uint32(alpha))
	}

	//urls
	urls = make([]map[string]string, 0)
	urlsPattern := regexp.MustCompile("url/[0-9a-zA-Z-_=]+")
	urlStrings := urlsPattern.FindAllString(cmd, -1)
	for _, urlStr := range urlStrings {
		uri, pErr := url.Parse(urlStr)
		if pErr != nil {
			err = errors.New(fmt.Sprintf("invalid imagecomp parameter 'url', wrong '%s'", url))
			return
		}

		urls = append(urls, map[string]string{
			"path": uri.Path,
			"url":  urlStr,
		})
	}

	//check row and col valid or not
	urlCount := len(urls)

	if urlCount > IMAGECOMP_MAX_URL_COUNT {
		err = errors.New(fmt.Sprintf("only allow url count not larger than %d", IMAGECOMP_MAX_URL_COUNT))
		return
	}

	if urlCount > rows*cols {
		err = errors.New("url count larger than rows*cols error")
		return
	}

	if urlCount < rows*cols {
		switch order {
		case 0:
			if urlCount < (rows-1)*cols {
				err = errors.New("url count less than (rows-1)*cols error")
				return
			}
		case 1:
			if urlCount < rows*(cols-1) {
				err = errors.New("url count less than rows*(cols-1) error")
				return
			}
		}
	}

	return
}

func (this *ImageComposer) Do(req UfopRequest) (result interface{}, contentType string, err error) {
	bucket, format, halign, valign, rows, cols, order, bgColor, urls, pErr := this.parse(req.Cmd)
	if pErr != nil {
		err = pErr
		return
	}

	formatMimes := map[string]string{
		"png":  "image/png",
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
	}

	//check urls validity, all should in bucket
	statItems := make([]rs.EntryPath, 0)
	for _, urlItem := range urls {
		iPath := urlItem["path"]
		entryPath := rs.EntryPath{
			bucket, iPath,
		}
		statItems = append(statItems, entryPath)
	}

	qclient := rs.New(this.mac)

	statRet, statErr := qclient.BatchStat(nil, statItems)
	if statErr != nil {
		err = errors.New(fmt.Sprintf("batch stat error, %s", statErr))
		return
	}

	//download images by url
	localImgPaths := make(map[string]string)
	remoteImgUrls := make(map[string]string)
	for _, urlItem := range urls {
		iUrl := urlItem["url"]
		iLocalName := fmt.Sprintf("imagecomp_tmp_%s_%d", utils.Md5Hex(iUrl), time.Now().UnixNano())
		iLocalPath := filepath.Join(os.TempDir(), iLocalName)
		contentType, dErr := utils.Download(iUrl, iLocalPath)
		if dErr != nil {
			err = dErr
			return
		}

		if !(contentType == "image/png" || contentType == "image/jpeg") {
			err = errors.New(fmt.Sprintf("unsupported mimetype of '%s', '%s'", iUrl, contentType))
			return
		}

		localImgPaths[iLocalPath] = contentType
		remoteImgUrls[iLocalPath] = iUrl
	}

	defer func() {
		for iPath, _ := range localImgPaths {
			os.Remove(iPath)
		}
	}()

	//layout the images
	localImgFps := make([]*os.File, 0)
	var localImgObjs [rows][cols]*image.Image
	var rowIndex int = 0
	var colIndex int = 0

	for iPath, iContentType := range localImgPaths {
		imgFp, openErr := os.Open(iPath)
		if openErr != nil {
			err = errors.New(fmt.Sprintf("open local image of remote '%s' failed, %s", remoteImgUrls[iPath], openErr.Error()))
			return
		}
		localImgFps = append(localImgFps, imgFp)

		var imgObj *image.Image
		var dErr error

		if iContentType == "image/png" {
			imgObj, dErr = png.Decode(imgFp)
			if dErr != nil {
				err = errors.New(fmt.Sprintf("decode png image of remote '%s' failed, %s", remoteImgUrls[iPath], dErr.Error()))
				return
			}
		} else if iContentType == "image/jpeg" {
			imgObj, dErr = jpeg.Decode(imgFp)
			if dErr != nil {
				err = errors.New(fmt.Sprintf("decode jpeg image of remote '%s' failed, %s", remoteImgUrls[iPath], dErr.Error()))
				return
			}
		}

		switch order {
		case IMAGECOMP_ORDER_BY_ROW:
			localImgObjs[rowIndex][colIndex] = imgObj

			if colIndex < cols {
				colIndex += 1
			} else {
				colIndex = 0
				rowIndex += 1
			}

		case IMAGECOMP_ORDER_BY_COL:
			localImgObjs[rowIndex][colIndex] = imgObj

			if rowIndex < rows {
				rowIndex += 1
			} else {
				rowIndex = 0
				colIndex += 1
			}
		}
	}

	//close file handlers
	defer func() {
		for _, fp := range localImgFps {
			fp.Close()
		}
	}()

	//calc the dst image size
	dstImageWidth := 0
	dstImageHeight := 0

	rowImageMaxWidths := make([]int, 0)
	colImageMaxHeights := make([]int, 0)
	for _, rowSlice := range localImgObjs {
		rowImageMaxWidth := 0
		colImageMaxHeight := 0
		for _, imgObj := range rowSlice {
			bounds := imgObj.Bounds()
			rowImageMaxWidth += bounds.Max.X - bounds.Min.X
			colImageMaxHeight += bounds.Max.Y - bounds.Min.Y
		}

		rowImageMaxWidths = append(rowImageMaxWidths, rowImageMaxWidth)
		colImageMaxHeights = append(colImageMaxHeights, colImageMaxHeight)
	}

	dstImageWidth = utils.MaxInt(rowImageMaxWidths...)
	dstImageHeight = utils.MaxInt(colImageMaxHeights...)

	//compose the dst image
	dstRect := image.Rect(0, 0, dstImageWidth, dstImageHeight)
	dstImage := image.NewRGBA(dstRect)

	//draw background
	draw.Draw(dstImage, dstImage.Bounds(), bgColor, image.ZP, draw.Src)

	drawStartPoint := image.ZP

	for _, rowSlice := range localImgObjs {
		for _, imgObj := range rowSlice {
			//calc the draw start point

			//draw
			draw.Draw(dstImage, imgObj.Bounds(), imgObj, drawStartPoint, draw.Src)
		}
	}

	contentType = formatMimes[format]

	var buffer = bytes.NewBuffer(nil)
	switch contentType {
	case "image/png":
		eErr := png.Encode(buffer, *dstImage)
		if eErr != nil {
			err = errors.New(fmt.Sprintf("create dst png image failed, %s", eErr))
			return
		}

	case "image/jpeg":
		eErr := jpeg.Encode(buffer, *dstImage, jpeg.Options{100})
		if eErr != nil {
			err = errors.New(fmt.Sprintf("create dst jpeg image failed, %s", eErr))
			return
		}
	}

	result = buffer.Bytes()
	return
}
