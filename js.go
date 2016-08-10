package rpk

var jsCode = `function rpk(url) {
	var result = {
		ready : false
	};
	
	// Calls callback with the error, or throws an exception if no callback.
	var callOrThrow = function(errorCallback, error) {
		if (!errorCallback) {
			throw error;
		}
		errorCallback(error);
	}
	
	// Calls an RPK function.
	var callRpk = function(name, param, callback, errorCallback) {
		var xhr = new XMLHttpRequest();
		xhr.onreadystatechange = function() {
			if (xhr.readyState == 4) {
				if (xhr.status != 200) {
					callOrThrow(errorCallback, "Got bad response status code: " + xhr.status);
					return;
				}
				var response = JSON.parse(xhr.responseText);
				if (response.error) {
					callOrThrow(errorCallback, response.error);
					return;
				}
				callback(response);
			}
		};
		xhr.open("POST", url+"?func=" + name + "&param=" + encodeURI(JSON.stringify(param)), true);
		xhr.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");
		xhr.send();
	};
	
	// Returns a function that calls a specific RPK function.
	var rpkCaller = function(name) {
		return function(param, callback, errorCallback) {
			callRpk(name, param, callback, errorCallback);
		};
	};
	
	// Prepare RPK functions for result.
	callRpk("funcs", "", function(funcs) {
		for (var i = 0; i < funcs.length; i++) {
			// TODO(amit): Lowercase first letter.
			result[funcs[i]] = rpkCaller(funcs[i]);
		}
		result.ready = true;
	});
	
	return result;
}
`
