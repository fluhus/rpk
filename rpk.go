package rpk

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

	for i := 0; i < n; i++ {
		method := value.Method(i)
		name := reflect.TypeOf(a).Method(i).Name
		typ := method.Type()

		// Must have at most 1 input argument.
		if typ.NumIn() > 1 {
			return nil, fmt.Errorf("Function '%s' must have 0 or 1 inputs. It has %d. %v %v",
				name, typ.NumIn(), typ.In(0), typ.In(1))
		}

		// Must have 2 outputs, the second one is an error.
		if typ.NumOut() != 2 {
			return nil, fmt.Errorf("Function '%s' does not have 2 outputs. It has %d.",
				name, typ.NumOut())
		}
		var perr *error
		if typ.Out(1) != reflect.TypeOf(perr).Elem() {
			return nil, fmt.Errorf("Function '%s' does not return an error as its"+
				" second output.", name)
		}

		// Passed. Register function.
		result[name] = method
	}

	return result, nil
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

	fType := f.Type()
	var out []reflect.Value

	// If function has an input argument.
	if fType.NumIn() == 1 {
		// Extract input parameter.
		inType := fType.In(0)
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

	// Check for error.
	if !out[1].IsNil() {
		return jsonError("%v", out[1].Interface())
	}

	// No error - encode result.
	result, _ := json.Marshal(out[0].Interface())

	return string(result)
}

// Generates a JSON string with an error field, which evaluates to the given
// format.
func jsonError(s string, a ...interface{}) string {
	result, _ := json.Marshal(map[string]string{"error": fmt.Sprintf(s, a...)})
	return string(result)
}

// Returns a handler function that calls a's methods upon getting POST requests.
// A request to the handler needs to have a parameter "func=FunctionName" and a JSON
// encoded function argument in the body.
func HandlerFuncFor(a interface{}) (http.HandlerFunc, error) {
	f, err := newFuncs(a)
	if err != nil {
		return nil, err
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// TODO(amit): Verify that request is POST.
		
		// Read parameter from body.
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.Write([]byte(jsonError("Error while reading request body: %v", err)))
			return
		}
		param := string(body)

		funcName := r.FormValue("func")
		result := f.call(funcName, param)
		w.Write([]byte(result))
	}, nil
}
