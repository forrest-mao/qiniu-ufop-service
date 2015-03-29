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
)

type UfopJobHandler interface {
	Do(ufopReq UfopRequest) (interface{}, error)
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
		mac: &mac,
	}

	jobHandlers[ufopPrefix+"unzip"] = unzipper
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

	ufopResult, err = handleJob(ufopReq)
	if err != nil {
		writeJsonError(w, 500, err.Error())
	} else {
		writeJsonResult(w, 200, ufopResult)
	}
}

func handleJob(ufopReq UfopRequest) (interface{}, error) {
	var ufopResult interface{}
	var err error
	cmd := ufopReq.Cmd

	items := strings.SplitN(cmd, "/", 2)
	fop := items[0]
	if jobHandler, ok := jobHandlers[fop]; ok {
		ufopReq.Cmd = strings.TrimPrefix(ufopReq.Cmd, ufopPrefix)
		ufopResult, err = jobHandler.Do(ufopReq)
	} else {
		err = errors.New("no fop available for the request")
	}
	return ufopResult, err
}

func writeJsonError(w http.ResponseWriter, statusCode int, message string) {
	log.Println(message)
	w.WriteHeader(statusCode)
	w.Header().Add("Content-Type", "application/json")
	io.WriteString(w, fmt.Sprintf(`{"error": "%s"}`, message))
}

func writeJsonResult(w http.ResponseWriter, statusCode int, result interface{}) {
	w.WriteHeader(statusCode)
	w.Header().Add("Content-Type", "application/json")
	data, err := json.Marshal(result)
	if err != nil {
		log.Println("Encode ufop result error,", err)
		writeJsonResult(w, 500, "Encode ufop result error")
	} else {
		io.WriteString(w, string(data))
	}
}
