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
	funcNames := []string{"Foo", "FooErr", "Bar", "BarErr"}
	if len(f) != len(funcNames) {
		t.Fatalf("Expected funcs to be of length %d, instead got %d.", len(funcNames), len(f))
	}
	for _, name := range funcNames {
		if _, ok := f[name]; !ok {
			t.Fatalf("Did not find func '%s'.", name)
		}
	}

	tests := []struct {
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
	}

	for _, test := range tests {
		result := f.call(test.f, test.arg)
		if test.shouldErr && !isJsonError(result) {
			t.Fatal("Expected error but got nil in test:", test)
		}
		if !test.shouldErr && isJsonError(result) {
			t.Fatal("Expected success but got error in test:", test)
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

func (t testType) Foo() (string, error) {
	return "Foo!", nil
}

func (t testType) FooErr() (string, error) {
	return "", fmt.Errorf("Foo error")
}

func (t testType) Bar(i *int) (string, error) {
	return fmt.Sprint("Bar ", *i), nil
}

func (t testType) BarErr(i *int) (string, error) {
	return "", fmt.Errorf("Bar error %d", *i)
}
