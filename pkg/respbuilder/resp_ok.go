package respbuilder

import "strings"

type RespCodeOk int

const (
	UnknownOk RespCodeOk = iota
	Success
)

var respMapOk = map[RespCodeOk]RespStructureOk{
	UnknownOk: {Code: "_", Status: "NotOk"},
	Success:   {Code: "0", Status: "Ok"},
}

// RespCodeOkStatus get RespStructureOk based on response code.
// If code not found, then UnknownOk will be used.
func RespCodeOkStatus(code RespCodeOk) RespStructureOk {
	r, exist := respMapOk[code]
	if !exist {
		r = respMapOk[UnknownOk]
	}

	return r
}

// GetAllRespCodeOk return all available response code for success response.
func GetAllRespCodeOk() []RespCodeOk {
	codes := make([]RespCodeOk, 0)
	for code := range respMapOk {
		codes = append(codes, code)
	}

	return codes
}

// RespStructureOk to ensure that json marshalled version will not sort the keys
type RespStructureOk struct {
	Code    string      `json:"code"`
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// Ok return RespStructureOk as contract when response is either fully success or partially success.
// So, every response will always have consistent data structure.
func Ok(respCode RespCodeOk, data interface{}, customMsg ...string) RespStructureOk {
	r := RespCodeOkStatus(respCode)
	r.Message = strings.Join(customMsg, "; ")
	r.Data = data
	return r
}
