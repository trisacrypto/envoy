/*
Application code for the transaction inbox dashboard page.
*/

import { createList, createPageSizeSelect } from '../modules/components.js';
import { isRequestFor } from '../htmx/helpers.js';
import Filter from './filter.js';

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
    if (cpList) {
      const list = createList(cpList);

      // Initialize Page Size Select
      const pageSizeSelect = document.getElementById('pageSizeSelect');
      createPageSizeSelect(pageSizeSelect, list);
    }

    // Initialize the status tooltips
    const tooltips = document.querySelectorAll('[data-bs-toggle="tooltip"]');
    tooltips.forEach(tooltip => {
      new Tooltip(tooltip);
    });

    // Initialize filters
    const filterForm = document.getElementById('filterListForm');
    if (filterForm) {
      new Filter(filterForm);
    }

    return;
  };
});

/*
Post-event handling when the transactions-updated event is fired.
*/
document.body.addEventListener("transactions-updated", function(e) {
  const elt = e.detail?.elt;
  if (elt) {
    if (elt.id === 'archiveBtn') {
      const confirmArchiveModal = Modal.getInstance(document.getElementById("confirmArchiveTransferModal"));
      confirmArchiveModal.hide();
    }
  }
});

/*
When the archive button is clicked, show the confirmation modal and populate the modal
contents with the data attributes from the row in the table. When the modal is hidden,
make sure to reset it to its previous ready state.
*/
const confirmArchiveModal = document.getElementById('confirmArchiveTransferModal');
if (confirmArchiveModal) {
  const fields = ["Originator", "Beneficiary", "Status", "Counterparty", "Amount", "Network"];
  const archiveBtn = confirmArchiveModal.querySelector('#archiveBtn');

  confirmArchiveModal.addEventListener('show.bs.modal', function (event) {
    const button = event.relatedTarget;
    fields.forEach(field => {
      const span = confirmArchiveModal.querySelector(`#archive${field}`);
      span.textContent = button.dataset[`bs${field}`];

      if (span.id === 'archiveStatus') {
        span.classList.remove('bg-secondary');
        span.classList.add('bg-'+button.dataset.bsStatusColor);
      }
    });

    archiveBtn.setAttribute("hx-post", "/v1/transactions/"+button.dataset.bsTransactionId+"/archive");
    htmx.process(archiveBtn);
  });

  confirmArchiveModal.addEventListener('hidden.bs.modal', function (event) {
    fields.forEach(field => {
      const span = confirmArchiveModal.querySelector(`#archive${field}`);
      span.textContent = "";

      if (span.id === 'archiveStatus') {
        spanStatus.classList.remove('bg-primary', 'bg-info', 'bg-warning', 'bg-danger', 'bg-success', 'bg-light');
        spanStatus.classList.add('bg-secondary');
      }
    });

    archiveBtn.removeAttribute("hx-post");
    htmx.process(archiveBtn);
  });
}