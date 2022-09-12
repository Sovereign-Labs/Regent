package main

import (
	"errors"
	"fmt"
	"path"
	"regent/db"
	"regent/rpc"
	"regent/rpc/jwt"
	"time"

	"github.com/ledgerwatch/erigon/cmd/rpcdaemon/commands"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/ledgerwatch/log/v3"
)

var (
	ERR_INVALID_PAYLOAD_STATUS   = errors.New("the Payload status was not a valid option. This indicates a client bug")
	ERR_INVALID_PAYLOAD          = errors.New("the execution client was unable to build a block because the fork choice update payload was invalid")
	ERR_EXECUTION_CLIENT_SYNCING = errors.New("the execution client was unable to build a block because it is syncing")
	ERR_INVALID_PAYLOAD_ID       = errors.New("the execution client returned an invalid payload ID")
	ERR_INVALID_TIMESTAMP        = errors.New("the consensus client provided an invalid timestamp for the payload to be built")
	ERR_INVALID_FORKCHOICE       = errors.New("the consensus client provided an invalid forkchoice update")
	// Test against this error when looking for ForkChoiceUpdateErrors
	ERR_FORKCHOICE_NOT_UPDATED = errors.New("the fork choice could not be updated")
	// Test against this error when looking for PayloadBuildErrors
	ERR_PAYLOAD_NOT_BUILT = errors.New("the execution client was unable to build a valid payload")
)

type ForkChoiceUpdateError struct {
	reason error
}

func (e *ForkChoiceUpdateError) Error() string {
	return fmt.Sprintf("%s: %s", ERR_FORKCHOICE_NOT_UPDATED, e.reason)
}

func (e *ForkChoiceUpdateError) Is(err error) bool {
	// All forkchoice update errors are also payload build errors
	return errors.Is(err, ERR_FORKCHOICE_NOT_UPDATED) || errors.Is(err, ERR_PAYLOAD_NOT_BUILT)
}

func (e *ForkChoiceUpdateError) Unwrap() error {
	return e.reason
}

// PayloadBuildErrors are a superset of ForkChoiceUpdateErrors.
// They can occur either because the forkchoice was not updated or because
// some payload attribute was incorrect.
type PayloadBuildError struct {
	reason error
}

func (e *PayloadBuildError) Error() string {
	return fmt.Sprintf("%s: %s", ERR_PAYLOAD_NOT_BUILT, e.reason)
}

func (e *PayloadBuildError) Is(err error) bool {
	return errors.Is(err, ERR_PAYLOAD_NOT_BUILT)
}

func (e *PayloadBuildError) Unwrap() error {
	return e.reason
}

type Regent struct {
	CurrentHead        common.Hash
	BlockNumber        uint
	NextPayloadId      string
	EngineRpc          rpc.Client
	BeneficiaryAddress common.Address
	DB                 db.SimpleDb
}

func Initialize() (*Regent, error) {
	r := &Regent{}
	r.EngineRpc = rpc.NewClient(EngineRpcPort)
	token, err := jwt.FromSecretFile(path.Join(ErigonDatadir, JWT_SECRET_FILENAME))
	if err != nil {
		return nil, err
	}
	r.EngineRpc.SetAuthToken(token)
	r.DB, err = db.NewLevelDB(DEFAULT_REGENT_DATADIR)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Regent) SetCurrentHead(newHead common.Hash) {
	r.CurrentHead = newHead
}

// TODO: The main event loop will eventually consist of the following steps:
//  1. Sync the consensus client (by downloading the latest block(s) from DA)
//  2. Send the hash of the latest block to the execution client. If the client was previously done syncing,
//     and it is our turn to sequence include PayloadAttributes. Otherwise, GOTO 1
//  3. Wait for block to build.
//  4. Fetch block from execution client.
//  5. Post block (+ optional proof) to DA
//
// For now, though we assume that...
//
//	The consensus client is always synced
//	It's always our turn to produce a block
//	DA happens by magic.
//
// This lets us use the following simplified loop.
func (r *Regent) run() error {
	log.Info("Starting block production loop")

	for {
		// Wait for next slot
		// TODO: This will eventually be a wait on the p2p network. For now, sleep to avoid a busy loop
		log.Info("Waiting for next slot")
		time.Sleep(5 * time.Second)
		log.Info("Done waiting")

		// TODO: don't bother getting a payload when this node isn't the sequencer
		payload, err := r.EngineRpc.GetPayload(r.NextPayloadId)
		log.Info("Getting next execution payload")
		if err != nil {
			log.Crit("encountered an error attempting retrive the next execution payload", "err", err)
		}

		// TODO: don't bother sending the payload to the sequencer when this node isn't the sequencer
		log.Info("Sending next payload to execution client", "blockhash", payload.BlockHash)
		_, err = r.EngineRpc.SendExecutionPayload(payload)
		if err != nil {
			log.Crit("encountered an error attempting to send the payload to the execution client", "err", err)
		}

		// TODO: Only start the builder when this node will be sequencer
		log.Info("Updating head", "blockhash", payload.BlockHash)
		err = r.ExtendChainAndStartBuilder(payload.BlockHash, r.BeneficiaryAddress)
		if errors.Is(err, ERR_FORKCHOICE_NOT_UPDATED) {
			if errors.Is(err, ERR_EXECUTION_CLIENT_SYNCING) {
				// TODO: re-enter the syncing loop.
				log.Warn("Unable to extend fork because the execution client is out of sync. Retrying.")
			}
			log.Crit("encountered an unrecoverable error attempting to extend the current chain", "err", err)
		}

	}
}

