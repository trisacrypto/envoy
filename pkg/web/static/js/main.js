document.body.addEventListener('htmx:configRequest', (e) => {
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

document.body.addEventListener('htmx:responseError', (e) => {
  // Close any open modals.
  document.querySelector('.modal')?.close()

  // Display error response to user.
  if (e.detail.xhr.response !== '') {
    const error = JSON.parse(e.detail.xhr.response)
    document.getElementById('toast')
      .insertAdjacentHTML('beforeend', `
      <div class="alert alert-error">
        <i class="fa-solid fa-circle-xmark"></i>
        <span>${error.error}</span>
      </div>`)

    setTimeout(() => {
      document.querySelector('.alert').remove()
    }, 5000)
  }
})

/*
 * Timestamp handling: localizes and formats datetimes on the page.
 */
const updateDatetimes = (e) => {
  const options = {};
  const dtfmt = Intl.DateTimeFormat(navigator.languages, {
    year: 'numeric', month: 'long', day: 'numeric',
    hour: 'numeric', minute:'numeric', second: 'numeric',
    hour12: false, timeZoneName: "short",
  });

  const datetimes = document.querySelectorAll('.datetime');
  datetimes?.forEach(elem => {
    if (elem.textContent !== '' || elem.textContent !== null) {
      const dt = new Date(elem.textContent);
      elem.textContent = dtfmt.format(dt);
      elem.classList.remove('datetime');
    }
  });
};

document.body.addEventListener('htmx:afterSettle', updateDatetimes);