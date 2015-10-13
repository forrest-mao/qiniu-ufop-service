package ufop

type UfopRequest struct {
	Cmd string         `json:"cmd"`
	Src UfopRequestSrc `json:"src"`
}

type UfopRequestSrc struct {
	Url      string `json:"url"`
	MimeType string `json:"mimetype"`
	Fsize    uint64 `json:"fsize"`
}

type UfopError struct {
	Request UfopRequest
	Error   string
}

type UfopJobHandler interface {
	Name() string
	InitConfig(jobConf string) error
	Do(ufopReq UfopRequest) (interface{}, string, error)
}
