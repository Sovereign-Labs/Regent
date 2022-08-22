package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/log/v3"
)

const VERSION_STRING string = "Regent/0.0.0"
const GET_BLOCK_BY_NUMBER RpcMethod = "eth_getBlockByNumber"
const FORK_CHOICE_UPDATED RpcMethod = "engine_forkchoiceUpdatedV1"
const NEW_EXECUTION_PAYLOAD RpcMethod = "engine_newPayloadV1"
const GET_EXECUTION_PAYLOAD RpcMethod = "engine_getPayloadV1"

type PayloadStatusString string
type RetryStrategy interface {
	Next() time.Duration
	Done() bool
}

// Retry forever with linear backoff, capped at 30 seconds between attempts
type InfiniteRetryStrategy struct {
	// The current attempt number, which is also the number of seconds to wait until the next attempt
	attempt time.Duration
}

func (s *InfiniteRetryStrategy) Next() time.Duration {
	if s.attempt >= 30*time.Second {
		return 30 * time.Second
	}
	s.attempt += time.Second
	return s.attempt - time.Second
}

func (s *InfiniteRetryStrategy) Done() bool {
	return false
}

// Retry 5 times with linear backoff
type SimpleRetryStrategy struct {
	// The current attempt number, which is also the number of seconds to wait until the next attempt
	attempt time.Duration
}

func (s *SimpleRetryStrategy) Next() time.Duration {
	s.attempt += time.Second
	return s.attempt - time.Second
}

func (s *SimpleRetryStrategy) Done() bool {
	return s.attempt > 5
}

const (
	VALID_PAYLOAD   PayloadStatusString = "VALID"
	INVALID_PAYLOAD PayloadStatusString = "INVALID"
	SYNCING_PAYLOAD PayloadStatusString = "SYNCING"
)

type PayloadStatus struct {
	Status          PayloadStatusString `json:"status"`
	LatestValidHash *common.Hash        `json:"latestValidHash"`
	ValidationError *string             `json:"validationError"`
}

type ForkChoiceUpdatedResult struct {
	PayloadStatus *PayloadStatus `json:"payloadStatus"`
	PayloadId     string         `json:"payloadId"`
}

type RpcMethod string

// An Ethereum Json-rpc message
type Request struct {
	JsonRPC string        `json:"jsonrpc"`
	Method  RpcMethod     `json:"method"`
	Params  []interface{} `json:"params"`
	Id      uint          `json:"id"`
}

// An Ethereum Json-rpc response
type Response[R comparable] struct {
	JsonRPC string        `json:"jsonrpc"`
	Result  R             `json:"result"`
	Id      uint          `json:"id"`
	Error   *JsonRpcError `json:"error"`
}

// Creates a Json-rpc message with the supplied method and parameters
func NewRequest(method RpcMethod, params ...interface{}) *Request {
	if params == nil {
		params = make([]interface{}, 0)
	}
	return &Request{
		JsonRPC: "2.0",
		Method:  method,
		Params:  params,
		Id:      1,
	}
}

// Gets a response of type R by using `client` to send the provided `request` with the given timeout and retry strategy
// This is a function rather than a method of client to workaround this limitation of Go's generics:
// https://go.googlesource.com/proposal/+/refs/heads/master/design/43651-type-parameters.md#No-parameterized-methods
func getResponse[R comparable](client *Client, request *Request, timeout time.Duration, retries RetryStrategy) (R, error) {
	var err error
	for !retries.Done() {
		time.Sleep(retries.Next())
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		var ret R
		ret, err = sendRequest[R](ctx, client, request)
		if err != nil {
			log.Warn("Error sending msg to execution client", "err", err)
			if IsRetryable(err) {
				continue
			}
		}
		return ret, err
	}
	return *new(R), err
}

// Sends a JSON-RPC method whose response is unmartialed into a Response with Result type R.
// This allows the caller to specify the type for the response.Result at compile time, rather than
// relying on runtime reflection to identify it
func sendRequest[R comparable](ctx context.Context, client *Client, msg *Request) (R, error) {
	marshalled, err := json.Marshal(msg)
	if err != nil {
		err = ErrFrom(ERR_MARSHALLING_FAILED, err)
		log.Crit(err.Error())
		return *new(R), err
	}
	fmt.Println(string(marshalled))
	req, err := http.NewRequest("POST", client.Endpoint, bytes.NewBuffer(marshalled))
	if err != nil {
		err = ErrFrom(ERR_REQUEST_CREATION_FAILED, err)
		log.Crit(err.Error())
		return *new(R), err
	}

	req.Header["Content-Type"] = []string{"application/json"}
	if client.authToken != nil {
		tokenString, err := client.authToken.TokenString()
		if err != nil {
			return *new(R), ErrFrom(ERR_JWT_REFRESH_FAILED, err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", tokenString))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return *new(R), ErrFrom(ERR_REQUEST_SEND_FAILED, err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return *new(R), ErrFrom(ERR_RESPONSE_READ_FAILED, fmt.Errorf("Error reading response to msg %v. %w", msg, err))
	}
	response := Response[R]{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return *new(R), ErrFrom(ERR_UNMARSHALLING_FAILED, fmt.Errorf("Error unmarshalling response to msg %v. response body %v. err %w", msg, string(body), err))
	}
	if response.Error != nil {
		return response.Result, response.Error
	}
	if response.Result == *new(R) {
		return response.Result, ErrFrom(ERR_UNMARSHALLING_FAILED, fmt.Errorf("The response to msg %v did not contain a value of type %T. response body %v. err %w", msg, response.Result, string(body), err))
	}
	return response.Result, nil
}
