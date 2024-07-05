const closeUserModal = document.getElementById('close-new-usr-mdl')

if (closeUserModal) {
  closeUserModal.addEventListener('click', () => {
    document.getElementById('new-user-form').reset()
  });
};