package models

type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"message"`
	Data any    `json:"data,omitempty"`
}

func NewResponse(code int, msg string, data any) *Response {
	return &Response{
		Code: code,
		Msg:  msg,
		Data: data,
	}
}
