package db

import (
	"errors"
	"fmt"
	"math"
	"regent/utils"
	"strings"

	"github.com/ledgerwatch/erigon/common"
)

const (
	RollupBlockNumberToHash = "RollupBlockNumberToHash"
	RollupBlockHashToNumber = "RollupBlockHashToNumber"

	serializedUint64Len = 8
	serializedHashLen   = 32
)

var (
	ERR_INVALID_U64        = errors.New("could not unmarshal slice as a uint64. The length must be exactly 8 bytes. This likely indicates db corruption")
	ERR_INVALID_BLOCK_HASH = errors.New("could not unmarshal slice as a hash. The length must be exactly 32 bytes. This likely indicates db corruption")
	ERR_NOT_FOUND          = errors.New("the requested item could note be found in the database. requested: ")
	ERR_EXHAUSTED          = errors.New("the iterator is exhausted")
	ERR_PAST_HEAD          = errors.New("the requested block is beyond the tip of the known chain")
	ERR_MISSING            = errors.New("the requested block was not found, but a more recent block was")
)

// The error returned when an item could not be found in the database
type NotFoundError struct {
	inner error
	msg   string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s. %s. ", e.msg, e.inner)
}

func (e *NotFoundError) Unwrap() error {
	return e.inner
}

func (e *NotFoundError) Is(err error) bool {
	return strings.HasPrefix(err.Error(), ERR_NOT_FOUND.Error()) || errors.Is(err, ERR_NOT_FOUND)
}

// An iterator over all block hashes, ordered by height
type BlockHashIterator struct {
	inner Iterator
}

// Get a DbIterator over the rollup block hashes, ordered by block number
func GetBlockHashIterator(db RangeDb) *BlockHashIterator {
	iter := &BlockHashIterator{
		inner: db.GetRange(RollupBlockNumberToHash, MarshalUint(0), MarshalUint(math.MaxUint64)),
	}
	iter.inner.First()
	return iter
}

// Read the current hash from the iterator, and advance it one spot
// Returns an error if and only if the iterator is exhausted
func (blocks *BlockHashIterator) Next() (common.Hash, error) {
	if blocks.inner.Key() != nil {
		res, err := UnmarshallHash(blocks.inner.Value())
		blocks.inner.Next()
		return check(res, err), nil
	}
	return common.Hash{}, ERR_EXHAUSTED
}

// Return the current hash from the iterator, and move it back one spot
// Returns an error if and only if the iterator is exhausted
func (blocks *BlockHashIterator) Prev() (common.Hash, error) {
	if blocks.inner.Key() != nil {
		res, err := UnmarshallHash(blocks.inner.Value())
		blocks.inner.Prev()
		return check(res, err), nil
	}
	return common.Hash{}, ERR_EXHAUSTED
}

// Return the block number that the iterator is currently at, without moving the iterator
// Returns an error if and only if the iterator is exhausted
func (blocks *BlockHashIterator) CursorHeight() (uint64, error) {
	if blocks.inner.Key() != nil {
		return check(UnmarshallUint(blocks.inner.Key())), nil
	}
	return 0, ERR_EXHAUSTED
}

// Return the current hash from the iterator without moving the iterator
// Returns an error if and only if the iterator is exhausted
func (blocks *BlockHashIterator) CursorHash() (common.Hash, error) {
	if blocks.inner.Key() != nil {
		return check(UnmarshallHash(blocks.inner.Value())), nil
	}
	return common.Hash{}, ERR_EXHAUSTED
}

// Set the iterator to the latest block, and return its hash
func (blocks *BlockHashIterator) HeadHash() common.Hash {
	if !blocks.inner.Last() {
		return utils.GENESIS_HASH
	}
	return check(UnmarshallHash(blocks.inner.Value()))
}

// Set the iterator to the latest block, and return its number
func (blocks *BlockHashIterator) HeadNumber() uint64 {
	if !blocks.inner.Last() {
		return 0
	}
	return check(UnmarshallUint(blocks.inner.Value()))
}

// Set the iterator to the genesis block, and return its hash
func (blocks *BlockHashIterator) Genesis() common.Hash {
	if !blocks.inner.First() {
		return utils.GENESIS_HASH
	}
	return check(UnmarshallHash(blocks.inner.Value()))
}

// Set the iterator to the block at the provided height, and return its hash
// Returns an error if the provided block number does not exist in the database
func (blocks *BlockHashIterator) Seek(number uint64) (common.Hash, error) {
	if number == 0 {
		return utils.GENESIS_HASH, nil
	}
	if !blocks.inner.Seek(MarshalUint(number)) {
		return common.Hash{}, &NotFoundError{
			inner: ERR_PAST_HEAD,
			msg:   fmt.Sprintf("block %v was not found in the database. The latest block is only %v", number, blocks.HeadNumber()),
		}
	}
	if check(UnmarshallUint(blocks.inner.Key())) != number {
		return common.Hash{}, &NotFoundError{
			inner: ERR_MISSING,
			msg:   fmt.Sprintf("block %v was missing from the database. Found %v in its place.", number, check(UnmarshallUint(blocks.inner.Key()))),
		}
	}
	return check(UnmarshallHash(blocks.inner.Value())), nil
}

// Store a new rollup block hash in the database. Stores the mapping from hash->number and from number->hash
func PutRollupBlockHashWithNumber(db SimpleDb, blockhash common.Hash, blocknumber uint64) error {
	err := db.Put(RollupBlockHashToNumber, blockhash[:], MarshalUint(blocknumber))
	if err != nil {
		return err
	}
	return db.Put(RollupBlockNumberToHash, MarshalUint(blocknumber), blockhash[:])
}

// Looks up the block number corresponding to the provided hash in the database
func GetRollupBlockNumber(db SimpleDb, blockhash common.Hash) (uint64, error) {
	raw, err := db.Get(RollupBlockHashToNumber, blockhash[:])
	if err != nil {
		return 0, &NotFoundError{
			inner: err,
			msg:   fmt.Sprintf("%s blockhash %v in table %s", ERR_NOT_FOUND, blockhash, RollupBlockHashToNumber),
		}
	}
	return UnmarshallUint(raw)
}

// Looks up the block hash for the provided block number in the database
func GetRollupBlockHash(db SimpleDb, blocknum uint64) (common.Hash, error) {
	raw, err := db.Get(RollupBlockNumberToHash, MarshalUint(blocknum))
	if err != nil {
		return common.Hash{}, &NotFoundError{
			inner: err,
			msg:   fmt.Sprintf("%s blocknumber %v in table %s", ERR_NOT_FOUND, blocknum, RollupBlockNumberToHash),
		}
	}
	return UnmarshallHash(raw)
}
