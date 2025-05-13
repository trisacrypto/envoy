/*
Application code for the sunrise review page which includes accept/reject logic.
*/

import Alerts from '../modules/alerts.js';
import { selectNetwork } from '../modules/networks.js';
import { selectCountry } from '../modules/countries.js';
import { createFlatpickr } from '../modules/components.js';
import { selectAddressType, selectNationalIdentifierType } from '../modules/ivms101.js';

// Initialize the alerts component.
const alerts = new Alerts("#alerts");
const rejectAlerts = new Alerts("#rejectAlerts");


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

  // Also initialize date of birth flatpickr
  document.querySelectorAll('[data-flatpickr]').forEach((elem) => {
    createFlatpickr(elem);
  });
}

/*
Handle collapsing and showing all extended information sections in the form.
*/
function initializeExtended() {
  // Find all buttons with the data-toggle extended attribute.
  document.querySelectorAll("[data-toggle='extended']").forEach((btn) => {
    // Get the target element to toggle.
    const target = document.querySelector(btn.getAttribute("data-bs-target"));
    const extend = new Collapse(target, {toggle: true});

    // Add a click event listener to each button.
    btn.addEventListener("click", (e) => {
      extend.toggle();
    });

    // Add a hidden event listener to the target element.
    target.addEventListener("hide.bs.collapse", (e) => {
      btn.innerHTML = '<i class="fe fe-eye"></i> Show Details';
    });

    // Add a shown event listener to the target element.
    target.addEventListener("show.bs.collapse", (e) => {
      btn.innerHTML = '<i class="fe fe-eye-off"></i> Hide Details';
    });

  });
}

/*
Handle htmx errors as JSON responses from the backend.
*/
document.body.addEventListener("htmx:responseError", (e) => {
  // Try to parse the error response.
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

  // Scroll to the top of the page.
  window.scrollTo(0, 0);

  switch (e.detail.xhr.status) {
    case 400:
      alerts.warning("Bad request", error.error);
      break;
    case 404:
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


// Initialize the page (don't have to wait for HTMX in this case).
initializeChoices(document);
initializeExtended();