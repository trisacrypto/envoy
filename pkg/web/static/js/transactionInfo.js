import { REJECT_CODES } from "./constants.js";
import { setSuccessToast } from "./utils.js";

const transactionEl = document.getElementById('transaction-id');
const transactionID = transactionEl?.value

// Add code to run after HTMX settles the DOM once a swap occurs.
document.body.addEventListener('htmx:afterSettle', (e) => {
  const envelopeListEP = `/v1/transactions/${transactionID}/secure-envelopes`
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

    // Reset transaction complete form if user closes the modal without submitting.
    const transactionCompleteCloseBtn = document.getElementById('transaction-complete-close-btn');
    transactionCompleteCloseBtn?.addEventListener('click', () => {
      const transactionCompleteForm = document.getElementById('transaction-complete-form');
      transactionCompleteForm.reset();
    });

    // TODO: Add code to toggle show/hide for PKS.
  }

  // Humanize the last update timestamp.
  if (e.detail.requestConfig.path === `/v1/transactions/${transactionID}?detail=full` && e.detail.requestConfig.verb === 'get') {
    const lastUpdate = document.querySelector('.trans-last-update');
    const humanizeLastUpdate = dayjs(lastUpdate.textContent).fromNow();
    lastUpdate.textContent = humanizeLastUpdate;
  };

  const envelopeID = document.getElementById('envelope-id')?.value;
  if (e.detail.requestConfig.path === `/v1/transactions/${transactionID}/secure-envelopes/${envelopeID}` && e.detail.requestConfig.verb === 'get') {
    // Humanize the reject error code.
    const errorCode = document.querySelectorAll('.error-code');
    errorCode?.forEach((code) => {
      const errorCodeText = code?.textContent;
      const readableErrorCode = REJECT_CODES[errorCodeText];
      code.textContent = readableErrorCode;
    });
  };

  // Disable the submit button after form submission.
  disableSubmitBtn();
});

// Add code to amend the request parameters before the request is sent.
const rejectEP = `/v1/transactions/${transactionID}/reject`
const transactionSendEP = `/v1/transactions/${transactionID}/send`
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

  // Amend data to include the txid and transfer state to complete the transaction.
  if (e.detail.path === transactionSendEP && e.detail.verb === 'post') {
    const params = e.detail.parameters;
    let envelope = JSON.parse(params.envelope)
    console.log(envelope)
    let data = {
      identity: envelope.identity,
      transaction: envelope.transaction,
    };

    data.transaction.txid = params.txid;
    data.transfer_state = 'completed';
    e.detail.parameters = data;
  };
});

// Reset the reject transaction form if the request is successful.
document.body.addEventListener('htmx:afterRequest', (e) => {
  if (e.detail.requestConfig.path === rejectEP && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    const transactionRejectForm = document.getElementById('transaction-reject-form')
    const transactionRejectionModal = document.getElementById('transaction_rejection_modal')
    transactionRejectionModal.close()
    transactionRejectForm.reset()
    setSuccessToast('Success! The secure envelope has been rejected.')
  }

  if (e.detail.requestConfig.path === transactionSendEP && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    const transactionCompleteForm = document.getElementById('transaction-complete-form');
    const transactionCompleteModal = document.getElementById('transaction_complete_modal');
    transactionCompleteModal.close();
    transactionCompleteForm.reset();
    enableSubmitBtn();
    setSuccessToast('Success! The on-chain transaction has been sent to the counterparty.');
  };
});

// Display success toast message if user is redirected to info page after accepting a transaction.
const transactionSend = Cookies.get('transaction_send_success')
if (transactionSend === 'true') {
  setSuccessToast('Success! The secure envelope has been accepted.')
}

function disableSubmitBtn() {
  const submitBtn = document.getElementById('transaction-complete-btn');
  const submitBtnText = document.getElementById('complete-btn-text');
  document.body.addEventListener('submit', () => {
    submitBtn?.setAttribute('disabled', 'disabled');
    submitBtnText?.classList.add('hidden');
  });
};

function enableSubmitBtn() {
  const submitBtn = document.getElementById('transaction-complete-btn');
  const submitBtnText = document.getElementById('complete-btn-text');
  submitBtn?.removeAttribute('disabled');
  submitBtnText?.classList.remove('hidden');
};