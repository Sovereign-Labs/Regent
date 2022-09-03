package rpc

import (
	"regent/utils"

	"github.com/ledgerwatch/erigon/cmd/rpcdaemon/commands"
)

func (client *Client) SendForkChoiceUpdated(forkchoiceState *commands.ForkChoiceState, payloadAttributes *commands.PayloadAttributes) (*Response, error) {
	var msg *Message
	if payloadAttributes != nil {
		msg = NewMessage(FORK_CHOICE_UPDATED, forkchoiceState, payloadAttributes)
	} else {
		msg = NewMessage(FORK_CHOICE_UPDATED, forkchoiceState)
	}
	return client.Send(msg)
}

func (client *Client) SendNewPayload(payload *commands.ExecutionPayload) (*Response, error) {
	return client.Send(NewMessage(NEW_EXECUTION_PAYLOAD, payload))
}

func (client *Client) SendGetPayload(payloadId uint64) (*Response, *commands.ExecutionPayload, error) {
	msg := NewMessage(GET_EXECUTION_PAYLOAD, utils.U64ToHex(payloadId))
	payload := &commands.ExecutionPayload{}
	r := &Response{
		Result: payload,
	}
	return r, payload, client.SendWithTypedResponse(msg, r)
}
