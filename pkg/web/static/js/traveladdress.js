function debounce(fn, wait) {
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


$(function() {
    const btn = $("#encode-decode-btn");
    const form = $("#traddr-encoder-form");
    const input = $("#input-content");
    const etext = $("#error-text");
    const output = $("#output-content");
    let prev = "";

    const clearError = function() {
      input.removeClass("errored");
      etext.removeClass("error-text");
      etext.empty();
    };

    const checkInput = function(e) {
      clearError();
      const val = input.val();

      if (val == prev) {
        return
      }

      // Clear out old output
      prev = val;
      output.empty();

      if (val.length > 2 && val.lastIndexOf("ta") === 0) {
        // Swap to "decode" mode
        form.attr("action", "/v1/utilities/travel-address/decode");
        input.attr("name", "encoded");
        btn.text("Decode");
      } else {
        // Swap to "encode mode
        form.attr("action", "/v1/utilities/travel-address/encode");
        input.attr("name", "decoded");
        btn.text("Encode");
      }
    };

    input.change(checkInput);
    input.keyup(debounce(checkInput, 400));

    form.submit(function(e) {
      e.preventDefault();

      // Disable the button and remove any error classes
      btn.attr("disabled", "true");
      clearError();

      // Get the data from the form
      const data = Object.fromEntries(new FormData(e.target).entries());
      const endpoint = $(e.target).attr("action");

      // Request the encoding/decoding from the server
      $.ajax({
        url: endpoint,
        method: "POST",
        data: JSON.stringify(data),
        dataType: "json",
        contentType: "application/json"
      }).done(function(reply) {
        console.debug(reply);

        // Determine the type of response
        if ("decoded" in data && "encoded" in reply) {
          output.text(reply.encoded);
        } else if ("encoded" in data && "decoded" in reply) {
          output.text(reply.decoded);
        } else {
          console.error("could not determine output");
        }
      }).fail(function(jqXHR, status, error) {
        input.addClass("errored");
        etext.addClass("error-text");

        if (jqXHR?.responseJSON.error) {
          etext.text(jqXHR.responseJSON.error);
        } else {
          etext.text("could not encode/decode your input");
        }

      }).always(function() {
        btn.removeAttr("disabled");
      });

      return false;
    });
});