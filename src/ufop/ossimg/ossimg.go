package ossimg

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qiniu/log"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"ufop"
)

/*
@TODO
basic image operation

1. brighten
2. darken
3. internal circle
4. round rectangle
5. crop by slice
6. crop by absolute position

//watermark operation
1. voffset, watermark middle offset
2. order, image text order
3. align, image text align
4. interval, image text interval
*/

//convert all the oss image operation to qiniu style
const (
	OSS_OPER_IMAGE     = "image"
	OSS_OPER_WATERMARK = "watermark"

	OSS_WM_IMAGE = 1
	OSS_WM_TEXT  = 2
	OSS_WM_MIX   = 3
)

var OSS_QINIU_GRAVITY = map[int]string{
	1: "NorthWest",
	2: "North",
	3: "NorthEast",
	4: "West",
	5: "Center",
	6: "East",
	7: "SouthWest",
	8: "South",
	9: "SouthEast",
}

//image basic operation
const (
	WIDTH_PATTERN              = `\d+w_{0,1}`
	HEIGHT_PATTERN             = `\d+h_{0,1}`
	LARGE_PATTERN              = `(0|1)l_{0,1}`
	QUALITY_PATTERN            = `\d+(q|Q)_{0,1}`
	EDGE_PATTERN               = `(0|1|2|4)e_{0,1}`
	PERCENT_PATTERN            = `\d+p_{0,1}`
	BACKGROUND_PATTERN         = `\d+\-\d+\-\d+bgc_{0,1}`
	AUTO_CROP_PATTERN          = `(0|1)c_{0,1}`
	AUTO_CROP_POSITION_PATTERN = `\d+\-\d+\-\d+\-\d+a`
	CROP_BY_GRAVITY_PATTERN    = `(\d+){0,1}x(\d+){0,1}\-(1|2|3|4|5|6|7|8|9)rc_{0,1}`
	ROTATE_PATTERN             = `\d+r[^c]_{0,1}`
	AUTO_ORIENT_PATTERN        = `(0|1|2)o_{0,1}`
	INTERLACE_PATTERN          = `(0|1)pr_{0,1}`
	SHARPEN_PATTERN            = `\d+sh_{0,1}`
	BLUR_PATTERN               = `\d+\-\d+bl_{0,1}`
	DEST_FORMAT_PATTERN        = `\.(jpg|png|webp|bmp|jpeg|src)`
)

type OSSImager struct {
	domain string
	path   string
}

type OSSImageConfig struct {
	Domain string `json:"domain"`
}

type OSSImageOperation struct {
	Name string

	//basic operation
	Width  int //w
	Height int //h

	//whether to enlarge
	DisableLarge int //l

	//q, relative quality
	RelQuality int

	//Q, absolute quality
	Quality int

	//e
	//edge=1
	//edge=2
	//edge=3
	Edge int

	//p, 100% is the original size
	Percent int

	//c, crop
	AutoCrop        int
	AutoCropOffsetX int
	AutoCropOffsetY int
	AutoCropWidth   int
	AutoCropHeight  int

	//r, rotate [0,360]
	Rotate int

	//pr, interlace
	Interlace int

	//o, auto orient
	AutoOrient int

	//jpg, png, webp, bmp, src
	//default jpg
	DestFormat string

	//rc, crop by gravity
	CropByPosWidth   int
	CropByPosHeight  int
	CropByPosGravity int

	//background
	BackgroundRed   int
	BackgroundGreen int
	BackgroundBlue  int

	//sharpen
	Sharpen int

	//blur
	BlurRadius int
	BlurSigma  int

	//watermark
	//1 image watermark
	//2 text watermark
	//3 image&text watermark
	WMType int

	//text, watermark text
	WMText string
	//s, text background transparency
	WMTextDissolve int
	//type, font type
	WMFontType string

	//color, font color
	WMFontColor string

	//size, font size
	WMFontSize int

	//object, watermark image
	WMImage string

	//t, transparency, (0,100]
	WMDissolve int

	//p, gravity
	WMGravity int

	//x, offset x
	WMOffsetX int
	//y, offset y
	WMOffsetY int
}

type ImageInfo struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (this *OSSImager) Name() string {
	return "ossimg"
}

