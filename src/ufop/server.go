package ufop

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qiniu/api/auth/digest"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	ufopPrefix  string
	jobHandlers map[string]UfopJobHandler
	unzipper    *UnZipper
	mkzipper    *Mkziper
)

type UfopJobHandler interface {
	Do(ufopReq UfopRequest) (interface{}, string, error)
}

type UfopServer struct {
	cfg *UfopConfig
}

func NewServer(cfg *UfopConfig) *UfopServer {
	serv := UfopServer{}
	serv.cfg = cfg
	return &serv
}

func (this *UfopServer) registerJobHandlers() {
	mac := digest.Mac{
		AccessKey: this.cfg.AccessKey,
		SecretKey: []byte(this.cfg.SecretKey),
	}
	jobHandlers = make(map[string]UfopJobHandler, 0)
	ufopPrefix = this.cfg.UfopPrefix
	//unzipper
	unzipper = &UnZipper{
		mac:              &mac,
		maxZipFileLength: this.cfg.UnzipMaxZipFileLength,
		maxFileLength:    this.cfg.UnzipMaxFileLength,
		maxFileCount:     this.cfg.UnzipMaxFileCount,
	}
	mkzipper = &Mkziper{
		mac:           &mac,
		maxFileLength: this.cfg.MkzipMaxFileLength,
		maxFileCount:  this.cfg.MkzipMaxFileCount,
	}

	jobHandlers[ufopPrefix+"unzip"] = unzipper
	jobHandlers[ufopPrefix+"mkzip"] = mkzipper
}

func (this *UfopServer) Listen() {
	//register
	this.registerJobHandlers()

	//define handler
	http.HandleFunc("/uop", serveUfop)

	//bind and listen
	endPoint := fmt.Sprintf("%s:%d", this.cfg.ListenHost, this.cfg.ListenPort)
	ufopServer := &http.Server{
		Addr:           endPoint,
		ReadTimeout:    time.Duration(this.cfg.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(this.cfg.WriteTimeout) * time.Second,
		MaxHeaderBytes: this.cfg.MaxHeaderBytes,
	}

	listenErr := ufopServer.ListenAndServe()
	if listenErr != nil {
		log.Println(listenErr)
	}
}

func serveUfop(w http.ResponseWriter, req *http.Request) {
	//check method
	if req.Method != "POST" {
		writeJsonError(w, 405, "method not allowed")
		return
	}

	defer req.Body.Close()
	var err error
	var ufopReq UfopRequest
	var ufopResult interface{}
	var ufopResultContentType string

	ufopReqData, err := ioutil.ReadAll(req.Body)
	if err != nil {
		writeJsonError(w, 500, "read ufop request body error")
		return
	}

	err = json.Unmarshal(ufopReqData, &ufopReq)
	if err != nil {
		writeJsonError(w, 500, "parse ufop request body error")
		return
	}

	ufopResult, ufopResultContentType, err = handleJob(ufopReq)
	if err != nil {
		writeJsonError(w, 400, err.Error())
	} else {
		switch ufopResultContentType {
		case "application/json":
			writeJsonResult(w, 200, ufopResult)
		default:
			writeOctetResult(w, 200, ufopResult)
		}
	}
}

func handleJob(ufopReq UfopRequest) (interface{}, string, error) {
	var ufopResult interface{}
	var contentType string
	var err error
	cmd := ufopReq.Cmd

	items := strings.SplitN(cmd, "/", 2)
	fop := items[0]
	if jobHandler, ok := jobHandlers[fop]; ok {
		ufopReq.Cmd = strings.TrimPrefix(ufopReq.Cmd, ufopPrefix)
		ufopResult, contentType, err = jobHandler.Do(ufopReq)
	} else {
		err = errors.New("no fop available for the request")
	}
	return ufopResult, contentType, err
}

func writeJsonError(w http.ResponseWriter, statusCode int, message string) {
	log.Println(message)
	w.WriteHeader(statusCode)
	if w.Header().Get("Content-Type") != "" {
		w.Header().Set("Content-Type", "application/json")
	} else {
		w.Header().Add("Content-Type", "application/json")
	}
	io.WriteString(w, fmt.Sprintf(`{"error": "%s"}`, message))
}

func writeJsonResult(w http.ResponseWriter, statusCode int, result interface{}) {
	w.WriteHeader(statusCode)
	w.Header().Add("Content-Type", "application/json")
	data, err := json.Marshal(result)
	if err != nil {
		log.Println("encode ufop result error,", err)
		writeJsonError(w, 500, "encode ufop result error")
	} else {
		_, err := io.WriteString(w, string(data))
		if err != nil {
			log.Println("write json response error", err)
			writeJsonError(w, 500, "write json response error")
		}
	}
}

func writeOctetResult(w http.ResponseWriter, statusCode int, result interface{}) {
	w.WriteHeader(statusCode)
	w.Header().Add("Content-Type", "application/octet-stream")
	if respData := result.([]byte); respData != nil {
		_, err := w.Write(respData)
		if err != nil {
			log.Println("write octect response error", err)
			writeJsonError(w, 500, "write octect response error")
		}
	}
}
