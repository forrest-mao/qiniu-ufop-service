package ufop

type UfopRequest struct {
	Cmd string         `json:"cmd"`
	Src UfopRequestSrc `json:"src"`
}

type UfopRequestSrc struct {
	Url      string `json:"url"`
	MimeType string `json:"mimetype"`
	Fsize    int64  `json:"fsize"`
}
