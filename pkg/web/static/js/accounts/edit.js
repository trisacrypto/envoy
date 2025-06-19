/*
Application code for the customer account management edit page.
*/

import { isRequestMatch } from '../htmx/helpers.js';
import Alerts from '../modules/alerts.js';
import { selectCountry } from '../modules/countries.js';
import { selectAddressType, selectNationalIdentifierType, encode } from '../modules/ivms101.js';

// Initialize the alerts component.
const alerts = new Alerts(document.getElementById("alerts"), {autoClose: true});

/*
Initialize the select elements with choices.js in the form element.
*/
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
}

// Initialize the form choices when the page loads.
initializeChoices(document);

/*
Pre-flight request configuration for htmx requests.
*/
document.body.addEventListener("htmx:configRequest", function(e) {
  /*
  Prepare IVMS101 record for updating the account endpoint.
  */
  if (isRequestMatch(e, "/v1/accounts/[0-7][0-9A-HJKMNP-TV-Z]{25}", "put")) {
    // Prepare the outgoing data as a nested JSON payload
    const form = e.target;
    const formData = new FormData(form);

    const data = {
      id: "",
      customer_id: "",
      first_name: "",
      last_name: "",
      ivms101: {
        naturalPerson: {
          name: {
            nameIdentifier: []
          },
          geographicAddress: [],
          nationalIdentification: {},
          customerIdentification: "",
          dateAndPlaceOfBirth: {},
          countryOfResidence: ""
        }
      }
    }

    var hasNationalIdentifier = false;
    var hasDateAndPlaceOfBirth = false;
    var hasGeographicAddress = false;

    formData.entries().forEach(([key, value]) => {
      // Filter choices.js search terms
      if (key === "search_terms") {
        return;
      }

      // Handle non-IVMS101 fields.
      if (!key.startsWith("ivms_")) {
        data[key] = value; // Keep non-IVMS101 fields as is.
        return;
      }

      // Handle IVMS101 fields, nested objects and arrays.
      key = key.replace("ivms_", "");
      let obj = data.ivms101.naturalPerson;

      if (key.startsWith("name_")) {
        key = key.replace("name_", "");
        obj = obj.name;

        if (key.startsWith("nameIdentifier_")) {
          key = key.replace("nameIdentifier_", "");
          const idx = parseInt(key.split("_", 1)[0]);
          key = key.replace(idx + "_", "");

          if (obj.nameIdentifier.length <= idx) {
            obj.nameIdentifier.push({});
          }

          obj = obj.nameIdentifier[idx];
        }

      } else if (key.startsWith("geographicAddress_")) {
        key = key.replace("geographicAddress_", "");
        const idx = parseInt(key.split("_", 1)[0]);
        key = key.replace(idx + "_", "");

        if (obj.geographicAddress.length <= idx) {
          obj.geographicAddress.push({"addressLines": ["", "", ""]});
        }

        obj = obj.geographicAddress[idx];

        if (key.startsWith("addressLines")) {
          key = parseInt(key.replace("addressLines_", ""));
          obj = obj.addressLines;

          if (value !== "") {
            hasGeographicAddress = true;
          }
        }

      } else if (key.startsWith("nationalIdentification_")) {
        key = key.replace("nationalIdentification_", "");
        obj = obj.nationalIdentification;

        if (key === "nationalIdentifier" && value !== "") {
          hasNationalIdentifier = true;
        }
      } else if (key.startsWith("dateAndPlaceOfBirth_")) {
        key = key.replace("dateAndPlaceOfBirth_", "");
        obj = obj.dateAndPlaceOfBirth;

        if (value !== "") {
          hasDateAndPlaceOfBirth = true;
        }
      }

      obj[key]  = value;
    });

    // Remove empty values from optional objects.
    if (!hasGeographicAddress) {
      data.ivms101.naturalPerson.geographicAddress = [];
    }

    if (!hasNationalIdentifier) {
      data.ivms101.naturalPerson.nationalIdentification = null;
    }

    if (!hasDateAndPlaceOfBirth) {
      data.ivms101.naturalPerson.dateAndPlaceOfBirth = null;
    }

    // Prepare the parameters to send via HTMX as string data
    e.detail.parameters = new FormData();
    for (const [key, value] of Object.entries(data))  {
      if (typeof value === "object") {
        e.detail.parameters.append(key, encode(value));
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


/*
Post-event handling when the accounts-updated event is fired.
*/
document.body.addEventListener("accounts-updated", function(e) {
  const elt = e.detail?.elt;
  if (elt) {
    if (elt.id === 'deleteBtn') {
      // Redirect to the accounts index page after the account is deleted.
      window.location.href = "/accounts";
    }

    if (elt.getAttribute("id") === 'editAccountForm') {
      alerts.success("", "Account updated successfully.");
    }
  }
});

/*
Handle any htmx errors that are not swapped by the htmx config.
*/
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