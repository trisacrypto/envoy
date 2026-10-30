// Application code for the counterparty management edit page.

import { isRequestMatch } from '../htmx/helpers.js';
import Alerts from '../modules/alerts.js';
import { selectBusinessCategory, selectVASPCategory } from '../modules/categories.js';
import { selectCountry } from '../modules/countries.js';
import { selectAddressType, selectNationalIdentifierType, encode, LegalPerson } from '../modules/ivms101.js';

// Initialize the alerts component.
const alerts = new Alerts(document.getElementById("alerts"), {autoClose: true});

// Initialize the select elements with choices.js in the form element.
function initializeChoices(elem) {
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

  // Initialize business category select.
  elem.querySelectorAll('[data-business-categories]').forEach(elem => {
    selectBusinessCategory(elem);
  });

  // Initialize VASP category select and restore multi-select values.
  elem.querySelectorAll('[data-vasp-categories]').forEach(elem => {
    const selected = elem.dataset.selected ? elem.dataset.selected.split(',').filter(Boolean) : [];
    // Avoid treating the comma-joined list as a single choice value.
    delete elem.dataset.selected;
    const choices = selectVASPCategory(elem);
    if (selected.length > 0) {
      choices.setChoiceByValue(selected);
    }
  });
}

// Initialize the form choices when the page loads.
initializeChoices(document);

// Pre-flight request configuration for htmx requests.
document.body.addEventListener("htmx:configRequest", function(e) {
  
  // Prepare IVMS101 legal person record for updating the counterparty endpoint.
  if (isRequestMatch(e, "/v1/counterparties/[0-7][0-9A-HJKMNP-TV-Z]{25}", "put")) {
    const form = e.target;
    const formData = new FormData(form);
    const legalPerson = new LegalPerson(form, {prefix: "ivms"});

    // Remove empty values from optional objects.
    if (!legalPerson.data.nationalIdentification?.nationalIdentifier) {
      legalPerson.data.nationalIdentification = null;
    }

    if (!legalPerson.data.geographicAddress?.some(addr => addr.addressLine?.some(line => line))) {
      legalPerson.data.geographicAddress = [];
    }

    const data = {
      id: "",
      source: "",
      protocol: "",
      common_name: "",
      endpoint: "",
      travel_address: "",
      name: "",
      website: "",
      country: "",
      business_category: "",
      vasp_categories: formData.getAll("vasp_categories").filter(Boolean),
      ivms101: encode(legalPerson.data),
    };

    formData.entries().forEach(([key, value]) => {
      // Filter choices.js search terms and IVMS/array fields handled above.
      if (key === "search_terms" || key.startsWith("ivms_") || key === "vasp_categories") {
        return;
      }

      if (Object.prototype.hasOwnProperty.call(data, key)) {
        data[key] = value;
      }
    });

    // Prepare the parameters to send via HTMX as string data
    e.detail.parameters = new FormData();
    for (const [key, value] of Object.entries(data))  {
      if (typeof value === "object") {
        e.detail.parameters.append(`json:${key}`, JSON.stringify(value));
      } else {
        e.detail.parameters.append(key, value);
      }
    }

    // Scroll to the top of the form to show alerts.
    window.scrollTo({
      top: 0,
      left: 0,
      behavior: 'smooth'
    });

  }
});


// Post-event handling when the counterparty-updated event is fired.
document.body.addEventListener("counterparty-updated", function(e) {
  const elt = e.detail?.elt;
  if (elt) {
    if (elt.id === 'deleteBtn') {
      // Redirect to the counterparties index page after the counterparty is deleted.
      window.location.href = "/counterparties";
      return;
    }

    if (elt.getAttribute("id") === 'editCounterpartyForm') {
      alerts.success("", "Counterparty updated successfully.");
    }
  }
});

// Handle any htmx errors that are not swapped by the htmx config.
document.body.addEventListener("htmx:responseError", function(e) {
  const error = JSON.parse(e.detail.xhr.response);
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
