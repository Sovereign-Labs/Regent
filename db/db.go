package db

import (
	"github.com/ledgerwatch/erigon-lib/kv"
)

const DEFAULT_REGENT_DATADIR = "/Users/prestonevans/sovereign/regent"

var regentDatadir = DEFAULT_REGENT_DATADIR

type SimpleDb interface {
	Get(table string, key []byte) (val []byte, err error)
	kv.Putter
	kv.Deleter
	kv.Closer
}

type BatchDb interface {
	SimpleDb
	WriteBatched([]struct {
		table string
		key   []byte
		val   []byte
	}) error
}

type Iterator interface {
	// First moves the iterator to the first key/value pair. If the iterator
	// only contains one key/value pair then First and Last move
	// to the same key/value pair.
	// Returns false if and only if the iterator is empty
	First() bool

	// Last moves the iterator to the last key/value pair. If the iterator
	// only contains one key/value pair then First and Last move
	// to the same key/value pair.
	// Returns false if and only if the iterator is empty
	Last() bool

	// Seek moves the iterator to the first key/value pair whose key is greater
	// than or equal to the given key.
	// Returns true only if such a key exists
	//
	// It is safe to modify the contents of the argument after Seek returns.
	Seek(key []byte) bool

	// Next moves the iterator to the next key/value pair.
	// Returns false if the iterator is exhausted.
	Next() bool

	// Prev moves the iterator to the previous key/value pair.
	// Returns false if the iterator is exhausted.
	Prev() bool

	// Key returns the key of the current key/value pair, or nil if done.
	// The caller should not modify the contents of the returned slice, and
	// its contents may change on the next call to the iterator
	Key() []byte

	// Value returns the value of the current key/value pair, or nil if done.
	// The caller should not modify the contents of the returned slice, and
	// its contents may change on the next call to the iterator
	Value() []byte

	// Releases any resources held by this iterator. Callers must invoke this method
	// before dropping the iterator
	Release()
}

type RangeDb interface {
	SimpleDb
	GetRange(table string) Iterator
}
