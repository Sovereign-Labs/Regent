package rpc

import (
	"fmt"
	"strings"
)

// https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#Errors
const (
	CODE_INTERNAL_ERROR             = -32603
	CODE_SERVER_ERROR               = -32000
	CODE_INVALID_PAYLOAD_ATTRIBUTES = -38003
	CODE_INVALID_FORKCHOICE_STATE   = -38002
)

const (
	ERR_TOKEN_STRING_RETRIEVAL_FAILED = "could not retrieve the engine JWT"
	ERR_MARSHALLING_FAILED            = "marshalling failed. This indicates a consensus client bug"
	ERR_REQUEST_CREATION_FAILED       = "HTTP Request creation failed. This indicates a client bug"
	ERR_REQUEST_SEND_FAILED           = "an error was encountered while sending the http request"
	ERR_RESPONSE_READ_FAILED          = "an error was encountered while sending the http request"
	ERR_UNMARSHALLING_FAILED          = "unmarshalling failed"
)

type MaybeRetryable interface {
	IsRetryable() bool
}

// The error from an Ethereum Json-rpc response
type JsonRpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *JsonRpcError) Error() string {
	return fmt.Sprintf("jsonrpc error, code: %v. msg: %v.", e.Code, e.Message)
}

// We only consider errors on the other client's side to be retryable
func (e *JsonRpcError) IsRetryable() bool {
	return e.Code == CODE_INTERNAL_ERROR || e.Code == CODE_SERVER_ERROR
}

// Indicates whether an error might be fixed by a retry.
// For now, we consider all out-of-protocol errors to be retryable
func IsRetryable(e error) bool {
	if rpcErr, ok := e.(MaybeRetryable); ok {
		return rpcErr.IsRetryable()
	}
	return false
}

type NonProtocolRpcError struct {
	inner error
	msg   string
}

func ErrFrom(msg string, err error) *NonProtocolRpcError {
	return &NonProtocolRpcError{
		inner: err,
		msg:   msg,
	}
}

func (e *NonProtocolRpcError) Error() string {
	return fmt.Sprintf("%s. %s. ", e.msg, e.inner)
}

// Indicates whether an error might be fixed by a retry.
// For now, we consider all out-of-protocol errors to be retryable, as well as in-protocol errors of the types listed below.
func (e *NonProtocolRpcError) IsRetryable() bool {
	return e.msg == ERR_REQUEST_SEND_FAILED || e.msg == ERR_RESPONSE_READ_FAILED
}

func (e *NonProtocolRpcError) Unwrap() error {
	return e.inner
}

func (e *NonProtocolRpcError) Is(err error) bool {
	return strings.HasPrefix(err.Error(), e.msg)
}
