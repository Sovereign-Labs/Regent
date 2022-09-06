package rpc

import (
	"fmt"
	"regent/rpc/jwt"
	"time"

	"github.com/ledgerwatch/erigon/cmd/rpcdaemon/commands"
)

type Client struct {
	authToken *jwt.EthJwt
	endpoint  string
}

var DefaultRetryStrategy = func() RetryStrategy {
	return &InfiniteRetryStrategy{}
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
	client.SetAuthToken(jwt.FromSecret(secret))
	return client
}

// Updates the execution client's current head.
func (client *Client) UpdateForkChoice(forkChoice *commands.ForkChoiceState) (*ForkChoiceUpdatedResult, error) {
	return getResponse[*ForkChoiceUpdatedResult](client, NewRequest(FORK_CHOICE_UPDATED, forkChoice), 8*time.Second, DefaultRetryStrategy())
}

// Updates the execution client's current head and starts the block building process
func (client *Client) UpdateForkChoiceAndBuildBlock(forkChoice *commands.ForkChoiceState, payloadAttributes *commands.PayloadAttributes) (*ForkChoiceUpdatedResult, error) {
	return getResponse[*ForkChoiceUpdatedResult](client, NewRequest(FORK_CHOICE_UPDATED, forkChoice, payloadAttributes), 8*time.Second, DefaultRetryStrategy())
}

// Passes a new `execution payload` (block) to the execution client
func (client *Client) SendExecutionPayload(payload *commands.ExecutionPayload) error {
	_, err := getResponse[*commands.PayloadAttributes](client, NewRequest(NEW_EXECUTION_PAYLOAD, payload), 8*time.Second, DefaultRetryStrategy())
	return err
}

// Requests a new block ("execution payload") from the client. This method will fail if
// the previous call was to UpdateForkChoice rather than UpdateForkChoiceAndBuildBlock
func (client *Client) GetPayload(payloadId string) (*commands.ExecutionPayload, error) {
	msg := NewRequest(GET_EXECUTION_PAYLOAD, payloadId)
	return getResponse[*commands.ExecutionPayload](client, msg, 1*time.Second, DefaultRetryStrategy())
}
