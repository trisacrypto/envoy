/*
Application code for the counterparty management edit page.
*/

import { isRequestMatch } from '../htmx/helpers.js';
import Alerts from '../modules/alerts.js';
import { selectCountry } from '../modules/countries.js';
import { selectAddressType, selectNationalIdentifierType, LegalPerson, encode } from '../modules/ivms101.js';


// Initialize the alerts component.
const alerts = new Alerts("#alerts", {autoClose: true});

/*
Initialize select elements with choices.js.
*/
function initializeChoices(elem) {
  // Initialize country select choices.
  elem.querySelectorAll('[data-countries]').forEach(choice => {
    selectCountry(choice);
  });

  // Initialize address type select choices.
  elem.querySelectorAll('[data-address-type]').forEach(choice => {
    selectAddressType(choice);
  });

  // Initialize national identifier type select choices.
  elem.querySelectorAll('[data-national-identifier-type]').forEach(choice => {
    selectNationalIdentifierType(choice);
  });
}

// Initialize page form choices.
initializeChoices(document);

function hasValue(value) {
  return value !== null && value !== undefined && String(value).trim() !== "";
}

function hasAddressData(address) {
  if (!address || typeof address !== "object") return false;
  if (hasValue(address.addressType) || hasValue(address.country)) return true;

  const lines = Array.isArray(address.addressLine) ? address.addressLine : [];
  return lines.some(line => hasValue(line));
}

function hasNationalIDData(nationalIdentification) {
  if (!nationalIdentification || typeof nationalIdentification !== "object") return false;
  return hasValue(nationalIdentification.nationalIdentifier) ||
    hasValue(nationalIdentification.nationalIdentifierType) ||
    hasValue(nationalIdentification.registrationAuthority) ||
    hasValue(nationalIdentification.countryOfIssue);
}

function hasIVMSData(legalPerson) {
  if (!legalPerson || typeof legalPerson !== "object") return false;

  const identifiers = legalPerson.name?.nameIdentifier || [];
  const hasName = identifiers.some(item => hasValue(item.legalPersonName));

  return hasName ||
    hasValue(legalPerson.customerNumber) ||
    hasValue(legalPerson.countryOfRegistration) ||
    hasNationalIDData(legalPerson.nationalIdentification) ||
    (Array.isArray(legalPerson.geographicAddress) && legalPerson.geographicAddress.some(address => hasAddressData(address)));
}

function sanitizeLegalPerson(legalPerson, fallbackName, fallbackCountry) {
  if (!legalPerson.name || !Array.isArray(legalPerson.name.nameIdentifier)) {
    legalPerson.name = {nameIdentifier: []};
  }

  if (!legalPerson.name.nameIdentifier[0]) {
    legalPerson.name.nameIdentifier[0] = {};
  }

  const primary = legalPerson.name.nameIdentifier[0];
  if (!hasValue(primary.legalPersonName) && hasValue(fallbackName)) {
    primary.legalPersonName = fallbackName;
  }

  if (!hasValue(primary.legalPersonNameIdentifierType)) {
    primary.legalPersonNameIdentifierType = "LEGL";
  }

  if (!hasValue(legalPerson.countryOfRegistration) && hasValue(fallbackCountry)) {
    legalPerson.countryOfRegistration = fallbackCountry;
  }

  if (!hasNationalIDData(legalPerson.nationalIdentification)) {
    legalPerson.nationalIdentification = null;
  }

  if (Array.isArray(legalPerson.geographicAddress)) {
    legalPerson.geographicAddress = legalPerson.geographicAddress.filter(address => hasAddressData(address));
  } else {
    legalPerson.geographicAddress = [];
  }

  if (!hasValue(legalPerson.customerNumber)) {
    delete legalPerson.customerNumber;
  }
}

/*
Pre-flight request configuration for htmx requests.
*/
document.body.addEventListener("htmx:configRequest", function(e) {
  /*
  Prepare IVMS101 record for updating the counterparty endpoint.
  */
  if (isRequestMatch(e, "/v1/counterparties/[0-7][0-9A-HJKMNP-TV-Z]{25}", "put")) {
    const form = e.target;
    const formData = new FormData(form);

    const data = {
      id: formData.get("id"),
      protocol: formData.get("protocol"),
      common_name: formData.get("common_name"),
      endpoint: formData.get("endpoint"),
      name: formData.get("name"),
      website: formData.get("website"),
      country: formData.get("country")
    };

    // Build legal person data from prefixed form fields.
    const legalPerson = new LegalPerson(formData, {prefix: "ivms"}).toJSON().legalPerson;
    const existingIVMSRecord = formData.get("has_ivms_record") === "true";
    if (existingIVMSRecord || hasIVMSData(legalPerson)) {
      sanitizeLegalPerson(legalPerson, data.name, data.country);
      data.ivms101 = legalPerson;
    }

    // Prepare parameters via HTMX as string data.
    e.detail.parameters = new FormData();
    for (const [key, value] of Object.entries(data)) {
      if (typeof value === "object") {
        e.detail.parameters.append(key, encode(value));
      } else {
        e.detail.parameters.append(key, value || "");
      }
    }

    // Scroll to top so users can see alerts.
    window.scrollTo({
      top: 0,
      left: 0,
      behavior: 'smooth'
    });
  }
});

/*
Post-event handling when counterparties-updated is fired.
*/
document.body.addEventListener("counterparties-updated", function(e) {
  const elt = e.detail?.elt;
  if (elt) {
    if (elt.id === 'deleteBtn') {
      // Redirect to the counterparty list after delete.
      window.location.href = "/counterparties";
    }

    if (elt.id === 'editCounterpartyForm') {
      alerts.success("", "Counterparty updated successfully.");
    }
  }
});

/*
Handle htmx errors that are not swapped by the htmx config.
*/
document.body.addEventListener("htmx:responseError", function(e) {
  if (isRequestMatch(e, "/v1/counterparties/[0-7][0-9A-HJKMNP-TV-Z]{25}", "put") || isRequestMatch(e, "/v1/counterparties/[0-7][0-9A-HJKMNP-TV-Z]{25}", "delete")) {
    const error = JSON.parse(e.detail.xhr.response);
    switch (e.detail.xhr.status) {
      case 400:
        alerts.danger("Error:", error.error);
        break;
      case 403:
        alerts.danger("Forbidden:", error.error);
        break;
      case 404:
        alerts.danger("Not found:", error.error);
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
    return;
  }

  throw new Error(`unhandled htmx error: status ${e.detail.xhr.status}`);
});
