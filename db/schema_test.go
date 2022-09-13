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

func assertThrows[T any](f func() (T, error), expected error, t *testing.T) {
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

func TestIteratorOperations_emptyDb(t *testing.T) {
	db, err := levelDbFromInner(leveldb.Open(storage.NewMemStorage(), nil))
	if err != nil {
		t.Fatal("unable to create test db in tempdir", err)
	}
	defer db.Close()
	blocksIter := GetBlockHashIterator(db)
	defer blocksIter.inner.Release()

	if blocksIter.ResetToGenesis() != blocksIter.HeadHash() ||
		blocksIter.HeadNumber() != 0 {
		t.Fatal("empty database should return genesis as its head block")
	}

	res, err := blocksIter.Next()
	if res != utils.GENESIS_HASH || err != nil {
		t.Fatalf("Cursor should have been at the genesis block. failed with the following error: %s", err)
	}
	assertThrows(blocksIter.Next, ERR_EXHAUSTED, t)
	res, err = blocksIter.Prev()
	if res != utils.GENESIS_HASH || err != nil {
		t.Fatalf("Cursor should have been at the genesis block. failed with the following error: %s", err)
	}
	assertThrows(blocksIter.Prev, ERR_EXHAUSTED, t)

	res, err = blocksIter.Peek()
	if res != utils.GENESIS_HASH || err != nil {
		t.Fatalf("Cursor should have been at the genesis block. failed with the following error: %s", err)
	}

	blocknum := blocksIter.Position()
	if blocknum != 0 {
		t.Fatalf("Cursor should have been at height 0. failed with the following error: %s", err)
	}

	res, err = blocksIter.Seek(0)
	if res != utils.GENESIS_HASH || err != nil {
		t.Fatalf("Seek 0 should always succeed, but failed with the following error: %s", err)
	}

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

	for i := uint64(1); i < 10; i++ {
		PutRollupBlockHashWithNumber(db, crypto.Keccak256Hash(MarshalUint(i)), i)
	}

	blocksIter := GetBlockHashIterator(db)
	defer blocksIter.inner.Release()
	if blocksIter.ResetToGenesis() != check(blocksIter.Next()) {
		t.Fatal("Iterator should return genesis block first")
	}

	for i := uint64(1); i < 10; i++ {
		if blocksIter.Position() != i {
			t.Fatalf("Incorrect next height. expected %v. got %v", i, blocksIter.Position())
		}
		expected := crypto.Keccak256Hash(MarshalUint(i))
		if check(blocksIter.Peek()) != expected {
			t.Fatalf("Peek returned the wrong value. expected: %v. got: %v", expected, check(blocksIter.Peek()))
		}
		if expected != check(blocksIter.Next()) {
			t.Fatal("Peek and check must return the same value")
		}
	}
	assertThrows(blocksIter.Next, ERR_EXHAUSTED, t)
	assertThrows(blocksIter.Next, ERR_EXHAUSTED, t)

	for i := uint64(9); i > 0; i-- {
		expected := crypto.Keccak256Hash(MarshalUint(i))
		if blocksIter.Position() != i+1 {
			t.Fatalf("Incorrect next height. expected: %v. got: %v", i, blocksIter.Position())
		}
		got := check(blocksIter.Prev())
		if got != expected {
			t.Fatalf("i: %v. Prev returned the wrong value. expected: %v. got: %v", i, expected, got)
		}
	}
	current := check(blocksIter.Prev())
	if current != utils.GENESIS_HASH {
		t.Fatalf("Iterator should be back to genesis, but was at position %v with hash %v", blocksIter.Position(), current)
	}
	assertThrows(blocksIter.Prev, ERR_EXHAUSTED, t)
	assertThrows(blocksIter.Prev, ERR_EXHAUSTED, t)
	current = check(blocksIter.Next())
	if current != utils.GENESIS_HASH {
		t.Fatalf("Iterator should be back to genesis, but was at position %v with hash %v", blocksIter.Position(), current)
	}
}
