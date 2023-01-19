package respbuilder

import (
	"errors"
	"fmt"

	"github.com/labstack/echo/v4"
)

type RespCodeErr int

const (
	ErrGeneral RespCodeErr = iota
)

// respMapErr must use prefix E to indicate the error
var respMapErr = map[RespCodeErr]RespStatus{
	ErrGeneral: {Code: "E0", Status: "ErrorGeneral"},
}

// RespStructureErr to ensure that json marshalled version will not sort the keys
type RespStructureErr struct {
	Code   string `json:"code"`
	Status string `json:"status"`
	Error  struct {
		Message string   `json:"message,omitempty"`
		Reasons []string `json:"reasons,omitempty"`
	} `json:"error,omitempty"`
}

func Error(respCode RespCodeErr, err error, reasons ...string) RespStructureErr {
	r, exist := respMapErr[respCode]
	if !exist {
		r = respMapErr[ErrGeneral]
	}

	if err == nil {
		err = fmt.Errorf("programmatically error, caller call this but with nil error")
	}

	msg := err.Error()
	internalReasons := make([]string, 0)
	var errHTTP *echo.HTTPError
	if errors.As(err, &errHTTP) {
		msg = fmt.Sprintf("%+v", errHTTP.Message)
		internalReasons = append(internalReasons, errHTTP.Error())
	}

	var errBinding *echo.BindingError
	if errors.As(err, &errBinding) {
		msg = fmt.Sprintf("%+v", errBinding.Message)
		internalReasons = append(internalReasons, errBinding.Error())
	}

	reasons = append(internalReasons, reasons...)

	out := RespStructureErr{
		Code:   r.Code,
		Status: r.Status,
		Error: struct {
			Message string   `json:"message,omitempty"`
			Reasons []string `json:"reasons,omitempty"`
		}{
			Message: msg,
			Reasons: reasons,
		},
	}

	return out
}
