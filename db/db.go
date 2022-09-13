package db

import (
	"errors"

	"github.com/ledgerwatch/erigon-lib/kv"
)

const DEFAULT_REGENT_DATADIR = "/Users/prestonevans/sovereign/regent"

var regentDatadir = DEFAULT_REGENT_DATADIR

// func (r *Regent) Sync() error {
// TODO: fetch latest block from DA
// Send block to execution client
// Update for choice with hash
// 	return nil
// }

// func testDb() {
// 	kv.AccountChangeSet
// }

type SimpleDb interface {
	Get(table string, key []byte) (val []byte, err error)
	kv.Putter
	kv.Deleter
	kv.Closer
}
type Iterator interface {
	// First moves the iterator to the first key/value pair. If the iterator
	// only contains one key/value pair then First and Last would move
	// to the same key/value pair.
	// Returns false if and only if the iterator is empty
	First() bool

	// Last moves the iterator to the last key/value pair. If the iterator
	// only contains one key/value pair then First and Last would moves
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
	GetRange(table string, from []byte, to []byte) Iterator
}

type MemDb struct {
	tables map[string]map[string][]byte
}

func (db *MemDb) Close() {}
func (db *MemDb) Put(table string, k []byte, v []byte) error {
	if t, ok := db.tables[table]; ok {
		t[string(k)] = v
		return nil
	}
	newTable := make(map[string][]byte)
	newTable[string(k)] = v
	db.tables[table] = newTable
	return nil
}
func (db *MemDb) Get(table string, key []byte) (val []byte, err error) {
	if t, ok := db.tables[table]; ok {
		return t[string(key)], nil
	}
	return nil, errors.New("Table does not exist")
}
func (db *MemDb) Delete(table string, k []byte) error {
	if t, ok := db.tables[table]; ok {
		delete(t, string(k))
		return nil
	}
	return errors.New("Table does not exist")
}

var _ SimpleDb = &MemDb{}
