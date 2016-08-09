package rpk

import (
	"fmt"
	"strings"
	"testing"
)

func TestFuncs(t *testing.T) {
	f, err := newFuncs(testType{})
	if err != nil {
		t.Fatal("Failed to create funcs:", err)
	}
	if len(f) != len(funcNames) {
		t.Fatalf("Expected funcs to be of length %d, instead got %d.", len(funcNames), len(f))
	}
	for _, name := range funcNames {
		if _, ok := f[name]; !ok {
			t.Fatalf("Did not find func '%s'.", name)
		}
	}

	for _, test := range tests {
		result := f.call(test.f, test.arg)
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

// ----- HELPERS --------------------------------------------------------------

// Checks if the given string looks like a JSON error.
func isJsonError(s string) bool {
	return strings.HasPrefix(s, "{\"error\":")
}

// ----- TEST TYPE ------------------------------------------------------------

type testType struct{}

type thing struct {
	I int
	S string
}

func (t testType) Foo() (string, error) {
	return "Foo!", nil
}

func (t testType) FooErr() (string, error) {
	return "", fmt.Errorf("Foo error")
}

func (t testType) Bar(i int) (string, error) {
	return fmt.Sprint("Bar ", i), nil
}

func (t testType) BarErr(i int) (string, error) {
	return "", fmt.Errorf("Bar error %d", i)
}

func (t testType) Baz(a []string) (string, error) {
	return fmt.Sprint("Baz ", a[0]), nil
}

func (t testType) BazErr(a []string) (string, error) {
	return "", fmt.Errorf("Bar error %s", a[0])
}

func (t testType) Fun(th *thing) (string, error) {
	return fmt.Sprint("Fun ", th.I, " ", th.S), nil
}

func (t testType) FunErr(th *thing) (string, error) {
	return "", fmt.Errorf("Fun error")
}

var funcNames = []string{"Foo", "FooErr", "Bar", "BarErr", "Baz", "BazErr",
	"Fun", "FunErr"}

// ----- TESTS -----------------------------------------------------------------

var tests = []struct {
	f         string
	arg       string
	result    string
	shouldErr bool
}{
	{"Foo", "", "\"Foo!\"", false},
	{"Foo", "1", "", true},
	{"FooErr", "", "", true},
	{"Bar", "7", "\"Bar 7\"", false},
	{"Bar", "", "", true},
	{"BarErr", "7", "", true},
	{"Baz", "[\"x\",\"y\"]", "\"Baz x\"", false},
	{"Baz", "", "", true},
	{"BazErr", "[\"x\",\"y\"]", "", true},
	{"Fun", "{\"i\":7,\"s\":\"aaa\"}", "\"Fun 7 aaa\"", false},
	{"Fun", "{\"i\":7,\"s\":\"aaa\"{", "", true},
	{"Fun", "", "", true},
	{"FunErr", "{\"i\":7,\"s\":\"aaa\"}", "", true},
}
