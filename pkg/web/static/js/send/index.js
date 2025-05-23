/*
Application code for the send TRISA/TRP forms.
*/

import Alerts from '../modules/alerts.js';
import { isRequestFor } from '../htmx/helpers.js';
import { selectNetwork } from '../modules/networks.js';
import { selectCountry } from '../modules/countries.js';
import { selectTRISACounterparty, createFlatpickr } from '../modules/components.js';
import { selectAddressType, selectNationalIdentifierType } from '../modules/ivms101.js';

const previewModal = document.getElementById('previewModal');
const alerts = new Alerts("#alerts", {autoClose: true, closeTime: 3000});

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
const vaspSelect = document.getElementById('routingCounterpartyID');
if (vaspSelect)  {
  selectTRISACounterparty(vaspSelect);
}

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

  /*
  Prepare the parameters for the preview transfer submission
  */
  if (isRequestFor(e, "/v1/transactions/prepare", "post")) {
    // Show the modal - this will show the loading spinner!
    const modal = new Modal(previewModal);
    modal.show();

    // Prepare the outgoing data as a nested JSON payload.
    const form = e.target;
    const formData = new FormData(form);

    const data = {
      "routing": {},
      "originator": {
        "identification": {},
        "addresses": []
      },
      "beneficiary": {
        "identification": {},
        "addresses": []
      },
      "transfer": {}
    }

    const keymap = {
      "name_nameIdentifier_0_secondaryIdentifier": "forename",
      "name_nameIdentifier_0_primaryIdentifier": "surname",
      "countryOfResidence": "country_of_residence",
      "customerIdentification": "customer_id",
      "dateAndPlaceOfBirth_dateOfBirth": "identification_dob",
      "dateAndPlaceOfBirth_placeOfBirth": "identification_birth_place",
      "nationalIdentification_nationalIdentifierType": "identification_type_code",
      "nationalIdentification_nationalIdentifier": "identification_number",
      "nationalIdentification_countryOfIssue": "identification_country",
    }

    formData.entries().forEach(([key, value]) => {
      // Skip keys from nested forms or choices.
      if (key == "search_terms") return;

      // Find the prefix to nest the object under.
      const prefix = key.split("_", 1)[0];
      key = key.replace(prefix + "_", "");

      // Get the object to update.
      let obj = data[prefix];

      // Remap keys from IVMS101 to the prepared keys.
      if (key in keymap) {
        key = keymap[key];
      }

      // Identification and Address are nested one level below
      if (key.startsWith("identification_")) {
        key = key.replace("identification_", "");
        obj = obj.identification;
      }

      if (key.startsWith("geographicAddress_")) {
        key = key.replace("geographicAddress_", "");
        const idx = parseInt(key.split("_", 1)[0]);
        key = key.replace(idx + "_", "");

        if (obj.addresses.length <= idx) {
          obj.addresses.push({"address_lines": ["", "", ""]});
        }

        if (key == "addressType") {
          key = "address_type";
        }

        obj = obj.addresses[idx];
        if (key.startsWith("addressLine_")) {
          key = parseInt(key.replace("addressLine_", ""));
          obj = obj.address_lines;
        }
      }

      // Need amount to be a float64; not a string
      if (key == "amount") {
        value = parseFloat(value);
      }

      obj[key] = value;
    });

    // Unfortunately, htmx converts everything into flattened FormData objects so
    // all values have to be strings. Our work around is to serialize JSON and prepend
    // the key with a json: prefix so the extension knows to parse it as JSON.
    e.detail.parameters = new FormData();
    for (const [key, value] of Object.entries(data)) {
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

  /*
  Sending a prepared transaction may take a few seconds, so we want to give as many
  indicators to the user that everything is working well as possible.
  */
 if (isRequestFor(e, "/v1/transactions/send-prepared", "post")) {
    // Mark the cancel link button as disabled.
    const cancelBtn = document.getElementById('cancelBtn');
    cancelBtn.classList.add('disabled');

    // Add an overlay to the modal body to indicate the transfer is being sent.
    const overlay = document.getElementById('previewModalBodyOverlay');
    overlay.classList.remove('d-none');
 }
});

/*
Post-event handling after htmx has settled the DOM.
*/
document.body.addEventListener("htmx:afterSettle", function(e) {
  // Re-initialize the form choices after account lookups.
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
  var error;
  try {
    error = JSON.parse(e.detail.xhr.response);
    console.error(e.detail.requestConfig.path, error.error);
  } catch (e) {
    console.error(e.detail.requestConfig.path, e);
    error = {
      error: "an unknown error occurred"
    }
  }

  const modal = Modal.getInstance(previewModal);
  if (modal) {
    modal.hide();
  }

  window.scrollTo(0, 0);

  switch (e.detail.xhr.status) {
    case 400:
      if (isRequestFor(e, "/v1/transactions/send-prepared", "post")) {
        alerts.danger("Transfer failed", error.error);
      } else {
        alerts.warning("Bad request", error.error);
      }
      break;
    case 404:
      if (isRequestFor(e, "/v1/accounts/lookup", "get")) {
        alerts.info("No account found", "crypto address not registered");
        break;
      }
      alerts.info("Not Found", error.error);
      break;
    case 409:
      alerts.warning("Conflict", error.error);
      break;
    case 422:
      alerts.warning("Validation error", error.error);
      break;
    case 500:
      window.location.href = "/error";
      break;
    case 502:
      alerts.danger("Counterparty unavailable", error.error);
      break;
    default:
      throw new Error("Unhandled error code: " + e.detail.xhr.status + " - " + error.error);
  }
});

/*
When the preview modal is closed, remove the modal body.
*/
previewModal.addEventListener('hidden.bs.modal', (e) => {
  previewModal.querySelector('.modal-body').innerHTML = "";
});