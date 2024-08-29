import { API_KEY_PERMISSIONS } from "./constants.js"
import { setSuccessToast } from "./utils.js";

const apiKeysEP = '/v1/apikeys';
const addApiKeyModal = document.getElementById('add_apikey_modal');
const revokeApiKeyModal = document.getElementById('revoke_key_modal');
const closeApiKeyModal = document.getElementById('close-apikey-modal');
const addApiKeyForm = document.getElementById('new-apikey-form');
const fullCheckbox = document.getElementById('full_access');
const customAccess = document.getElementById('custom-access');
const customCheckbox = document.querySelectorAll('.custom-access');

// Reset create API key form if user closes modal without submitting.
if (closeApiKeyModal) {
  closeApiKeyModal.addEventListener('click', () => {
    addApiKeyModal.close();
    addApiKeyForm.reset();
    fullCheckbox.disabled = false;
    customCheckbox.forEach((checkbox) => {
      checkbox.disabled = false;
    });
  });
  // Scroll to the top of the custom access section in the modal.
  customAccess?.scrollTo(0, 0);
};

// Toggle disabled state of full and custom access checkboxes depending on the user's selection.
function toggleCheckboxState(isChecked) {
  customCheckbox.forEach((checkbox) => {
    checkbox.disabled = isChecked;
    if (isChecked) {
      checkbox.checked = false;
    };
  });
};

// Check if any custom access checkboxes are checked.
function isCustomChecked() {
  return Array.from(customCheckbox).some((checkbox) => checkbox.checked);
}

fullCheckbox?.addEventListener('change', () => {
  toggleCheckboxState(fullCheckbox.checked);
});

customCheckbox.forEach((checkbox) => {
  checkbox.addEventListener('change', () => {
    fullCheckbox.disabled = isCustomChecked();
    if (checkbox.checked) {
      fullCheckbox.checked = false;
    };
  });
})

// Add code to amend htmx requests before they are sent.
document.body.addEventListener('htmx:configRequest', (e) => {
  // Set API key permissions if the user selects full access.
  if (e.detail.path === apiKeysEP && e.detail.verb === 'post') {
    const params = e.detail.parameters

    // If full access is selected, send all permissions values to the BE.
    if (params.permissions === 'full') {
      params.permissions = API_KEY_PERMISSIONS || params.permissions
    };

    // If only 1 param is selected and its value isn't full, send it as an array.
    if (params.permission !== 'full' && typeof (params.permissions) === 'string') {
      params.permissions = [params.permissions]
    };
  };
});

document.body.addEventListener('htmx:afterRequest', (e) => {
  if (e.detail.requestConfig.path === apiKeysEP && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    addApiKeyModal.close();
    addApiKeyForm.reset();
    fullCheckbox.disabled = false;
    customCheckbox.forEach((checkbox) => {
      checkbox.disabled = false;
    });
    // Scroll to the top of the custom access section in the modal.
    customAccess.scrollTo(0, 0);
    setSuccessToast('Success! The API key has been created.');
  };

  if (e.detail.requestConfig.verb === 'put' && e.detail.successful) {
    const editApiKeyModal = document.getElementById('edit_key_modal');
    const editApiKeyForm = document.getElementById('edit-key-form');
    editApiKeyModal.close();
    editApiKeyForm.reset();
    setSuccessToast('Success! The API key description has been updated.');
  };
});

// Add code to run after htmx settles the DOM once a swap occurs.
document.body.addEventListener('htmx:afterSettle', (e) => {
  // Copy client secret and client ID to clipboard.
  if (e.detail.requestConfig.path === apiKeysEP && e.detail.requestConfig.verb === 'post') {
    const copyIdBtn = document.getElementById('copy-id-btn');
    const copySecretBtn = document.getElementById('copy-secret-btn');
    copyIdBtn.addEventListener('click', copyClientID);
    copySecretBtn.addEventListener('click', copyClientSecret);
  };

  if (e.detail.requestConfig.verb === 'delete' && e.detail.successful) {
    revokeApiKeyModal.close();
    setSuccessToast('Success! The API key has been revoked.');
  };

  // Get API key ID and set the delete API key endpoint.
  const openRevokeBtn = document.getElementById('open-revoke-btn');
  openRevokeBtn?.addEventListener('click', () => {
    const deleteKeyEP = openRevokeBtn.dataset.deleteEp;
    const revokeKeyBtn = document.getElementById('revoke-key-btn');
    revokeKeyBtn.setAttribute('hx-delete', deleteKeyEP);
    // Initialize htmx attribute added to the revoke key button.
    htmx.process(revokeKeyBtn);

    // Get and set the API key description in the revoke key modal.
    const keyDescription = openRevokeBtn.dataset.description
    const keyDescriptionEl = document.getElementById('key-description');
    keyDescriptionEl.textContent = keyDescription;
  });
});

function copyClientID() {
  const clientID = document.getElementById('client-id').textContent;
  // The clipboard API is only available in secure contexts.
  navigator.clipboard.writeText(`Client ID: ${clientID}`);

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
  // The clipboard API is only available in secure contexts.
  navigator.clipboard.writeText(`Client Secret: ${clientSecret}`);

  const copySecretIcon = document.getElementById('copy-secret-icon');
  copySecretIcon.classList.remove('fa-copy');
  copySecretIcon.classList.add('fa-circle-check');

  setTimeout(() => {
    copySecretIcon.classList.remove('fa-circle-check');
    copySecretIcon.classList.add('fa-copy');
  }, 1000);
};