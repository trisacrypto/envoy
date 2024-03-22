const openModal = document.getElementById('open-envelope-modal')
const closeModal = document.getElementById('close-envelope-modal')
const previewModal = document.getElementById('preview-envelope-modal')
const modalOverlay = document.getElementById('preview-envelope-overlay')

openModal.addEventListener('click', () => {
  previewModal.classList.remove('hidden');
  modalOverlay.classList.remove('hidden');

  // Prevent users from scrolling when the modal is open.
  document.body.style.overflow = 'hidden';
});

if (closeModal) {
  closeModal.addEventListener('click', () => {
    previewModal.classList.add('hidden');
    modalOverlay.classList.add('hidden');

    // Reset scrolling behavior when the modal is closed.
    document.body.style.overflow = 'auto';
  })
};