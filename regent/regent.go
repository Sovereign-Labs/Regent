package main

import (
	"os"
	"path"
	"regent/rpc"
	"regent/rpc/jwt"
	"time"

	"github.com/ledgerwatch/erigon/cmd/rpcdaemon/commands"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/ledgerwatch/log/v3"
)

type Regent struct {
	CurrentHead        common.Hash
	NextPayloadId      string
	EngineRpc          rpc.Client
	BeneficiaryAddress common.Address
}

func Initialize() (*Regent, error) {
	r := &Regent{}
	r.EngineRpc = rpc.NewClient(EngineRpcPort)
	token, err := jwt.FromSecretFile(path.Join(ErigonDatadir, JWT_SECRET_FILENAME))
	if err != nil {
		return nil, err
	}
	r.EngineRpc.SetAuthToken(token)
	return r, nil
}

func (r *Regent) run() error {
	log.Info("Starting block production loop")

	for {
		// TODO: don't bother getting a payload when this node isn't the sequencer
		payload, err := r.EngineRpc.GetPayload(r.NextPayloadId)
		log.Info("Getting next execution payload")
		if err != nil {
			log.Crit("encountered an error attempting retrive the next execution payload", "err", err)
			return err
		}

		// TODO: don't bother sending the payload to the sequencer when this node isn't the sequencer
		log.Info("Sending next payload to execution client", "blockhash", payload.BlockHash)
		err = r.EngineRpc.SendExecutionPayload(payload)
		if err != nil {
			log.Crit("encountered an error attempting to send the payload to the execution client", "err", err)
			return err
		}

		// TODO: Only start the builder when this node will be sequencer
		log.Info("Updating head", "blockhash", payload.BlockHash)
		newErr := r.ExtendChainAndStartBuilder(payload.BlockHash, r.BeneficiaryAddress)
		if newErr != nil {
			log.Crit("encountered an error attempting to extend the current chain", "err", newErr)
			return newErr
		}

		log.Info("Waiting for next slot")
		time.Sleep(5 * time.Second)
		log.Info("Done waiting")
	}
}

// Add a new block to the chain using engine_forkChoiceUpdated. Re-orgs are impossible,
// so the last finalized block is just the previous head
func (r *Regent) ExtendChainAndStartBuilder(hash common.Hash, suggestedRecipient common.Address) error {
	nextState := commands.ForkChoiceState{
		HeadHash:           hash,
		FinalizedBlockHash: r.CurrentHead,
		SafeBlockHash:      r.CurrentHead,
	}
	result, err := r.EngineRpc.UpdateForkChoiceAndBuildBlock(&nextState, &commands.PayloadAttributes{
		Timestamp:             hexutil.Uint64(time.Now().Unix()),
		SuggestedFeeRecipient: suggestedRecipient,
	})
	if err != nil {
		if err.(*rpc.JsonRpcError) != nil {
			// TODO: Handle the various expected errors
			log.Crit("Fatal err trying to update fork choice", "err", err)
			os.Exit(1)
		} else {
			return err
		}
	}
	r.NextPayloadId = result.PayloadId
	r.CurrentHead = hash
	return nil
}
