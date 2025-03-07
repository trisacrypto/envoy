const debounce = (fn, wait) =>{
  var timeout;
  return function() {
    var _this = this;
    var args = arguments;
    var later = function() {
      timeout = null;
      fn.apply(_this, args);
    };
    clearTimeout(timeout);
    timeout = setTimeout(later, wait || 100);
  };
};

(function() {
  const form = document.getElementById('traddrEncodeForm');
  const encodeDecodeBtn = document.getElementById('encodeDecodeBtn');
  const resetBtn = document.getElementById('resetBtn');
  const inputContent = document.getElementById('inputContent');
  const errorText = document.getElementById('errorText');
  const outputLabel = document.getElementById('outputLabel');
  const outputContent = document.getElementById('outputContent');
  const copyOutputBtn = document.getElementById('copyOutputBtn');
  let prev = "";

  const clearError = function() {
    inputContent.classList.remove('is-invalid');
    errorText.textContent = "";
  };

  const checkInput = function(e) {
    clearError();
    const val = inputContent.value.trim();
    if (val == prev) {
      return;
    }

    prev = val;
    outputContent.textContent = "";

    if (val.length === 0) {
      encodeDecodeBtn.textContent = "Encode/Decode";
      outputLabel.textContent = "Output:";
      return;
    }

    // NOTE: if a valid URL starts with TA this will be a problem.
    // TODO: switch to a regular expression to check if it's a travel address.
    if (val.length > 2 && val.lastIndexOf("ta") === 0) {
      // Swap to "Decode Mode"
      form.action = "/v1/utilities/travel-address/decode";
      inputContent.setAttribute("name", "encoded");
      encodeDecodeBtn.textContent = "Decode";
      outputLabel.textContent = "Decoded Output:";
    } else {
      // Swap to "Encode Mode"
      form.action = "/v1/utilities/travel-address/encode";
      inputContent.setAttribute("name", "decoded");
      encodeDecodeBtn.textContent = "Encode";
      outputLabel.textContent = "Encoded Output:";
    }
  };

  inputContent.addEventListener('change', checkInput);
  inputContent.addEventListener('keyup', debounce(checkInput, 400));

  resetBtn.addEventListener('click', function(e) {
    clearError();
    outputContent.textContent = "";
    prev = "";
  });

  form.addEventListener('submit', function(e) {
    e.preventDefault();

    // Disable the button and remove any error classes
    encodeDecodeBtn.disabled = true;
    resetBtn.disabled = true;
    clearError();

    // Get the data from the form
    const data = Object.fromEntries(new FormData(e.target).entries());
    const endpoint = e.target.getAttribute('action');
    const method = e.target.getAttribute('method');

    // Check to make sure there is something to submit
    if (!data.decoded && !data.encoded) {
      inputContent.classList.add('is-invalid');
      errorText.textContent = "please enter a value to encode or decode";
      encodeDecodeBtn.disabled = false;
      resetBtn.disabled = false;
      return false;
    }

    // Make the request
    fetch(endpoint, {
      method: method,
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(data)
    }).then(response => {
      if (!response.ok) {
        switch (response.status) {
          case 400:
            // Errors are returned as JSON
            return response.json();
          case 401:
            document.location = "/login";
            break;
          case 403:
            document.location = "/login";
          default:
            document.location = "/error";
        }
      }
      return response.json();
    }).then(reply => {
      if (reply.error) {
        throw new Error(reply.error);
      }

      // Determine the type of response
      if ("decoded" in data && "encoded" in reply) {
        outputContent.textContent = reply.encoded;
      } else if ("encoded" in data && "decoded" in reply) {
        outputContent.textContent = reply.decoded;
      } else {
        outputContent.textContent = JSON.stringify(reply);
      }
    }).catch(error => {
      inputContent.classList.add('is-invalid');
      errorText.textContent = error.message;
    }).finally(() => {
      encodeDecodeBtn.disabled = false;
      resetBtn.disabled = false;
    });

    return false;
  });

  copyOutputBtn.addEventListener('click', function(e) {
    const output = outputContent.innerText;
    if (!output) {
      return;
    }

    navigator.clipboard.writeText(output)
      .then(() => {
        console.info("output content copied to clipboard");
        copyOutputBtn.innerHTML = '<i class="fe fe-copy"></i> Copied!';
      })
      .catch(err => {
        console.error("failed to copy output content to clipboard", err);
        copyOutputBtn.classList.remove('btn-primary');
        copyOutputBtn.classList.add('btn-danger');
        copyOutputBtn.innerHTML = '<i class="fe fe-x-octagon"></i> Copy Failed';
      })
      .finally(() => {
        setTimeout(() => {
          copyOutputBtn.classList.remove('btn-danger');
          copyOutputBtn.classList.add('btn-primary');
          copyOutputBtn.innerHTML = '<i class="fe fe-clipboard"></i> Copy Output';
        }, 1000);
      });
  });

})();