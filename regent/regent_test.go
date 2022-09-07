package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"regent/rpc"
	"regent/utils"
	"regent/utils/test"

	"github.com/ledgerwatch/erigon/common"
)

var TestRpcClient = rpc.NewClient("8545")
var TestRegent Regent

func init() {
	TestRpcClient.Endpoint = test.TestServer.URL
	rpc.DefaultRetryStrategy = func() rpc.RetryStrategy { return &test.NoRetryStrategy{} }
	TestRegent = Regent{
		EngineRpc: TestRpcClient,
	}
}

func TestExtendChainAndStartBuilder_successResponse(t *testing.T) {
	hash := common.HexToHash(utils.GENESIS_HASH_STRING)
	response, _ := json.Marshal(rpc.Response[rpc.ForkChoiceUpdatedResult]{
		Result: rpc.ForkChoiceUpdatedResult{PayloadStatus: &rpc.PayloadStatus{
			Status:          rpc.VALID_PAYLOAD,
			LatestValidHash: &hash,
		},
			PayloadId: "0x0000000000000001",
		}})
	test.TestHandler.Response = response

	err := TestRegent.ExtendChainAndStartBuilder(common.HexToHash(utils.GENESIS_HASH_STRING), utils.DEV_ADDRESS)
	if err != nil {
		t.Fatalf("ExtendChainAndStartBuilder - expected: %v, got: %v", nil, err)
	}
}

func TestExtendChainAndStartBuilder_syncingResponse(t *testing.T) {
	response := `{"result": {"payloadStatus": {"status": "SYNCING", "latestValidHash": null, "validationError": null}, "payloadId": null}}`
	test.TestHandler.Response = []byte(response)

	err := TestRegent.ExtendChainAndStartBuilder(common.HexToHash(utils.GENESIS_HASH_STRING), utils.DEV_ADDRESS)
	if !errors.Is(err, ERR_EXECUTION_CLIENT_SYNCING) || !errors.Is(err, ERR_FORKCHOICE_NOT_UPDATED) || !errors.Is(err, ERR_PAYLOAD_NOT_BUILT) {
		t.Fatalf("ExtendChainAndStartBuilder - expected: %v, got: %v", ERR_EXECUTION_CLIENT_SYNCING, err)
	}
}

func TestExtendChainAndStartBuilder_invalidPayloadResponse(t *testing.T) {
	response := `{"result": {"payloadStatus": {"status": "INVALID", "latestValidHash": null, "validationError": null}, "payloadId": null}}`
	test.TestHandler.Response = []byte(response)

	err := TestRegent.ExtendChainAndStartBuilder(common.HexToHash(utils.GENESIS_HASH_STRING), utils.DEV_ADDRESS)
	if !errors.Is(err, ERR_INVALID_PAYLOAD) || !errors.Is(err, ERR_FORKCHOICE_NOT_UPDATED) || !errors.Is(err, ERR_PAYLOAD_NOT_BUILT) {
		t.Fatalf("ExtendChainAndStartBuilder - expected: %v, got: %v", ERR_INVALID_PAYLOAD, err)
	}
}

func TestExtendChainAndStartBuilder_invalidPayloadId(t *testing.T) {
	response := `{"result": {"payloadStatus": {"status": "VALID", "latestValidHash": null, "validationError": null}, "payloadId": null}}`
	test.TestHandler.Response = []byte(response)

	err := TestRegent.ExtendChainAndStartBuilder(common.HexToHash(utils.GENESIS_HASH_STRING), utils.DEV_ADDRESS)
	if !errors.Is(err, ERR_INVALID_PAYLOAD_ID) || errors.Is(err, ERR_FORKCHOICE_NOT_UPDATED) || !errors.Is(err, ERR_PAYLOAD_NOT_BUILT) {
		t.Fatalf("ExtendChainAndStartBuilder - expected: %v, got: %v", ERR_INVALID_PAYLOAD_ID, err)
	}
}

func TestExtendChainAndStartBuilder_valid(t *testing.T) {
	response := `{"result": {"payloadStatus": {"status": "VALID", "latestValidHash": null, "validationError": null}, "payloadId": "0x0000000000000001"}}`
	test.TestHandler.Response = []byte(response)

	err := TestRegent.ExtendChainAndStartBuilder(common.HexToHash(utils.GENESIS_HASH_STRING), utils.DEV_ADDRESS)
	if err != nil {
		t.Fatalf("ExtendChainAndStartBuilder - expected: %v, got: %v", nil, err)
	}
}

