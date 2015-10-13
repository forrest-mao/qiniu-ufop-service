package amerge

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qiniu/api.v6/auth/digest"
	"github.com/qiniu/api.v6/rs"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"ufop"
	"ufop/utils"
)

const (
	AUDIO_MERGE_MAX_FIRST_FILE_LENGTH  = 100 * 1024 * 1024
	AUDIO_MERGE_MAX_SECOND_FILE_LENGTH = 100 * 1024 * 1024
)

type AudioMerger struct {
	mac                 *digest.Mac
	maxFirstFileLength  uint64
	maxSecondFileLength uint64
}

type AudioMergerConfig struct {
	//ak & sk
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`

	AmergeMaxFirstFileLength  uint64 `json:"amerge_max_first_file_length,omitempty"`
	AmergeMaxSecondFileLength uint64 `json:"amerge_max_second_file_length,omitempty"`
}

func (this *AudioMerger) Name() string {
	return "amerge"
}

func (this *AudioMerger) InitConfig(jobConf string) (err error) {
	confFp, openErr := os.Open(jobConf)
	if openErr != nil {
		err = errors.New(fmt.Sprintf("Open amerge config failed, %s", openErr.Error()))
		return
	}

	config := AudioMergerConfig{}
	decoder := json.NewDecoder(confFp)
	decodeErr := decoder.Decode(&config)
	if decodeErr != nil {
		err = errors.New(fmt.Sprintf("Parse amerge config failed, %s", decodeErr.Error()))
		return
	}

	if config.AmergeMaxFirstFileLength <= 0 {
		this.maxFirstFileLength = AUDIO_MERGE_MAX_FIRST_FILE_LENGTH
	} else {
		this.maxFirstFileLength = config.AmergeMaxFirstFileLength
	}

	if config.AmergeMaxSecondFileLength <= 0 {
		this.maxSecondFileLength = AUDIO_MERGE_MAX_SECOND_FILE_LENGTH
	} else {
		this.maxSecondFileLength = config.AmergeMaxSecondFileLength
	}

	this.mac = &digest.Mac{config.AccessKey, []byte(config.SecretKey)}

	return
}

/*

amerge
/format/<string>
/mime/<encoded mime>
/bucket/<encoded bucket>
/url/<encoded url>
/duration/<[first|shortest|longest]>

*/

func (this *AudioMerger) parse(cmd string) (format string, mime string, bucket string, url string, duration string, err error) {
	pattern := "^amerge/format/[a-zA-Z0-9]+/mime/[0-9a-zA-Z-_=]+/bucket/[0-9a-zA-Z-_=]+/url/[0-9a-zA-Z-_=]+(/duration/(first|shortest|longest)){0,1}$"
	matched, _ := regexp.Match(pattern, []byte(cmd))
	if !matched {
		err = errors.New("invalid amerge command format")
		return
	}

	var decodeErr error
	format = utils.GetParam(cmd, "format/[a-zA-Z0-9]+", "format")
	mime, decodeErr = utils.GetParamDecoded(cmd, "mime/[0-9a-zA-Z-_=]+", "mime")
	if decodeErr != nil {
		err = errors.New("invalid amerge parameter 'mime'")
		return
	}
	bucket, decodeErr = utils.GetParamDecoded(cmd, "bucket/[0-9a-zA-Z-_=]+", "bucket")
	if decodeErr != nil {
		err = errors.New("invalid amerge parameter 'bucket'")
		return
	}
	url, decodeErr = utils.GetParamDecoded(cmd, "url/[0-9a-zA-Z-_=]+", "url")
	if decodeErr != nil {
		err = errors.New("invalid amerge parameter 'url'")
		return
	}
	duration = utils.GetParam(cmd, "duration/(first|shortest|longest)", "duration")
	if duration == "" {
		duration = "longest"
	}
	return
}

func (this *AudioMerger) Do(req ufop.UfopRequest) (result interface{}, contentType string, err error) {
	//check first file
	if req.Src.Fsize > this.maxFirstFileLength {
		err = errors.New("first file length exceeds the limit")
		return
	}
	if !strings.HasPrefix(req.Src.MimeType, "audio/") {
		err = errors.New("first file mimetype not supported")
		return
	}
	//parse command
	dstFormat, dstMime, secondFileBucket, secondFileUrl, dstDuration, pErr := this.parse(req.Cmd)
	if pErr != nil {
		err = pErr
		return
	}

	secondFileUri, pErr := url.Parse(secondFileUrl)
	if pErr != nil {
		err = errors.New("second file resource url not valid")
		return
	}
	secondFileKey := strings.TrimPrefix(secondFileUri.Path, "/")
	client := rs.New(this.mac)
	sEntry, sErr := client.Stat(nil, secondFileBucket, secondFileKey)
	if sErr != nil || sEntry.Hash == "" {
		err = errors.New("second file not in the specified bucket")
		return
	}
	//check second file
	if uint64(sEntry.Fsize) > this.maxSecondFileLength {
		err = errors.New("second file length exceeds the limit")
		return
	}
	if !strings.HasPrefix(sEntry.MimeType, "audio/") {
		err = errors.New("second file mimetype not supported")
		return
	}
	//download first and second file
	fResp, fRespErr := http.Get(req.Src.Url)
	if fRespErr != nil || fResp.StatusCode != 200 {
		if fRespErr != nil {
			err = errors.New(fmt.Sprintf("retrieve first file resource data failed, %s", fRespErr.Error()))
		} else {
			err = errors.New(fmt.Sprintf("retrieve first file resource data failed, %s", fResp.Status))
			if fResp.Body != nil {
				fResp.Body.Close()
			}
		}
		return
	}

	fTmpFp, fErr := ioutil.TempFile("", "first")
	if fErr != nil {
		err = errors.New(fmt.Sprintf("open first file temp file failed, %s", fErr.Error()))
		return
	}
	_, fCpErr := io.Copy(fTmpFp, fResp.Body)
	if fCpErr != nil {
		err = errors.New(fmt.Sprintf("save first temp file failed, %s", fCpErr.Error()))
		return
	}
	//close first one
	fTmpFname := fTmpFp.Name()
	fTmpFp.Close()
	fResp.Body.Close()

	sResp, sRespErr := http.Get(secondFileUrl)
	if sRespErr != nil || sResp.StatusCode != 200 {
		if sRespErr != nil {
			err = errors.New(fmt.Sprintf("retrieve second file resource data failed, %s", sRespErr.Error()))
		} else {
			err = errors.New(fmt.Sprintf("retrieve second file resource data failed, %s", sResp.Status))
			if sResp.Body != nil {
				sResp.Body.Close()
			}
		}
		return
	}
	sTmpFp, sErr := ioutil.TempFile("", "second")
	if sErr != nil {
		err = errors.New(fmt.Sprintf("open second file temp file failed, %s", sErr.Error()))
		return
	}
	_, sCpErr := io.Copy(sTmpFp, sResp.Body)
	if sCpErr != nil {
		err = errors.New(fmt.Sprintf("save second first tmp file failed, %s", sCpErr.Error()))
		return
	}
	//close second one
	sTmpFname := sTmpFp.Name()
	sTmpFp.Close()
	sResp.Body.Close()

	//do conversion
	oTmpFp, oErr := ioutil.TempFile("", "output")
	if oErr != nil {
		err = errors.New(fmt.Sprintf("open output file temp file failed, %s", oErr.Error()))
		return
	}
	oTmpFname := oTmpFp.Name()
	oTmpFp.Close()
	//be sure to delete temp files
	defer os.Remove(fTmpFname)
	defer os.Remove(sTmpFname)
	defer os.Remove(oTmpFname)
	//prepare command
	mergeCmdParams := []string{
		"-y",
		"-v", "error",
		"-i", fTmpFname,
		"-i", sTmpFname,
		"-filter_complex", fmt.Sprintf("amix=inputs=2:duration=%s:dropout_transition=2", dstDuration),
		"-f", dstFormat,
		oTmpFname,
	}

	//exec command
	mergeCmd := exec.Command("ffmpeg", mergeCmdParams...)

	stdErrPipe, pipeErr := mergeCmd.StderrPipe()
	if pipeErr != nil {
		err = errors.New(fmt.Sprintf("open exec stderr pipe error, %s", pipeErr.Error()))
		return
	}
	if startErr := mergeCmd.Start(); startErr != nil {
		err = errors.New(fmt.Sprintf("start ffmpeg command error, %s", startErr.Error()))
		return
	}

	stdErrData, readErr := ioutil.ReadAll(stdErrPipe)
	if readErr != nil {
		err = errors.New(fmt.Sprintf("read ffmpeg command stderr error, %s", readErr.Error()))
		return
	}

	//check stderr output & output file
	if string(stdErrData) != "" {
		log.Println(string(stdErrData))
	}

	if waitErr := mergeCmd.Wait(); waitErr != nil {
		err = errors.New(fmt.Sprintf("wait ffmpeg to exit error, %s", waitErr))
		return
	}

	if oFileInfo, statErr := os.Stat(oTmpFname); statErr == nil {
		if oFileInfo.Size() > 0 {
			oTmpFp, openErr := os.Open(oTmpFname)
			if openErr != nil {
				err = errors.New(fmt.Sprintf("open ffmpeg output result error, %s", openErr.Error()))
				return
			}
			defer oTmpFp.Close()
			outputBytes, readErr := ioutil.ReadAll(oTmpFp)
			if readErr != nil {
				err = errors.New(fmt.Sprintf("read ffmpeg output result error, %s", readErr.Error()))
				return
			}
			result = outputBytes
		} else {
			err = errors.New("audio merge with no valid output result")
		}
	}
	contentType = dstMime
	return
}
