// Disable submit button to prevent multiple form submissions.
function disableSubmitButton() {
  const acceptSbmtBtn = document.getElementById('accept-sbmt-btn');
  const acceptBtnText = document.getElementById('accept-btn-text');
  const acceptForm = document.getElementById('accept-form');

  acceptForm?.addEventListener('submit', () => {
    acceptBtnText?.classList.add('hidden');
    acceptSbmtBtn.disabled = true;
  });
};

// Enable submit button after a request.
function enableSubmitButton() {
  const acceptSbmtBtn = document.getElementById('accept-sbmt-btn');
  const acceptBtnText = document.getElementById('accept-btn-text');

  acceptBtnText?.classList.remove('hidden');
  acceptSbmtBtn.disabled = false;
}

document.body.addEventListener('htmx:afterSettle', disableSubmitButton);
document.body.addEventListener('htmx:afterRequest', enableSubmitButton);