func (this *OSSImager) InitConfig(jobConf string) (err error) {
	confFp, openErr := os.Open(jobConf)
	if openErr != nil {
		err = errors.New(fmt.Sprintf("Open ossimg config failed, %s", openErr.Error()))
		return
	}

	config := OSSImageConfig{}
	decoder := json.NewDecoder(confFp)
	decodeErr := decoder.Decode(&config)
	if decodeErr != nil {
		err = errors.New(fmt.Sprintf("Parse ossimg config failed, %s", decodeErr.Error()))
		return
	}

	this.domain = config.Domain

	return
}

/**
ossimg/2015/10/20/test.jpg@120w_120h_80q_1l_1c.src
*/
func (this *OSSImager) parse(cmd string, operations *[]OSSImageOperation) (err error) {
	cmdParam := strings.TrimPrefix(cmd, this.Name())
	items := strings.Split(cmdParam, "@")
	if len(items) < 1 {
		err = errors.New("invalid rewrite url")
		return
	}

	this.path = items[0]

	operStrItems := items[1:]
	for _, operStr := range operStrItems {
		if strings.HasPrefix(operStr, "watermark") {
			//watermark operation
			operation := this.parseWatermarkOperation(operStr)
			*operations = append(*operations, operation)
		} else {
			//image operation
			operation := this.parseImageOperation(operStr)
			*operations = append(*operations, operation)
		}
	}

	return
}

func (this *OSSImager) Do(req ufop.UfopRequest) (result interface{}, resultType int, contentType string, err error) {
	operations := make([]OSSImageOperation, 0)
	pErr := this.parse(req.Cmd, &operations)
	if pErr != nil {
		err = pErr
		return
	}

	srcUrl := fmt.Sprintf("%s%s", this.domain, this.path)
	qiniuUrl := srcUrl

	for _, oper := range operations {
		var fop string
		switch oper.Name {
		case OSS_OPER_IMAGE:
			fop = this.formatQiniuImageFop(oper)
		case OSS_OPER_WATERMARK:
			fop = this.formatQiniuWatermarkFop(oper)
		}

		if fop != "" {
			if strings.Contains(qiniuUrl, "?") {
				qiniuUrl = fmt.Sprintf("%s|%s", qiniuUrl, fop)
			} else {
				qiniuUrl = fmt.Sprintf("%s?%s", qiniuUrl, fop)
			}
		}
	}

	result = qiniuUrl
	resultType = ufop.RESULT_TYPE_OCTECT_URL

	//for debug
	result = []byte(qiniuUrl)
	resultType = ufop.RESULT_TYPE_OCTECT_BYTES
	return
}

