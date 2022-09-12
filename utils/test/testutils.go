package test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"time"
)

var TestHandler = MockHandler{}
var TestServer = httptest.NewServer(&TestHandler)

type NoRetryStrategy struct{}

func (s *NoRetryStrategy) Next() time.Duration {
	return time.Duration(0)
}

func (s *NoRetryStrategy) Done() bool {
	return true
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