// Check whether the fork choice update was applied. Return an error if not.
func validateForkChoiceUpdate(err error, result *rpc.ForkChoiceUpdatedResult, nextState *commands.ForkChoiceState) error {
	if err != nil {
		// If the error was an in protocol error, then the error code tells us whether the the fork choice was
		// updated: "invalid payload attributes" don't prevent a fork choice from being applied, but all other errors do
		if rpcErr, ok := err.(*rpc.JsonRpcError); ok {
			switch rpcErr.Code {
			case rpc.CODE_INVALID_PAYLOAD_ATTRIBUTES:
				return nil
			case rpc.CODE_INVALID_FORKCHOICE_STATE:
				return ERR_INVALID_FORKCHOICE
			default:
				return err
			}
		}
		// If the error is not a defined protocol error, assume the fork choice was not updated
		return fmt.Errorf("Unknown error prevented a fork choice update: %w", err)
	}

	// If there was no error, the payload status.status field tells us whether the update succeeded
	if result.PayloadStatus == nil {
		return ERR_INVALID_PAYLOAD_STATUS
	}
	switch result.PayloadStatus.Status {
	case rpc.VALID_PAYLOAD:
		return nil
	case rpc.INVALID_PAYLOAD:
		return ERR_INVALID_PAYLOAD
	case rpc.SYNCING_PAYLOAD:
		return ERR_EXECUTION_CLIENT_SYNCING
	default:
		return ERR_INVALID_PAYLOAD_STATUS
	}
}

// Add a new block to the chain using engine_forkChoiceUpdated. Re-orgs are impossible,
// so the last finalized block is just the previous head
func (r *Regent) ExtendChainAndStartBuilder(newHead common.Hash, suggestedRecipient common.Address) error {
	r.DB.Put(db.RollupBlockHashToNumber, newHead[:], db.VarUint(BlockNumber+1))
	return r.tryExtendChainAndStartBuilder(newHead, suggestedRecipient)
}

// Add a new block to the chain using engine_forkChoiceUpdated. Re-orgs are impossible,
// so the last finalized block is just the previous head
func (r *Regent) tryExtendChainAndStartBuilder(newHead common.Hash, suggestedRecipient common.Address) error {
	// Construct and send the Rpc Message
	nextState := commands.ForkChoiceState{
		HeadHash:           newHead,
		FinalizedBlockHash: r.CurrentHead,
		SafeBlockHash:      r.CurrentHead,
	}
	result, err := r.EngineRpc.UpdateForkChoiceAndBuildBlock(&nextState, &commands.PayloadAttributes{
		Timestamp:             hexutil.Uint64(time.Now().Unix()),
		SuggestedFeeRecipient: suggestedRecipient,
	})

	// Verify that the fork choice was updated
	forkChoiceErr := validateForkChoiceUpdate(err, result, &nextState)
	if forkChoiceErr != nil {
		log.Crit(ERR_FORKCHOICE_NOT_UPDATED.Error(), "err", forkChoiceErr)
		return &ForkChoiceUpdateError{forkChoiceErr}
	}
	r.SetCurrentHead(newHead)
	r.BlockNumber += 1

	// If `err` is not nil but we reached this point, the error must have been "invalid payload attributes".
	if err != nil {
		log.Crit(ERR_INVALID_TIMESTAMP.Error(), "err", err, "forkChoiceState", nextState)
		return &PayloadBuildError{ERR_INVALID_TIMESTAMP}
	}
	// Sanity check that the payload ID looks like a valid DATA[8] object
	if len(result.PayloadId) != 18 {
		log.Crit("The execution client returned an invalid payload id", "forkChoiceState", nextState, "response", result)
		return &PayloadBuildError{ERR_INVALID_PAYLOAD_ID}
	}
	r.NextPayloadId = result.PayloadId
	return nil
}
