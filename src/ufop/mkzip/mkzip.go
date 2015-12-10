package mkzip

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qiniu/api.v6/auth/digest"
	"github.com/qiniu/api.v6/rs"
	"github.com/qiniu/rpc"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"ufop"
	"ufop/utils"
)

/*

mkzip/bucket/<encoded bucket>/encoding/<encoded encoding[gbk|utf8]>
/url/<encoded url>/alias/<encoded alias>/url/<encoded url>/alias/<encoded alias>

*/
const (
	MKZIP_MAX_FILE_LENGTH int64 = 100 * 1024 * 1024 //100MB
	MKZIP_MAX_FILE_COUNT  int   = 100               //100
	MKZIP_MAX_FILE_LIMIT  int   = 1000              //1000
)

type Mkzipper struct {
	mac           *digest.Mac
	maxFileLength int64
	maxFileCount  int
}

type MkzipperConfig struct {
	//ak & sk
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`

	MkzipMaxFileLength int64 `json:"mkzip_max_file_length,omitempty"`
	MkzipMaxFileCount  int   `json:"mkzip_max_file_count,omitempty"`
}

type ZipFile struct {
	url   string
	key   string
	alias string
}

func (this *Mkzipper) Name() string {
	return "mkzip"
}

func (this *Mkzipper) InitConfig(jobConf string) (err error) {
	confFp, openErr := os.Open(jobConf)
	if openErr != nil {
		err = errors.New(fmt.Sprintf("Open mkzip config failed, %s", openErr.Error()))
		return
	}

	config := MkzipperConfig{}
	decoder := json.NewDecoder(confFp)
	decodeErr := decoder.Decode(&config)
	if decodeErr != nil {
		err = errors.New(fmt.Sprintf("Parse mkzip config failed, %s", decodeErr.Error()))
		return
	}

	if config.MkzipMaxFileCount <= 0 {
		this.maxFileCount = MKZIP_MAX_FILE_COUNT
	} else {
		this.maxFileCount = config.MkzipMaxFileCount
	}

	if config.MkzipMaxFileLength <= 0 {
		this.maxFileLength = MKZIP_MAX_FILE_LENGTH
	} else {
		this.maxFileLength = config.MkzipMaxFileLength
	}

	this.mac = &digest.Mac{config.AccessKey, []byte(config.SecretKey)}

	return
}

func (this *Mkzipper) parse(cmd string) (bucket string, encoding string, zipFiles []ZipFile, err error) {
	pattern := "^mkzip/bucket/[0-9a-zA-Z-_=]+(/encoding/[0-9a-zA-Z-_=]+){0,1}(/url/[0-9a-zA-Z-_=]+(/alias/[0-9a-zA-Z-_=]+){0,1})+$"
	matched, _ := regexp.MatchString(pattern, cmd)
	if !matched {
		err = errors.New("invalid mkzip command format")
		return
	}

	var decodeErr error
	//get bucket
	bucket, decodeErr = utils.GetParamDecoded(cmd, "bucket/[0-9a-zA-Z-_=]+", "bucket")
	if decodeErr != nil {
		err = errors.New("invalid mkzip paramter 'bucket'")
		return
	}
	//get encoding
	encoding, decodeErr = utils.GetParamDecoded(cmd, "encoding/[0-9a-zA-Z-_=]+", "encoding")
	if decodeErr != nil {
		err = errors.New("invalid mkzip parameter 'encoding'")
		return
	}
	//get url & alias
	urlAliasRegx := regexp.MustCompile("url/[0-9a-zA-Z-_=]+(/alias/[0-9a-zA-Z-_=]+){0,1}")
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
			err = errors.New("mkzip parameter 'url' format error")
			return
		}

		//parse key
		path := uri.Path
		ldx := strings.Index(path, "/")
		if ldx != -1 {
			key = path[ldx+1:]
			if palias == "" {
				palias = key
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

func (this *Mkzipper) Do(req ufop.UfopRequest) (result interface{}, resultType int, contentType string, err error) {
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
	statUrls := make([]string, 0)
	for _, zipFile := range zipFiles {
		entryPath := rs.EntryPath{
			bucket, zipFile.key,
		}
		statItems = append(statItems, entryPath)
		statUrls = append(statUrls, zipFile.url)
	}
	qclient := rs.New(this.mac)

	statRet, statErr := qclient.BatchStat(nil, statItems)

	if statErr != nil {
		if _, ok := statErr.(*rpc.ErrorInfo); !ok {
			err = errors.New(fmt.Sprintf("batch stat error, %s", statErr.Error()))
			return
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

	//retrieve resource and create zip file
	var tErr error
	zipBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuffer)

	for _, zipFile := range zipFiles {
		//convert encoding
		fname := zipFile.alias
		if encoding == "gbk" {
			fname, tErr = utils.Utf82Gbk(fname)
			if tErr != nil {
				err = errors.New(fmt.Sprintf("unsupported encoding gbk, %s", tErr))
				return
			}
		}

		//create each zip file writer
		fw, fErr := zipWriter.Create(fname)
		if fErr != nil {
			err = errors.New(fmt.Sprintf("create zip file error, %s", fErr))
			return
		}
		//read data and write
		resResp, respErr := http.Get(zipFile.url)
		if respErr != nil || resResp.StatusCode != 200 {
			if respErr != nil {
				err = errors.New("get zip file resource error, " + respErr.Error())
			} else {
				err = errors.New(fmt.Sprintf("get zip file resource error, %s", resResp.Status))
				if resResp.Body != nil {
					resResp.Body.Close()
				}
			}
			return
		}
		respData, readErr := ioutil.ReadAll(resResp.Body)
		if readErr != nil {
			err = errors.New(fmt.Sprintf("read zip file resource content error, %s", readErr))
			return
		}
		resResp.Body.Close()

		_, writeErr := fw.Write(respData)
		if writeErr != nil {
			err = errors.New(fmt.Sprintf("write zip file content error, %s", writeErr))
			return
		}
	}
	//close zip file
	if cErr := zipWriter.Close(); cErr != nil {
		err = errors.New(fmt.Sprintf("close zip file error, %s", cErr))
		return
	}

	//write result
	result = zipBuffer.Bytes()
	resultType = ufop.RESULT_TYPE_OCTECT_BYTES
	contentType = "application/zip"
	return
}
