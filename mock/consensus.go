package main

import (
	"regent/common"
	"regent/rpc"
)

var CURRENT_HEAD_STRING string = common.GENESIS_HASH

// Add a new block to the chain using engine_forkChoiceUpdated. Re-orgs are impossible,
// so the last finalized block is just the previous head
func AddBlockToChain(hash string) (*rpc.Response, error) {
	nextState := rpc.ForkchoiceStateV1{
		HeadBlockHash:      hash,
		FinalizedBlockHash: CURRENT_HEAD_STRING,
		SafeBlockHash:      CURRENT_HEAD_STRING,
	}
	return EngineRpc.SendForkChoiceUpdated(&nextState, nil)
}
