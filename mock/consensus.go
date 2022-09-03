package main

import (
	"regent/rpc"
	"regent/utils"
	"time"

	"github.com/ledgerwatch/erigon/cmd/rpcdaemon/commands"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
)

var CURRENT_HEAD = common.HexToHash(utils.GENESIS_HASH_STRING)

// Add a new block to the chain using engine_forkChoiceUpdated. Re-orgs are impossible,
// so the last finalized block is just the previous head
func AddBlockToChain(hash common.Hash) (*rpc.Response, error) {
	nextState := commands.ForkChoiceState{
		HeadHash:           hash,
		FinalizedBlockHash: CURRENT_HEAD,
		SafeBlockHash:      CURRENT_HEAD,
	}
	CURRENT_HEAD = hash
	return EngineRpc.SendForkChoiceUpdated(&nextState, &commands.PayloadAttributes{
		Timestamp:             hexutil.Uint64((time.Now().Add(12 * time.Second).Unix())),
		SuggestedFeeRecipient: utils.DEV_ADDRESS,
	})
}
