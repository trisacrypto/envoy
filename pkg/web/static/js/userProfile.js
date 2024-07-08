// Helper function to add the same event handler for multiple events
const addEventListeners = (el, evts, fn) => {
  evts.split(' ').forEach(e => el.addEventListener(e, fn, false));
}

// Elements referenced by this script
const changePwdModal = document.getElementById('change_pwd_modal');
const changePwdForm = document.getElementById('change-pwd-form')
const closePwdModalBtn = document.getElementById('close-pwd-mdl-btn');
const passwordInput = document.getElementById('password');
const confirmPassword = document.getElementById('confirm-password');
const changePasswordBtn = document.getElementById('change-password-btn');

// Reset the form when change password modal is closed in case user entered some values without submitting.
closePwdModalBtn.addEventListener('click', () => {
  changePwdForm.reset();
});

// Enable the change password button when the password and confirm password inputs are identical.
// TODO: should we add any other visual indication that the passwords do not match?
const checkPasswords = () => {
  if (passwordInput.value == confirmPassword.value && passwordInput.value != '') {
    changePasswordBtn.removeAttribute('disabled');
  } else {
    changePasswordBtn.setAttribute('disabled', 'true');
  };
};

// TODO: is this a complete list of events that should be used to detect if the passwords match?
addEventListeners(passwordInput, 'change keyup paste cut', checkPasswords);
addEventListeners(confirmPassword, 'change keyup paste cut', checkPasswords);

// Write code to run after an htmx request event.
// TODO: should we do a pre-flight check as a last sanity check that the passwords match?
document.body.addEventListener('htmx:afterRequest', (e) => {
    changePwdForm.reset();
    changePasswordBtn.setAttribute('disabled', 'true');

  if (e.detail.requestConfig.path === '/v1/change-password' && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    // Close the modal and reset the form.
    changePwdModal.close();
    changePwdForm.reset();

    // Show success toast message
    const successToast = document.getElementById('success-toast');
    const successToastMessage = document.getElementById('success-toast-msg');
    successToast.classList.remove('hidden');
    successToastMessage.textContent = 'Success! The password has been changed.';

    setTimeout(() => {
      successToast.classList.add('hidden');
    }, 2250);
  };
});