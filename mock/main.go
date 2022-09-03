package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regent/rpc"
	"regent/rpc/jwt"
	"regent/utils"
	"time"

	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/log/v3"
)

const DEFAULT_ERIGON_DATADIR = "/Users/prestonevans/sovereign"
const JWT_SECRET_FILENAME = "jwt.hex"

var ErigonDatadir = DEFAULT_ERIGON_DATADIR
var EngineRpc rpc.Client
var EngineRpcPort string = "8551"

func mainLoop() error {
	log.Info("Starting block production loop")
	currentBlockNumber := 0
	currentBlockHash := common.HexToHash(utils.GENESIS_HASH_STRING)

	for {
		response, payload, err := EngineRpc.SendGetPayload(uint64(currentBlockNumber) + 1)
		log.Info("Getting next execution payload")
		if err != nil {
			fmt.Println(err)
		} else {
			printable, _ := json.Marshal(response)
			fmt.Println(string(printable))
		}

		log.Info("Sending next payload to execution client", "blockhash", currentBlockHash)
		response, err = EngineRpc.SendNewPayload(payload)
		if err != nil {
			fmt.Println(err)
		} else {
			printable, _ := json.Marshal(response)
			fmt.Println(string(printable))
		}

		currentBlockHash = payload.BlockHash

		log.Info("Updating head", "blockhash", currentBlockHash)
		response, err = AddBlockToChain(currentBlockHash)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		} else {
			printable, _ := json.Marshal(response)
			fmt.Println(string(printable))
		}

		currentBlockNumber += 1

		log.Info("Waiting for next slot")
		time.Sleep(5 * time.Second)
		log.Info("Done waiting", "slot number", currentBlockNumber+1)
	}
}

func startRpc() {
	EngineRpc = rpc.NewClient(EngineRpcPort)
	token, err := jwt.FromSecretFile(path.Join(ErigonDatadir, JWT_SECRET_FILENAME))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	EngineRpc.SetAuthToken(token)
}

func main() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StderrHandler))
	startRpc()

	// Finalize the genesis block and start the building process
	response, err := AddBlockToChain(common.HexToHash(utils.GENESIS_HASH_STRING))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	} else {
		printable, _ := json.Marshal(response)
		fmt.Println(string(printable))
	}

	time.Sleep(time.Second)

	err = mainLoop()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Done. Goodbye for now!")

	// nextHeader := utils.EmptyBlockHeader(common.HexToHash("03cbee8fac5256aa39823eb6437acf0918f2829e1775554cdd08f9519bf3e9e1"), 1, 30000000, 875000000)
	// nextHeader.Root = common.HexToHash("862fbac79527db3a6c72192e64146ed4fbfda1acdbedc7b9d673b4066ae54313")
	// block := types.NewBlockWithHeader(nextHeader)
	// payload := utils.BlockToExecutionPayload(block)
	// nextBlockhash := nextHeader.Hash()

	// response, err := EngineRpc.SendNewPayload(payload)
	// if err != nil {
	// 	fmt.Println(err)
	// } else {
	// 	printable, _ := json.Marshal(response)
	// 	fmt.Print("Sending new payload")
	// 	fmt.Println(string(printable))
	// }

	// response, err = AddBlockToChain(block.Hash().String())
	// if err != nil {
	// 	fmt.Println(err)
	// } else {
	// 	printable, _ := json.Marshal(response)
	// 	fmt.Println(string(printable))
	// }

	// response, err = EngineRpc.SendGetPayload(1)
	// fmt.Print("Sending get payload")
	// if err != nil {
	// 	fmt.Println(err)
	// } else {
	// 	printable, _ := json.Marshal(response)
	// 	fmt.Println(string(printable))
	// }
	// fmt.Print(nextBlockhash)

	// // w := bytes.NewBuffer(make([]byte, 0, 100))
	// // secondHeader.EncodeRLP(w)

	// // fmt.Println(common.Bytes2Hex(w.Bytes()))
	// // fmt.Println(secondHeader.Hash())
}
