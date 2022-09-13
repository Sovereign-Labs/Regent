package db

import (
	"errors"
	"os"
	"regent/utils"
	"testing"

	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/crypto"
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
	db, err := NewLevelDB(os.TempDir())
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
	db, err := NewLevelDB(os.TempDir())
	if err != nil {
		t.Fatal("unable to create test db in tempdir", err)
	}
	defer db.Close()

	blocksIter := GetBlockHashIterator(db)
	defer blocksIter.inner.Release()
	if blocksIter.Genesis() != blocksIter.HeadHash() || blocksIter.HeadNumber() != 0 {
		t.Fatal("empty database should return genesis as its head block")
	}
	assertThrows(blocksIter.Next, ERR_EXHAUSTED, t)
	assertThrows(blocksIter.Prev, ERR_EXHAUSTED, t)
	assertThrows(blocksIter.CursorHash, ERR_EXHAUSTED, t)
	assertThrows(blocksIter.CursorHeight, ERR_EXHAUSTED, t)

	res, err := blocksIter.Seek(0)
	if res != utils.GENESIS_HASH || err != nil {
		t.Fatalf("Seek 0 should always succeed, but failed with the following error: %s", err)
	}

	_, err = blocksIter.Seek(1)
	if err == nil || !errors.Is(err, ERR_NOT_FOUND) || !errors.Is(err, ERR_PAST_HEAD) {
		t.Fatalf("Seek 1 failed with the wrong error. expected: %v. got: %s", ERR_PAST_HEAD, ERR_MISSING)
	}
}

func TestIteratorOperations_completeDb(t *testing.T) {
	db, err := NewLevelDB(os.TempDir())
	if err != nil {
		t.Fatal("unable to create test db in tempdir", err)
	}
	defer db.Close()

	blocksIter := GetBlockHashIterator(db)
	if blocksIter.Genesis() != blocksIter.HeadHash() || blocksIter.HeadNumber() != 0 {
		t.Fatal("empty database should return genesis as its head block")
	}
	assertThrows(blocksIter.Next, ERR_EXHAUSTED, t)
	assertThrows(blocksIter.Prev, ERR_EXHAUSTED, t)
	assertThrows(blocksIter.CursorHash, ERR_EXHAUSTED, t)
	assertThrows(blocksIter.CursorHeight, ERR_EXHAUSTED, t)

	res, err := blocksIter.Seek(0)
	if res != utils.GENESIS_HASH || err != nil {
		t.Fatalf("Seek 0 should always succeed, but failed with the following error: %s", err)
	}

	_, err = blocksIter.Seek(1)
	if err == nil || !errors.Is(err, ERR_NOT_FOUND) || !errors.Is(err, ERR_PAST_HEAD) {
		t.Fatalf("Seek 1 failed with the wrong error. expected: %v. got: %s", ERR_PAST_HEAD, ERR_MISSING)
	}
}
