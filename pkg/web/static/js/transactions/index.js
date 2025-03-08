/*
Application code for the transaction inbox dashboard page.
*/

import { createList, createPageSizeSelect } from '../modules/components.js';
import { isRequestFor } from '../htmx/helpers.js';
// import { Tooltip } from 'bootstrap';

/*
Post-event handling after htmx has settled the DOM.
*/
document.addEventListener("htmx:afterSettle", function(e) {
  /*
  Whenever the apikey list is refreshed, make sure the pagination and list controls are
  re-initialized since the list table is coming from the HTMX request.
  */
  if (isRequestFor(e, "/v1/transactions", "get")) {
    const cpList = document.getElementById('transactionList');
    const list = createList(cpList);

    // Initialize Page Size Select
    const pageSizeSelect = document.getElementById('pageSizeSelect');
    createPageSizeSelect(pageSizeSelect, list);

    // Initialize the status tooltips
    const tooltips = document.querySelectorAll('[data-bs-toggle="tooltip"]');
    tooltips.forEach(tooltip => {
      new Tooltip(tooltip);
    });

    return;
  };
});