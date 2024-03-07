document.body.addEventListener('htmx:configRequest', function(e) {
  e.detail.headers['Accept'] = 'text/html'
});