package rpc

import (
	"time"

	"github.com/ledgerwatch/erigon/cmd/rpcdaemon/commands"
)

func (client *Client) SendForkChoiceUpdated(forkchoiceState *commands.ForkChoiceState, payloadAttributes *commands.PayloadAttributes) (*ForkChoiceUpdatedResult, *RpcError) {
	var msg *Message
	if payloadAttributes != nil {
		msg = NewMessage(FORK_CHOICE_UPDATED, forkchoiceState, payloadAttributes)
	} else {
		msg = NewMessage(FORK_CHOICE_UPDATED, forkchoiceState)
	}
	result := &ForkChoiceUpdatedResult{}
	return result, client.SendWithTypedResponse(msg, 8*time.Second, &InfiniteRetryStrategy{}, result)
}

func (client *Client) SendNewPayload(payload *commands.ExecutionPayload) *RpcError {
	_, err := client.Send(NewMessage(NEW_EXECUTION_PAYLOAD, payload), 8*time.Second, &InfiniteRetryStrategy{})
	return err
}

func (client *Client) SendGetPayload(payloadId string) (*commands.ExecutionPayload, *RpcError) {
	msg := NewMessage(GET_EXECUTION_PAYLOAD, payloadId)
	payload := &commands.ExecutionPayload{}
	return payload, client.SendWithTypedResponse(msg, 1*time.Second, &InfiniteRetryStrategy{}, payload)
}
