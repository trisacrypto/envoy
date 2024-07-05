const usersEP = '/v1/users';
const newUserModal = document.getElementById('new_user_modal');
const newUserForm = document.getElementById('new-user-form');

// Reset new user modal form if user closes the modal.
const closeUserModal = document.getElementById('close-new-user-modal')
if (closeUserModal) {
  closeUserModal.addEventListener('click', () => {
    newUserForm?.reset()
  });
};

// Add code to run after HTMX settles the DOM once a swap occurs. 
document.addEventListener('htmx:afterSettle', (e) => {
  if (e.detail.requestConfig.path === usersEP && e.detail.requestConfig.verb === 'post') {
    
    // Copy the new user password to the clipboard if user clicks the copy icon.
    const copyPasswordBtn = document.getElementById('copy-password-btn');
    if (copyPasswordBtn) {
      copyPasswordBtn.addEventListener('click', copyUserPassword);
    };

    // Copy the new user password to the clipboard if user clicks the close button.
    const closePwdModalBtn = document.getElementById('close-pwd-modal');
    if (closePwdModalBtn) {
      closePwdModalBtn.addEventListener('click', (e) => {
        e.preventDefault();
        copyUserPassword();
        document.getElementById('user_pwd_modal').close();
      });
    };
  };
});

function copyUserPassword() {
  const newPassword = document.getElementById('new-user-pwd').innerHTML;
  navigator.clipboard.writeText(newPassword);

  const copyIcon = document.getElementById('copy-icon');
  copyIcon.classList.remove('fa-copy');
  copyIcon.classList.add('fa-circle-check');
  setTimeout(() => {
    copyIcon.classList.remove('fa-circle-check');
    copyIcon.classList.add('fa-copy');
  }, 1000);
}

// Add code to run after htmx:afterRequest event.
document.addEventListener('htmx:afterRequest', (e) => {
  if (e.detail.requestConfig.path === usersEP && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    // Close the add user modal and reset the form.
    newUserModal.close();
    newUserForm.reset();

    // Display success toast message.
    const successToast = document.getElementById('success-toast');
    const successToastMsg = document.getElementById('success-toast-msg');
    successToast.classList.remove('hidden');
    successToastMsg.textContent = 'Success! The new user has been created.'

    // Remove the toast after 5 seconds.
    setTimeout(() => {
      successToast.classList.add('hidden');
    }, 5000);
  }
})