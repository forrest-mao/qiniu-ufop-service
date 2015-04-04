package ufop

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"errors"
	"github.com/qiniu/api/auth/digest"
	"github.com/qiniu/api/rs"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

//mkzip/bucket/xxx/encoding/[gbk|utf8]/url/alias/url/alias

const (
	MKZIP_MAX_FILE_LENGTH int64 = 100 * 1024 * 1024 //100MB
	MKZIP_MAX_FILE_COUNT  int   = 100               //100
	MKZIP_MAX_FILE_LIMIT  int   = 1000              //1000
)

type Mkziper struct {
	mac           *digest.Mac
	maxFileLength int64
	maxFileCount  int
}

type ZipFile struct {
	url   string
	key   string
	alias string
}

func (this *Mkziper) parse(cmd string) (bucket string, encoding string, zipFiles []ZipFile, err error) {
	pattern := "^mkzip/bucket/[0-9a-zA-Z-_=]+(/encoding/[0-9a-zA-Z-_=]+){0,1}(/url/[0-9a-zA-Z-_=]+(/alias/[0-9a-zA-Z-_=]+){0,1})+$"
	matched, _ := regexp.Match(pattern, []byte(cmd))
	if !matched {
		err = errors.New("invalid mkzip command format")
		return
	}
	//get bucket
	bucketRegx := regexp.MustCompile("bucket/[0-9a-zA-Z-_=]+")
	bucketPairItems := strings.Split(bucketRegx.FindString(cmd), "/")
	bucketBytes, decodeErr := base64.URLEncoding.DecodeString(bucketPairItems[1])
	if decodeErr != nil {
		err = errors.New("invalid mkzip paramter 'bucket'")
		return
	}
	bucket = string(bucketBytes)
	//get encoding
	encodingRegx := regexp.MustCompile("encoding/[0-9a-zA-Z-_=]+")
	encodingPair := encodingRegx.FindString(cmd)
	if encodingPair != "" {
		encodingPairItems := strings.Split(encodingPair, "/")
		encodingBytes, decodeErr := base64.URLEncoding.DecodeString(encodingPairItems[1])
		if decodeErr != nil {
			err = errors.New("invalid mkzip parameter 'encoding'")
			return
		}
		encoding = string(encodingBytes)
	}
	//get url & alias
	urlAliasRegx := regexp.MustCompile("(url/[0-9a-zA-Z-_=]+(/alias/[0-9a-zA-Z-_=]+){0,1})")
	urlAliasPairs := urlAliasRegx.FindAllString(cmd, -1)
	paliasMap := make(map[string]string, 0)
	for _, urlAliasPair := range urlAliasPairs {
		urlAliasItems := strings.Split(urlAliasPair, "/")
		zipFile := ZipFile{}
		var purl string
		var palias string
		var key string
		switch len(urlAliasItems) {
		case 2:
			urlBytes, decodeErr := base64.URLEncoding.DecodeString(urlAliasItems[1])
			if decodeErr != nil {
				err = errors.New("invalid mkzip parameter 'url'")
				return
			}
			purl = string(urlBytes)
		case 4:
			urlBytes, decodeErr := base64.URLEncoding.DecodeString(urlAliasItems[1])
			if decodeErr != nil {
				err = errors.New("invalid mkzip parameter 'url'")
				return
			}
			aliasBytes, decodeErr := base64.URLEncoding.DecodeString(urlAliasItems[3])
			if decodeErr != nil {
				err = errors.New("invalid mkzip parameter 'alias'")
				return
			}
			purl = string(urlBytes)
			palias = string(aliasBytes)
		}
		uri, parseErr := url.Parse(purl)
		if parseErr != nil {
			err = errors.New("invalid mkzip parameter 'url'")
			return
		}
		if palias == "" {
			path := uri.Path
			ldx := strings.Index(path, "/")
			if ldx != -1 {
				palias = path[ldx+1:]
				key = palias
			}
		}
		if key == "" {
			err = errors.New("invalid mkzip resource url")
			return
		}
		if _, ok := paliasMap[palias]; ok {
			err = errors.New("duplicate mkzip resource alias")
			return
		}
		paliasMap[palias] = palias

		//set zip file
		zipFile.alias = palias
		zipFile.url = purl
		zipFile.key = key
		zipFiles = append(zipFiles, zipFile)
	}
	return
}

func (this *Mkziper) Do(req UfopRequest) (result interface{}, contentType string, err error) {
	contentType = "application/json"
	//set mkzip check criteria
	if this.maxFileCount <= 0 {
		this.maxFileCount = MKZIP_MAX_FILE_COUNT
	}
	if this.maxFileLength <= 0 {
		this.maxFileLength = MKZIP_MAX_FILE_LENGTH
	}
	//parse command
	bucket, encoding, zipFiles, pErr := this.parse(req.Cmd)
	if pErr != nil {
		err = pErr
		return
	}

	//check file count
	if len(zipFiles) > this.maxFileCount {
		err = errors.New("zip file count exceeds the limit")
		return
	}
	if len(zipFiles) > MKZIP_MAX_FILE_LIMIT {
		err = errors.New("only support items less than 1000")
		return
	}
	//check whether file in bucket and exceeds the limit
	statItems := make([]rs.EntryPath, 0)
	for _, zipFile := range zipFiles {
		entryPath := rs.EntryPath{
			bucket, zipFile.key,
		}
		statItems = append(statItems, entryPath)
	}
	qclient := rs.New(this.mac)

	statRet, statErr := qclient.BatchStat(nil, statItems)
	if statErr != nil {
		err = errors.New("batch stat error")
		return
	}
	for _, ret := range statRet {
		if ret.Error != "" {
			err = errors.New("stat resource in bucket error")
			return
		}
		if ret.Data.Fsize > this.maxFileLength {
			err = errors.New("stat resource length exceeds the limit")
			return
		}
	}
	//retrieve resource and create zip file
	var tErr error
	zipBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuffer)

	for _, zipFile := range zipFiles {
		//convert encoding
		fname := zipFile.alias
		if encoding == "gbk" {
			fname, tErr = utf82GBK(fname)
			if tErr != nil {
				err = errors.New("unsupported encoding gbk")
				return
			}
		}

		//create each zip file writer
		fw, fErr := zipWriter.Create(fname)
		if fErr != nil {
			err = errors.New("create zip file error")
			return
		}
		//read data and write
		resp, respErr := http.Get(zipFile.url)
		if respErr != nil {
			err = errors.New("get zip file resource error")
			return
		}
		respData, readErr := ioutil.ReadAll(resp.Body)
		if readErr != nil {
			err = errors.New("read zip file resource content error")
			return
		}
		resp.Body.Close()

		_, writeErr := fw.Write(respData)
		if writeErr != nil {
			err = errors.New("write zip file content error")
			return
		}
	}
	//close zip file
	if cErr := zipWriter.Close(); cErr != nil {
		err = errors.New("close zip file error")
		return
	}
	result = zipBuffer.Bytes()
	contentType = "application/octect-stream"
	return
}
