/*
Application code for the counterparty management dashboard page.
*/

import { createList, createPageSizeSelect } from '../modules/components.js';
import { isRequestFor, isRequestMatch } from '../htmx/helpers.js';
import { selectCountry } from '../modules/countries.js';
import Alerts from '../modules/alerts.js';


// Alert managers for create modal and page-level actions.
const createCounterpartyAlerts = new Alerts("#createCounterpartyAlerts");
const pageAlerts = new Alerts("#pageAlerts");

// Initialize create modal choices on first page load.
document.querySelectorAll('[data-countries]').forEach(elem => {
  selectCountry(elem);
});

/*
Pre-flight request configuration for htmx requests.
*/
document.body.addEventListener("htmx:configRequest", function(e) {
  // Filter choices.js search terms before create requests.
  if (isRequestFor(e, "/v1/counterparties", "post")) {
    e.detail.parameters.delete("search_terms");
  }
});

/*
Post-event handling after htmx has settled the DOM.
*/
document.addEventListener("htmx:afterSettle", function(e) {
  // Initialize list controls whenever the counterparty list is refreshed.
  if (isRequestFor(e, "/v1/counterparties", "get")) {
    const cpList = document.getElementById('counterpartyList');
    if (cpList) {
      const list = createList(cpList);
      const pageSizeSelect = document.getElementById('pageSizeSelect');
      if (pageSizeSelect) {
        createPageSizeSelect(pageSizeSelect, list);
      }
    }
  }

  // Clear create form state when a create request succeeds.
  if (isRequestFor(e, "/v1/counterparties", "post")) {
    const createCounterpartyForm = document.getElementById('createCounterpartyForm');
    if (createCounterpartyForm) {
      createCounterpartyForm.reset();
    }
  }
});

/*
Post-event handling when the counterparties-updated event is fired.
*/
document.body.addEventListener("counterparties-updated", function(e) {
  const elt = e.detail?.elt;
  if (elt && elt.id === 'deleteCounterpartyBtn') {
    const modal = Modal.getInstance(document.getElementById("deleteCounterpartyModal"));
    if (modal) modal.hide();
  }
});

/*
Handle any htmx errors that are not swapped by the htmx config.
*/
document.body.addEventListener("htmx:responseError", function(e) {
  // Create counterparty errors.
  if (isRequestFor(e, "/v1/counterparties", "post")) {
    const error = JSON.parse(e.detail.xhr.response);
    switch (e.detail.xhr.status) {
      case 400:
        createCounterpartyAlerts.danger("Error:", error.error);
        break;
      case 409:
        createCounterpartyAlerts.danger("Conflict:", error.error);
        break;
      case 422:
        createCounterpartyAlerts.danger("Validation error:", error.error);
        break;
      default:
        throw new Error(`unhandled htmx error: ${error.error}`);
    }
    return;
  }

  // Delete counterparty errors.
  if (isRequestMatch(e, "/v1/counterparties/[0-7][0-9A-HJKMNP-TV-Z]{25}", "delete")) {
    const error = JSON.parse(e.detail.xhr.response);
    pageAlerts.danger("Delete Counterparty Error:", error.error);
    return;
  }

  // If the error is unhandled; throw it.
  throw new Error(`unhandled htmx error: status ${e.detail.xhr.status}`);
});

/*
Reset create modal alerts on close.
*/
const createCounterpartyForm = document.getElementById('createCounterpartyForm');
if (createCounterpartyForm) {
  createCounterpartyForm.addEventListener('reset', function() {
    const alert = document.querySelector('#createCounterpartyAlerts .alert');
    if (alert) alert.remove();
  });
}

/*
Configure delete modal button to target the selected counterparty.
*/
const deleteCounterpartyModal = document.getElementById("deleteCounterpartyModal");
if (deleteCounterpartyModal) {
  const deleteBtn = deleteCounterpartyModal.querySelector("#deleteCounterpartyBtn");

  deleteCounterpartyModal.addEventListener("show.bs.modal", function(event) {
    const button = event.relatedTarget;
    deleteCounterpartyModal.querySelector("#counterpartyName").textContent = button.dataset.bsCounterpartyName || "—";
    deleteCounterpartyModal.querySelector("#counterpartyProtocol").textContent = (button.dataset.bsCounterpartyProtocol || "—").toUpperCase();
    deleteCounterpartyModal.querySelector("#counterpartyEndpoint").textContent = button.dataset.bsCounterpartyEndpoint || "—";

    deleteBtn.setAttribute("hx-delete", "/v1/counterparties/" + button.dataset.bsCounterpartyId);
    htmx.process(deleteBtn);
  });

  deleteCounterpartyModal.addEventListener("hidden.bs.modal", function() {
    deleteCounterpartyModal.querySelector("#counterpartyName").textContent = "";
    deleteCounterpartyModal.querySelector("#counterpartyProtocol").textContent = "";
    deleteCounterpartyModal.querySelector("#counterpartyEndpoint").textContent = "";

    deleteBtn.removeAttribute("hx-delete");
    htmx.process(deleteBtn);
  });
}