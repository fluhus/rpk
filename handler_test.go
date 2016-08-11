package rpk

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestHandler(t *testing.T) {
	handler, err := NewHandlerFunc(testType{})
	if err != nil {
		t.Fatal("Failed to create handler:", err)
	}

	for _, test := range tests {
		req, err := http.NewRequest("POST", "", nil)
		if err != nil {
			t.Fatal("Failed to create HTTP request:", err)
		}
		req.PostForm = map[string][]string{
			"func":  {test.f},
			"param": {test.arg},
		}
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

func TestHandler_funcs(t *testing.T) {
	handler, err := NewHandlerFunc(testType{})
	if err != nil {
		t.Fatal("Failed to create handler:", err)
	}

	req, err := http.NewRequest("POST", "", strings.NewReader(""))
	if err != nil {
		t.Fatal("Failed to create HTTP request:", err)
	}
	req.PostForm = map[string][]string{"func": {"funcs"}}
	res := &mockResponseWriter{bytes.NewBuffer(nil)}

	handler(res, req)
	resultJson := res.buf.String()

	var result []string
	err = json.Unmarshal([]byte(resultJson), &result)
	if err != nil {
		t.Fatal("Failed to parse JSON response:", err)
	}

	expected := sliceToMap(funcNames)
	actual := sliceToMap(result)

	if len(actual) != len(expected) {
		t.Fatalf("Bad result length: %d, expected %d.", len(actual), len(expected))
	}
	for s := range expected {
		if !actual[s] {
			t.Fatalf("Result does not contain function '%s'.", s)
		}
	}
}

// ----- HELPERS ---------------------------------------------------------------

func sliceToMap(a []string) map[string]bool {
	result := map[string]bool{}
	for _, s := range a {
		result[s] = true
	}
	return result
}

type mockResponseWriter struct {
	buf *bytes.Buffer
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	return m.buf.Write(b)
}
func (m *mockResponseWriter) Header() http.Header {
	return map[string][]string{}
}
func (m *mockResponseWriter) WriteHeader(i int) {}
