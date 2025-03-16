/*
Application code for the transaction detail page.
*/

import { isRequestMatch } from '../htmx/helpers.js';
import { createChoices } from '../modules/components.js';


/*
Specialized add alerts function for the transfer detail page.

TODO: consider refactoring this into a more general alerts class.
*/
function alertError(id, title, message) {
  const alerts = document.getElementById(id);
  alerts.insertAdjacentHTML('beforeend', `
    <div class="alert alert-danger alert-dismissible fade show" role="alert">
      <strong>${title}</strong> ${message}.
      <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
    </div>
  `);
}


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
      createChoices(select);
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
    alertError("rejectAlerts", "Error:", error.error);
  }
});