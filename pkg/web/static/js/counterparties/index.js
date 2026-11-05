import { isRequestFor, isRequestMatch } from '../htmx/helpers.js';
import { selectBusinessCategory, selectVASPCategory } from '../modules/categories.js';
import { createList, createPageSizeSelect } from '../modules/components.js';
import { selectCountry } from '../modules/countries.js';
import { selectProtocol } from '../modules/protocols.js';
import Alerts from '../modules/alerts.js';

const pageAlerts = new Alerts(document.getElementById("pageAlerts"));

document.addEventListener("htmx:afterSettle", function (e) {
  // Initialize List.js
  const cpList = document.getElementById('counterpartyList');
  const list = createList(cpList);

  // Initialize Page Size Select
  const pageSizeSelect = document.getElementById('pageSizeSelect');
  createPageSizeSelect(pageSizeSelect, list);
});

const protocolSelect = document.getElementById('protocol');

// Display Sunrise or TRP form fields after a user selects the protocol.
function updateProtocolFields(protocol) {
  const sunriseFields = document.getElementById('sunriseFields');
  const trpFields = document.getElementById('trpFields');
  if (protocol === 'sunrise') {
    sunriseFields.classList.remove('d-none');
    trpFields.classList.add('d-none');
  } else if (protocol === 'trp') {
    sunriseFields.classList.add('d-none');
    trpFields.classList.remove('d-none');
  } else {
    sunriseFields.classList.add('d-none');
    trpFields.classList.add('d-none');
  }
};

protocolSelect.addEventListener('change', function (e) {
  updateProtocolFields(e.target.value);
});

// Initialize choices.js and keep the instances.
const protocolChoices = selectProtocol(document.getElementById('protocol'));
const countryChoices = selectCountry(document.getElementById('country'));
const businessChoices = selectBusinessCategory(document.getElementById('business_category'));
const vaspChoices = selectVASPCategory(document.getElementById('vasp_categories'));
const cpForm = document.getElementById('createCounterpartyForm');

// Manually reset the form. This allows select elements with options populated by choices.js
// to continue to be displayed after a reset.
function clearCPForm() {
  cpForm.querySelectorAll('input').forEach((input) => {
    input.value = '';
  });

  // Reset the value for the select elements using choices.
  protocolChoices.setChoiceByValue('');
  countryChoices.setChoiceByValue('');
  businessChoices.setChoiceByValue('');
  vaspChoices.removeActiveItems();
  updateProtocolFields('');
};

document.getElementById('cancelCreateCounterparty').addEventListener('click', clearCPForm);

// Modify create counterparty request payload.
document.body.addEventListener('htmx:configRequest', function (e) {
  if (isRequestFor(e, "/v1/counterparties", "post")) {
    const params = e.detail.parameters
    const name = params.get("contact_name");
    const email = params.get("contact_email");
    const vaspCategories = params.getAll("vasp_categories");

    // If contact name or contact email is provided, create a contacts array with an object.
    if (name || email) {
      params.append("json:contacts", JSON.stringify([{ name, email }]));

      // Delete the contact name and contact email parameters.
      params.delete("contact_name");
      params.delete("contact_email");
    }


    // If VASP categories are provided, create a vasp_categories array.
    if (vaspCategories.length > 0) {
      params.append("json:vasp_categories", JSON.stringify(Array.isArray(vaspCategories) ? vaspCategories : [vaspCategories]));

      // Delete the VASP categories parameters.
      params.delete("vasp_categories");
    }
  }
});

// Hide the counterparty modal and clear the form after a counterparty is created.
document.body.addEventListener("counterparty-created", function () {
  const cpModal = Modal.getInstance(document.getElementById("createCounterpartyModal"));
  if (cpModal) {
    cpModal.hide();
  }
  clearCPForm();
});

// Post-event handling when the counterparty-updated event is fired.
document.body.addEventListener("counterparty-updated", function (e) {
  const elt = e.detail?.elt;
  if (elt) {
    if (elt.id === "deleteBtn") {
      const deleteCounterpartyModal = Modal.getInstance(document.getElementById("deleteCounterpartyModal"));
      if (deleteCounterpartyModal) {
        deleteCounterpartyModal.hide();
      }
    }
  }
});

// Handle any htmx errors that are not swapped by the htmx config.
document.body.addEventListener("htmx:responseError", function (e) {
  // Handle errors for delete counterparty by showing a toast alert.
  if (isRequestMatch(e, "/v1/counterparties/[0-7][0-9A-HJKMNP-TV-Z]{25}", "delete")) {
    const error = JSON.parse(e.detail.xhr.response);
    pageAlerts.danger("Delete Counterparty Error:", error.error);
    return;
  }

  // If the error is unhandled; throw it
  throw new Error(`unhandled htmx error: status ${e.detail.xhr.status}`);
});


// Show the delete modal and populate its contents with the data attributes from the table row. 
// When hidden, make sure the modal is reset to its initial state.
const deleteCounterpartyModal = document.getElementById("deleteCounterpartyModal");
if (deleteCounterpartyModal) {
  deleteCounterpartyModal.addEventListener("show.bs.modal", function (event) {
    const button = event.relatedTarget;
    deleteCounterpartyModal.querySelector("#counterpartyID").textContent = button.dataset.bsCounterpartyId || "—";
    deleteCounterpartyModal.querySelector("#name").textContent = button.dataset.bsName || "—";

    const deleteBtn = deleteCounterpartyModal.querySelector("#deleteBtn");
    deleteBtn.setAttribute("hx-delete", "/v1/counterparties/" + button.dataset.bsCounterpartyId);
    htmx.process(deleteBtn);
  });

  deleteCounterpartyModal.addEventListener("hidden.bs.modal", function () {
    deleteCounterpartyModal.querySelector("#counterpartyID").textContent = "";
    deleteCounterpartyModal.querySelector("#name").textContent = "";

    const deleteBtn = deleteCounterpartyModal.querySelector("#deleteBtn");
    deleteBtn.removeAttribute("hx-delete");
    htmx.process(deleteBtn);
  });
}