//get image operation parameters
//w, width int
//h, height int
//l, large bool
//e, edge int
//p, percent int, affected by l
//bgc, red int,green int,blue int background filling
//c, crop bool
//rc, crop (int) by position
//r, rotate
//o, orient
//sh, sharpen
func (this *OSSImager) parseImageOperation(oper string) (operation OSSImageOperation) {
	operation = OSSImageOperation{
		Name: OSS_OPER_IMAGE,
	}

	if strings.HasSuffix(oper, "r") {
		oper += ".jpg"
	}

	//parse
	width := this.scanImageParamInt(oper, "w", WIDTH_PATTERN)
	operation.Width = width

	height := this.scanImageParamInt(oper, "h", HEIGHT_PATTERN)
	operation.Height = height

	large := this.scanImageParamInt(oper, "l", LARGE_PATTERN)
	operation.DisableLarge = large

	edge := this.scanImageParamInt(oper, "e", EDGE_PATTERN)
	operation.Edge = edge

	percent := this.scanImageParamInt(oper, "p", PERCENT_PATTERN)
	operation.Percent = percent

	background := this.scanImageParam(oper, "bgc", BACKGROUND_PATTERN)
	if background != "" {
		colorItems := strings.Split(background, "-")
		red, _ := strconv.Atoi(colorItems[0])
		green, _ := strconv.Atoi(colorItems[1])
		blue, _ := strconv.Atoi(colorItems[2])
		operation.BackgroundRed = red
		operation.BackgroundGreen = green
		operation.BackgroundBlue = blue
	}

	crop := this.scanImageParamInt(oper, "c", AUTO_CROP_PATTERN)
	operation.AutoCrop = crop

	cropPos := this.scanImageParam(oper, "a", AUTO_CROP_POSITION_PATTERN)
	if cropPos != "" {
		items := strings.Split(cropPos, "-")
		cropOffsetX, _ := strconv.Atoi(items[0])
		cropOffsetY, _ := strconv.Atoi(items[1])
		cropWidth, _ := strconv.Atoi(items[2])
		cropHeight, _ := strconv.Atoi(items[3])

		operation.AutoCropOffsetX = cropOffsetX
		operation.AutoCropOffsetY = cropOffsetY
		operation.AutoCropWidth = cropWidth
		operation.AutoCropHeight = cropHeight
	}

	quality := this.scanImageParamInt(oper, "qQ", QUALITY_PATTERN)
	qLIndex := strings.LastIndex(oper, "q")
	QLIndex := strings.LastIndex(oper, "Q")
	if qLIndex < QLIndex {
		operation.Quality = quality
	} else if qLIndex > QLIndex {
		operation.RelQuality = quality
	}

	rcStr := this.scanImageParam(oper, "rc", CROP_BY_GRAVITY_PATTERN)
	if rcStr != "" {
		rcItems := strings.Split(rcStr, "-")
		cropGravity, _ := strconv.Atoi(rcItems[1])
		cropWidth := 0
		cropHeight := 0
		if matched, _ := regexp.MatchString(`\d+x\d+`, rcItems[0]); matched {
			items := strings.Split(rcItems[0], "x")
			cropWidth, _ = strconv.Atoi(items[0])
			cropHeight, _ = strconv.Atoi(items[1])
		} else if matched, _ := regexp.MatchString(`\d+x`, rcItems[0]); matched {
			items := strings.Split(rcItems[0], "x")
			cropWidth, _ = strconv.Atoi(items[0])
		} else if matched, _ := regexp.MatchString(`x\d+`, rcItems[0]); matched {
			items := strings.Split(rcItems[0], "x")
			cropHeight, _ = strconv.Atoi(items[0])
		}

		operation.CropByPosGravity = cropGravity
		operation.CropByPosWidth = cropWidth
		operation.CropByPosHeight = cropHeight
	}

	rotate := this.scanImageParamInt(oper, "r.", ROTATE_PATTERN)
	operation.Rotate = rotate

	orient := this.scanImageParamInt(oper, "o", AUTO_ORIENT_PATTERN)
	operation.AutoOrient = orient

	sharpen := this.scanImageParamInt(oper, "sh", SHARPEN_PATTERN)
	operation.Sharpen = sharpen

	interlace := this.scanImageParamInt(oper, "pr", INTERLACE_PATTERN)
	operation.Interlace = interlace

	blurStr := this.scanImageParam(oper, "bl", BLUR_PATTERN)
	if blurStr != "" {
		items := strings.Split(blurStr, "-")
		blurRadius, _ := strconv.Atoi(items[0])
		blurSigma, _ := strconv.Atoi(items[1])
		operation.BlurRadius = blurRadius
		operation.BlurSigma = blurSigma
	}

	dstFormat := this.scanImageParam(oper, "", DEST_FORMAT_PATTERN)
	dstFormat = strings.TrimPrefix(dstFormat, ".")
	operation.DestFormat = dstFormat

	//fix the default values according to the ali oss image operation doc
	//{@link http://help.aliyun.com/document_detail/oss/oss-img-guide/crop/auto-crop.html}
	if operation.AutoCrop == 1 && !strings.Contains(oper, "e") {
		operation.Edge = 1
	}
	return
}

func (this *OSSImager) scanImageParam(cmd string, tag string, pattern string) (val string) {
	regx, _ := regexp.Compile(pattern)
	allFound := regx.FindAllString(cmd, -1)
	if len(allFound) > 0 {
		val = strings.TrimRight(allFound[len(allFound)-1], fmt.Sprintf("%s_", tag))
	}
	return
}

func (this *OSSImager) scanImageParamInt(cmd string, tag string, pattern string) (val int) {
	valStr := this.scanImageParam(cmd, tag, pattern)
	if v, err := strconv.Atoi(valStr); err == nil {
		val = v
	}
	return
}

