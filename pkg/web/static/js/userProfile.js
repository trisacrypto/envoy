// Write code to run after an htmx request event.
document.body.addEventListener('htmx:afterRequest', (e) => {
  if (e.detail.requestConfig.path === 'v1/change-password' && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    // Close the change password modal and reset the form.
    const changePwdModal = document.getElementById('change_pwd_modal');
    const changePwdForm = document.getElementById('change-pwd-form');
    changePwdModal.close();
    changePwdForm.reset();

    // Show success toast message
    const successToast = document.getElementById('success-toast');
    const successToastMessage = document.getElementById('success-toast-message');
    successToast.classList.remove('hidden');
    successToastMessage.textContent = 'Success! The password has been changed!';

    setTimeout(() => {
      successToast.classList.add('hidden');
    }, 5000);
  };
});

