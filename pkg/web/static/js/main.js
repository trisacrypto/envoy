document.body.addEventListener('htmx:configRequest', function(e) {
  e.detail.headers['Accept'] = 'text/html'
});

const hideWelcomeMsg = 'hideWelcomeMsg'
const welcomeMsg = document.getElementById('welcome-msg')
const closeMsg = document.getElementById('close-welcome-msg')
const logoutBttn = document.getElementById('logout-bttn')

// Hide welcome message during a session if user clicks the close button.
if (closeMsg) {
  closeMsg.addEventListener('click', () => {
    welcomeMsg.classList.add('hidden')
    localStorage.setItem(hideWelcomeMsg, 'true')
  })
}

if (localStorage.getItem(hideWelcomeMsg) === 'true') {
  welcomeMsg?.classList.add('hidden')
}

if (logoutBttn) {
  logoutBttn.addEventListener('click', () => {
    localStorage.removeItem(hideWelcomeMsg)
  })
}
