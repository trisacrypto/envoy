const bttn = document.getElementById('preview-envelope-bttn')

bttn.addEventListener('click', () => {
  const modal = document.getElementById('default-modal')
  const overlay = document.getElementById('preview-envelope-overlay')

  modal.classList.remove('hidden')
  overlay.classList.remove('hidden')

  document.body.style.overflow = 'hidden';
})

const closeBttn = document.getElementById('close-envelope-modal')

closeBttn.addEventListener('click', () => {
  const modal = document.getElementById('default-modal')
  const overlay = document.getElementById('preview-envelope-overlay')

  modal.classList.add('hidden')
  overlay.classList.add('hidden')

  document.body.style.overflow = 'auto';
})