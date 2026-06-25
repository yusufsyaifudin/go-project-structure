package httpservermw

import "net/url"

type AccessLog struct {
	Method      string            `json:"method,omitempty"`
	Host        string            `json:"host,omitempty"`
	Path        string            `json:"path,omitempty"`
	StatusCode  int               `json:"statusCode,omitempty"`
	Header      map[string]string `json:"header,omitempty"`
	Body        any               `json:"body,omitempty"`
	BodyLen     int64             `json:"bodyLen,omitempty"`
	QueryParams url.Values        `json:"queryParams,omitempty"`
	Error       string            `json:"error,omitempty"`
	ElapsedTime int64             `json:"elapsedTime,omitempty"`
}
