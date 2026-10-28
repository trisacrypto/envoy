import { isRequestFor } from '../htmx/helpers.js';
import { selectBusinessCategory, selectVASPCategory } from '../modules/categories.js';
import { createList, createPageSizeSelect } from '../modules/components.js';
import { selectCountry } from '../modules/countries.js';
import { selectProtocol } from '../modules/protocols.js';

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