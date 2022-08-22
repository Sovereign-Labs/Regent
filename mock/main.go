package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regent/rpc"
	"regent/rpc/jwt"

	"github.com/ledgerwatch/log/v3"
)

const DEFAULT_ERIGON_DATADIR = "/Users/prestonevans/sovereign"
const JWT_SECRET_FILENAME = "jwt.hex"

var ErigonDatadir = DEFAULT_ERIGON_DATADIR
var EngineRpc rpc.Client
var EngineRpcPort string = "8551"

func main() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StderrHandler))

	token, err := jwt.FromSecretFile(path.Join(ErigonDatadir, JWT_SECRET_FILENAME))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	EngineRpc = rpc.NewClient(EngineRpcPort)
	EngineRpc.SetAuthToken(token)

	response, err := AddBlockToChain("0xfeedbeeffeedbeef8ea0b98a3409290e39dce6be7f558493aeb6e4b99a172a87")
	if err != nil {
		fmt.Println(err)
	} else {
		printable, _ := json.Marshal(response)
		fmt.Println(string(printable))
	}

}
