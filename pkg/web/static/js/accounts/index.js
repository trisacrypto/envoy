/*
Application code for the customer account management dashboard page.
*/
import WalletRows from './walletrows.js';
import { createList, createPageSizeSelect } from '../modules/components.js';
import { isRequestFor } from '../htmx/helpers.js';


// Create the WalletRows manager for the create account form.
const walletRows = new WalletRows();

/*
Specialized add alerts function for the create account modal.

TODO: consider refactoring this into a more general alerts class.
*/
function alertError(id, title, message) {
  const alerts = document.getElementById(id);
  alerts.insertAdjacentHTML('beforeend', `
    <div class="alert alert-danger alert-dismissible fade show" role="alert">
      <strong>${title}</strong> ${message}.
      <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
    </div>
  `);
}

/*
Pre-flight request configuration for htmx requests.
*/
document.body.addEventListener("htmx:configRequest", function(e) {
  /*
  When creating a customer account, collect the wallet addresses into a nested JSON
  array to be posted to the backend.
  */
  if (isRequestFor(e, "/v1/accounts", "post")) {
    const data = e.detail.parameters;
    data.delete('search_terms');

    const wallets = data.getAll("crypto_addresses_crypto_address");
    const networks = data.getAll("crypto_addresses_network");

    const addresses = [];
    for (let i = 0; i < wallets.length; i++) {
      addresses.push({
        crypto_address: wallets[i],
        network: networks[i]
      });
    }

    data.append("json:crypto_addresses", JSON.stringify(addresses));
    data.delete('crypto_addresses_crypto_address');
    data.delete('crypto_addresses_network');
    e.detail.parameters = data;
  }

});

/*
Post-event handling after htmx has settled the DOM.
*/
document.addEventListener("htmx:afterSettle", function(e) {
  // Initialize List.js
  const cpList = document.getElementById('accountList');
  const list = createList(cpList);

  // Initialize Page Size Select
  const pageSizeSelect = document.getElementById('pageSizeSelect');
  createPageSizeSelect(pageSizeSelect, list);
});

/*
Post-event handling when the accounts-updated event is fired.
*/
document.body.addEventListener("accounts-updated", function(e) {
  const elt = e.detail?.elt;
  if (elt) {
    if (elt.id === 'deleteBtn') {
      const deleteAccountModal = Modal.getInstance(document.getElementById("deleteAccountModal"));
      deleteAccountModal.hide();
    }
  }
});

/*
Handle any htmx errors that are not swapped by the htmx config.
*/
document.body.addEventListener("htmx:responseError", function(e) {
  // Handle errors for create API key modal
  if (isRequestFor(e, "/v1/accounts", "post")) {
    const error = JSON.parse(e.detail.xhr.response);
    switch (e.detail.xhr.status) {
      case 400:
        alertError("createAccountAlerts", "Error:", error.error);
        break;
      case 409:
        alertError("createAccountAlerts", "Conflict:", error.error);
        break;
      case 422:
        alertError("createAccountAlerts", "Validation error:", error.error);
        break;
      default:
        throw new Error(`unhandled htmx error: ${error.error}`);
    }
    return;
  }

  // Handle errors for delete user by showing a toast alert.
  if (isRequestMatch(e, "/v1/accounts/[0-7][0-9A-HJKMNP-TV-Z]{25}", "delete")) {
    const error = JSON.parse(e.detail.xhr.response);
    alertError("pageAlerts", "Delete Account Error:", error.error);
    return;
  }

  // If the error is unhandled; throw it
  throw new Error(`unhandled htmx error: status ${e.detail.xhr.status}`);
});

/*
Ensure that when the create account form gets reset, so do the wallet rows in the modal.
*/
document.getElementById('createAccountForm').addEventListener('reset', function(e) {
  walletRows.reset(e);
});

/*
When the delete button is pressed in the list, show the modal and populate the modal
contents with the data attributes from the row in the table. When hidden, make sure
the modal is reset to its previous ready state.
*/
const deleteAccountModal = document.getElementById("deleteAccountModal");
if (deleteAccountModal) {
  deleteAccountModal.addEventListener("show.bs.modal", function(event) {
    const button = event.relatedTarget;
    deleteAccountModal.querySelector("#customerID").textContent = button.dataset.bsCustomerId || "—";
    deleteAccountModal.querySelector("#firstName").textContent = button.dataset.bsFirstName || "—";
    deleteAccountModal.querySelector("#lastName").textContent = button.dataset.bsLastName || "—";

    const deleteBtn = deleteAccountModal.querySelector("#deleteBtn");
    deleteBtn.setAttribute("hx-delete", "/v1/accounts/" + button.dataset.bsAccountId);
    htmx.process(deleteBtn);
  });

  deleteAccountModal.addEventListener("hidden.bs.modal", function(event) {
    deleteAccountModal.querySelector("#customerID").textContent = "";
    deleteAccountModal.querySelector("#firstName").textContent = "";
    deleteAccountModal.querySelector("#lastName").textContent = "";

    const deleteBtn = deleteAccountModal.querySelector("#deleteBtn")
    deleteBtn.removeAttribute("hx-delete");
    htmx.process(deleteBtn);
  });
}