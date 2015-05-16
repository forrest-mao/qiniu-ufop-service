package ufop

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

//default ufop config
var defaultUfopConfig UfopConfig = UfopConfig{
	ListenPort:     9100,
	ListenHost:     "0.0.0.0",
	ReadTimeout:    1800,
	WriteTimeout:   1800,
	MaxHeaderBytes: 1 << 16,
}

type UfopConfig struct {
	ListenPort int    `json:"listen_port,omitempty"`
	ListenHost string `json:"listen_host,omitempty"`

	ReadTimeout  int `json:"read_timeout,omitempty"`
	WriteTimeout int `json:"write_timeout,omitempty"`

	MaxHeaderBytes int `json:"max_header_bytes,omitempty"`

	UfopPrefix string `json:"ufop_prefix"`
	AccessKey  string `json:"access_key"`
	SecretKey  string `json:"secret_key"`

	UnzipMaxZipFileLength int64 `json:"unzip_max_zip_file_length,omitempty"`
	UnzipMaxFileLength    int64 `json:"unzip_max_file_length,omitempty"`
	UnzipMaxFileCount     int   `json:"unzip_max_file_count,omitempty"`

	MkzipMaxFileLength int64 `json:"mkzip_max_file_length,omitempty"`
	MkzipMaxFileCount  int   `json:"mkzip_max_file_count,omitempty"`

	AmergeMaxFirstFileLength  int64 `json:"amerge_max_first_file_length,omitempty"`
	AmergeMaxSecondFileLength int64 `json:"amerge_max_second_file_length,omitempty"`
}

func (this *UfopConfig) LoadFromFile(configFilePath string) (err error) {
	confFp, openErr := os.Open(configFilePath)
	if openErr != nil {
		err = errors.New(fmt.Sprintf("Open ufop config failed, %s", openErr))
		return
	}
	defer confFp.Close()

	decoder := json.NewDecoder(confFp)
	decodeErr := decoder.Decode(this)
	if decodeErr != nil {
		err = errors.New(fmt.Sprintf("Parse ufop config failed, %s", decodeErr))
	}
	if this.ListenPort <= 0 {
		this.ListenPort = defaultUfopConfig.ListenPort
	}
	if this.ReadTimeout <= 0 {
		this.ReadTimeout = defaultUfopConfig.ReadTimeout
	}
	if this.WriteTimeout <= 0 {
		this.WriteTimeout = defaultUfopConfig.WriteTimeout
	}
	return
}
