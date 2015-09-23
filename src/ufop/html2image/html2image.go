package html2image

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"ufop"
	"ufop/utils"
)

const (
	HTML2IMAGE_MAX_PAGE_SIZE = 10 * 1024 * 1024
)

type Html2Imager struct {
	maxPageSize int64
}

type Html2ImagerConfig struct {
	Html2ImageMaxPageSize int64 `json:"html2image_max_page_size,omitempty"`
}

type Html2ImageOptions struct {
	CropH   int
	CropW   int
	CropX   int
	CropY   int
	Format  string
	Height  int
	Width   int
	Quality int
	Force   bool
}

func (this *Html2Imager) Name() string {
	return "html2image"
}

func (this *Html2Imager) InitConfig(jobConf string) (err error) {
	confFp, openErr := os.Open(jobConf)
	if openErr != nil {
		err = errors.New(fmt.Sprintf("Open html2image config failed, %s", openErr.Error()))
		return
	}

	config := Html2ImagerConfig{}
	decoder := json.NewDecoder(confFp)
	decodeErr := decoder.Decode(&config)
	if decodeErr != nil {
		err = errors.New(fmt.Sprintf("Parse html2image config failed, %s", decodeErr.Error()))
		return
	}

	if config.Html2ImageMaxPageSize <= 0 {
		this.maxPageSize = HTML2IMAGE_MAX_PAGE_SIZE
	} else {
		this.maxPageSize = config.Html2ImageMaxPageSize
	}

	return
}

func (this *Html2Imager) parse(cmd string) (options *Html2ImageOptions, err error) {
	pattern := `^html2image(/croph/\d+|/cropw/\d+|/cropx/\d+|/cropy/\d+|/format/(png|jpg|jpeg)|/height/\d+|/quality/\d+|/width/\d+|/force/[0|1]){0,9}$`
	matched, _ := regexp.Match(pattern, []byte(cmd))
	if !matched {
		err = errors.New("invalid html2image command format")
		return
	}

	options = &Html2ImageOptions{
		Format: "jpg",
	}

	//croph
	cropHStr := utils.GetParam(cmd, `croph/\d+`, "croph")
	if cropHStr != "" {
		cropH, _ := strconv.Atoi(cropHStr)
		if cropH <= 0 {
			err = errors.New("invalid html2image parameter 'croph'")
			return
		} else {
			options.CropH = cropH
		}
	}

	//cropw
	cropWStr := utils.GetParam(cmd, `cropw/\d+`, "cropw")
	if cropWStr != "" {
		cropW, _ := strconv.Atoi(cropWStr)
		if cropW <= 0 {
			err = errors.New("invalid html2image parameter 'cropw'")
			return
		} else {
			options.CropW = cropW
		}
	}

	//cropx
	cropXStr := utils.GetParam(cmd, `cropx/\d+`, "cropx")
	fmt.Println(cropXStr)
	if cropXStr != "" {
		cropX, _ := strconv.Atoi(cropXStr)
		if cropX <= 0 {
			err = errors.New("invalid html2image parameter 'cropx'")
			return
		} else {
			options.CropX = cropX
		}
	}

	//cropy
	cropYStr := utils.GetParam(cmd, `cropy/\d+`, "cropy")
	if cropYStr != "" {
		cropY, _ := strconv.Atoi(cropYStr)
		if cropY <= 0 {
			err = errors.New("invalid html2image parameter 'cropy'")
			return
		} else {
			options.CropY = cropY
		}
	}

	//format
	formatStr := utils.GetParam(cmd, "format/(png|jpg|jpeg)", "format")
	if formatStr != "" {
		options.Format = formatStr
	}

	//height
	heightStr := utils.GetParam(cmd, `height/\d+`, "height")
	if heightStr != "" {
		height, _ := strconv.Atoi(heightStr)
		if height <= 0 {
			err = errors.New("invalid html2image parameter 'height'")
			return
		} else {
			options.Height = height
		}
	}

	//width
	widthStr := utils.GetParam(cmd, `width/\d+`, "width")
	if widthStr != "" {
		width, _ := strconv.Atoi(widthStr)
		if width <= 0 {
			err = errors.New("invalid html2image parameter 'width'")
			return
		} else {
			options.Width = width
		}
	}

	//quality
	qualityStr := utils.GetParam(cmd, `quality/\d+`, "quality")
	if qualityStr != "" {
		quality, _ := strconv.Atoi(qualityStr)
		if quality > 100 || quality <= 0 {
			err = errors.New("invalid html2image parameter 'quality'")
			return
		} else {
			options.Quality = quality
		}
	}

	//force
	forceStr := utils.GetParam(cmd, "force/[0|1]", "force")
	if forceStr != "" {
		force, _ := strconv.Atoi(forceStr)
		if force == 1 {
			options.Force = true
		}
	}

	return

}

