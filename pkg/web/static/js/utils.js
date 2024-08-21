export function setSuccessToast(msg) {
  const successToast = document.getElementById('success-toast');
  const successToastMsg = document.getElementById('success-toast-msg');
  successToast.classList.remove('hidden');
  successToastMsg.textContent = msg;

  // Remove the toast message.
  setTimeout(() => {
    successToast.classList.add('hidden');
  }, 2250);
}