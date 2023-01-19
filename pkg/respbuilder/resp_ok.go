package respbuilder

import "strings"

type RespCodeOk int

const (
	UnknownOk RespCodeOk = iota
	Success
)

var respMapOk = map[RespCodeOk]RespStatus{
	UnknownOk: {Code: "_", Status: "NotOk"},
	Success:   {Code: "0", Status: "Ok"},
}

// RespStructureOk to ensure that json marshalled version will not sort the keys
type RespStructureOk struct {
	Code    string      `json:"code"`
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func Ok(respCode RespCodeOk, data interface{}, customMsg ...string) RespStructureOk {
	r, exist := respMapOk[respCode]
	if !exist {
		r = respMapOk[UnknownOk]
	}

	out := RespStructureOk{
		Code:    r.Code,
		Status:  r.Status,
		Message: strings.Join(customMsg, "; "),
		Data:    data,
	}

	return out
}
