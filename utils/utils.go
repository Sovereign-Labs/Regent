package utils

import (
	"encoding/hex"
	"log"
	"strings"

	"github.com/ledgerwatch/erigon/common"
)

const VERSION_STRING string = "Regent/0.0.0"
const GENESIS_HASH_STRING = "0x03cbee8fac5256aa39823eb6437acf0918f2829e1775554cdd08f9519bf3e9e1"

var GENESIS_HASH = common.HexToHash("0x03cbee8fac5256aa39823eb6437acf0918f2829e1775554cdd08f9519bf3e9e1")
var DEV_ADDRESS = common.HexToAddress("0x013068165Fe8257f960C6831745927f924b2dd0d")

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
