package rpc

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ledgerwatch/erigon/cmd/rpcdaemon/commands"
)

var handler = MockHandler{}
var rpcClient = NewClientWithJwt("8545", make([]byte, 30))
var testServer = httptest.NewServer(&handler)

type NoRetryStrategy struct {
	HasTried bool
}

func (s *NoRetryStrategy) Next() time.Duration {
	return time.Duration(0)
}

func (s *NoRetryStrategy) Done() bool {
	if !s.HasTried {
		s.HasTried = true
		return false
	}
	return true
}

func init() {
	rpcClient.endpoint = testServer.URL
}

type MockHandler struct {
	Response    []byte
	HandlerFunc func(resp http.ResponseWriter, req *http.Request)
}

func (m *MockHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if m.HandlerFunc != nil {
		m.HandlerFunc(resp, req)
		return
	}
	resp.WriteHeader(200)
	resp.Write(m.Response)
}

func ErrorIs(err error, kind string) bool {
	return errors.Is(err, errors.New(kind))
}

func TestUpdateForkChoice_emptyResponse(t *testing.T) {
	handler.Response = make([]byte, 0)
	_, err := rpcClient.UpdateForkChoice(&commands.ForkChoiceState{})
	if !ErrorIs(err, ERR_UNMARSHALLING_FAILED) {
		t.Fatalf("UpdateForkChoice - expected %v, got %v", ERR_UNMARSHALLING_FAILED, err)
	}
}

func TestUpdateForkChoice_wrongResponseMessage(t *testing.T) {
	handler.Response = []byte(`{"jsonrpc":"2.0","method":"engine_forkchoiceUpdatedV1","params":[{"headBlockHash":"0x0000000000000000000000000000000000000000000000000000000000000000","safeBlockHash":"0x0000000000000000000000000000000000000000000000000000000000000000","finalizedBlockHash":"0x0000000000000000000000000000000000000000000000000000000000000000"}],"id":1}`)
	_, err := rpcClient.UpdateForkChoice(&commands.ForkChoiceState{})
	if !ErrorIs(err, ERR_UNMARSHALLING_FAILED) {
		t.Fatalf("UpdateForkChoice - expected %v, got %v", ERR_UNMARSHALLING_FAILED, err)
	}
}

func TestUpdateForkChoice_invalidEndpoint(t *testing.T) {
	// Disable retries and override the handler function
	previousRetryStrategy := DefaultRetryStrategy
	DefaultRetryStrategy = func() RetryStrategy { return &NoRetryStrategy{} }
	rpcClient.endpoint = "not-a-url"
	defer func() {
		DefaultRetryStrategy = previousRetryStrategy
		rpcClient.endpoint = testServer.URL
	}()
	_, err := rpcClient.UpdateForkChoice(&commands.ForkChoiceState{})
	if !ErrorIs(err, ERR_REQUEST_SEND_FAILED) {
		t.Fatalf("UpdateForkChoice - expected %v, got %v", ERR_REQUEST_SEND_FAILED, err)
	}
}

func TestUpdateForkChoice_unexpectedEOF(t *testing.T) {
	// Disable retries and override the handler function
	previousRetryStrategy := DefaultRetryStrategy
	DefaultRetryStrategy = func() RetryStrategy { return &NoRetryStrategy{} }
	handler.HandlerFunc = func(resp http.ResponseWriter, req *http.Request) {
		testServer.CloseClientConnections()
	}
	// Restore state after the test ends
	defer func() {
		DefaultRetryStrategy = previousRetryStrategy
		handler.HandlerFunc = nil
	}()

	_, err := rpcClient.UpdateForkChoice(&commands.ForkChoiceState{})
	if !ErrorIs(err, ERR_RESPONSE_READ_FAILED) {
		t.Fatalf("UpdateForkChoice - expected %v, got %v", ERR_RESPONSE_READ_FAILED, err)
	}
}

