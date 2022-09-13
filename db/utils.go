package db

import (
	"encoding/binary"

	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/log/v3"
)

// Marshall a uint64 into a slice of 8 bytes in BigEndian order
func MarshalUint(num uint64) []byte {
	out := make([]byte, 0, 8)
	return binary.BigEndian.AppendUint64(out, num)
}

// Unmarshall a slice of 8 bytes into a uint64. Return an error
// if the length of the slice is not equal to 8.
// Uses BigEndian order
func UnmarshallUint(raw []byte) (uint64, error) {
	if len(raw) != serializedUint64Len {
		return 0, ERR_INVALID_U64
	}
	return binary.BigEndian.Uint64(raw), nil
}

// Unmarshall a slice of 32 bytes into a Hash. Return an error
// if the length of the slice is not equal to 32
func UnmarshallHash(raw []byte) (common.Hash, error) {
	output := common.Hash{}
	if len(raw) != serializedHashLen {
		return output, ERR_INVALID_BLOCK_HASH
	}
	copy(output[:], raw)
	return output, nil
}

// Convert a tuple of (result, error) into only a result by panicking if the error is non-nil
// This function should only be used in situations where the error is both unexpected and unrecoverable
// such as when database corruption occurs.
func check[T any](res T, err error) T {
	if err != nil {
		log.Crit("An unrecoverable error occurred. Panicking", "err", err)
		panic(err)
	}
	return res
}
