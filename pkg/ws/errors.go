package ws

import "fmt"

const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603

	MinCustomErrorCode = -32099
	MaxCustomErrorCode = -32000
)

type HandlerError interface {
	error
	Code() int
}

func toRPCError(err error) *jsonRPCError {
	if err == nil {
		return nil
	}

	if rpcErr, ok := err.(*jsonRPCError); ok {
		return rpcErr
	}

	if handlerErr, ok := err.(HandlerError); ok {
		code := handlerErr.Code()
		message := handlerErr.Error()
		if code < MinCustomErrorCode || code > MaxCustomErrorCode {
			code = CodeInternalError
			message = fmt.Errorf("invalid error code: %d :%w", handlerErr.Code(), handlerErr).Error()
		}
		return &jsonRPCError{Code: code, Message: message}
	}

	return &jsonRPCError{Code: CodeInternalError, Message: err.Error()}
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// newJSONRPCError creates a JSON-RPC error
func newJSONRPCError(code int, message string) *jsonRPCError {
	return &jsonRPCError{Code: code, Message: message}
}

func (e *jsonRPCError) Error() string {
	return e.Message
}
