package rpc

import "fmt"

// https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#Errors
const (
	CODE_INTERNAL_ERROR = -32603
	CODE_SERVER_ERROR   = -32000
)

// The error from an Ethereum Json-rpc response
type JsonRpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *JsonRpcError) Error() string {
	return fmt.Sprintf("jsonrpc error, code: %v. msg: %v.", e.Code, e.Message)
}

// Indicates whether an error might be fixed by a retry.
// For now, we consider all out-of-protocol errors to be retryable, as well as in-protocol errors of the types listed below.
func (e *JsonRpcError) IsRetryable() bool {
	return e.Code == CODE_INTERNAL_ERROR || e.Code == CODE_SERVER_ERROR
}

func IsRetryable(e error) bool {
	if rpcErr, ok := e.(*JsonRpcError); ok {
		return rpcErr.IsRetryable()
	}
	return true
}
