package respbuilder

import (
	"errors"
	"fmt"

	"github.com/labstack/echo/v4"
)

type RespCodeErr int

const (
	ErrUnknown RespCodeErr = iota
	ErrGeneral
)

// respMapErr must use prefix E to indicate the error
var respMapErr = map[RespCodeErr]RespStructureErr{
	ErrUnknown: {Code: "E", Status: "ErrUnknown"},
	ErrGeneral: {Code: "E0", Status: "ErrorGeneral"},
}

// RespCodeErrStatus get RespStructureErr based on response code.
// If code not found, then ErrUnknown will be used.
func RespCodeErrStatus(code RespCodeErr) RespStructureErr {
	r, exist := respMapErr[code]
	if !exist {
		r = respMapErr[ErrUnknown]
	}

	return r
}

// GetAllRespCodeErr return all available response code for error
func GetAllRespCodeErr() []RespCodeErr {
	codes := make([]RespCodeErr, 0)
	for code := range respMapErr {
		codes = append(codes, code)
	}

	return codes
}

type RespError struct {
	Message string   `json:"message,omitempty"`
	Reasons []string `json:"reasons,omitempty"`
}

// RespStructureErr to ensure that json marshalled version will not sort the keys
type RespStructureErr struct {
	Code   string     `json:"code"`
	Status string     `json:"status"`
	Error  *RespError `json:"error,omitempty"`
}

// Error return RespStructureErr as contract when response is not success.
// So, every error response will always have consistent data structure.
func Error(respCode RespCodeErr, err error, reasons ...string) RespStructureErr {
	r := RespCodeErrStatus(respCode)

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

	r.Error = &RespError{
		Message: msg,
		Reasons: reasons,
	}

	return r
}
