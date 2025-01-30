import { countriesArray, addressTypeArray, naturalPersonNtlIdTypeArray, nationalIdentifierTypeArray } from "./constants.js";
import { setSuccessToast } from "./utils.js";

// Set data for SlimSelect elements that appear in the send envelope and send message forms.
const country = 'country';
const natnitc = 'natnitc';
const legnitc = 'legnitc';
const address = 'address';

function initializeSlimSelects()  {
  const elements = document.querySelectorAll('[data-select-type]');
  elements.forEach((element) => setSlimSelect(element));
}

function setSlimSelect(element) {
  // Initialize SlimSelect only for select elements that exist in the DOM.
  if (!element) {
    return
  };

  const selectType = element.getAttribute('data-select-type');
  const newDropdown = new SlimSelect({
    select: '#' + element.getAttribute('id'),
  });

  if (selectType === country) {
    countriesArray.unshift({ 'placeholder': true, 'text': 'Select a country', 'value': '' });
    newDropdown.setData(countriesArray);
  };

  if (selectType === address) {
    addressTypeArray.unshift({ 'placeholder': true, 'text': 'Select address type', 'value': '' });
    newDropdown.setData(addressTypeArray);
  };

  if (selectType === natnitc) {
    naturalPersonNtlIdTypeArray.unshift({ 'placeholder': true, 'text': 'Select national identifier type', 'value': '' });
    newDropdown.setData(naturalPersonNtlIdTypeArray);
  };

  if (selectType === legnitc) {
    nationalIdentifierTypeArray.unshift({ 'placeholder': true, 'text': 'Select national identifier type', 'value': '' });
    newDropdown.setData(nationalIdentifierTypeArray);
  };

  // Set the default value on the dropdown if present.
  const value = element?.getAttribute('value');
  if (value) {
    newDropdown.setSelected(value);
  };
};


// Initialie all dropdown slim selects in DOM
initializeSlimSelects()

// HTMX event listeners
const rejectEP = "/sunrise/reject";
const acceptEP = "/sunrise/accept";
const rejectBtn = "#reject-btn";

document.body.addEventListener('htmx:configRequest', (e) => {
  const isRetryChecked = document.getElementById('retry');
  if (e.detail.path === rejectEP && e.detail.verb === 'post') {
    const retryTransaction = isRetryChecked?.checked;
    e.detail.parameters = {
      ...e.detail.parameters,
      retry: retryTransaction,
    };
  }
});

// Reset the reject transaction form if the request is successful.
document.body.addEventListener('htmx:afterRequest', (e) => {
  if (e.detail.requestConfig.path === rejectEP && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    const transactionRejectForm = document.getElementById('transaction-reject-form')
    const transactionRejectionModal = document.getElementById('transaction_rejection_modal')
    transactionRejectionModal.close()
    transactionRejectForm.reset()
    setSuccessToast('Thank you! The sunrise message has been rejected.')
  }
});
