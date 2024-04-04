document.body.addEventListener('htmx:afterRequest', (e) => {
  const addCpartyForm = 'new-cparty-form';
  // Check if the request to add a new counterparty was successful.
  if (e.detail.elt.id === addCpartyForm && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    // Close the add counterparty modal and reset the form.
    document.getElementById('add_cparty_modal').close();
    document.getElementsByClassName(addCpartyForm).reset();
  }
});

document.body.addEventListener('htmx:afterRequest', (e) => {
  const editCpartyForm = 'edit-cparty-form';
  // Check if the request to update a user added counterparty was successful.
  if (e.detail.elt.id === editCpartyForm && e.detail.requestConfig.verb === 'put' && e.detail.successful) {
    // Close the edit counterparty modal and reset the form.
    document.getElementById('cparty_modal').close();
    document.getElementById(editCpartyForm).reset();
  }
});