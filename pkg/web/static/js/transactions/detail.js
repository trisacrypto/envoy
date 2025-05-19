/*
Application code for the transaction detail page.
*/

import { isRequestMatch } from '../htmx/helpers.js';
import { createChoices } from '../modules/components.js';
import { Transaction } from '../modules/ivms101.js';

import Alerts from '../modules/alerts.js';


// Initialize the alerts component.
const rejectAlerts = new Alerts("#rejectAlerts");
const completeAlerts = new Alerts("#completeAlerts");

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

    console.log(e.detail.parameters)
    e.preventDefault();
    return false;
  }

});


/*
Post-event handling after htmx has settled the DOM.
*/
document.addEventListener("htmx:afterSettle", function(e) {
    if (isRequestMatch(e, /\/v1\/transactions\/[A-Fa-f0-9-]{36}/, "get")) {
      // Initialize the status tooltips
      const tooltips = document.querySelectorAll('[data-bs-toggle="tooltip"]');
      tooltips.forEach(tooltip => {
        new Tooltip(tooltip);
      });

      const select = document.querySelector("select[name='code']");
      if (select) createChoices(select);
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
  if (isRequestMatch(e, /\/v1\/transactions\/[A-Fa-f0-9-]{36}\/reject/, "post")) {
    const error = JSON.parse(e.detail.xhr.response);
    rejectAlerts.danger("Error:", error.error);
    return;
  }

  if (isRequestMatch(e, /\/v1\/transactions\/[A-Fa-f0-9-]{36}\/complete/, "post")) {
    const error = JSON.parse(e.detail.xhr.response);
    completeAlerts.danger("Error:", error.error);
  }
});