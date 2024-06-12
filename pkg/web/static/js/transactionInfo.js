const transactionID = document.getElementById('transaction-id');


// Add code to run after HTMX settles the DOM once a swap occurs.
document.body.addEventListener('htmx:afterSettle', (e) => {
  const envelopeListEP = `/v1/transactions/${transactionID?.value}/secure-envelopes`
  if (e.detail.requestConfig.path === envelopeListEP && e.detail.requestConfig.verb === 'get') {
    // Toggle the collapse-close class when an accordion is clicked.
    document.querySelectorAll('.envelope-accordion').forEach((accordion) => {
      accordion.addEventListener('click', () => {
        accordion.classList.toggle('collapse-close');

        // If an accordion is open, close all other accordions.
        if (!accordion.classList.contains('collapse-close')) {
          document.querySelectorAll('.envelope-accordion').forEach((otherAccordion) => {
            if (otherAccordion !== accordion) {
              otherAccordion.classList.add('collapse-close');
            }
          });
        }
      });
    });

    // TODO: Add code to toggle show/hide for PKS.
  }
});

// Add code to amend the request parameters before the request is sent.
const rejectEP = `/v1/transactions/${transactionID?.value}/reject`

document.body.addEventListener('htmx:configRequest', (e) => {
  // Determine if the request repair checkbox is checked and add to the request parameters.
  const isRetryChecked = document.getElementById('request_retry')
  if (e.detail.path === rejectEP && e.detail.verb === 'post') {
    const retryTransaction = isRetryChecked?.checked;
    e.detail.parameters = {
      ...e.detail.parameters,
      request_retry: retryTransaction,
    };
  };
});

// Reset the reject transaction form if the request is successful.
document.body.addEventListener('htmx:afterRequest', (e) => {
  if (e.detail.requestConfig.path === rejectEP && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    const transactionRejectForm = document.getElementById('transaction-reject-form')
    transactionRejectForm.reset()
  }
});
