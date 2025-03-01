// Ensure the accept type for all HTMX requests is HTML partials.
document.body.addEventListener('htmx:configRequest', (e) => {
  e.detail.headers['Accept'] = 'text/html'
});