package json

// Code mirrors one.rewind.nio.web.serialization.Msg for API envelope compatibility.

type Code int

const (
	CodeSuccess Code = 1
	CodeFailure Code = 0

	CodeInvalidParameters Code = 213
	CodeInsertFailure     Code = 221
	CodeUpdateFailure     Code = 222
	CodeDeleteFailure     Code = 223
	CodeBadRequest        Code = 400
	CodeTokenInvalid      Code = 401
	CodeAccessAnonymous   Code = 402
	CodeAccessDenied      Code = 403
	CodeNotFound          Code = 404
	CodeMethodRejected    Code = 405
	CodeTooManyReq        Code = 406
	CodeObjectExists      Code = 409
	CodeServerError       Code = 500
)

var codeMessages = map[Code]string{
	CodeSuccess:           "SUCCESS",
	CodeFailure:           "FAILURE",
	CodeInvalidParameters: "INVALID_PARAMETERS",
	CodeInsertFailure:     "INSERT_FAILURE",
	CodeUpdateFailure:     "UPDATE_FAILURE",
	CodeDeleteFailure:     "DELETE_FAILURE",
	CodeBadRequest:        "BAD_REQUEST",
	CodeTokenInvalid:      "TOKEN_INVALID",
	CodeAccessAnonymous:   "ACCESS_ANONYMOUS",
	CodeAccessDenied:      "ACCESS_DENIED",
	CodeNotFound:          "NOT_FOUND",
	CodeMethodRejected:    "METHOD_REJECTED",
	CodeTooManyReq:        "TOO_MANGY_REQ",
	CodeObjectExists:      "OBJECT_EXITS",
	CodeServerError:       "SERVER_ERROR",
}

type Msg struct {
	Code int                    `json:"code"`
	Msg  string                 `json:"msg"`
	Data any                    `json:"data,omitempty"`
	Meta map[string]int64       `json:"_meta,omitempty"`
}

func NewMsg(code Code, data any, meta map[string]int64) Msg {
	msg, ok := codeMessages[code]
	if !ok {
		msg = "FAILURE"
	}
	return Msg{
		Code: int(code),
		Msg:  msg,
		Data: data,
		Meta: meta,
	}
}

func Success(data any) Msg {
	return NewMsg(CodeSuccess, data, nil)
}

func SuccessList(data any, total int64) Msg {
	return NewMsg(CodeSuccess, data, map[string]int64{"total": total})
}

func SuccessPage(data any, page, size, total int64) Msg {
	totalPage := int64(0)
	if page > 0 && size > 0 && total > 0 {
		totalPage = total / size
		if total%size > 0 {
			totalPage++
		}
	}
	return NewMsg(CodeSuccess, data, map[string]int64{
		"page":       page,
		"size":       size,
		"total":      total,
		"total_page": totalPage,
	})
}

func Failure() Msg {
	return NewMsg(CodeFailure, nil, nil)
}

func FailureString(message string) Msg {
	if message == "" {
		message = "FAILURE"
	}
	return NewMsg(CodeFailure, message, nil)
}

func FailureErr(err error) Msg {
	if err == nil {
		return Failure()
	}
	return NewMsg(CodeFailure, err.Error(), nil)
}
