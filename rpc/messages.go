package rpc

func (client *Client) SendForkChoiceUpdated(forkchoiceState *ForkchoiceStateV1, payloadAttributes *PayloadAttributesV1) (*Response, error) {
	var msg *Message
	if payloadAttributes != nil {
		msg = NewMessage(FORK_CHOICE_UPDATED, forkchoiceState, payloadAttributes)
	} else {
		msg = NewMessage(FORK_CHOICE_UPDATED, forkchoiceState)
	}
	return client.Send(msg)
}
