// Reset new user modal form if user closes the modal.
const closeUserModal = document.getElementById('close-new-usr-mdl')
if (closeUserModal) {
  closeUserModal.addEventListener('click', () => {
    document.getElementById('new-user-form').reset()
  });
};

// Add code to run after HTMX settles the DOM once a swap occurs. 
document.addEventListener('htmx:afterSettle', (e) => {
  if (e.detail.requestConfig.path === '/v1/users' && e.detail.requestConfig.verb === 'post') {
    
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