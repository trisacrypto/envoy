const changePwdModal = document.getElementById('change_pwd_modal');
const changePwdForm = document.getElementById('change-pwd-form')
const closePwdModalBtn = document.getElementById('close-pwd-mdl-btn');

// Reset the form when change password modal is closed in case user entered some values without submitting.
closePwdModalBtn.addEventListener('click', () => {
  changePwdForm.reset();
});

// Write code to run after an htmx request event.
  document.body.addEventListener('htmx:afterRequest', (e) => {
    changePwdForm.reset();

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
    }, 5000);
  };
});

