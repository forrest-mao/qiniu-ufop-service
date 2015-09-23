package imagecomp

import (
	"encoding/hex"
	"errors"
	"image/color"
	"net/url"
	"regexp"
	"strconv"
	"ufop"
	"ufop/utils"
)

const (
	H_ALIGN_LEFT   = "Left"
	H_ALIGN_RIGHT  = "Right"
	H_ALIGN_CENTER = "Center"
	V_ALIGN_TOP    = "Top"
	V_ALIGN_BOTTOM = "Bottom"
	V_ALIGN_MIDDLE = "Middle"
)

type ImageComposer struct {
}

func (this *ImageComposer) Name() string {
	return "imagecomp"
}

func (this *ImageComposer) InitConfig(jobConf string) (err error) {
	return
}

/*

imagecomp
/bucket/<string>
/format/<string> 	optional, default jpg
/halign/<string> 	optional, default Center
/valign/<string> 	optional, default Middle
/row/<int>			optional, default 1
/col/<int>			optional, default 1
/order/<int>		optional, default 1
/alpha/<int> 		optional, default 0
/bgcolor/<string>	optional, default gray
/url/<string>
/url/<string>

*/
func (this *ImageComposer) parse(cmd string) (bucket, format, halign, valign string,
	row, col, order int, alpha bool, bgColor *color.Color, urls []map[string]string, err error) {
	pattern = `^imagecomp/bucket/[0-9a-zA-Z-_=]+(/format/(png|jpg|jpeg)|/halign/(left|right|center)|/valign/(top|bottom|middle)|/row/\d+|/col/\d+|/order/(0|1)|/alpha/\d+|/bgcolor/[0-9a-zA-Z-_=]+){0,8}(/url/[0-9a-zA-Z-_=]+)+$`

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
	if rowStr := utils.GetParam(cmd, `row/\d+`, "row"); rowStr != "" {
		row, _ = strconv.Atoi(rowStr)
	}

	//col
	if colStr := utils.GetParam(cmd, `col/\d+`, "col"); colStr != "" {
		col, _ = strconv.Atoi(colStr)
	}

	//order
	order = 0
	if orderStr := utils.GetParam(cmd, "order/(0|1)", "order"); orderStr != "" {
		order, _ = strconv.Atoi(orderStr)
	}

	//alpha
	alpha = 0
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

	return
}

func (this *ImageComposer) Do(req UfopRequest) (result interface{}, contentType string, err error) {
	return
}
