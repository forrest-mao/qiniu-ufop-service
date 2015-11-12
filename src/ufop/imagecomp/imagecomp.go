package imagecomp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qiniu/api.v6/auth/digest"
	"github.com/qiniu/api.v6/rs"
	"github.com/qiniu/rpc"
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
	"io/ioutil"
)

const (
	IMAGECOMP_MAX_URL_COUNT = 1000

	IMAGECOMP_ORDER_BY_ROW = 0
	IMAGECOMP_ORDER_BY_COL = 1
)

const (
	H_ALIGN_LEFT = "left"
	H_ALIGN_RIGHT = "right"
	H_ALIGN_CENTER = "center"
	V_ALIGN_TOP = "top"
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
/rows/<int>			optional, default 1
/cols/<int>			optional, default 1
/halign/<string> 	optional, default left
/valign/<string> 	optional, default top
/order/<int>		optional, default 1
/alpha/<int> 		optional, default 0
/bgcolor/<string>	optional, default gray
/margin/<int>		optional, default 0
/url/<string>
/url/<string>

*/
func (this *ImageComposer) parse(cmd string) (bucket, format, halign, valign string,
rows, cols, order int, bgColor color.Color, margin int, urls []map[string]string, err error) {
	pattern := `^imagecomp/bucket/[0-9a-zA-Z-_=]+(/format/(png|jpg|jpeg)|/halign/(left|right|center)|/valign/(top|bottom|middle)|/rows/\d+|/cols/\d+|/order/(0|1)|/alpha/\d+|/margin/\d+|/bgcolor/[0-9a-zA-Z-_=]+){0,9}(/url/[0-9a-zA-Z-_=]+)+$`

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

	//check later by url count
	//rows
	rows = 0
	if rowsStr := utils.GetParam(cmd, `rows/\d+`, "rows"); rowsStr != "" {
		rows, _ = strconv.Atoi(rowsStr)
	}

	//cols
	cols = 0
	if colsStr := utils.GetParam(cmd, `cols/\d+`, "cols"); colsStr != "" {
		cols, _ = strconv.Atoi(colsStr)
	}

	//halign
	halign = H_ALIGN_LEFT
	if v := utils.GetParam(cmd, "halign/(left|right|center)", "halign"); v != "" {
		halign = v
	}

	//valign
	valign = V_ALIGN_TOP
	if v := utils.GetParam(cmd, "valign/(top|bottom|middle)", "valign"); v != "" {
		valign = v
	}

	//order
	order = IMAGECOMP_ORDER_BY_COL
	if orderStr := utils.GetParam(cmd, "order/(0|1)", "order"); orderStr != "" {
		order, _ = strconv.Atoi(orderStr)
	}

	//alpha
	alpha := 255

	if format == "png" {
		alpha = 0
	}

	if alphaStr := utils.GetParam(cmd, `alpha/\d+`, "alpha"); alphaStr != "" {
		alpha, _ = strconv.Atoi(alphaStr)
	}

	if alpha < 0 || alpha > 255 {
		err = errors.New("invalid imagecomp parameter 'alhpa', should between [0,255]")
		return
	}

	//bgcolor, default white
	bgColor = color.RGBA{0xFF, 0xFF, 0xFF, uint8(alpha)}

	var bgColorStr string
	bgColorStr, decodeErr = utils.GetParamDecoded(cmd, "bgcolor/[0-9a-zA-Z-_=]+", "bgcolor")
	if decodeErr != nil {
		err = errors.New("invalid imagecomp parameter 'bgcolor'")
		return
	} else {
		if bgColorStr != "" {
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

			bgColor = color.RGBA{
				uint8(redInt),
				uint8(greenInt),
				uint8(blueInt),
				uint8(alpha),
			}
		}
	}

	//margin
	margin = 0
	if marginStr := utils.GetParam(cmd, `margin/\d+`, "margin"); marginStr != "" {
		margin, _ = strconv.Atoi(marginStr)
	}

	//urls
	urls = make([]map[string]string, 0)
	urlsPattern := regexp.MustCompile("url/[0-9a-zA-Z-_=]+")
	urlStrings := urlsPattern.FindAllString(cmd, -1)
	for _, urlString := range urlStrings {
		urlBytes, _ := base64.URLEncoding.DecodeString(urlString[4:])
		urlStr := string(urlBytes)
		uri, pErr := url.Parse(urlStr)
		if pErr != nil {
			err = errors.New(fmt.Sprintf("invalid imagecomp parameter 'url', wrong '%s'", urlStr))
			return
		}

		urls = append(urls, map[string]string{
			"path": uri.Path[1:],
			"url":  urlStr,
		})
	}

	//check rows and cols valid or not
	urlCount := len(urls)

	if urlCount > IMAGECOMP_MAX_URL_COUNT {
		err = errors.New(fmt.Sprintf("only allow url count not larger than %d", IMAGECOMP_MAX_URL_COUNT))
		return
	}

	if rows == 0 && cols == 0 {
		cols = 1
		rows = urlCount / cols
	} else if rows == 0 && cols != 0 {
		if cols > urlCount {
			err = errors.New("cols larger than url count error")
			return
		}
		if urlCount % cols == 0 {
			rows = urlCount / cols
		} else {
			rows = urlCount / cols + 1
		}
	} else if rows != 0 && cols == 0 {
		if rows > urlCount {
			err = errors.New("rows larger than url count error")
			return
		}
		if urlCount % rows == 0 {
			cols = urlCount / rows
		} else {
			cols = urlCount / rows + 1
		}
	} else {
		if urlCount > rows * cols {
			err = errors.New("url count larger than rows*cols error")
			return
		}

		if urlCount < rows * cols {
			switch order {
			case IMAGECOMP_ORDER_BY_ROW:
				if urlCount < (rows - 1) * cols + 1 {
					err = errors.New("url count less than (rows-1)*cols+1 error")
					return
				}
			case IMAGECOMP_ORDER_BY_COL:
				if urlCount < rows * (cols - 1) + 1 {
					err = errors.New("url count less than rows*(cols-1)+1 error")
					return
				}
			}
		}
	}

	return
}

func (this *ImageComposer) Do(req ufop.UfopRequest) (result interface{}, contentType string, err error) {
	bucket, format, halign, valign, rows, cols, order, bgColor, margin, urls, pErr := this.parse(req.Cmd)
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
	statUrls := make([]string, 0)
	for _, urlItem := range urls {
		iPath := urlItem["path"]
		iUrl := urlItem["url"]
		entryPath := rs.EntryPath{
			bucket, iPath,
		}
		statItems = append(statItems, entryPath)
		statUrls = append(statUrls, iUrl)
	}

	qclient := rs.New(this.mac)

	statRet, statErr := qclient.BatchStat(nil, statItems)

	if statErr != nil {
		if sErr, ok := statErr.(*rpc.ErrorInfo); !ok {
			err = errors.New(fmt.Sprintf("batch stat error, %s", statErr.Error()))
			return
		} else {
			if sErr.Err != "" {
				err = errors.New(fmt.Sprintf("batch stat error, %s", sErr.Err))
				return
			}
		}
	}

	for index := 0; index < len(statRet); index++ {
		ret := statRet[index]
		if ret.Code != 200 {
			if ret.Code == 612 {
				err = errors.New(fmt.Sprintf("batch stat '%s' error, no such file or directory", statUrls[index]))
			} else if ret.Code == 631 {
				err = errors.New(fmt.Sprintf("batch stat '%s' error, no such bucket", statUrls[index]))
			} else {
				err = errors.New(fmt.Sprintf("batch stat '%s' error, %d", statUrls[index], ret.Code))
			}
			return
		}
	}

	//download images by url
	localImgPathTypeMap := make(map[string]string)
	localImgPaths := make([]string, 0)
	remoteImgUrls := make(map[string]string)
	for _, urlItem := range urls {
		iUrl := urlItem["url"]
		iLocalName := fmt.Sprintf("imagecomp_tmp_%s_%d", utils.Md5Hex(iUrl), time.Now().UnixNano())
		iLocalPath := filepath.Join(os.TempDir(), iLocalName)
		dContentType, dErr := utils.Download(iUrl, iLocalPath)
		if dErr != nil {
			err = dErr
			return
		}

		if !(dContentType == "image/png" || dContentType == "image/jpeg") {
			err = errors.New(fmt.Sprintf("unsupported mimetype of '%s', '%s'", iUrl, dContentType))
			return
		}

		localImgPaths = append(localImgPaths, iLocalPath)
		localImgPathTypeMap[iLocalPath] = dContentType
		remoteImgUrls[iLocalPath] = iUrl
	}

	defer func() {
		for iPath, _ := range localImgPathTypeMap {
			os.Remove(iPath)
		}
	}()

	//layout the images
	localImgFps := make([]*os.File, 0)

	var localImgObjs [][]image.Image = make([][]image.Image, rows * cols)

	for index := 0; index < rows; index++ {
		localImgObjs[index] = make([]image.Image, cols)
	}

	var rowIndex int = 0
	var colIndex int = 0

	for _, iLocalPath := range localImgPaths {
		iContentType := localImgPathTypeMap[iLocalPath]
		imgFp, openErr := os.Open(iLocalPath)
		if openErr != nil {
			err = errors.New(fmt.Sprintf("open local image of remote '%s' failed, %s", remoteImgUrls[iLocalPath], openErr.Error()))
			return
		}
		localImgFps = append(localImgFps, imgFp)

		var imgObj image.Image
		var dErr error

		if iContentType == "image/png" {
			imgObj, dErr = png.Decode(imgFp)
			if dErr != nil {
				err = errors.New(fmt.Sprintf("decode png image of remote '%s' failed, %s", remoteImgUrls[iLocalPath], dErr.Error()))
				return
			}
		} else if iContentType == "image/jpeg" {
			imgObj, dErr = jpeg.Decode(imgFp)
			if dErr != nil {
				err = errors.New(fmt.Sprintf("decode jpeg image of remote '%s' failed, %s", remoteImgUrls[iLocalPath], dErr.Error()))
				return
			}
		} else {
			err = errors.New(fmt.Sprintf("unsupported src image format '%s' of url '%s'", iContentType, remoteImgUrls[iLocalPath]))
			return
		}

		localImgObjs[rowIndex][colIndex] = imgObj

		//update index
		switch order {
		case IMAGECOMP_ORDER_BY_ROW:
			if colIndex < cols - 1 {
				colIndex += 1
			} else {
				colIndex = 0
				rowIndex += 1
			}

		case IMAGECOMP_ORDER_BY_COL:
			if rowIndex < rows - 1 {
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
	rowImageMaxHeights := make([]int, 0)

	for _, rowSlice := range localImgObjs {
		if len(rowSlice) == 0 {
			continue
		}

		rowImageColWidths := make([]int, 0)
		rowImageColHeights := make([]int, 0)

		for _, imgObj := range rowSlice {
			if imgObj != nil {
				bounds := imgObj.Bounds()
				rowImageColWidths = append(rowImageColWidths, bounds.Max.X - bounds.Min.X)
				rowImageColHeights = append(rowImageColHeights, bounds.Max.Y - bounds.Min.Y)
			}
		}

		rowImageColMaxWidth := utils.MaxInt(rowImageColWidths...)
		rowImageColMaxHeight := utils.MaxInt(rowImageColHeights...)

		rowImageMaxWidths = append(rowImageMaxWidths, rowImageColMaxWidth)
		rowImageMaxHeights = append(rowImageMaxHeights, rowImageColMaxHeight)
	}

	blockWidth := utils.MaxInt(rowImageMaxWidths...)
	blockHeight := utils.MaxInt(rowImageMaxHeights...)

	//dest image width & height with margin
	dstImageWidth = blockWidth * cols + (cols + 1) * margin
	dstImageHeight = blockHeight * rows + (rows + 1) * margin

	//compose the dst image
	dstRect := image.Rect(0, 0, dstImageWidth, dstImageHeight)
	dstImage := image.NewRGBA(dstRect)

	draw.Draw(dstImage, dstImage.Bounds(), image.NewUniform(bgColor), image.ZP, draw.Src)

	for rowIndex, rowSlice := range localImgObjs {
		for colIndex := 0; colIndex < len(rowSlice); colIndex++ {
			imgObj := rowSlice[colIndex]

			//check nil
			if imgObj == nil {
				continue
			}

			imgWidth := imgObj.Bounds().Max.X - imgObj.Bounds().Min.X
			imgHeight := imgObj.Bounds().Max.Y - imgObj.Bounds().Min.Y

			//calc the draw rect start point
			p1 := image.Point{
				colIndex * blockWidth + (colIndex + 1) * margin,
				rowIndex * blockHeight + (rowIndex + 1) * margin,
			}

			//check halign and valign
			//default is left and top
			switch halign {
			case H_ALIGN_CENTER:
				offset := (blockWidth - imgWidth) / 2
				p1.X += offset
			case H_ALIGN_RIGHT:
				offset := (blockWidth - imgWidth)
				p1.X += offset
			}

			switch valign {
			case V_ALIGN_MIDDLE:
				offset := (blockHeight - imgHeight) / 2
				p1.Y += offset
			case V_ALIGN_BOTTOM:
				offset := (blockHeight - imgHeight)
				p1.Y += offset
			}

			//calc the draw rect end point
			p2 := image.Point{}
			p2.X = p1.X + blockWidth
			p2.Y = p1.Y + blockHeight

			drawRect := image.Rect(p1.X, p1.Y, p2.X, p2.Y)

			//draw
			draw.Draw(dstImage, drawRect, imgObj, imgObj.Bounds().Min, draw.Src)
		}
	}

	contentType = formatMimes[format]

	var buffer = bytes.NewBuffer(nil)
	switch contentType {
	case "image/png":
		eErr := png.Encode(buffer, dstImage)
		if eErr != nil {
			err = errors.New(fmt.Sprintf("create dst png image failed, %s", eErr))
			return
		}

	case "image/jpeg":
		eErr := jpeg.Encode(buffer, dstImage, &jpeg.Options{
			Quality: 100,
		})
		if eErr != nil {
			err = errors.New(fmt.Sprintf("create dst jpeg image failed, %s", eErr))
			return
		}
	}

	result = buffer.Bytes()

	ioutil.WriteFile("/home/jemy/test.png", buffer.Bytes(), 0666)
	return
}
