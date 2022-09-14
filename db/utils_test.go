package db

import (
	"errors"
	"testing"
)

func TestUnmarshalUint_wrongLen(t *testing.T) {
	raw := make([]byte, 50)
	_, err := UnmarshallUint(raw)
	if err == nil || !errors.Is(err, ERR_INVALID_U64) {
		t.Fatalf("unmarshalling a uint64 must fail if the slice length is not 8. expected: %v. got: %v.", ERR_INVALID_U64, err)
	}
}

func TestUnmarshalHash_wrongLen(t *testing.T) {
	raw := make([]byte, 50)
	_, err := UnmarshallHash(raw)
	if err == nil || !errors.Is(err, ERR_INVALID_BLOCK_HASH) {
		t.Fatalf("unmarshalling a uint64 must fail if the slice length is not 8. expected: %v. got: %v.", ERR_INVALID_BLOCK_HASH, err)
	}
}
