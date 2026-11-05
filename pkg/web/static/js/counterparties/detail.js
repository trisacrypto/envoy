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

const contactDetailPath = /^\/v1\/counterparties\/[0-7][0-9A-HJKMNP-TV-Z]{25}\/contacts\/[0-7][0-9A-HJKMNP-TV-Z]{25}$/;
const contactListPath = /^\/v1\/counterparties\/[0-7][0-9A-HJKMNP-TV-Z]{25}\/contacts$/;

document.addEventListener("htmx:afterSettle", function (e) {
  // After fetching a contact, display the edit contact modal.
  if (isRequestMatch(e, contactDetailPath, "get")) {
    // Edit contact content is loaded into an empty modal shell via HTMX. Create the 
    // Bootstrap modal instance and show it after settle.
    const editContactModal = new Modal("#editContactModal", {});
    editContactModal.show();
    return;
  }

  // Initialize List.js.
  if (isRequestMatch(e, contactListPath, "get")) {
    const table = document.getElementById("contactList");
    if (table) createList(table);
  }
});

const createContactModal = document.getElementById("createContactModal");
if (createContactModal) {
  // Reset the form when the modal is hidden.
  createContactModal.addEventListener("hidden.bs.modal", resetCreateContactForm);
}

const editContactModal = document.getElementById("editContactModal");
if (editContactModal) {
  // Clear the HTMX-loaded dialog when the modal is hidden.
  editContactModal.addEventListener("hidden.bs.modal", function () {
    editContactModal.innerHTML = "";
  });
}


// Show delete contact modal and populate its content with the data attributes from the table row. 
// When hidden, make sure the modal is reset to its initial state.
const deleteContactModal = document.getElementById("deleteContactModal");
if (deleteContactModal) {
  const deleteBtn = deleteContactModal.querySelector("#deleteContactBtn");
  const baseDeleteURL = deleteBtn.getAttribute("hx-delete");

  deleteContactModal.addEventListener("show.bs.modal", function (event) {
    const button = event.relatedTarget;
    const label = button.dataset.bsContactName || button.dataset.bsContactEmail || "—";
    deleteContactModal.querySelector("#deleteContactLabel").textContent = label;

    deleteBtn.setAttribute("hx-delete", baseDeleteURL + button.dataset.bsContactId);
    htmx.process(deleteBtn);
  });

  deleteContactModal.addEventListener("hidden.bs.modal", function () {
    deleteContactModal.querySelector("#deleteContactLabel").textContent = "";

    deleteBtn.removeAttribute("hx-delete");
    htmx.process(deleteBtn);
  });
}

document.body.addEventListener("contact-created", function () {
  // Hide the modal and reset the form when a contact is created.
  const modal = Modal.getInstance(createContactModal);
  if (modal) {
    modal.hide();
  }
  resetCreateContactForm();
});

document.body.addEventListener("contact-updated", function () {
  // Hide the modal when a contact is updated.
  const modal = Modal.getInstance(editContactModal);
  if (modal) {
    modal.hide();
  }
});

document.body.addEventListener("contact-deleted", function () {
  // Hide the modal when a contact is deleted.
  const modal = Modal.getInstance(deleteContactModal);
  if (modal) {
    modal.hide();
  }
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
