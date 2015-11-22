package roundpic

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gographics/imagick/imagick"
	"github.com/qiniu/bytes"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"ufop"
	"ufop/utils"
	"path/filepath"
	"time"
)

const (
	ROUND_PIC_MAX_FILE_SIZE = 100 * 1024 * 1024
)

type RoundPicer struct {
	maxFileSize uint64
}

type RoundPicConfig struct {
	RoundPicMaxFileSize uint64 `json:"round_pic_max_file_size"`
}

type RoundPicParams struct {
	RadiusX string
	RadiusY string
	Radius  string
}

func (this *RoundPicer) Name() string {
	return "roundpic"
}

func (this *RoundPicer) InitConfig(jobConf string) (err error) {
	confFp, openErr := os.Open(jobConf)
	if openErr != nil {
		err = errors.New(fmt.Sprintf("Open roundpic config failed, %s", openErr.Error()))
		return
	}

	config := RoundPicConfig{}

	decoder := json.NewDecoder(confFp)
	decodeErr := decoder.Decode(&config)
	if decodeErr != nil {
		err = errors.New(fmt.Sprintf("Parse roundpic config failed, %s", decodeErr.Error()))
		return
	}

	if config.RoundPicMaxFileSize <= 0 {
		this.maxFileSize = ROUND_PIC_MAX_FILE_SIZE
	} else {
		this.maxFileSize = config.RoundPicMaxFileSize
	}

	return
}

func (this *RoundPicer) parse(cmd string) (params RoundPicParams, err error) {
	pattern := `^roundpic((/radius/\d+(\.\d+){0,1}%{0,1})|(/radius-x/\d+(\.\d+){0,1}%{0,1}/radius-y/\d+(\.\d+){0,1}%{0,1}))$`
	if matched, _ := regexp.MatchString(pattern, cmd); !matched {
		err = errors.New("invalid roundpic command")
		return
	}

	params = RoundPicParams{}

	//get radius if specified
	params.Radius = utils.GetParam(cmd, `radius/\d+(\.\d+){0,1}%{0,1}`, "radius")

	//get radius-x, radius-y
	params.RadiusX = utils.GetParam(cmd, `radius-x/\d+(\.\d+){0,1}%{0,1}`, "radius-x")
	params.RadiusY = utils.GetParam(cmd, `radius-y/\d+(\.\d+){0,1}%{0,1}`, "radius-y")

	if params.Radius == "" && (params.RadiusX == "" || params.RadiusY == "") {
		err = errors.New("roundpic radius or radius-x or radius-y empty error")
		return
	}

	return
}

