// Get the transaction ID from the URL and add it to the htmx request.
const id = window.location.pathname.split('/').pop()
document.getElementById('transaction-accept').setAttribute('hx-get', `/v1/transactions/${id}/accept`)

