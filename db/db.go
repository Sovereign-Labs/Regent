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
type DbIterator interface {
	Value() []byte
	Next() bool
}

type RangeDb interface {
	SimpleDb
	GetRange(table string, from []byte, to []byte) DbIterator
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
