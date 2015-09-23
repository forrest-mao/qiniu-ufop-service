package imagecomp

import (
	"image/color"
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

func (this *ImageComposer) parse(cmd string) (bucket, format, halign, valign string,
	row, col, order int, alpha bool, bgColor color.Color, urls []string, err error) {
	pattern = ``
	return
}

func (this *ImageComposer) Do(req UfopRequest) (result interface{}, contentType string, err error) {
	return
}