//get watermark operation parameters
func (this *OSSImager) parseWatermarkOperation(oper string) (operation OSSImageOperation) {
	paramItems := strings.Split(oper, "&")
	params := map[string]string{}
	for _, paramItem := range paramItems {
		kvp := strings.Split(paramItem, "=")
		if len(kvp) == 2 {
			key := strings.TrimSpace(kvp[0])
			value := strings.TrimSpace(kvp[1])
			params[key] = value
		}
	}

	operation = OSSImageOperation{}
	operation.WMType = this.wmInt(params["watermark"])

	//wmText
	operation.WMText = this.wmBase64Decode("text", params["text"])
	//wmFontType
	operation.WMFontType = this.wmBase64Decode("type", params["type"])
	//wmFontColor
	operation.WMFontColor = this.wmBase64Decode("color", params["color"])

	//wmFontSize
	if wmFontSize, pErr := strconv.Atoi(params["size"]); pErr != nil {
		log.Error(fmt.Sprintf("invalid watermark font size, '%s'", params["size"]))
	} else {
		operation.WMFontSize = wmFontSize
	}

	//wmImage
	operation.WMImage = this.wmBase64Decode("object", params["object"])

	//position
	operation.WMGravity = this.wmInt(params["p"])
	//dissolve
	operation.WMDissolve = this.wmInt(params["t"])
	//offsetX
	operation.WMOffsetX = this.wmInt(params["x"])
	//offsetY
	operation.WMOffsetY = this.wmInt(params["y"])

	return
}

func (this *OSSImager) wmInt(value string) (result int) {
	if v, err := strconv.Atoi(value); err == nil {
		result = v
	}
	return
}

func (this *OSSImager) wmBase64Decode(key string, value string) (result string) {
	fLen := len(value)
	toDecodeStr := value
	if (fLen+1)*6%8 == 0 {
		toDecodeStr = fmt.Sprintf("%s=", value)
	} else if (fLen+2)*6%8 == 0 {
		toDecodeStr = fmt.Sprintf("%s==", value)
	}

	resultBytes, pErr := base64.URLEncoding.DecodeString(toDecodeStr)
	if pErr != nil {
		log.Error(fmt.Sprintf("invalid watermark base64 param value for '%s'", key))
	}

	result = string(resultBytes)
	return
}

/*
get image width or height
*/
func (this *OSSImager) getImageInfo(imageUrl string) (imageInfo *ImageInfo, err error) {
	imageInfoUrl := fmt.Sprintf("%s?imageInfo", imageUrl)
	log.Debug(imageInfoUrl)
	resp, respErr := http.Get(imageInfoUrl)
	if respErr != nil {
		err = respErr
		return
	}
	defer resp.Body.Close()
	buffer := bytes.NewBuffer(nil)
	_, cpErr := io.Copy(buffer, resp.Body)
	if cpErr != nil {
		err = cpErr
		return
	}
	imageInfo = &ImageInfo{}
	decodeErr := json.Unmarshal(buffer.Bytes(), imageInfo)
	if decodeErr != nil {
		err = decodeErr
		return
	}
	return
}

