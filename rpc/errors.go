package rpc

import (
	"fmt"
)

// https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#Errors
const (
	CODE_INTERNAL_ERROR = -32603
	CODE_SERVER_ERROR   = -32000
)

// An error which retains contextual information about its cause
type RpcError struct {
	InProtocolError *JsonRpcError
	Exception       error
}

// Indicates whether an error might be fixed by a retry.
// For now, we consider all out-of-protocol errors to be retryable, as well as in-protocol errors of the types listed below.
func (e *RpcError) IsRetryable() bool {
	return e.InProtocolError == nil || e.InProtocolError.Code == CODE_INTERNAL_ERROR || e.InProtocolError.Code == CODE_SERVER_ERROR
}

func (e *RpcError) Error() string {
	if e.Exception != nil {
		return e.Exception.Error()
	} else {
		return fmt.Sprintf("%v", e.InProtocolError)
	}
}

func InProtocolError(err *JsonRpcError) *RpcError {
	return &RpcError{
		InProtocolError: err,
	}
}

func Exception(ex error) *RpcError {
	return &RpcError{
		Exception: ex,
	}
}
