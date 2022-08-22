package common

import (
	"encoding/hex"
	"log"
	"strings"
)

const VERSION_STRING string = "Regent/0.0.0"

const GENESIS_HASH = "0x03cbee8fac5256aa39823eb6437acf0918f2829e1775554cdd08f9519bf3e9e1"

type Hash [32]byte
type Address = [20]byte

func FromHex(input string) []byte {
	input = strings.TrimPrefix(input, "0x")
	if len(input)%2 != 0 {
		input = "0" + input
	}
	result, err := hex.DecodeString(input)
	if err != nil {
		log.Fatal(err)
	}
	return result
}
