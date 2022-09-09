package db

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var defaultReadOptions = &opt.ReadOptions{
	Strict: opt.DefaultStrict,
}
var defaultWriteOptions = &opt.WriteOptions{}

// A simple wrapper over the levelDB package.
// The underlying DB can be read and written to concurrently.
type LevelDB struct {
	inner *leveldb.DB
}

// Opens a new DB at the provided path.
//
// The returned DB instance is safe for concurrent use.
// The DB must be closed after use, by calling Close method.
func NewLevelDB(path string) (*LevelDB, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	return &LevelDB{inner: db}, nil
}

// Close closes the DB. This will also release any outstanding snapshots, abort any in-flight compaction, and discard any open transactions.
// It is not safe to close a DB until all outstanding iterators are released.
// It is valid to call Close multiple times. Other methods should not be called after the DB has been closed.
func (db *LevelDB) Close() {
	db.inner.Close()
}

// Get gets the value for the given key. It returns ErrNotFound if the DB does not contains the key.
// The returned slice is a copy, so it is safe to modify the contents of the returned slice.
// It is safe to modify the contents of the argument after Get returns.
func (db *LevelDB) Get(table string, key []byte) ([]byte, error) {
	return db.inner.Get(KeyFor(table, key), defaultReadOptions)
}

// Put sets the value for the given key. It overwrites any previous value for that key;
// When Put is used concurrently and the batch is small enough, leveldb will try to merge the batches.
// Set the NoWriteMerge option to true to disable this behavior.
// It is safe to modify the contents of the arguments after Put returns but not before.
func (db *LevelDB) Put(table string, k []byte, v []byte) error {
	return db.inner.Put(KeyFor(table, k), v, defaultWriteOptions)
}

// Delete deletes the value for the given key. Delete will not returns error if key doesn't exist.
// Write merge also applies to Delete. See the doc comment on Put for more information.
// It is safe to modify the contents of the arguments after Delete returns but not before.
func (db *LevelDB) Delete(table string, k []byte) error {
	return db.inner.Delete(KeyFor(table, k), defaultWriteOptions)
}

// Get all keys from start up to (but not including) end
// Remember that the contents of the returned slice should not be modified, and
// are only valid until the next call to Next.
func (db *LevelDB) GetRange(table string, start []byte, end []byte) DbIterator {
	return db.inner.NewIterator(&util.Range{
		Start: KeyFor(table, start),
		Limit: KeyFor(table, end),
	}, defaultReadOptions)
}

// Combine a tablename and key into a single string
func KeyFor(table string, key []byte) []byte {
	output := make([]byte, 0, len(table)+len(key))
	output = append(output, []byte(table)...)
	return append(output, key...)
}

var _ RangeDb = &LevelDB{}