func TestInfiniteRetryStrategy_isLinearWithCap(t *testing.T) {
	var nextSleep time.Duration
	s := InfiniteRetryStrategy{}
	for i := 0; i <= 30; i++ {
		nextSleep = s.Next()
		if s.Done() || nextSleep != time.Duration(i)*time.Second {
			t.Fatalf("Retry strategy failed: expected %d, got %d", time.Duration(i)*time.Second, nextSleep)
		}
	}
	nextSleep = s.Next()
	if nextSleep != 30*time.Second || s.Done() {
		t.Fatalf("Retry strategy failed: expected %d, got %d", 30*time.Second, nextSleep)
	}
}

func TestUpdateForkChoice_success(t *testing.T) {
	resp, _ := json.Marshal(Response[ForkChoiceUpdatedResult]{})
	handler.Response = []byte(resp)
	_, err := rpcClient.UpdateForkChoice(&commands.ForkChoiceState{})
	if err != nil {
		t.Fatalf("UpdateForkChoice - expected %v, got %v", nil, err)
	}
}

func TestUpdateForkChoiceAndBuildBlock_success(t *testing.T) {
	resp, _ := json.Marshal(Response[ForkChoiceUpdatedResult]{})
	handler.Response = []byte(resp)
	_, err := rpcClient.UpdateForkChoiceAndBuildBlock(&commands.ForkChoiceState{}, &commands.PayloadAttributes{})
	if err != nil {
		t.Fatalf("UpdateForkChoiceAndBuildBlock - expected %v, got %v", nil, err)
	}
}

func TestUpdateForkChoiceAndBuildBlock_invalidResponse(t *testing.T) {
	resp, _ := json.Marshal(Response[*commands.ExecutionPayload]{})
	handler.Response = []byte(resp)
	_, err := rpcClient.UpdateForkChoiceAndBuildBlock(&commands.ForkChoiceState{}, &commands.PayloadAttributes{})
	if !ErrorIs(err, ERR_UNMARSHALLING_FAILED) {
		t.Fatalf("UpdateForkChoiceAndBuildBlock - expected %v, got %v", ERR_UNMARSHALLING_FAILED, err)
	}
}

func TestSendExecutionPayload_success(t *testing.T) {
	resp, _ := json.Marshal(Response[commands.PayloadAttributes]{})
	handler.Response = []byte(resp)
	err := rpcClient.SendExecutionPayload(&commands.ExecutionPayload{})
	if err != nil {
		t.Fatalf("SendExecutionPayload - expected %v, got %v", nil, err)
	}
}

func TestSendExecutionPayload_invalidResponse(t *testing.T) {
	resp, _ := json.Marshal(Response[*commands.ExecutionPayload]{})
	handler.Response = []byte(resp)
	err := rpcClient.SendExecutionPayload(&commands.ExecutionPayload{})
	if !ErrorIs(err, ERR_UNMARSHALLING_FAILED) {
		t.Fatalf("SendExecutionPayload - expected %v, got %v", ERR_UNMARSHALLING_FAILED, err)
	}
}

func TestGetPayload_success(t *testing.T) {
	resp, _ := json.Marshal(Response[*commands.ExecutionPayload]{
		Result: new(commands.ExecutionPayload),
	})
	handler.Response = []byte(resp)
	_, err := rpcClient.GetPayload("0x0000000000000000")
	if err != nil {
		t.Fatalf("GetPayload - expected %v, got %v", nil, err)
	}
}

func TestGetPayload_invalidResponse(t *testing.T) {
	resp, _ := json.Marshal(Response[int]{})
	handler.Response = []byte(resp)
	result, err := rpcClient.GetPayload("0x0000000000000000")
	if !ErrorIs(err, ERR_UNMARSHALLING_FAILED) {
		t.Error("result: ", result)
		t.Fatalf("GetPayload - expected %v, got %v", ERR_UNMARSHALLING_FAILED, err)
	}
}
