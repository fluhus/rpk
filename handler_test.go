package rpk

import (
	//"fmt"
	"bytes"
	"net/http"
	"strings"
	"testing"
)

func TestHandler(t *testing.T) {
	handler, err := HandlerFuncFor(testType{})
	if err != nil {
		t.Fatal("Failed to create handler:", err)
	}

	for _, test := range tests {
		req, err := http.NewRequest("POST", "", strings.NewReader(test.arg))
		if err != nil {
			t.Fatal("Failed to create HTTP request:", err)
		}
		req.PostForm = map[string][]string{"func": {test.f}}
		res := &mockResponseWriter{bytes.NewBuffer(nil)}

		handler(res, req)
		result := res.buf.String()

		if test.shouldErr && !isJsonError(result) {
			t.Fatal("Expected error but got nil in test:", test)
		}
		if !test.shouldErr && isJsonError(result) {
			t.Fatal("Expected success but got error in test:", test, result)
		}
		if !test.shouldErr && result != test.result {
			t.Fatalf("Bad result for test: %v Got: %s", test, result)
		}
	}
}

type mockResponseWriter struct {
	buf *bytes.Buffer
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	return m.buf.Write(b)
}
func (m *mockResponseWriter) Header() http.Header {
	return nil
}
func (m *mockResponseWriter) WriteHeader(i int) {}
