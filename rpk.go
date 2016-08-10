package rpk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
)

// TODO(amit): Organize documentation.

// Represents a set of callable functions, that communicates in JSON.
// Maps from function name to the reflection of that function.
type funcs map[string]reflect.Value

// Creates a funcs instance from the methods of the given interface.
//
// All methods must:
// (1) Be exported.
// (2) Have at most 1 input argument, which should be JSON encodable.
// (3) Have 2 output arguments, the first is JSON-encodable and the second is an
// error.
func newFuncs(a interface{}) (funcs, error) {
	result := funcs{}
	value := reflect.ValueOf(a)
	n := value.NumMethod()

	// Go over functions.
	for i := 0; i < n; i++ {
		method := value.Method(i)
		name := reflect.TypeOf(a).Method(i).Name
		typ := method.Type()
		// TODO(amit): Check that function is exported.

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
			return jsonError("Error encoding JSON: %v", err)
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

// Returns a handler function that calls a's methods upon getting POST requests.
// The "Content-Type" header field should read "application/x-www-form-urlencoded".
// The content should be "func=FunctionName&param=JsonEncodedParam".
func HandlerFuncFor(a interface{}) (http.HandlerFunc, error) {
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

// TODO(amit): Consider a better name for HandleJs.

// An http.HandlerFunc for fetching the Javascript client code.
func HandleJs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Write([]byte(jsCode))
}
