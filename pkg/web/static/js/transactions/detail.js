/*
Application code for the transaction detail page.
*/

import { isRequestMatch } from '../htmx/helpers.js';
import { createChoices } from '../modules/components.js';
import { Transaction } from '../modules/ivms101.js';

import Alerts from '../modules/alerts.js';


// Initialize the alerts component.
var rejectAlerts;
var completeAlerts;

/*
Pre-flight request configuration for htmx requests.
*/
document.body.addEventListener("htmx:configRequest", function(e) {
  /*
  Retry checkbox needs to be stored as a boolean value.
  */
 if (isRequestMatch(e, /\/v1\/transactions\/[A-Fa-f0-9-]{36}\/reject/, "post")) {
    const params = new FormData();
    e.detail.parameters.forEach((value, key) => {
      if (key === "retry") {
        value = value === "on" ? "true" : "false";
        key = "json:retry";
      }
      params.append(key, value);
    });

    e.detail.parameters = params;
    return;
  }

  /*
  Handle the complete transaction IVMS101 data request.
  */
  if (isRequestMatch(e, /\/v1\/transactions\/[A-Fa-f0-9-]{36}\/complete/, "post")) {
    const tx = new Transaction(e.target);

    // Convert to a flattened form data object for htmx.
    e.detail.parameters = new FormData();
    for (const [key, value] of tx.entries()) {
      if (typeof value === "object") {
        e.detail.parameters.append(`json:${key}`, JSON.stringify(value));
      } else {
        e.detail.parameters.append(key, value);
      }
    }

    return;
  }

});


/*
Post-event handling after htmx has settled the DOM.
*/
document.addEventListener("htmx:afterSettle", function(e) {
    if (isRequestMatch(e, /^\/v1\/transactions\/[A-Fa-f0-9-]{36}$/, "get")) {
      // Initialize the status tooltips
      const tooltips = document.querySelectorAll('[data-bs-toggle="tooltip"]');
      tooltips.forEach(tooltip => {
        new Tooltip(tooltip);
      });

      const select = document.querySelector("select[name='code']");
      if (select) createChoices(select);

      // Initialize the alerts components
      rejectAlerts = new Alerts("#rejectAlerts");
      completeAlerts = new Alerts("#completeAlerts");
      return;
    }

    if (isRequestMatch(e, /\/v1\/transactions\/[A-Fa-f0-9-]{36}\/secure-envelopes\/[0-7][0-9A-HJKMNP-TV-Z]{25}/, "get")) {
      const element = document.getElementById("secureEnvelopePayload");
      element.scrollIntoView({ behavior: "smooth", block: "start" });
      return;
    }
});

/*
Post-event handling when the transactions-updated event is fired.
*/
document.body.addEventListener("transactions-updated", function(e) {
  const elt = e.detail?.elt;
  if (elt) {
    if (elt.id === 'rejectForm') {
      const modal = Modal.getInstance(document.getElementById("rejectTransferModal"));
      modal.hide();
    }
  }
});

/*
Handle any htmx errors that are not swapped by the htmx config.
*/
document.body.addEventListener("htmx:responseError", function(e) {
  // Try to parse the error response.
  var error;
  try {
    error = JSON.parse(e.detail.xhr.response);
    error.statusCode = e.detail.xhr.status;
  } catch (e) {
    if (e.detail && e.detail.requestConfig && e.detail.requestConfig.path) {
      console.error(e.detail.requestConfig.path, e);
    } else {
      console.error("could not parse JSON response", e);
    }

    error = {
      error: "an unknown error occurred",
      statusCode: e.detail.xhr.status
    }
  }

  if (error.statusCode >= 500) {
    window.location.href = "/error";
    return;
  }

  if (isRequestMatch(e, /\/v1\/transactions\/[A-Fa-f0-9-]{36}\/reject/, "post")) {
    switch (error.statusCode) {
      case 400:
        rejectAlerts.warning("Bad request", error.error);
        break;
      case 409:
        rejectAlerts.danger("Conflict", error.error);
        break;
      case 422:
        rejectAlerts.warning("Validation error", error.error);
        break;
      default:
        throw new Error("Unhandled reject error code: " + error.statusCode + " - " + error.error);
    }

    // Make sure to exit so we don't rethrow the error.
    return;
  }

  if (isRequestMatch(e, /\/v1\/transactions\/[A-Fa-f0-9-]{36}\/complete/, "post")) {
    switch (error.statusCode) {
      case 400:
        completeAlerts.warning("Bad request", error.error);
        break;
      case 409:
        completeAlerts.danger("Conflict", error.error);
        break;
      case 422:
        completeAlerts.warning("Validation error", error.error);
        break;
      default:
        throw new Error("Unhandled complete error code: " + error.statusCode + " - " + error.error);
    }

    // Make sure to exit so we don't rethrow the error.
    return;
  }

  throw new Error("Unhandled error code: " + error.statusCode + " - " + error.error);
});