// Application code for the transfer policy form.

import { choicesDefaultOptions } from '../modules/components.js';
import { selectCountry } from '../modules/countries.js';
import { selectNetwork } from '../modules/networks.js';

// Initialize the searchable jurisdiction picker.
selectCountry(document.getElementById('policy-jurisdictions'));

// Show the IVMS101 field picker only when there is an IVMS101 condition.
const ivmsMatch = document.getElementById('policy-ivms-match');
const ivmsFields = document.getElementById('policy-ivms-fields');

function toggleIVMSFields() {
  ivmsFields.hidden = ivmsFields.disabled = ivmsMatch.value === 'any';
}

toggleIVMSFields();
ivmsMatch.addEventListener('change', toggleIVMSFields);

// Enable the amount threshold fields when limit by amount is enabled.
const limitAmount = document.getElementById('policy-limit-amount');
const amountFields = document.getElementById('policy-amount-fields');

function toggleAmountFields() {
  amountFields.hidden = !limitAmount.checked;
  amountFields.disabled = !limitAmount.checked;
}

toggleAmountFields();
limitAmount.addEventListener('change', toggleAmountFields);
selectNetwork(document.getElementById('policy-amount-network'));

// Show settings for the selected action. Only Reject and Request repair show response fields.
const autoAction = document.getElementById('policy-auto');
const labelAction = document.getElementById('policy-label');
const autoFields = document.getElementById('policy-auto-fields');
const labelFields = document.getElementById('policy-label-fields');
const autoResponseFields = document.getElementById('policy-auto-response-fields');

function toggleAutoResponseFields() {
  const decision = autoFields.querySelector('input[name="autoDecision"]:checked')?.value;
  autoResponseFields.hidden = autoResponseFields.disabled = decision !== 'reject' && decision !== 'repair';
}

function toggleActionFields() {
  autoFields.hidden = autoFields.disabled = !autoAction.checked;
  labelFields.hidden = labelFields.disabled = !labelAction.checked;
}

toggleActionFields();
toggleAutoResponseFields();
autoFields.addEventListener('change', toggleAutoResponseFields);
document.querySelectorAll('input[name="action"]').forEach(action => {
  action.addEventListener('change', toggleActionFields);
});

const counterpartyOptions = document.getElementById('policy-counterparty-options');
// Reuse the disabled picker from the form for loading and error states.
const counterpartyPlaceholder = counterpartyOptions.querySelector('select').cloneNode(true);
let counterpartyRequest;
let counterpartyChoices;

function showCounterpartyStatus(message, errorMessage) {
  const select = counterpartyPlaceholder.cloneNode(true);
  select.options[0].textContent = message;
  counterpartyOptions.replaceChildren(select);
  if (errorMessage) {
    const error = document.createElement('div');
    error.id = 'policy-counterparties-error';
    error.className = 'form-text text-danger';
    error.textContent = errorMessage;
    select.setAttribute('aria-describedby', error.id);
    counterpartyOptions.append(error);
  }
}

// Clear selections for the previous protocol and tear down its Choices instance.
counterpartyOptions.addEventListener('htmx:beforeRequest', function (e) {
  counterpartyRequest = e.detail.xhr;
  counterpartyChoices?.destroy();
  counterpartyChoices = undefined;
  showCounterpartyStatus('Loading counterparties…');
});

// A late response for a previous protocol should not replace the current picker.
counterpartyOptions.addEventListener('htmx:beforeSwap', function (e) {
  if (e.detail.xhr !== counterpartyRequest) {
    e.preventDefault();
  }
});

// Initialize Choices when the replacement partial contains selectable counterparties.
counterpartyOptions.addEventListener('htmx:afterSettle', function (e) {
  if (e.detail.xhr === counterpartyRequest && !counterpartyChoices) {
    const select = counterpartyOptions.querySelector('select[data-choices]');
    if (!select) return;
    counterpartyChoices = new Choices(select, {
      ...JSON.parse(select.dataset.choices),
      ...choicesDefaultOptions(select),
      // Use the default Choices templates to render counterparty names as plain text.
      callbackOnCreateTemplates: null,
    });
  }
});

// Show an inline fallback when an HTTP or network error leaves no partial to swap.
counterpartyOptions.addEventListener('htmx:afterRequest', function (e) {
  if (e.detail.xhr === counterpartyRequest && (e.detail.failed || e.detail.xhr.status === 0)) {
    showCounterpartyStatus(
      'No counterparties available',
      'Unable to load counterparties for the selected protocol',
    );
  }
});
