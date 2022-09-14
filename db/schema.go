package db

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ledgerwatch/erigon/common"
)

const (
	RollupBlockNumberToHash = "RollupBlockNumberToHash"
	RollupBlockHashToNumber = "RollupBlockHashToNumber"
)

var (
	ERR_INVALID_U64        = errors.New("could not unmarshal slice as a uint64. The length must be exactly 8 bytes. This likely indicates db corruption")
	ERR_INVALID_BLOCK_HASH = errors.New("could not unmarshal slice as a hash. The length must be exactly 32 bytes. This likely indicates db corruption")
	ERR_NOT_FOUND          = errors.New("the requested item could note be found in the database. requested: ")
	ERR_EXHAUSTED          = errors.New("the iterator is exhausted")
	ERR_PAST_HEAD          = errors.New("the requested block is beyond the tip of the known chain")
	ERR_MISSING            = errors.New("the requested block was not found, but a more recent block was.")
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

// An double-ended iterator over all block hashes, ordered by height.
// The iterator may panic if the databse is empty, so callers should ensure
// that the underlying database contains at least the Genesis block at all times.
type BlockHashIterator struct {
	inner    Iterator
	position uint64
}

// Get a DbIterator over the rollup block hashes, ordered by block number
func GetBlockHashIterator(db RangeDb) *BlockHashIterator {
	iter := &BlockHashIterator{
		inner: db.GetRange(RollupBlockNumberToHash),
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
		blocks.position += 1
		return unwrap(res, err), nil
	}
	return common.Hash{}, ERR_EXHAUSTED
}

// Return the current hash from the iterator, and move it back one spot
// Returns an error if and only if the iterator is exhausted
func (blocks *BlockHashIterator) Prev() (common.Hash, error) {
	if blocks.position == 0 {
		return common.Hash{}, ERR_EXHAUSTED
	}
	blocks.inner.Prev()
	blocks.position -= 1
	res, err := UnmarshallHash(blocks.inner.Value())
	return unwrap(res, err), nil
}

// Return the block number of the hash that the iterator will yield next, without moving the iterator
func (blocks *BlockHashIterator) Position() uint64 {
	return uint64(blocks.position)
}

// Return the current hash from the iterator without moving the iterator
// Returns an error if and only if the iterator is exhausted
func (blocks *BlockHashIterator) Peek() (common.Hash, error) {
	if blocks.inner.Key() != nil {
		return unwrap(UnmarshallHash(blocks.inner.Value())), nil
	}
	return common.Hash{}, ERR_EXHAUSTED
}

// Set the iterator to the latest block, and return its hash
func (blocks *BlockHashIterator) Head() common.Hash {
	blocks.inner.Last()
	pos := unwrap(extractBlockNumFromKey(blocks.inner.Key()))
	blocks.position = pos
	return unwrap(UnmarshallHash(blocks.inner.Value()))
}

// Set the iterator to the latest block, and return its number
func (blocks *BlockHashIterator) HeadNumber() uint64 {
	blocks.inner.Last()
	pos := unwrap(extractBlockNumFromKey(blocks.inner.Key()))
	blocks.position = pos
	return pos
}

// Set the iterator to the genesis block, and return its hash
func (blocks *BlockHashIterator) Genesis() common.Hash {
	blocks.inner.First()
	blocks.position = 0
	return unwrap(UnmarshallHash(blocks.inner.Value()))
}

// Set the iterator to the block at the provided height, and return its hash
// Returns an error only if the provided block number does not exist in the database
func (blocks *BlockHashIterator) Seek(number uint64) (common.Hash, error) {
	if !blocks.inner.Seek(keyFor(RollupBlockNumberToHash, MarshalUint(number))) {
		return common.Hash{}, &NotFoundError{
			inner: ERR_PAST_HEAD,
			msg:   fmt.Sprintf("block %v was not found in the database. The latest block is only %v", number, blocks.HeadNumber()),
		}
	}
	if unwrap(extractBlockNumFromKey(blocks.inner.Key())) != number {
		return common.Hash{}, &NotFoundError{
			inner: ERR_MISSING,
			msg:   fmt.Sprintf("block %v was missing from the database. Found %v in its place.", number, unwrap(extractBlockNumFromKey(blocks.inner.Key()))),
		}
	}
	return unwrap(UnmarshallHash(blocks.inner.Value())), nil
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

// Remove the tablename prefix from a key in the `RollupBlockNumberToHash` table,
// then unmarshall the remaining bytes as a blocknumber
func extractBlockNumFromKey(raw []byte) (uint64, error) {
	if len(raw) < len(RollupBlockNumberToHash) {
		return 0, fmt.Errorf("%s did not contain a valid block number with prefix %s. %w", raw, RollupBlockNumberToHash, ERR_INVALID_U64)
	}
	return UnmarshallUint(raw[len(RollupBlockNumberToHash):])
}
