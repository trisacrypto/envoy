/*
Application code for the customer account management detail page.
*/

import { createList } from '../modules/components.js';
import { isRequestMatch } from '../htmx/helpers.js';
import EditWallet from './editwallet.js';
import Alerts from '../modules/alerts.js';


/*
When the edit button is pressed in the crypto addresses list, show the edit crypto
address modal and populate the modal contents with the data attributes from the row in
the table. When hidden, make sure the modal is reset to its previous ready state so that
it can also be used as the create crypto address modal.
*/
const editCryptoAddressModal = new EditWallet("#editCryptoAddressModal");


// Create alert managers for the page
const editCryptoAddressAlerts = new Alerts("#editCryptoAddressAlerts");

/*
Post-event handling after htmx has settled the DOM.
*/
document.addEventListener("htmx:afterSettle", function(e) {
  // Initialize the counterparties list.
  if (isRequestMatch(e, "/v1/accounts/[0-7][0-9A-HJKMNP-TV-Z]{25}/crypto-addresses", "get")) {
    const table = document.getElementById('cryptoAddressList');
    if (table) createList(table);
  }
});

/*
Post-event handling when the crypto-addresses-updated event is fired.
*/
document.body.addEventListener("crypto-addresses-updated", function(e) {
  const elt = e.detail?.elt;
  if (elt) {
    if (elt.id === 'editCryptoAddressForm') {
      const modal = Modal.getInstance(document.getElementById("editCryptoAddressModal"));
      modal.hide();
    }

    if (elt.id === 'deleteCryptoAddressBtn') {
      const modal = Modal.getInstance(document.getElementById("deleteCryptoAddressModal"));
      modal.hide();
    }
  }
});

/*
Handle any htmx errors that are not swapped by the htmx config.
*/
document.body.addEventListener("htmx:responseError", function(e) {
  // Handle errors for create API key modal
  if (isRequestMatch(e, "/v1/accounts/[0-7][0-9A-HJKMNP-TV-Z]{25}/crypto-addresses", "post") || isRequestMatch(e, "/v1/accounts/[0-7][0-9A-HJKMNP-TV-Z]{25}/crypto-addresses/[0-7][0-9A-HJKMNP-TV-Z]{25}", "put")) {
    const error = JSON.parse(e.detail.xhr.response);
    switch (e.detail.xhr.status) {
      case 400:
        editCryptoAddressAlerts.danger("Error:", error.error);
        break;
      case 409:
        editCryptoAddressAlerts.danger("Conflict:", error.error);
        break;
      case 422:
        editCryptoAddressAlerts.danger("Validation error:", error.error);
        break;
      default:
        break;
    }
    return;
  }

  // If the error is unhandled; throw it
  throw new Error(`unhandled htmx error: status ${e.detail.xhr.status}`);
});

/*
When the delete crypto address button is pressed in the list, show the modal and
populate the modal contents with the data attributes from the row in the table.
When hidden, make sure the modal is reset to its previous ready state.
*/
const deleteCryptoAddressModal = document.getElementById("deleteCryptoAddressModal");
if (deleteCryptoAddressModal) {
  const deleteBtn = deleteCryptoAddressModal.querySelector("#deleteCryptoAddressBtn");
  const baseDeleteURL = deleteBtn.getAttribute("hx-delete");

  deleteCryptoAddressModal.addEventListener("show.bs.modal", function(event) {
    const button = event.relatedTarget;
    deleteCryptoAddressModal.querySelector("#deleteCryptoAddress").textContent = button.dataset.bsCryptoAddress || "—";

    deleteBtn.setAttribute("hx-delete", baseDeleteURL + button.dataset.bsCryptoAddressId);
    htmx.process(deleteBtn);
  });

  deleteCryptoAddressModal.addEventListener("hidden.bs.modal", function(event) {
    deleteCryptoAddressModal.querySelector("#deleteCryptoAddress").textContent = "";

    deleteBtn.removeAttribute("hx-delete");
    htmx.process(deleteBtn);
  });
}