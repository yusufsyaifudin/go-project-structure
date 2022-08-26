package respbuilder

import (
	"fmt"
	"strings"
)

type RespStatus struct {
	Code   string
	Status string
}

type RespCode int

const (
	Success RespCode = iota
	ErrGeneral
)

var respMap = map[RespCode]RespStatus{
	Success: {Code: "0", Status: "OK"},

	// error must use prefix E for the convention
	ErrGeneral: {Code: "E0", Status: "GeneralError"},
}

// RespStructure to ensure that json marshalled version will not sort the keys
type RespStructure struct {
	Code    string      `json:"code"`
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func Ok(respCode RespCode, data interface{}, customMsg ...string) RespStructure {
	r := respMap[respCode]

	out := RespStructure{
		Code:    r.Code,
		Status:  r.Status,
		Message: strings.Join(customMsg, "; "),
		Data:    data,
	}

	return out
}

func Error(err error) RespStructure {
	r := respMap[ErrGeneral]

	if err == nil {
		err = fmt.Errorf("programmatically error, caller call this but with nil error")
	}

	out := RespStructure{
		Code:    r.Code,
		Status:  r.Status,
		Message: err.Error(),
	}

	return out
}
