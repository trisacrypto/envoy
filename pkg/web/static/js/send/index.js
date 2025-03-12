/*
Application code for the send TRISA/TRP forms.
*/

import { isRequestFor } from '../htmx/helpers.js';
import { selectNetwork } from '../modules/networks.js';
import { selectCountry } from '../modules/countries.js';
import { selectTRISATravelAddress, createFlatpickr } from '../modules/components.js';
import { selectAddressType, selectNationalIdentifierType } from '../modules/ivms101.js';

/*
Create pop up toast alerts for errors or info when the form is being handled.
*/
const alerts = document.getElementById("alerts");
function alert(level, title, message) {
  if (alerts) {
    alerts.insertAdjacentHTML('beforeend', `
      <div class="alert alert-${level} alert-dismissible fade show" role="alert">
          <strong>${title}</strong>: <span>${message}</span>.
          <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
      </div>
    `);

    setTimeout(() => {
      document.querySelector('.alert').remove()
    }, 3000);
  }
}

/*
Initialize the select elements with choices.js in the form element.
*/
function initializeChoices(elem) {
  // Initialize the network select choices.
  elem.querySelectorAll('[data-networks]').forEach(elem => {
    selectNetwork(elem);
  });

  // Initialize the country select choices.
  elem.querySelectorAll('[data-countries]').forEach(elem => {
    selectCountry(elem);
  });

  // Initialize the address type select choices.
  elem.querySelectorAll('[data-address-type]').forEach(elem => {
    selectAddressType(elem);
  });

  // Initialize the national identifier type select choices.
  elem.querySelectorAll('[data-national-identifier-type]').forEach(elem => {
    selectNationalIdentifierType(elem);
  });
}

// Initialize the form choices when the page loads.
initializeChoices(document);

// Initialize the TRISA VASP Selection choices.
const vaspSelect = document.getElementById('trisaTravelAddress');
if (vaspSelect)  {
  selectTRISATravelAddress(vaspSelect);
}

// Handle form submission
document.getElementById('sendTransferForm').addEventListener('submit', function(e) {
  e.preventDefault();
  const form = e.target;
  const formData = new FormData(form);
  const data = Object.fromEntries(formData.entries());
  const json = JSON.stringify(data, null, 2);
  console.log(json);
  return false;
});

/*
Configure requests made by htmx before they are sent.
*/
document.body.addEventListener('htmx:configRequest', function(e) {
  /*
  Update the parameter for account lookups by crypto address
  */
  if (isRequestFor(e, "/v1/accounts/lookup", "get")) {
    const elem = e.detail.elt;
    const prefix = elem.name.split("_", 1)[0];
    const params = e.detail.parameters;

    params.set("prefix", prefix);
    params.set("crypto_address", elem.value);
    params.delete(elem.name);

    return
  }

});

/*
Handle pre-flight checks before making htmx requests
*/
document.body.addEventListener('htmx:beforeRequest', function(e) {
  /*
  If the lookup request doesn't have a value, cancel the request.
  */
  if (isRequestFor(e, "/v1/accounts/lookup", "get")) {
    if (!e.detail.requestConfig.parameters['crypto_address']) e.preventDefault();
    return
  }
});

/*
Post-event handling after htmx has settled the DOM.
*/
document.body.addEventListener("htmx:afterSettle", function(e) {
  // Re-initialize the form choices aftrer account lookups.
  if (isRequestFor(e, "/v1/accounts/lookup", "get")) {
    initializeChoices(e.target);

    // Also initialize date of birth flatpickr
    // This is not done in initializeChoices because it is handled by DashKit
    document.querySelectorAll('[data-flatpickr]').forEach((elem) => {
      createFlatpickr(elem);
    });
  }
});

/*
Handle htmx errors as JSON responses from the backend.
*/
document.body.addEventListener("htmx:responseError", (e) => {
  const error = JSON.parse(e.detail.xhr.response);
  console.error(e.detail.requestConfig.path, error.error);

  switch (e.detail.xhr.status) {
    case 400:
      alert("warning", "Bad request", error.error);
      break;
    case 404:
      if (isRequestFor(e, "/v1/accounts/lookup", "get")) {
        alert("info", "No account found", "crypto address not registered");
        break;
      }
      alert("info", "Not Found", error.error);
      break;
    case 422:
      alert("warning", "Validation error", error.error);
      break;
    case 500:
      window.location.href = "/error";
      break;
    default:
      break;
  }
});