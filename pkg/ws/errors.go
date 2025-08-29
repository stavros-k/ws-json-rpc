package ws

const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
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
		return &jsonRPCError{Code: handlerErr.Code(), Message: handlerErr.Error()}
	}

	return &jsonRPCError{Code: codeInternalError, Message: err.Error()}
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
