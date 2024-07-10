import { addEventListeners } from "./userProfile.js";
import { setSuccessToast } from "./utils.js";

const usersEP = '/v1/users';
const newUserModal = document.getElementById('new_user_modal');
const newUserForm = document.getElementById('new-user-form');
const closeUserModal = document.getElementById('close-new-user-modal')
const userModal = document.getElementById('user_modal');

// Reset new user modal form if user closes the modal.
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

  // Get user ID, if it exists, after the DOM settles.
  const userID = document.getElementById('user-id');
  const userDetailEP = `/v1/users/${userID?.value}?detail=password`;

  if (e.detail.requestConfig.path === userDetailEP && e.detail.requestConfig.verb === 'get') {
    const passwordInput = document.getElementById('password');
    const confirmPassword = document.getElementById('confirm-password');
    const changePasswordBtn = document.getElementById('change-password-btn');

    // TODO: Convert to a reusable function for password matching.
    const checkPasswords = () => {
      if (passwordInput?.value == confirmPassword?.value && passwordInput?.value != '') {
        changePasswordBtn?.removeAttribute('disabled');
      } else {
        changePasswordBtn?.setAttribute('disabled', 'true');
      };
    };

    // TODO: Should input also be included in the events lists?
    addEventListeners(passwordInput, 'change keyup paste cut', checkPasswords);
    addEventListeners(confirmPassword, 'change keyup paste cut', checkPasswords);
  };
});

function copyUserPassword() {
  const newPassword = document.getElementById('new-user-pwd').innerHTML;
  // The clipboard API is only available in secure contexts.
  navigator.clipboard.writeText(newPassword);

  const copyIcon = document.getElementById('copy-icon');
  copyIcon.classList.remove('fa-copy');
  copyIcon.classList.add('fa-circle-check');

  // Reset the copy icon after 1 second.
  setTimeout(() => {
    copyIcon.classList.remove('fa-circle-check');
    copyIcon.classList.add('fa-copy');
  }, 1000);
};

// Add code to run after htmx:afterRequest event.
document.addEventListener('htmx:afterRequest', (e) => {
  if (e.detail.requestConfig.path === usersEP && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    // Close the add user modal and reset the form.
    newUserModal.close();
    newUserForm.reset();
    setSuccessToast('Success! The new user has been created.');
  };

  // Get user ID, if it exists, after the DOM settles.
  const userID = document.getElementById('user-id');
  const userChangePwd = `/v1/users/${userID?.value}/password`;

  if (e.detail.requestConfig.path === userChangePwd && e.detail.requestConfig.verb === 'post') {
    userModal.close();
  };

  if (e.detail.requestConfig.path === userChangePwd && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    userModal.close();
    setSuccessToast('Success! The password has been changed.');
  };
});