func (this *Html2Imager) Do(req ufop.UfopRequest) (result interface{}, contentType string, err error) {
	//if not text format, error it
	if !strings.HasPrefix(req.Src.MimeType, "text/") {
		err = errors.New("unsupported file mime type, only text/* allowed")
		return
	}

	//if file size exceeds, error it
	if req.Src.Fsize > this.maxPageSize {
		err = errors.New("page file length exceeds the limit")
		return
	}

	options, pErr := this.parse(req.Cmd)
	if pErr != nil {
		err = pErr
		return
	}

	//get page file content save it into temp dir
	resp, respErr := http.Get(req.Src.Url)
	if respErr != nil || resp.StatusCode != 200 {
		if respErr != nil {
			err = errors.New(fmt.Sprintf("retrieve page file resource data failed, %s", respErr.Error()))
		} else {
			err = errors.New(fmt.Sprintf("retrieve page file resource data failed, %s", resp.Status))
			if resp.Body != nil {
				resp.Body.Close()
			}
		}
		return
	}

	jobPrefix := utils.Md5Hex(req.Src.Url)

	pageSuffix := "txt"
	if strings.HasPrefix(req.Src.MimeType, "text/html") {
		pageSuffix = "html"
	}

	localPageTmpFname := fmt.Sprintf("%s%d.page.%s", jobPrefix, time.Now().UnixNano(), pageSuffix)
	localPageTmpFpath := filepath.Join(os.TempDir(), localPageTmpFname)
	defer os.Remove(localPageTmpFpath)

	localPageTmpFp, openErr := os.OpenFile(localPageTmpFpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0655)
	if openErr != nil {
		err = errors.New(fmt.Sprintf("open page file temp file failed, %s", openErr.Error()))
		return
	}
	_, cpErr := io.Copy(localPageTmpFp, resp.Body)
	if cpErr != nil {
		err = errors.New(fmt.Sprintf("save page file content to tmp file failed, %s", cpErr.Error()))
		return
	}

	localPageTmpFp.Close()
	resp.Body.Close()

	//prepare command
	cmdParams := make([]string, 0)
	cmdParams = append(cmdParams, "-q")

	if options.CropH > 0 {
		cmdParams = append(cmdParams, "--crop-h", fmt.Sprintf("%d", options.CropH))
	}

	if options.CropW > 0 {
		cmdParams = append(cmdParams, "--crop-w", fmt.Sprintf("%d", options.CropW))
	}

	if options.CropX > 0 {
		cmdParams = append(cmdParams, "--crop-x", fmt.Sprintf("%d", options.CropX))
	}

	if options.CropY > 0 {
		cmdParams = append(cmdParams, "--crop-y", fmt.Sprintf("%d", options.CropY))
	}

	if options.Format != "" {
		cmdParams = append(cmdParams, "--format", options.Format)
	}

	if options.Quality > 0 {
		cmdParams = append(cmdParams, "--quality", fmt.Sprintf("%d", options.Quality))
	}

	if options.Height > 0 {
		cmdParams = append(cmdParams, "--height", fmt.Sprintf("%d", options.Height))
	}

	if options.Width > 0 {
		cmdParams = append(cmdParams, "--width", fmt.Sprintf("%d", options.Width))
	}

	if options.Force {
		cmdParams = append(cmdParams, "--disable-smart-width")
	}

	//result tmp file
	resultTmpFname := fmt.Sprintf("%s%d.result.%s", jobPrefix, time.Now().UnixNano(), options.Format)
	resultTmpFpath := filepath.Join(os.TempDir(), resultTmpFname)
	defer os.Remove(resultTmpFpath)

	cmdParams = append(cmdParams, localPageTmpFpath, resultTmpFpath)

	//cmd
	convertCmd := exec.Command("wkhtmltoimage", cmdParams...)
	log.Println(convertCmd.Path, convertCmd.Args)

	stdErrPipe, pipeErr := convertCmd.StderrPipe()
	if pipeErr != nil {
		err = errors.New(fmt.Sprintf("open exec stderr pipe error, %s", pipeErr.Error()))
		return
	}

	if startErr := convertCmd.Start(); startErr != nil {
		err = errors.New(fmt.Sprintf("start html2image command error, %s", startErr.Error()))
		return
	}

	stdErrData, readErr := ioutil.ReadAll(stdErrPipe)
	if readErr != nil {
		err = errors.New(fmt.Sprintf("read html2image command stderr error, %s", readErr.Error()))
		return
	}

	//check stderr output & output file
	if string(stdErrData) != "" {
		log.Println(string(stdErrData))
	}

	if waitErr := convertCmd.Wait(); waitErr != nil {
		err = errors.New(fmt.Sprintf("wait html2image to exit error, %s", waitErr.Error()))
		return
	}

	if _, statErr := os.Stat(resultTmpFpath); statErr == nil {
		oTmpFp, openErr := os.Open(resultTmpFpath)
		if openErr != nil {
			err = errors.New(fmt.Sprintf("open html2image output result error, %s", openErr.Error()))
			return
		}
		defer oTmpFp.Close()

		outputBytes, readErr := ioutil.ReadAll(oTmpFp)
		if readErr != nil {
			err = errors.New(fmt.Sprintf("read html2image output result error, %s", readErr.Error()))
			return
		}
		result = outputBytes
	} else {
		err = errors.New("html2image with no valid output result")
	}

	if options.Format == "png" {
		contentType = "image/png"
	} else {
		contentType = "image/jpeg"
	}

	return
}
