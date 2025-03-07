// Ensure the accept type for all HTMX requests is HTML partials.
document.body.addEventListener('htmx:configRequest', (e) => {
  e.detail.headers['Accept'] = 'text/html'
});

// Ensure that all 500 errors redirect to the error page.
document.body.addEventListener('htmx:responseError', (e) => {
  switch (e.detail.xhr.status) {
    case 500:
      window.location.href = '/error';
      break;
    case 501:
      window.location.href = '/not-allowed';
      break;
  }
});
