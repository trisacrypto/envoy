
// Add code to run after HTMX settles the DOM once a swap occurs.
document.body.addEventListener('htmx:afterSettle', (e) => {
  const transactionID = document.getElementById('transaction-id');
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
})