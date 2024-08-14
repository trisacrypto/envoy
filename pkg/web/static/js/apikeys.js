import { API_KEY_PERMISSIONS } from "./constants.js"
import { setSuccessToast } from "./utils.js";

const apiKeysEP = '/v1/apikeys';
const addApiKeyModal = document.getElementById('add_apikey_modal');
const addApiKeyForm = document.getElementById('new-apikey-form');

// Reset add api key modal form if user closes the modal.
if (addApiKeyModal) {
  addApiKeyModal.addEventListener('click', () => {
    addApiKeyForm?.reset()
  });
};

// Add code to amend htmx requests before they are sent.
document.body.addEventListener('htmx:configRequest', (e) => {
  // Set API key permissions if the user selects full access.
  if (e.detail.path === apiKeysEP && e.detail.verb === 'post') {
    const params = e.detail.parameters
    const permissions = params.permissions === 'full' ? API_KEY_PERMISSIONS : params.permissions
    params.permissions = permissions
  };
});

// Add code to run after an htmx request.
document.body.addEventListener('htmx:afterRequest', (e) => {
  // Close the add API key modal and reset the form after a successful request.
  if (e.detail.requestConfig.path === apiKeysEP && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    addApiKeyModal.close()
    addApiKeyModal.reset()
    setSuccessToast('Success! The API key has been created.')
  };
});

// Add code to run after htmx settles the DOM once a swap occurs.
document.body.addEventListener('htmx:afterSettle', (e) => {
  // Add code to copy client secret and client ID to clipboard.
  if (e.detail.requestConfig.path === apiKeysEP && e.detail.requestConfig.verb === 'post') {
    const copyIdBtn= document.getElementById('copy-id-btn');
    const copySecretBtn = document.getElementById('copy-secret-btn');

    if (copyIdBtn) {
      copyIdBtn.addEventListener('click', copyClientID);
    };

    if (copySecretBtn) {
      copySecretBtn.addEventListener('click', copyClientSecret);
    };
  };
});

// TODO: Create one function to copy client ID and client secret.
function copyClientID() {
  const clientID = document.getElementById('client-id').textContent;
  navigator.clipboard.writeText(clientID);

  const copyIdIcon = document.getElementById('copy-id-icon');
  copyIdIcon.classList.remove('fa-copy');
  copyIdIcon.classList.add('fa-circle-check');

  setTimeout(() => {
    copyIdIcon.classList.remove('fa-circle-check');
    copyIdIcon.classList.add('fa-copy');
  }, 1000);
}

function copyClientSecret() {
  const clientSecret = document.getElementById('client-secret').textContent;
  navigator.clipboard.writeText(clientSecret);

  const copySecretIcon = document.getElementById('copy-secret-icon');
  copySecretIcon.classList.remove('fa-copy');
  copySecretIcon.classList.add('fa-circle-check');

  setTimeout(() => {
    copySecretIcon.classList.remove('fa-circle-check');
    copySecretIcon.classList.add('fa-copy');
  }, 1000);
};