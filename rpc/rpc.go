package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regent/rpc/jwt"

	"github.com/ledgerwatch/log/v3"
)

const VERSION_STRING string = "Regent/0.0.0"
const GET_BLOCK_BY_NUMBER RpcMethod = "eth_getBlockByNumber"

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

func (client *Client) Send(msg *Message) (*Response, error) {
	marshalled, err := json.Marshal(msg)
	if err != nil {
		log.Crit("Marshalling failed. This indicates a bug: ", err)
		return nil, err
	}
	req, err := http.NewRequest("POST", client.endpoint, bytes.NewBuffer(marshalled))
	if err != nil {
		log.Crit("Creating an http request failed. This indicates a bug: ", err)
		return nil, err
	}

	req.Header["Content-Type"] = []string{"application/json"}
	if client.authToken != nil {
		tokenString, err := client.authToken.TokenString()
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", tokenString))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error sending message to execution client: %w", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response to msg %v. %w", msg, err)
	}
	response := Response{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling response to msg %v. response body %v. err %w", msg, body, err)
	}
	return &response, nil
}
