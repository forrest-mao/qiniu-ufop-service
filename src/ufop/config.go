package ufop

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

//default ufop config
var DefaultUfopConfig UfopConfig = UfopConfig{
	ListenPort:     9011,
	ListenHost:     "0.0.0.0",
	ReadTimeout:    30,
	WriteTimeout:   30,
	MaxHeaderBytes: 1 << 16,
}

type UfopConfig struct {
	ListenPort int    `json:"listen_port,omitempty"`
	ListenHost string `json:"listen_host,omitempty"`

	ReadTimeout  int64 `json:"read_timeout,omitempty"`
	WriteTimeout int64 `json:"write_timeout,omitempty"`

	MaxHeaderBytes int `json:"max_header_bytes,omitempty"`

	UfopPrefix string `json:"ufop_prefix"`
	AccessKey  string `json:"access_key"`
	SecretKey  string `json:"secret_key"`
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
	return
}
