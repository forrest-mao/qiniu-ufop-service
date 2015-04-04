package ufop

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"errors"
	"github.com/qiniu/api/auth/digest"
	fio "github.com/qiniu/api/io"
	rio "github.com/qiniu/api/resumable/io"
	"github.com/qiniu/api/rs"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	UNZIP_MAX_ZIP_FILE_LENGTH int64 = 1 * 1024 * 1024 * 1024
	UNZIP_MAX_FILE_LENGTH     int64 = 100 * 1024 * 1024 //100MB
	UNZIP_MAX_FILE_COUNT      int   = 10                //10
)

type UnZipResult struct {
	Files []UnZipFile `json:"files"`
}

type UnZipFile struct {
	Key   string `json:"key"`
	Hash  string `json:"hash,omitempty"`
	Error string `json:"error,omitempty"`
}

type UnZipper struct {
	mac              *digest.Mac
	maxZipFileLength int64
	maxFileLength    int64
	maxFileCount     int
}

func (this *UnZipper) parse(cmd string) (bucket string, overwrite bool, err error) {
	pattern := "^unzip/bucket/[0-9a-zA-Z-_=]+(/overwrite/[0|1]){0,1}$"
	matched, _ := regexp.Match(pattern, []byte(cmd))
	if !matched {
		err = errors.New("invalid unzip command format")
		return
	}
	items := strings.Split(cmd, "/")

	if len(items) >= 3 {
		bucketBytes, paramErr := base64.URLEncoding.DecodeString(items[2])
		if paramErr != nil {
			err = errors.New("invalid unzip parameter 'bucket'")
			return
		}
		bucket = string(bucketBytes)
	}
	if len(items) == 5 {
		overwriteVal, paramErr := strconv.ParseInt(items[4], 10, 64)
		if paramErr != nil {
			err = errors.New("invalid unzip parameter 'overwrite'")
			return
		}
		if overwriteVal == 1 {
			overwrite = true
		}
	}
	return
}

func (this *UnZipper) Do(req UfopRequest) (result interface{}, contentType string, err error) {
	contentType = "application/json"
	//set zip file check criteria
	if this.maxFileCount <= 0 {
		this.maxFileCount = UNZIP_MAX_FILE_COUNT
	}
	if this.maxFileLength <= 0 {
		this.maxFileLength = UNZIP_MAX_FILE_LENGTH
	}
	if this.maxZipFileLength <= 0 {
		this.maxZipFileLength = UNZIP_MAX_ZIP_FILE_LENGTH
	}
	//check mimetype
	if req.Src.MimeType != "application/zip" {
		err = errors.New("unsupported mimetype to unzip")
		return
	}
	//check zip file length
	if req.Src.Fsize > this.maxZipFileLength {
		err = errors.New("src zip file length exceeds the limit")
		return
	}

	//parse command
	bucket, overwrite, pErr := this.parse(req.Cmd)
	if pErr != nil {
		err = pErr
		return
	}

	//get resource
	resUrl := req.Src.Url
	resResp, respErr := http.Get(resUrl)
	if respErr != nil {
		err = errors.New("retrieve resource data failed")
		return
	}
	defer resResp.Body.Close()

	respData, respErr := ioutil.ReadAll(resResp.Body)
	if respErr != nil {
		err = errors.New("read resource data failed")
		return
	}

	//read zip
	respReader := bytes.NewReader(respData)
	zipReader, zipErr := zip.NewReader(respReader, int64(respReader.Len()))
	if zipErr != nil {
		err = errors.New("invalid zip file")
		return
	}
	zipFiles := zipReader.File
	//check file count
	zipFileCount := len(zipFiles)
	if zipFileCount > this.maxFileCount {
		err = errors.New("zip files count exceeds the limit")
		return
	}
	//check file size
	for _, zipFile := range zipFiles {
		fileInfo := zipFile.FileHeader.FileInfo()
		fileSize := fileInfo.Size()
		//check file size
		if fileSize > this.maxFileLength {
			err = errors.New("zip file length exceeds the limit")
			return
		}
	}

	//parse zip
	rputSettings := rio.Settings{
		ChunkSize: 4 * 1024 * 1024,
		Workers:   1,
	}
	rio.SetSettings(&rputSettings)
	var rputThreshold int64 = 100 * 1024 * 1024
	policy := rs.PutPolicy{
		Scope: bucket,
	}
	var unzipResult UnZipResult
	unzipResult.Files = make([]UnZipFile, 0)
	var tErr error
	//iterate the zip file
	for _, zipFile := range zipFiles {
		fileInfo := zipFile.FileHeader.FileInfo()
		fileName := zipFile.FileHeader.Name
		fileSize := fileInfo.Size()

		if !utf8.Valid([]byte(fileName)) {
			fileName, tErr = gbk2Utf8(fileName)
			if tErr != nil {
				err = errors.New("unsupported file name encoding")
				return
			}
		}

		if fileInfo.IsDir() {
			continue
		}

		zipFileReader, zipErr := zipFile.Open()
		if zipErr != nil {
			err = errors.New("open zip file content failed")
			return
		}
		defer zipFileReader.Close()

		unzipData, unzipErr := ioutil.ReadAll(zipFileReader)
		if unzipErr != nil {
			err = errors.New("unzip the file content failed")
			return
		}
		unzipReader := bytes.NewReader(unzipData)

		//save file to bucket
		if overwrite {
			policy.Scope = bucket + ":" + fileName
		}
		uptoken := policy.Token(this.mac)
		var unzipFile UnZipFile
		unzipFile.Key = fileName
		if fileSize <= rputThreshold {
			var fputRet fio.PutRet
			fErr := fio.Put(nil, &fputRet, uptoken, fileName, unzipReader, nil)
			if fErr != nil {
				unzipFile.Error = "save unzip file to bucket error"
			} else {
				unzipFile.Hash = fputRet.Hash
			}

		} else {
			var rputRet rio.PutRet
			rErr := rio.Put(nil, &rputRet, uptoken, fileName, unzipReader, fileSize, nil)
			if rErr != nil {
				unzipFile.Error = "save unzip file to bucket error"
			} else {
				unzipFile.Hash = rputRet.Hash
			}
		}
		unzipResult.Files = append(unzipResult.Files, unzipFile)
	}
	result = unzipResult
	return
}
