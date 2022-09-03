package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ledgerwatch/erigon/cmd/rpcdaemon/commands"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/ledgerwatch/erigon/core/types"
)

const VERSION_STRING string = "Regent/0.0.0"
const GENESIS_HASH_STRING = "0x03cbee8fac5256aa39823eb6437acf0918f2829e1775554cdd08f9519bf3e9e1"

var DEV_ADDRESS = common.HexToAddress("0x013068165Fe8257f960C6831745927f924b2dd0d")

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

func EmptyBlockHeader(prevhash common.Hash, number int64, gasLimit uint64, basefee int64) *types.Header {
	return &types.Header{
		ParentHash:  prevhash,
		UncleHash:   types.EmptyUncleHash,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash, // TODO: verify the receipt hash for an empty block
		Time:        uint64(time.Now().Unix()),
		Coinbase:    DEV_ADDRESS,
		GasLimit:    gasLimit,
		BaseFee:     big.NewInt(int64(basefee)),
		Number:      big.NewInt(int64(number)),
		Bloom:       types.CreateBloom(nil),
		Eip1559:     true,
	}
}

func BlockToExecutionPayload(block *types.Block) *commands.ExecutionPayload {
	txs := make([]hexutil.Bytes, 0, block.Transactions().Len())
	for _, tx := range block.Transactions() {
		out := hexutil.Bytes{}
		tx.MarshalBinary(bytes.NewBuffer(out))
		txs = append(txs, out)
	}
	return &commands.ExecutionPayload{
		ParentHash:    block.Header().ParentHash,
		FeeRecipient:  block.Header().Coinbase,
		StateRoot:     block.Header().Root,
		ReceiptsRoot:  block.ReceiptHash(),
		LogsBloom:     block.Bloom().Bytes(),
		PrevRandao:    common.Hash{}, // TODO: Add actual randoa
		BlockNumber:   hexutil.Uint64(block.NumberU64()),
		GasLimit:      hexutil.Uint64(block.Header().GasLimit),
		GasUsed:       hexutil.Uint64(block.Header().GasUsed),
		Timestamp:     hexutil.Uint64(block.Header().Time),
		ExtraData:     block.Extra(),
		BaseFeePerGas: (*hexutil.Big)(block.BaseFee()),
		BlockHash:     block.Header().Hash(),
		Transactions:  txs,
	}
}

func U64ToHex(num uint64) string {
	output := make([]byte, 0, 8)
	output = binary.BigEndian.AppendUint64(output, num)
	h := hexutil.Bytes(output)
	res, _ := h.MarshalText()
	return string(res)
}
