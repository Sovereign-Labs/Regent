package rpc

import (
	"encoding/json"
	"net/http"
	"regent/utils/test"
	"testing"
	"time"

	"github.com/ledgerwatch/erigon/cmd/rpcdaemon/commands"
)

var TestRpcClient = NewClientWithJwt("8545", make([]byte, 32))

func init() {
	TestRpcClient.Endpoint = test.TestServer.URL
}

func TestUpdateForkChoice_emptyResponse(t *testing.T) {
	test.TestHandler.Response = make([]byte, 0)
	_, err := TestRpcClient.UpdateForkChoice(&commands.ForkChoiceState{})
	if !test.ErrorIs(err, ERR_UNMARSHALLING_FAILED) {
		t.Fatalf("UpdateForkChoice - expected %v, got %v", ERR_UNMARSHALLING_FAILED, err)
	}
}

func TestUpdateForkChoice_wrongResponseMessage(t *testing.T) {
	// Return request message instead of response
	test.TestHandler.Response = []byte(`{"jsonrpc":"2.0","method":"engine_forkchoiceUpdatedV1","params":[{"headBlockHash":"0x0000000000000000000000000000000000000000000000000000000000000000","safeBlockHash":"0x0000000000000000000000000000000000000000000000000000000000000000","finalizedBlockHash":"0x0000000000000000000000000000000000000000000000000000000000000000"}],"id":1}`)
	_, err := TestRpcClient.UpdateForkChoice(&commands.ForkChoiceState{})
	if !test.ErrorIs(err, ERR_UNMARSHALLING_FAILED) {
		t.Fatalf("UpdateForkChoice - expected %v, got %v", ERR_UNMARSHALLING_FAILED, err)
	}
}

func TestUpdateForkChoice_invalidEndpoint(t *testing.T) {
	// Disable retries and override the handler function
	previousRetryStrategy := DefaultRetryStrategy
	DefaultRetryStrategy = func() RetryStrategy { return &test.NoRetryStrategy{} }
	TestRpcClient.Endpoint = "not-a-url"
	defer func() {
		DefaultRetryStrategy = previousRetryStrategy
		TestRpcClient.Endpoint = test.TestServer.URL
	}()
	_, err := TestRpcClient.UpdateForkChoice(&commands.ForkChoiceState{})
	if !test.ErrorIs(err, ERR_REQUEST_SEND_FAILED) {
		t.Fatalf("UpdateForkChoice - expected %v, got %v", ERR_REQUEST_SEND_FAILED, err)
	}
}

func TestUpdateForkChoice_unexpectedEOF(t *testing.T) {
	// Disable retries and override the handler function
	previousRetryStrategy := DefaultRetryStrategy
	DefaultRetryStrategy = func() RetryStrategy { return &test.NoRetryStrategy{} }
	test.TestHandler.HandlerFunc = func(resp http.ResponseWriter, req *http.Request) {
		test.TestServer.CloseClientConnections()
	}
	// Restore state after the test ends
	defer func() {
		DefaultRetryStrategy = previousRetryStrategy
		test.TestHandler.HandlerFunc = nil
	}()

	_, err := TestRpcClient.UpdateForkChoice(&commands.ForkChoiceState{})
	if !test.ErrorIs(err, ERR_RESPONSE_READ_FAILED) {
		t.Fatalf("UpdateForkChoice - expected %v, got %v", ERR_RESPONSE_READ_FAILED, err)
	}
}

func TestInfiniteRetryStrategy_isLinearWithCap(t *testing.T) {
	var nextSleep time.Duration
	s := InfiniteRetryStrategy{}
	for i := 0; i < 30; i++ {
		nextSleep = s.Next()
		if s.Done() || nextSleep != time.Duration(i+1)*time.Second {
			t.Fatalf("Retry strategy failed: expected %d, got %d", time.Duration(i+1)*time.Second, nextSleep)
		}
	}
	nextSleep = s.Next()
	if nextSleep != 30*time.Second || s.Done() {
		t.Fatalf("Retry strategy failed: expected %d, got %d", 30*time.Second, nextSleep)
	}
}

func TestUpdateForkChoice_success(t *testing.T) {
	resp, _ := json.Marshal(Response[ForkChoiceUpdatedResult]{})
	test.TestHandler.Response = []byte(resp)
	_, err := TestRpcClient.UpdateForkChoice(&commands.ForkChoiceState{})
	if err != nil {
		t.Fatalf("UpdateForkChoice - expected %v, got %v", nil, err)
	}
}

func TestUpdateForkChoiceAndBuildBlock_success(t *testing.T) {
	resp, _ := json.Marshal(Response[ForkChoiceUpdatedResult]{})
	test.TestHandler.Response = []byte(resp)
	_, err := TestRpcClient.UpdateForkChoiceAndBuildBlock(&commands.ForkChoiceState{}, &commands.PayloadAttributes{})
	if err != nil {
		t.Fatalf("UpdateForkChoiceAndBuildBlock - expected %v, got %v", nil, err)
	}
}

func TestUpdateForkChoiceAndBuildBlock_invalidResponse(t *testing.T) {
	resp, _ := json.Marshal(Response[*commands.ExecutionPayload]{})
	test.TestHandler.Response = []byte(resp)
	_, err := TestRpcClient.UpdateForkChoiceAndBuildBlock(&commands.ForkChoiceState{}, &commands.PayloadAttributes{})
	if !test.ErrorIs(err, ERR_UNMARSHALLING_FAILED) {
		t.Fatalf("UpdateForkChoiceAndBuildBlock - expected %v, got %v", ERR_UNMARSHALLING_FAILED, err)
	}
}

func TestSendExecutionPayload_success(t *testing.T) {
	resp, _ := json.Marshal(Response[commands.PayloadAttributes]{})
	test.TestHandler.Response = []byte(resp)
	_, err := TestRpcClient.SendExecutionPayload(&commands.ExecutionPayload{})
	if err != nil {
		t.Fatalf("SendExecutionPayload - expected %v, got %v", nil, err)
	}
}

func TestSendExecutionPayload_invalidResponse(t *testing.T) {
	resp, _ := json.Marshal(Response[*commands.ExecutionPayload]{})
	test.TestHandler.Response = []byte(resp)
	_, err := TestRpcClient.SendExecutionPayload(&commands.ExecutionPayload{})
	if !test.ErrorIs(err, ERR_UNMARSHALLING_FAILED) {
		t.Fatalf("SendExecutionPayload - expected %v, got %v", ERR_UNMARSHALLING_FAILED, err)
	}
}

func TestGetPayload_success(t *testing.T) {
	resp, _ := json.Marshal(Response[*commands.ExecutionPayload]{
		Result: new(commands.ExecutionPayload),
	})
	test.TestHandler.Response = []byte(resp)
	_, err := TestRpcClient.GetPayload("0x0000000000000000")
	if err != nil {
		t.Fatalf("GetPayload - expected %v, got %v", nil, err)
	}
}

func TestGetPayload_invalidResponse(t *testing.T) {
	resp, _ := json.Marshal(Response[int]{})
	test.TestHandler.Response = []byte(resp)
	result, err := TestRpcClient.GetPayload("0x0000000000000000")
	if !test.ErrorIs(err, ERR_UNMARSHALLING_FAILED) {
		t.Error("result: ", result)
		t.Fatalf("GetPayload - expected %v, got %v", ERR_UNMARSHALLING_FAILED, err)
	}
}
