/*
Application code for the counterparty detail page.
*/

import { createList } from '../modules/components.js';
import { isRequestMatch } from '../htmx/helpers.js';
import Alerts from '../modules/alerts.js';

const createContactAlerts = new Alerts("#createContactAlerts");
const createContactForm = document.getElementById("createContactForm");

function resetCreateContactForm() {
  if (createContactForm) {
    createContactForm.reset();
  }
  const alerts = document.getElementById("createContactAlerts");
  if (alerts) {
    alerts.innerHTML = "";
  }
}

document.addEventListener("htmx:afterSettle", function (e) {
  // Initialize List.js.
  if (isRequestMatch(e, "/v1/counterparties/[0-7][0-9A-HJKMNP-TV-Z]{25}/contacts", "get")) {
    const table = document.getElementById("contactList");
    if (table) createList(table);
  }
});

const createContactModal = document.getElementById("createContactModal");
if (createContactModal) {
  // Reset the form when the modal is hidden.
  createContactModal.addEventListener("hidden.bs.modal", resetCreateContactForm);
}

document.body.addEventListener("contact-created", function () {
  // Hide the modal and reset the form when a contact is created.
  const modal = Modal.getInstance(createContactModal);
  if (modal) {
    modal.hide();
  }
  resetCreateContactForm();
});

// Handle errors from the create counterparty contact endpoint.
document.body.addEventListener("htmx:responseError", function (e) {
  if (isRequestMatch(e, "/v1/counterparties/[0-7][0-9A-HJKMNP-TV-Z]{25}/contacts", "post")) {
    const error = JSON.parse(e.detail.xhr.response);
    switch (e.detail.xhr.status) {
      case 400:
        createContactAlerts.danger("Error:", error.error);
        break;
      case 409:
        createContactAlerts.danger("Conflict:", error.error);
        break;
      case 422:
        createContactAlerts.danger("Validation error:", error.error);
        break;
      default:
        break;
    }
    return;
  }

  throw new Error(`unhandled htmx error: status ${e.detail.xhr.status}`);
});