func (this *OSSImager) formatQiniuImageFop(oper OSSImageOperation) (qFop string) {
	srcUrl := fmt.Sprintf("%s%s", this.domain, this.path)

	imageInfo, gErr := this.getImageInfo(srcUrl)
	if gErr != nil {
		log.Error("get image info error", gErr.Error())
		return
	}

	width := oper.Width
	height := oper.Height

	//check crop by gravity
	//{@link http://helpcdn.aliyun.com/document_detail/oss/oss-img-guide/crop/area-crop.html}
	var qCropFop string

	if oper.CropByPosGravity != 0 {
		var cropx string
		var cropy string
		if oper.CropByPosWidth != 0 {
			cropx = fmt.Sprintf("%d", oper.CropByPosWidth)
		}
		if oper.CropByPosHeight != 0 {
			cropy = fmt.Sprintf("%d", oper.CropByPosHeight)
		}

		if cropx != "" && cropy != "" {
			qCropFop = fmt.Sprintf("imageMogr2/gravity/%s/crop/%sx%s", OSS_QINIU_GRAVITY[oper.CropByPosGravity], cropx, cropy)
		}
	}

	//check percent
	//{@link http://help.aliyun.com/document_detail/oss/oss-img-guide/resize/resize-scale.html}
	if oper.Percent > 0 {
		width = int(float64(width) * float64(oper.Percent) / 100)
		height = int(float64(height) * float64(oper.Percent) / 100)
	}

	if width != 0 && height != 0 {
		if oper.Edge == 0 {
			qFop = fmt.Sprintf("imageMogr2/thumbnail/%dx%d", width, height)
		} else if oper.Edge == 1 {
			qFop = fmt.Sprintf("imageMogr2/thumbnail/!%dx%dr", width, height)
		} else if oper.Edge == 2 {
			qFop = fmt.Sprintf("imageMogr2/thumbnail/%dx%d!", width, height)
		} else if oper.Edge == 4 {
			qFop = fmt.Sprintf("imageMogr2/thumbnail/%dx%d/extent/%dx%d", width, height, width, height)
			if oper.BackgroundBlue != 0 || oper.BackgroundGreen != 0 || oper.BackgroundRed != 0 {
				background := fmt.Sprintf("#%02x%02x%02x", oper.BackgroundRed, oper.BackgroundGreen, oper.BackgroundBlue)
				qFop = fmt.Sprintf("%s/background/%s", qFop, base64.URLEncoding.EncodeToString([]byte(background)))
			}
		}

		if oper.AutoCrop == 1 {
			qFop = fmt.Sprintf("%s/gravity/Center/crop/%dx%d", qFop, width, height)
		}

		//enlarge disabled
		if oper.DisableLarge == 1 {
			if width > imageInfo.Width || height > imageInfo.Height {
				qFop = ""
				return
			}
		}
	} else if width != 0 || height != 0 {
		if width == 0 {
			qFop = fmt.Sprintf("imageMogr2/thumbnail/x%d", height)
		} else {
			qFop = fmt.Sprintf("imageMogr2/thumbnail/%dx", width)
		}
		//enlarge disabled
		if oper.DisableLarge == 1 {
			if width > imageInfo.Width || height > imageInfo.Height {
				qFop = ""
				return
			}
		}
	} else {
		if oper.Percent > 0 {
			qFop = fmt.Sprintf("imageMogr2/thumbnail/!%dp", oper.Percent)
		}
	}

	if qCropFop != "" {
		qFop = fmt.Sprintf("%s|%s", qCropFop, qFop)
	}

	if qFop == "" {
		qFop = "imageMogr2"
	}

	if oper.Rotate != 0 {
		qFop = fmt.Sprintf("%s/rotate/%d", qFop, oper.Rotate)
	}

	if oper.RelQuality != 0 {
		qFop = fmt.Sprintf("%s/quality/%d", qFop, oper.RelQuality)
	}

	if oper.Quality != 0 {
		qFop = fmt.Sprintf("%s/quality/%d!", qFop, oper.Quality)
	}

	if oper.Interlace == 1 {
		qFop = fmt.Sprintf("%s/interlace/%d", qFop, oper.Interlace)
	}

	if oper.Sharpen > 0 {
		qFop = fmt.Sprintf("%s/sharpen/1", qFop)
	}

	if oper.BlurRadius != 0 && oper.BlurSigma != 0 {
		qFop = fmt.Sprintf("%s/blur/%dx%d", qFop, oper.BlurRadius, oper.BlurSigma)
	}

	if oper.DestFormat == "" {
		qFop = fmt.Sprintf("%s/format/jpg", qFop)
	} else if oper.DestFormat != "src" {
		qFop = fmt.Sprintf("%s/format/%s", qFop, oper.DestFormat)
	}

	//inject the auto-orient
	//{@link http://help.aliyun.com/document_detail/oss/oss-img-guide/rotation/auto-orient.html}
	if oper.AutoOrient == 1 {
		if qFop != "" {
			qFop = fmt.Sprintf("%s/auto-orient", qFop)
		} else {
			qFop = fmt.Sprintf("imageMogr2/auto-orient")
		}
	} else if oper.AutoOrient == 2 {
		if qFop != "" {
			qFop = fmt.Sprintf("imageMogr2/auto-orient%s", strings.TrimPrefix(qFop, "imageMogr2"))
		} else {
			qFop = fmt.Sprintf("imageMogr2/auto-orient")
		}
	}

	if qFop == "imageMogr2" {
		qFop = ""
	}

	//check auto crop
	if oper.AutoCropWidth != 0 && oper.AutoCropHeight != 0 {
		qCropFop = fmt.Sprintf("imageMogr2/crop/!%dx%da%da%d", oper.AutoCropWidth, oper.AutoCropHeight,
			oper.AutoCropOffsetX, oper.AutoCropOffsetY)

		if qFop == "" {
			qFop = qCropFop
		} else {
			qFop = fmt.Sprintf("%s|%s", qFop, qCropFop)
		}
	}

	return
}

func (this *OSSImager) formatQiniuWatermarkFop(oper OSSImageOperation) (qFop string) {
	return
}
