import { REJECT_CODES } from "./constants.js"

document.body.addEventListener('htmx:afterSettle', (e) => {
  const idEl = document.getElementById('envelope-id')
  const id = idEl?.value
  if (e.detail.requestConfig.path === `/v1/transactions/${id}/repair` && e.detail.requestConfig.verb === 'get') {
  const errorCode = document.getElementById('repair-error')
    const errorMsg = errorCode.textContent.trim()
    const readableError = REJECT_CODES[errorMsg]
    errorCode.textContent = readableError
  };
});



// Disable submit button to prevent multiple form submissions.
function disableSubmitButton() {
  const repairSbmtBtn = document.getElementById('repair-sbmt-btn');
  const repairBtnText = document.getElementById('repair-btn-text');
  const repairForm = document.getElementById('repair-form');
  repairForm?.addEventListener('submit', () => {
    repairBtnText?.classList.add('hidden');
    repairSbmtBtn.disabled = true;
  });
};

// Enable submit button after a request.
function enableSubmitButton() {
  const repairSbmtBtn = document.getElementById('repair-sbmt-btn');
  const repairBtnText = document.getElementById('repair-btn-text');
  repairBtnText?.classList.remove('hidden');
  repairSbmtBtn.disabled = false;
}

document.body.addEventListener('htmx:afterSettle', disableSubmitButton);
document.body.addEventListener('htmx:afterRequest', enableSubmitButton);
