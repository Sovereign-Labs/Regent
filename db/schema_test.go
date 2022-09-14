package db

import (
	"errors"
	"regent/utils"
	"testing"

	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/crypto"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

var testHashes []common.Hash

func init() {
	for i := uint64(0); i < 10; i++ {
		testHashes = append(testHashes, crypto.Keccak256Hash(MarshalUint(i)))
	}
}

func assertErrs[T any](f func() (T, error), expected error, t *testing.T) {
	_, err := f()
	if !errors.Is(err, expected) {
		t.Fatalf("function should have failed with %v but returned the following err: %v", expected, err)
	}
}

func testRead(db SimpleDb, expectedHash common.Hash, expectedNumber uint64, t *testing.T) {
	actualHash, err := GetRollupBlockHash(db, expectedNumber)
	if err != nil {
		t.Fatal("unable to retrieve block hash from db", err)
	}
	if actualHash != expectedHash {
		t.Fatalf("incorrect genesis hash. expected: %v. got: %v", expectedHash, actualHash)
	}

	actualNumber, err := GetRollupBlockNumber(db, expectedHash)
	if err != nil {
		t.Fatal("unable to retrieve block from db", err)
	}
	if actualNumber != expectedNumber {
		t.Fatalf("incorrect block number. expected: %v. got: %v", expectedNumber, actualNumber)
	}
}

func testDelete(db SimpleDb, hash common.Hash, number uint64, t *testing.T) {
	err := db.Delete(RollupBlockNumberToHash, MarshalUint(number))
	if err != nil {
		t.Fatal("unable to delete block number from db:", number)
	}
	db.Delete(RollupBlockHashToNumber, hash[:])
	if err != nil {
		t.Fatal("unable to delete block hash from db:", hash)
	}

	_, err = GetRollupBlockHash(db, number)
	if err == nil || !errors.Is(err, ERR_NOT_FOUND) {
		t.Fatalf("deletion failed. lookup receieved: %v. expected: %v.", err, ERR_NOT_FOUND)
	}

	_, err = GetRollupBlockNumber(db, hash)
	if err == nil || !errors.Is(err, ERR_NOT_FOUND) {
		t.Fatalf("deletion failed. lookup receieved: %v. expected: %v.", err, ERR_NOT_FOUND)
	}
}

func testCrud(db SimpleDb, hash common.Hash, number uint64, t *testing.T) {
	// Insert the items and read them back
	err := PutRollupBlockHashWithNumber(db, hash, number)
	if err != nil {
		t.Fatal("unable to insert block hash into db:", err)
	}
	testRead(db, hash, number, t)

	// Change the hash and read the items back again
	updatedHash := crypto.Keccak256Hash(hash[:])
	err = PutRollupBlockHashWithNumber(db, updatedHash, number)
	if err != nil {
		t.Fatal("unable to update block hash in db:", err)
	}
	testRead(db, updatedHash, number, t)
	// Delete the items, and verify that they disappear
	testDelete(db, hash, number, t)
}

func TestCrudOperations(t *testing.T) {
	db, err := levelDbFromInner(leveldb.Open(storage.NewMemStorage(), nil))
	if err != nil {
		t.Fatal("unable to create test db in tempdir", err)
	}
	defer db.Close()

	// Run all CRUD operations on the genesis hash (which may be special cased in impls)
	testCrud(db, utils.GENESIS_HASH, 0, t)
	// Run all CRUD operations on a different hash
	testCrud(db, crypto.Keccak256Hash(MarshalUint(100)), 100, t)
}

// Test that a fresh DB contains the genesis block and behaves as expected
func TestIteratorOperations_emptyDb(t *testing.T) {
	db, err := levelDbFromInner(leveldb.Open(storage.NewMemStorage(), nil))
	if err != nil {
		t.Fatal("unable to create test db in tempdir", err)
	}
	defer db.Close()
	blocksIter := GetBlockHashIterator(db)
	defer blocksIter.inner.Release()

	// The genesis block should be equal to the head block
	if blocksIter.Genesis() != blocksIter.Head() ||
		blocksIter.HeadNumber() != 0 {
		t.Fatal("empty database should return genesis as its head block")
	}
	// The first call to next should return the genesis block and move us to the `done` marker
	res, err := blocksIter.Next()
	if res != utils.GENESIS_HASH || err != nil {
		t.Fatalf("Cursor should have been at the genesis block. failed with the following error: %s", err)
	}
	// Any susbsequent calls to next should not move the cursor or return a value
	assertErrs(blocksIter.Next, ERR_EXHAUSTED, t)
	assertErrs(blocksIter.Next, ERR_EXHAUSTED, t)
	if int(blocksIter.Position()) != 1 {
		t.Fatalf("The cursor should not keep moving past the head of the chain. expected position: %v got: %v", 1, blocksIter.Position())
	}

	// The first call to Prev should get us back to the genesis block
	res, err = blocksIter.Prev()
	if res != utils.GENESIS_HASH || err != nil {
		t.Fatalf("Cursor should have been at the genesis block. failed with the following error: %s", err)
	}

	// Any additional calls to Prev should should not move the cursor beyond the beginning of the array
	assertErrs(blocksIter.Prev, ERR_EXHAUSTED, t)
	assertErrs(blocksIter.Prev, ERR_EXHAUSTED, t)
	if int(blocksIter.Position()) != 0 {
		t.Fatalf("The cursor should not keep moving before the genesis block. expected position: %v got: %v", 0, blocksIter.Position())
	}

	res, err = blocksIter.Peek()
	if res != utils.GENESIS_HASH || err != nil {
		t.Fatalf("Cursor should have been at the genesis block. failed with the following error: %s", err)
	}

	// After calling peeek the cursor should be in the same place
	blocknum := blocksIter.Position()
	if blocknum != 0 {
		t.Fatalf("Cursor should have been at height 0. failed with the following error: %s", err)
	}

	// Ensure that Seek is able to get us to the genesis block
	res, err = blocksIter.Seek(0)
	if res != utils.GENESIS_HASH || err != nil {
		t.Fatalf("Seek 0 should always succeed, but failed with the following error: %s", err)
	}

	// Seeking the next block should fail
	_, err = blocksIter.Seek(1)
	if err == nil || !errors.Is(err, ERR_NOT_FOUND) || !errors.Is(err, ERR_PAST_HEAD) {
		t.Fatalf("Seek 1 failed with the wrong error. expected: %v. got: %s", ERR_PAST_HEAD, err)
	}
}

func TestIteratorOperations_completeDb(t *testing.T) {
	db, err := levelDbFromInner(leveldb.Open(storage.NewMemStorage(), nil))
	if err != nil {
		t.Fatal("unable to create test db in tempdir", err)
	}
	defer db.Close()

	// Overwrite the genesis block and add some additional test hashes to the db
	for i, hash := range testHashes {
		PutRollupBlockHashWithNumber(db, hash, uint64(i))
	}

	blocksIter := GetBlockHashIterator(db)
	defer blocksIter.inner.Release()

	// Check each method while iterating forwards
	for idx, expected := range testHashes {
		i := uint64(idx)
		if blocksIter.Position() != i {
			t.Fatalf("Incorrect next height. expected %v. got %v", i, blocksIter.Position())
		}
		got := unwrap(blocksIter.Peek())
		if got != expected {
			t.Fatalf("Peek returned the wrong value. expected: %v. got: %v", expected, got)
		}
		got = unwrap(blocksIter.Next())
		if expected != got {
			t.Fatalf("Peek and next must return the same value. expected: %v. got: %v", expected, got)
		}
	}
	assertErrs(blocksIter.Next, ERR_EXHAUSTED, t)
	assertErrs(blocksIter.Peek, ERR_EXHAUSTED, t)

	// Check each method while iterating backwards
	for i := uint64(len(testHashes)); i != 0; i-- {
		expected := testHashes[i-1]
		if blocksIter.Position() != i {
			t.Fatalf("Incorrect next height. expected: %v. got: %v", i, blocksIter.Position())
		}
		got := unwrap(blocksIter.Prev())
		if got != expected {
			t.Fatalf("i: %v. Prev returned the wrong value. expected: %v. got: %v", i, expected, got)
		}
	}
	assertErrs(blocksIter.Prev, ERR_EXHAUSTED, t)

	got := unwrap(blocksIter.Peek())
	if got != testHashes[0] {
		t.Fatalf("Iterator should be back to genesis, but was at position %v with hash %v", blocksIter.Position(), got)
	}
}

func TestBlocksIetartor_SeekMissingValue(t *testing.T) {
	db, err := levelDbFromInner(leveldb.Open(storage.NewMemStorage(), nil))
	if err != nil {
		t.Fatal("unable to create test db in tempdir", err)
	}
	defer db.Close()

	err = PutRollupBlockHashWithNumber(db, testHashes[5], 5)
	if err != nil {
		t.Fatal("insertion into an open db should succeed but failed with err:", err)
	}

	blocksIter := GetBlockHashIterator(db)
	defer blocksIter.inner.Release()

	_, err = blocksIter.Seek(3)
	if err == nil || !errors.Is(err, ERR_NOT_FOUND) || !errors.Is(err, ERR_MISSING) {
		t.Fatalf("seeking a missing block must return an error. expected: %v. got: %v.", ERR_MISSING, err)
	}
}

func TestInsertIntoClosedDb(t *testing.T) {
	db, err := levelDbFromInner(leveldb.Open(storage.NewMemStorage(), nil))
	if err != nil {
		t.Fatal("unable to create test db in tempdir", err)
	}
	db.Close()

	err = PutRollupBlockHashWithNumber(db, utils.GENESIS_HASH, 0)
	if err == nil {
		t.Fatal("inserting into a closed db must throw")
	}
}

func TestExtractBlockNumFromKey_notPrefixedWithKey(t *testing.T) {
	_, err := extractBlockNumFromKey(make([]byte, 8))
	if err == nil || !errors.Is(err, ERR_INVALID_U64) {
		t.Fatalf("must not remove a non-existent prefix. expected err: %v. got %v", ERR_INVALID_U64, err)
	}
}
