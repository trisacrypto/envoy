/*
Application code for the customer account related transfers page.
*/

import { createList } from '../modules/components.js';

/*
Post-event handling after htmx has settled the DOM.
*/
document.addEventListener("htmx:afterSettle", function(e) {
  // Initialize the transfers list.
  const table = document.getElementById('transactionsList');
  if (table) createList(table);

  // Initialize the status tooltips
  const tooltips = document.querySelectorAll('[data-bs-toggle="tooltip"]');
  tooltips.forEach(tooltip => {
    new Tooltip(tooltip);
  });
});