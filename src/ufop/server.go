package ufop

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qiniu/log"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type UfopServer struct {
	cfg         *UfopConfig
	jobHandlers map[string]UfopJobHandler
}

func NewServer(cfg *UfopConfig) *UfopServer {
	serv := UfopServer{}
	serv.cfg = cfg
	serv.jobHandlers = make(map[string]UfopJobHandler, 0)
	return &serv
}

func (this *UfopServer) RegisterJobHandler(jobConf string, jobHandler interface{}) (err error) {
	if h, ok := jobHandler.(UfopJobHandler); ok {
		initErr := h.InitConfig(jobConf)
		if initErr != nil {
			err = errors.New(fmt.Sprintf("init job handler for cmd '%s' error, %s", h.Name(), initErr.Error()))
			return
		}

		this.jobHandlers[this.cfg.UfopPrefix+h.Name()] = h
	} else {
		err = errors.New("job handler must implement interface UfopJobHandler")
	}
	return
}

func (this *UfopServer) Listen() {
	//define handler
	http.HandleFunc("/uop", this.serveUfop)

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

func (this *UfopServer) serveUfop(w http.ResponseWriter, req *http.Request) {
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
	log.Info(string(ufopReqData))
	err = json.Unmarshal(ufopReqData, &ufopReq)
	if err != nil {
		writeJsonError(w, 500, "parse ufop request body error")
		return
	}

	ufopResult, ufopResultContentType, err = handleJob(ufopReq, this.cfg.UfopPrefix, this.jobHandlers)
	if err != nil {
		ufopErr := UfopError{
			Request: ufopReq,
			Error:   err.Error(),
		}
		logBytes, _ := json.Marshal(&ufopErr)
		log.Error(string(logBytes))
		writeJsonError(w, 400, err.Error())
	} else {
		switch ufopResultContentType {
		case "application/json":
			writeJsonResult(w, 200, ufopResult)
		default:
			writeOctetResultWithMime(w, 200, ufopResult, ufopResultContentType)
		}
	}
}

func handleJob(ufopReq UfopRequest, ufopPrefix string, jobHandlers map[string]UfopJobHandler) (interface{}, string, error) {
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	io.WriteString(w, fmt.Sprintf(`{"error":"%s"}`, message))
}

func writeJsonResult(w http.ResponseWriter, statusCode int, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	data, err := json.Marshal(result)
	if err != nil {
		log.Error("encode ufop result error,", err)
		writeJsonError(w, 500, "encode ufop result error")
	} else {
		_, err := io.WriteString(w, string(data))
		if err != nil {
			log.Error("write json response error", err)
		}
	}
}

func writeOctetResultWithMime(w http.ResponseWriter, statusCode int, result interface{}, mimeType string) {
	if mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	}
	if respData := result.([]byte); respData != nil {
		_, err := w.Write(respData)
		if err != nil {
			log.Error("write octect response error", err)
		}
	}
}
