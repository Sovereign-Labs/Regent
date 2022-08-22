package main

import (
	"fmt"
	"os"
	"regent/utils"
	"time"

	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/log/v3"
)

const DEFAULT_ERIGON_DATADIR = "/Users/prestonevans/sovereign"
const JWT_SECRET_FILENAME = "jwt.hex"

var ErigonDatadir = DEFAULT_ERIGON_DATADIR
var EngineRpcPort string = "8551"

func main() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StderrHandler))

	regent, err := Initialize()
	if err != nil {
		log.Crit("Fatal error attempting to start app", "err", err)
	}

	// Finalize the genesis block and start the building process
	err = regent.ExtendChainAndStartBuilder(common.HexToHash(utils.GENESIS_HASH_STRING), utils.DEV_ADDRESS)
	if err != nil {
		log.Crit("Could not finalize genesis block", "err", err)
		os.Exit(1)
	}
	// Wait for one second to ensure that the next payload builds with a future timestamp
	time.Sleep(time.Second)

	err = regent.run()
	if err != nil {
		os.Exit(1)
	}

	fmt.Println("Done. Goodbye for now!")
}
