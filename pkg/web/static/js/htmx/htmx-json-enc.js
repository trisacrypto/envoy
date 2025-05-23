// Check HTMX version and warn if it is not 2.x
if (htmx.version && !htmx.version.startsWith("2.")) {
    console.warn("WARNING: You are using an htmx 2 extension with htmx " + htmx.version +
        ".  It is recommended that you move to the version of this extension found on https://htmx.org/extensions")
}

// Define the json-enc extension for HTMX.
htmx.defineExtension('json-enc', {
  onEvent: function (name, evt) {
    if (name === "htmx:configRequest") {
      evt.detail.headers['Content-Type'] = "application/json";
    }
  },

  encodeParameters : function(xhr, parameters, elt) {
    // Ensure the MIME type is set correctly
    xhr.overrideMimeType('text/json');

    // Handle serialization of FormData objects
    // This will ensure arrays of values are serialized correctly into arrays
    // However it cannot handle non string or []string objects (right now).
    // If that is needed in the future, we'd have to add the JSON as a Blob to
    // the form data and deserialize it by checking if it was a blob type (async).
    if (parameters instanceof FormData) {
      const obj = {};
      for (var key of parameters.keys()) {
        // Get all values for this key if needed to pass them as an array.
        const values = parameters.getAll(key);

        // If it starts with a json: prefix, then we need to parse it as JSON.
        const isJSON = key.startsWith("json:");
        if (isJSON) {
          key = key.substring(5);
        }

        if (values.length === 1) {
          obj[key] = isJSON ? JSON.parse(values[0]) : values[0];
        } else {
          obj[key] = values.map(value => isJSON ? JSON.parse(value) : value);
        }
      }
      return (JSON.stringify(obj));
    }

    // Otherwise this is probably a ProxyForm object from HTMX.
    return (JSON.stringify(parameters));
  }
});
