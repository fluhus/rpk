package rpk

// TODO(amit): Document client code.

var jsCode = `function rpk(url) {
	var result = {
		ready : false
	};
	
	// Calls callback with the parameters, or throws an exception if no callback.
	var callOrThrow = function(callback, data, error) {
		if (callback) {
			callback(data, error);
		}
		if (!callback && error) {
			throw error;
		}
	}
	
	// Calls an RPK function.
	var callRpk = function(name, param, callback) {
		var xhr = new XMLHttpRequest();
		xhr.onreadystatechange = function() {
			if (xhr.readyState == 4) {
				if (xhr.status != 200) {
					callOrThrow(callback, null, "Got bad response status code: " + xhr.status);
					return;
				}
				try {
					var response = JSON.parse(xhr.responseText);
				} catch (error) {
					callOrThrow(callback, null, "Error parsing response: " + error);
					return;
				}
				if (response.error) {
					callOrThrow(callback, null, response.error);
					return;
				}
				callOrThrow(callback, response, null);
			}
		};
		if (typeof param == "undefined") {
			param = "";
		} else {
			param = encodeURI(JSON.stringify(param));
		}
		xhr.open("POST", url+"?func=" + name + "&param=" + param, true);
		xhr.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");
		xhr.send();
	};
	
	// Returns a function that calls a specific RPK function.
	var rpkCaller = function(name) {
		return function(param, callback) {
			if (arguments.length != 1 && arguments.length != 2) {
				throw "Bad number of arguments: " + arguments.length 
					+ ", expected 1 or 2.";
			}
			if (arguments.length == 1) {
				callback = param;
				param = undefined;
			}
			callRpk(name, param, callback);
		};
	};

	// Prepare RPK functions for result.
	var initError = null;
	var initCallbacks = [];
	callRpk("funcs", "", function(funcs, error) {
		if (error) {
			initError = error;
		} else {
			for (var i = 0; i < funcs.length; i++) {
				result[funcs[i]] = rpkCaller(funcs[i]);
			}
			result.ready = true;
		}
		for (var i = 0; i < initCallbacks.length; i++) {
			initCallbacks[i](initError);
		}
	});

	result.onReady = function(callback) {
		if (result.ready || initError) {
			callback(initError);
			return;
		}
		initCallbacks.push(callback);
	};

	return result;
}
`
