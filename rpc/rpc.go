package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regent/rpc/jwt"
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

type InfiniteRetryStrategy struct {
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

// // The standard retry strategy performs 5 retries with linear backoff.
// // The first retry waits 1 second, the 2nd waits two seconds, etc.
// type StandardRetryStrategy struct {
// 	attempt int
// }

// func (s StandardRetryStrategy) Next() time.Duration {
// 	s.attempt += 1
// 	return time.Duration(s.attempt-1) * time.Second
// }

// func (s StandardRetryStrategy) Done() bool {
// 	return s.attempt >= 5
// }

// type NoRetriesStrategy struct{}

// func (s NoRetriesStrategy) Done() bool {
// 	return true
// }
// func (s NoRetriesStrategy) Next() int {
// 	log.Crit("Bug! called 'retry' when the strategy was 'no retries'")
// 	panic("Bug detected...")
// }

const (
	VALID_PAYLOAD   PayloadStatusString = "VALID"
	INVALID_PAYLOAD PayloadStatusString = "INVALID"
	SYNCING_PAYLOAD PayloadStatusString = "SYNCING"
)

type PayloadStatus struct {
	Status          PayloadStatusString `json:"status"`
	LatestValidHash common.Hash         `json:"latestValidHash"`
}

type ForkChoiceUpdatedResult struct {
	PayloadStatus *PayloadStatus `json:"payloadStatus"`
	PayloadId     string         `json:"payloadId"`
}

type RpcMethod string

type Client struct {
	authToken *jwt.EthJwt
	endpoint  string
}

func NewClient(port string) Client {
	return Client{
		endpoint: fmt.Sprintf("http://localhost:%v", port),
	}
}

func (client *Client) SetAuthToken(newToken *jwt.EthJwt) {
	client.authToken = newToken
}

func NewClientWithJwt(port string, secret []byte) Client {
	client := NewClient(port)
	client.authToken = jwt.FromSecret(secret)
	return client
}

// An Ethereum Json-rpc message
type Message struct {
	JsonRPC string        `json:"jsonrpc"`
	Method  RpcMethod     `json:"method"`
	Params  []interface{} `json:"params"`
	Id      uint          `json:"id"`
}

// An Ethereum Json-rpc response
type Response struct {
	JsonRPC string        `json:"jsonrpc"`
	Result  interface{}   `json:"result"`
	Id      uint          `json:"id"`
	Error   *JsonRpcError `json:"error"`
}

// The error from an Ethereum Json-rpc response
type JsonRpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Creates a Json-rpc message with the supplied method and parameters
func NewMessage(method RpcMethod, params ...interface{}) *Message {
	if params == nil {
		params = make([]interface{}, 0)
	}
	return &Message{
		JsonRPC: "2.0",
		Method:  method,
		Params:  params,
		Id:      1,
	}
}

// Sends a JSON-RPC method whose response is unmartialed into the provided rpc.Response object.
// This allows the caller to specify the type for the response.Result at compile time, rather than
// relying on runtime reflection to identify it
func (client *Client) SendWithTypedResponse(msg *Message, timeout time.Duration, retries RetryStrategy, result interface{}) *RpcError {
	for !retries.Done() {
		time.Sleep(retries.Next())
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := client.sendWithTypedResponse(ctx, msg, result)
		if err != nil {
			log.Warn("Error sending msg to execution client", "err", err)
			if err.IsRetryable() {
				continue
			}
		}
		return err
	}
	return nil
}

func (client *Client) sendWithTypedResponse(ctx context.Context, msg *Message, result interface{}) *RpcError {
	marshalled, err := json.Marshal(msg)
	if err != nil {
		log.Crit("Marshalling failed. This indicates a bug: ", err)
		return Exception(err)
	}
	fmt.Println(string(marshalled))
	req, err := http.NewRequest("POST", client.endpoint, bytes.NewBuffer(marshalled))
	if err != nil {
		log.Crit("Creating an http request failed. This indicates a bug: ", err)
		return Exception(err)
	}

	req.Header["Content-Type"] = []string{"application/json"}
	if client.authToken != nil {
		tokenString, err := client.authToken.TokenString()
		if err != nil {
			return Exception(err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", tokenString))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Exception(fmt.Errorf("Error sending message to execution client: %w", err))
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Exception(fmt.Errorf("Error reading response to msg %v. %w", msg, err))
	}
	response := &Response{
		Result: result,
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return Exception(fmt.Errorf("Error unmarshalling response to msg %v. response body %v. err %w", msg, string(body), err))
	}
	if response.Error != nil {
		return InProtocolError(response.Error)
	}
	return nil
}

// Sends a JSON rpc message. Should be used when the caller doesn't need static access to the response.Result.
// If you do rely on having a response.Result of a certain type, use `SendWithTypedResponse` and pass in your own
// response object.
func (client *Client) Send(msg *Message, timeout time.Duration, retries RetryStrategy) ([]byte, *RpcError) {
	r := []byte{}
	return r, client.SendWithTypedResponse(msg, timeout, retries, r)
}