func (this *RoundPicer) Do(req ufop.UfopRequest) (result interface{}, resultType int, contentType string, err error) {
	//parse cmd
	cmdParams, pErr := this.parse(req.Cmd)
	if pErr != nil {
		err = pErr
		return
	}

	//check src image
	if matched, _ := regexp.MatchString("image/(png|jpeg)", req.Src.MimeType); !matched {
		err = errors.New("unsupported mimetype, only 'image/png' and 'image/jpeg' supported")
		return
	}

	if req.Src.Fsize > this.maxFileSize {
		err = errors.New("src image size too large, exceeds the limit")
		return
	}

	//download the image
	resp, respErr := http.Get(req.Src.Url)
	if respErr != nil || resp.StatusCode != http.StatusOK {
		if respErr != nil {
			err = errors.New(fmt.Sprintf("get image data failed, %s", respErr.Error()))
		} else {
			err = errors.New(fmt.Sprintf("get image data failed, %s", resp.Status))
			if resp.Body != nil {
				resp.Body.Close()
			}
		}
		return
	}
	defer resp.Body.Close()

	srcImgData, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		err = errors.New(fmt.Sprintf("read image data failed, %s", readErr.Error()))
		return
	}

	var srcImg image.Image
	var decodeErr error

	switch req.Src.MimeType {
	case "image/png":
		srcImg, decodeErr = png.Decode(bytes.NewReader(srcImgData))
	case "image/jpeg":
		srcImg, decodeErr = jpeg.Decode(bytes.NewReader(srcImgData))
	}

	if decodeErr != nil {
		err = errors.New(fmt.Sprintf("decode image failed, %s", decodeErr.Error()))
		return
	}

	srcImgWidth := srcImg.Bounds().Dx()
	srcImgHeight := srcImg.Bounds().Dy()

	//parse cmd params, radius can be pixels or percentage
	radiusX, radiusY := getRadius(cmdParams, srcImgWidth, srcImgHeight)

	//init imagick
	imagick.Initialize()
	defer imagick.Terminate()

	//create mask
	maskDraw := imagick.NewMagickWand()
	defer maskDraw.Destroy()

	backDraw := imagick.NewPixelWand()
	defer backDraw.Destroy()
	backDraw.SetColor("none")

	//draw mask
	nErr := maskDraw.NewImage(uint(srcImgWidth), uint(srcImgHeight), backDraw)
	if nErr != nil {
		err = errors.New(fmt.Sprintf("create mask image failed, %s", nErr.Error()))
		return
	}

	backDraw.SetColor("white")
	roundDraw := imagick.NewDrawingWand()
	defer roundDraw.Destroy()
	roundDraw.SetFillColor(backDraw)
	roundDraw.RoundRectangle(0, 0, float64(srcImgWidth-1), float64(srcImgHeight-1), radiusX, radiusY)

	//draw round pic
	dErr := maskDraw.DrawImage(roundDraw)
	if dErr != nil {
		err = errors.New(fmt.Sprintf("draw mask image failed, %s", dErr.Error()))
		return
	}

	//load src image
	srcDraw := imagick.NewMagickWand()
	defer srcDraw.Destroy()
	rErr := srcDraw.ReadImageBlob(srcImgData)
	if rErr != nil {
		err = errors.New(fmt.Sprintf("read src image failed, %s", rErr.Error()))
		return
	}

	//composite the mask and the src image
	cErr := maskDraw.CompositeImage(srcDraw, imagick.COMPOSITE_OP_SRC_IN, 0, 0)
	if cErr != nil {
		err = errors.New(fmt.Sprintf("composite mask and src image failed, %s", cErr.Error()))
		return
	}

	//write dest image
	oTmpFpath := filepath.Join(os.TempDir(), fmt.Sprintf("roundpic_tmp_result_%d.png", time.Now().UnixNano()))
	wErr := maskDraw.WriteImage(oTmpFpath)
	if wErr != nil {
		err = errors.New(fmt.Sprintf("write dest image failed, %s", wErr.Error()))
		defer os.Remove(oTmpFpath)
		return
	}

	//write result
	result = oTmpFpath
	resultType = ufop.RESULT_TYPE_OCTECT
	contentType = "image/png"

	return
}

func getRadius(cmdParams RoundPicParams, srcImgWidth, srcImgHeight int) (radiusX, radiusY float64) {
	if cmdParams.Radius != "" {
		var radius float64
		if strings.HasSuffix(cmdParams.Radius, "%") {
			percentStr := cmdParams.Radius[:len(cmdParams.Radius) -1]
			percent, _ := strconv.ParseFloat(percentStr, 64)
			if percent > 50 {
				percent = 50
			}
			radius = float64(utils.MinInt(srcImgWidth, srcImgHeight)) * percent / 100
		} else {
			pixels, _ := strconv.ParseFloat(cmdParams.Radius, 64)
			maxPixelsAllowed := float64(utils.MinInt(srcImgWidth, srcImgHeight)) / 2
			if pixels > maxPixelsAllowed {
				pixels = maxPixelsAllowed
			}
			radius = pixels
		}
		radiusX = radius
		radiusY = radius
	} else {
		//radius-x
		if strings.HasSuffix(cmdParams.RadiusX, "%") {
			percentStr := cmdParams.RadiusX[:len(cmdParams.RadiusX) - 1]
			percent, _ := strconv.ParseFloat(percentStr, 64)
			if percent > 50 {
				percent = 50
			}
			radiusX = float64(srcImgWidth) * percent / 100
		} else {
			pixels, _ := strconv.ParseFloat(cmdParams.RadiusX, 64)
			maxPixelsAllowed := float64(srcImgWidth) / 2
			if pixels > maxPixelsAllowed {
				pixels = maxPixelsAllowed
			}
			radiusX = pixels
		}

		//radius-y
		if strings.HasSuffix(cmdParams.RadiusY, "%") {
			percentStr := cmdParams.RadiusY[:len(cmdParams.RadiusY) - 1]
			percent, _ := strconv.ParseFloat(percentStr, 64)
			if percent > 50 {
				percent = 50
			}
			radiusY = float64(srcImgHeight) * percent / 100
		} else {
			pixels, _ := strconv.ParseFloat(cmdParams.RadiusY, 64)
			maxPixelsAllowed := float64(srcImgHeight) / 2
			if pixels > maxPixelsAllowed {
				pixels = maxPixelsAllowed
			}
			radiusY = pixels
		}
	}

	return
}
