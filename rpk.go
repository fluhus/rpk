// Simple RPC between Javascript and Go.
// The package converts objects to RPC handlers that call their exported methods.
//
// Restrictions on RPC methods
//
// The methods of an RPC object must:
// (1) have at most 1 input argument, which should be JSON encodable
// (2) have at most 2 outputs: 1 optional value of any JSON encodable type, and an optional
// error. If using 2 outputs, the error should come second.
//
// Unexported methods are ignored and do not have any restriction.
//
// Server code example
//
// The server defines the exported RPC interface through the methods of a type.
//
//  type myAPI struct{}
//
//  func (m myAPI) Half(i int) int {
//    return i / 2
//  }
//
//  func main() {
//    http.HandleFunc("/api/client.js", rpk.HandleJs)  // Serves client code.
//    handler, _ := rpk.NewHandlerFunc(myAPI{})
//    http.HandleFunc("/api", handler)
//    http.ListenAndServe(":8080", nil)
//  }
//
// Client code example
//
// The client needs to fetch the complementary Javascript code.
//
//  <script type="text/javascript" src="/api/client.js"></script>
//  <script type="text/javascript">
//
//  api = rpk("/api")
//  api.onReady(function(error) {...});
//
//  // ... After ready ...
//  api.Half(10, function(result, error) {
//    if (error) {
//      console.error("error=" + error);
//    } else {
//      console.log("result=" + result);
//    }
//  });
//
//  </script>
//
// Javascript API
//
// The Javascript code exposes a single function.
//  rpk([string] url)
// Returns an RPK object, which will have the exported methods of the Go object that
// handles that URL.
//
//  rpkObject.ready
// Boolean. Indicates whether this RPK object is ready to be called.
//
//  rpkObject.onReady( callback(error) )
// Adds a listener that will be called when myRpkObject finishes initializing.
// If successful, error will be null. Else, error will be a string describing
// the problem. Several listeners can be added. They will be called by order of
// adding.
//
//  rpkObject.FuncName(param, callback(data, error))
// Calls a Go method.
// Param should be of the type expected by the Go method. If the Go method expects
// no input, then param should be omitted. On success, error will be null and data
// will contain the output (if any). On error, error will be a string describing
// the problem.
package rpk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// TODO(amit): Test with bad types.
// TODO(amit): Consider a better name for HandleJs.

// Represents a set of callable functions, that communicates in JSON.
// Maps from function name to the reflection of that function.
type funcs map[string]reflect.Value

// Creates a funcs instance from the methods of the given interface.
// Returns an error if a method does not match the requirements (see package description).
func newFuncs(a interface{}) (funcs, error) {
	result := funcs{}
	value := reflect.ValueOf(a)
	n := value.NumMethod()

	// Go over functions.
	for i := 0; i < n; i++ {
		method := value.Method(i)
		name := reflect.TypeOf(a).Method(i).Name
		typ := method.Type()

		// Check if exported.
		if name[:1] == strings.ToLower(name[:1]) {
			continue
		}

		// Check that function matches the requirements.
		if err := checkInputs(typ); err != nil {
			return nil, fmt.Errorf("Function '%s': %v", name, err)
		}
		if err := checkOutputs(typ); err != nil {
			return nil, fmt.Errorf("Function '%s': %v", name, err)
		}

		// Passed. Register function.
		result[name] = method
	}

	return result, nil
}

// Checks if a function's input argument match the requirements of RPK.
func checkInputs(f reflect.Type) error {
	// Must have at most 1 input argument.
	if f.NumIn() > 1 {
		return fmt.Errorf("Must have 0 or 1 inputs. It has %d. %v %v",
			f.NumIn(), f.In(0), f.In(1))
	}
	return nil
}

// Checks if a function's outputs match the requirements of RPK.
func checkOutputs(f reflect.Type) error {
	// Must have at most 2 outputs.
	if f.NumOut() > 2 {
		return fmt.Errorf("More than 2 outputs: %d", f.NumOut())
	}
	// If 2 outputs, then the second must be an error.
	if f.NumOut() == 2 && !isError(f.Out(1)) {
		return fmt.Errorf("Second output should be an error, but found %v.", f.Out(1))
	}
	return nil
}

// Checks if the given type is error.
func isError(t reflect.Type) bool {
	var perr *error
	return t == reflect.TypeOf(perr).Elem()
}

// Calls a function with the given JSON encoded parameter.
// Functions with no parameters should get an empty string.
// On error, returns a JSON object with an error field.
func (fs funcs) call(funcName string, param string) string {
	// Get function.
	f, ok := fs[funcName]
	if !ok {
		return jsonError("No such function '%s'.", funcName)
	}

	typ := f.Type()
	var out []reflect.Value

	// If function has an input argument.
	if typ.NumIn() == 1 {
		// Extract input parameter.
		inType := typ.In(0)
		in := reflect.New(inType)
		err := json.Unmarshal([]byte(param), in.Interface())
		if err != nil {
			return jsonError("Error decoding JSON: %v", err)
		}

		// Call method.
		out = f.Call([]reflect.Value{in.Elem()})

	} else {
		// Argument not expected.
		if param != "" {
			return jsonError("Function '%s' does not accept parameters.", funcName)
		}
		out = f.Call(nil)
	}

	// Sort out outputs.
	var outVal, outErr reflect.Value
	if len(out) == 2 {
		outVal, outErr = out[0], out[1]
	} else if len(out) == 1 {
		if isError(out[0].Type()) {
			outErr = out[0]
		} else {
			outVal = out[0]
		}
	}

	if outErr.IsValid() && !outErr.IsNil() {
		return jsonError("%v", outErr.Interface())
	}
	if outVal.IsValid() {
		result, err := json.Marshal(outVal.Interface())
		if err != nil {
			return jsonError("Error encoding result: %v", err)
		}
		return string(result)
	}
	return ""
}

// Generates a JSON string with an error field, which evaluates to the given
// format.
func jsonError(s string, a ...interface{}) string {
	result, _ := json.Marshal(map[string]string{"error": fmt.Sprintf(s, a...)})
	return string(result)
}

// Returns a handler function that calls a's exported methods. Access this handler using
// the Javascript code served by HandleJs. Returns an error if a's methods do not match
// the requirements - see package description.
func NewHandlerFunc(a interface{}) (http.HandlerFunc, error) {
	// The "Content-Type" header field should read "application/x-www-form-urlencoded".
	// The content should be "func=FunctionName&param=JsonEncodedParam".
	f, err := newFuncs(a)
	if err != nil {
		return nil, err
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// TODO(amit): Verify that request is POST.
		funcName := r.FormValue("func")

		// Special value - "funcs" - returns the names of registered functions.
		if funcName == "funcs" {
			names := make([]string, 0, len(f))
			for name := range f {
				names = append(names, name)
			}
			json.NewEncoder(w).Encode(names)
			return
		}

		param := r.FormValue("param")
		result := f.call(funcName, param)
		w.Write([]byte(result))
	}, nil
}

// An http.HandlerFunc for serving the Javascript client code.
func HandleJs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Write([]byte(jsCode))
}