func TestExtendChainAndStartBuilder_invalidForkchoice(t *testing.T) {
	response := `{"error": {"code": -38002, "message": "Invalid forkchoice state"}}`
	test.TestHandler.Response = []byte(response)

	err := TestRegent.ExtendChainAndStartBuilder(common.HexToHash(utils.GENESIS_HASH_STRING), utils.DEV_ADDRESS)
	if !errors.Is(err, ERR_INVALID_FORKCHOICE) || !errors.Is(err, ERR_FORKCHOICE_NOT_UPDATED) || !errors.Is(err, ERR_PAYLOAD_NOT_BUILT) {
		t.Fatalf("ExtendChainAndStartBuilder - expected: %v, got: %v", ERR_INVALID_FORKCHOICE, err)
	}
}

func TestExtendChainAndStartBuilder_noPayloadStatus(t *testing.T) {
	response := `{"result": {"payloadStatus": null, "payloadId": "0x0000000000000001"}}`
	test.TestHandler.Response = []byte(response)

	err := TestRegent.ExtendChainAndStartBuilder(common.HexToHash(utils.GENESIS_HASH_STRING), utils.DEV_ADDRESS)
	if !errors.Is(err, ERR_INVALID_PAYLOAD_STATUS) || !errors.Is(err, ERR_FORKCHOICE_NOT_UPDATED) || !errors.Is(err, ERR_PAYLOAD_NOT_BUILT) {
		t.Fatalf("ExtendChainAndStartBuilder - expected: %v, got: %v", ERR_INVALID_PAYLOAD_STATUS, err)
	}
}

func TestExtendChainAndStartBuilder_invalidPayloadAttributes(t *testing.T) {
	response := `{"error": {"code": -38003, "message": "Invalid payload attributes"}}`
	test.TestHandler.Response = []byte(response)

	err := TestRegent.ExtendChainAndStartBuilder(common.HexToHash(utils.GENESIS_HASH_STRING), utils.DEV_ADDRESS)
	if !errors.Is(err, ERR_INVALID_TIMESTAMP) || !errors.Is(err, ERR_PAYLOAD_NOT_BUILT) || errors.Is(err, ERR_FORKCHOICE_NOT_UPDATED) {
		t.Fatalf("ExtendChainAndStartBuilder - expected: %v, got: %v", ERR_INVALID_TIMESTAMP, err)
	}
}

func TestExtendChainAndStartBuilder_invalidPayloadStatus(t *testing.T) {
	response := `{"result": {"payloadStatus": {"status": "INVALID_BLOCK_HASH", "latestValidHash": null, "validationError": null}, "payloadId": "0x0000000000000001"}}`
	test.TestHandler.Response = []byte(response)

	err := TestRegent.ExtendChainAndStartBuilder(common.HexToHash(utils.GENESIS_HASH_STRING), utils.DEV_ADDRESS)
	if !errors.Is(err, ERR_INVALID_PAYLOAD_STATUS) {
		t.Fatalf("ExtendChainAndStartBuilder - expected: %v, got: %v", ERR_INVALID_PAYLOAD_STATUS, err)
	}
}

func TestExtendChainAndStartBuilder_errorReadingBody(t *testing.T) {
	test.TestHandler.HandlerFunc = func(resp http.ResponseWriter, req *http.Request) {
		test.TestServer.CloseClientConnections()
	}
	defer func() { test.TestHandler.HandlerFunc = nil }()

	err := TestRegent.ExtendChainAndStartBuilder(common.HexToHash(utils.GENESIS_HASH_STRING), utils.DEV_ADDRESS)
	if !errors.Is(err, ERR_FORKCHOICE_NOT_UPDATED) || !errors.Is(err, ERR_PAYLOAD_NOT_BUILT) {
		t.Fatalf("ExtendChainAndStartBuilder - expected: %v, got: %v", ERR_FORKCHOICE_NOT_UPDATED, err)
	}
}

func TestExtendChainAndStartBuilder_executionClientError(t *testing.T) {
	response := `{"error": {"code": -32000, "message": "Generic client error while processing request"}}`
	test.TestHandler.Response = []byte(response)

	err := TestRegent.ExtendChainAndStartBuilder(common.HexToHash(utils.GENESIS_HASH_STRING), utils.DEV_ADDRESS)
	if !errors.Is(err, ERR_PAYLOAD_NOT_BUILT) || !errors.Is(err, ERR_FORKCHOICE_NOT_UPDATED) {
		t.Fatalf("ExtendChainAndStartBuilder - expected: %v, got: %v", ERR_FORKCHOICE_NOT_UPDATED, err)
	}
}
