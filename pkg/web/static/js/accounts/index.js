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
      case 422:
        alertError("createAccountAlerts", "Validation error:", error.error);
        break;
      default:
        break;
    }
    return;
  }
});

/*
Ensure that when the create account form gets reset, so do the wallet rows in the modal.
*/
document.getElementById('createAccountForm').addEventListener('reset', function(e) {
  walletRows.reset(e);
});
