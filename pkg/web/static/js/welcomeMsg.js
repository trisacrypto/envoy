const closeMsg = document.getElementById('close-welcome-msg')
const welcomeMsg = document.getElementById('welcome-msg')

if (closeMsg) {
  closeMsg.addEventListener('click', () => {
    welcomeMsg.classList.add('hidden')
    localStorage.setItem('hideWelcomeMsg', 'true')
  })
}

if (localStorage.getItem('hideWelcomeMsg') === 'true') {
  welcomeMsg.classList.add('hidden')
}

