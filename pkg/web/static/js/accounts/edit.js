/*
Application code for the customer account management edit page.
*/

import Alerts from '../modules/alerts.js';

// Initialize the alerts component.
const alerts = new Alerts(document.getElementById("alerts"), {autoClose: true});

/*
Post-event handling when the accounts-updated event is fired.
*/
document.body.addEventListener("accounts-updated", function(e) {
  const elt = e.detail?.elt;
  if (elt) {
    console.log(elt.id);
    if (elt.id === 'deleteBtn') {
      // Redirect to the accounts index page after the account is deleted.
      window.location.href = "/accounts";
    }

    if (elt.getAttribute("id") === 'editAccountForm') {
      alerts.success("", "Account updated successfully.");
    }
  }
});

/*
Handle any htmx errors that are not swapped by the htmx config.
*/
document.body.addEventListener("htmx:responseError", function(e) {
  const error = JSON.parse(e.detail.xhr.response);
  console.log(alerts);
  switch (e.detail.xhr.status) {
    case 400:
      alerts.danger("Error:", error.error);
      break;
    case 409:
      alerts.danger("Conflict:", error.error);
      break;
    case 422:
      alerts.danger("Validation error:", error.error);
      break;
    default:
      throw new Error(`unhandled htmx error: ${error.error}`);
  }
});