package rpc

import (
	"regent/common"
	"time"
)

const FORK_CHOICE_UPDATED RpcMethod = "engine_forkchoiceUpdatedV1"

type ForkchoiceStateV1 struct {
	HeadBlockHash string `json:"headBlockHash"`
	// This value must be either equal to or an ancestor of headBlockHash
	SafeBlockHash string `json:"safeBlockHash"`
	// Since our chain finalizes instantly, this is a duplicate of the SafeBlockHash
	FinalizedBlockHash string `json:"finalizedBlockHash"`
}

type PayloadAttributesV1 struct {
	Timestamp             time.Time      `json:"Timestamp"`
	PrevRandao            common.Hash    `json:"prevRandao"`
	SuggestedFeeRecipient common.Address `json:"suggestedFeeRecipient"`
}
