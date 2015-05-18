package ufop

import (
	"errors"
	"fmt"
	"github.com/qiniu/api/auth/digest"
	"github.com/qiniu/api/rs"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	AUDIO_MERGE_MAX_FIRST_FILE_LENGTH  = 100 * 1024 * 1024
	AUDIO_MERGE_MAX_SECOND_FILE_LENGTH = 100 * 1024 * 1024
)

type AudioMerger struct {
	mac                 *digest.Mac
	maxFirstFileLength  int64
	maxSecondFileLength int64
}

/*

amerge/format/<format>/mime/<encoded mime>/bucket/<encoded bucket>/url/<encoded url>/duration/<[first|shortest|longest]>

*/

func (this *AudioMerger) parse(cmd string) (format string, mime string, bucket string, url string, duration string, err error) {
	pattern := "^amerge/format/[a-zA-Z0-9]+/mime/[0-9a-zA-Z-_=]+/bucket/[0-9a-zA-Z-_=]+/url/[0-9a-zA-Z-_=]+(/duration/(first|shortest|longest)){0,1}$"
	matched, _ := regexp.Match(pattern, []byte(cmd))
	if !matched {
		err = errors.New("invalid amerge command format")
		return
	}

	var decodeErr error
	format = getParam(cmd, "format/[a-zA-Z0-9]+", "format")
	mime, decodeErr = getParamDecoded(cmd, "mime/[0-9a-zA-Z-_=]+", "mime")
	if decodeErr != nil {
		err = errors.New("invalid amerge parameter 'mime'")
		return
	}
	bucket, decodeErr = getParamDecoded(cmd, "bucket/[0-9a-zA-Z-_=]+", "bucket")
	if decodeErr != nil {
		err = errors.New("invalid amerge parameter 'bucket'")
		return
	}
	url, decodeErr = getParamDecoded(cmd, "url/[0-9a-zA-Z-_=]+", "url")
	if decodeErr != nil {
		err = errors.New("invalid amerge parameter 'url'")
		return
	}
	duration = getParam(cmd, "duration/(first|shortest|longest)", "duration")
	if duration == "" {
		duration = "longest"
	}
	return
}

func (this *AudioMerger) Do(req UfopRequest) (result interface{}, contentType string, err error) {
	contentType = "application/json"
	//check first file &second file length criteria
	if this.maxFirstFileLength <= 0 {
		this.maxFirstFileLength = AUDIO_MERGE_MAX_FIRST_FILE_LENGTH
	}
	if this.maxSecondFileLength <= 0 {
		this.maxSecondFileLength = AUDIO_MERGE_MAX_SECOND_FILE_LENGTH
	}
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
	if sEntry.Fsize > this.maxSecondFileLength {
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
		if fResp.Body != nil {
			fResp.Body.Close()
		}
		err = errors.New("retrieve first file resource data failed, " + fRespErr.Error())
		return
	}
	fTmpFp, fErr := ioutil.TempFile("", "first")
	if fErr != nil {
		err = errors.New("open first file temp file failed, " + fErr.Error())
		return
	}
	_, fCpErr := io.Copy(fTmpFp, fResp.Body)
	if fCpErr != nil {
		err = errors.New("save first temp file failed, " + fCpErr.Error())
		return
	}
	//close first one
	fTmpFname := fTmpFp.Name()
	fTmpFp.Close()
	fResp.Body.Close()

	sResp, sRespErr := http.Get(secondFileUrl)
	if sRespErr != nil || sResp.StatusCode != 200 {
		if sResp.Body != nil {
			sResp.Body.Close()
		}
		err = errors.New("retrieve second file resource data failed, " + sRespErr.Error())
		return
	}
	sTmpFp, sErr := ioutil.TempFile("", "second")
	if sErr != nil {
		err = errors.New("open second file temp file failed, " + sErr.Error())
		return
	}
	_, sCpErr := io.Copy(sTmpFp, sResp.Body)
	if sCpErr != nil {
		err = errors.New("save second first tmp file failed, " + sCpErr.Error())
		return
	}
	//close second one
	sTmpFname := sTmpFp.Name()
	sTmpFp.Close()
	sResp.Body.Close()

	//do conversion
	oTmpFp, oErr := ioutil.TempFile("", "output")
	if oErr != nil {
		err = errors.New("open output file temp file failed, " + oErr.Error())
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
		err = errors.New("open exec stderr pipe error, " + pipeErr.Error())
		return
	}
	if startErr := mergeCmd.Start(); startErr != nil {
		err = errors.New("start ffmpeg command error, " + startErr.Error())
		return
	}
	stdErrData, readErr := ioutil.ReadAll(stdErrPipe)
	if readErr != nil {
		err = errors.New("read ffmpeg command stderr error, " + readErr.Error())
		return
	}
	if waitErr := mergeCmd.Wait(); waitErr != nil {
		err = errors.New("wait ffmpeg to exit error")
		return
	}
	//check stderr output & output file
	if string(stdErrData) != "" {
		log.Println(string(stdErrData))
	}
	if oFileInfo, statErr := os.Stat(oTmpFname); statErr == nil {
		if oFileInfo.Size() > 0 {
			oTmpFp, openErr := os.Open(oTmpFname)
			if openErr != nil {
				err = errors.New("open ffmpeg output result error, " + openErr.Error())
				return
			}
			defer oTmpFp.Close()
			outputBytes, readErr := ioutil.ReadAll(oTmpFp)
			if readErr != nil {
				err = errors.New("read ffmpeg output result error, " + readErr.Error())
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
