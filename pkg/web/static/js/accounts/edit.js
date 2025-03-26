/*
Application code for the customer account management edit page.
*/

/*
Specialized add alerts function for the edit account page.

TODO: consider refactoring this into a more general alerts class.
*/
function alertError(id, color, title, message) {
  const alerts = document.getElementById(id);
  alerts.insertAdjacentHTML('beforeend', `
    <div class="alert alert-${color} alert-dismissible fade show" role="alert">
      <strong>${title}</strong> ${message}.
      <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
    </div>
  `);
}

/*
Post-event handling when the accounts-updated event is fired.
*/
document.body.addEventListener("accounts-updated", function(e) {
  const elt = e.detail?.elt;
  if (elt) {
    console.log(elt.id);
    if (elt.id === 'deleteBtn') {
      // Redirect to the accounts index page after the account is deleted.
      window.location.href = "/accounts";
    }

    if (elt.getAttribute("id") === 'editAccountForm') {
      console.log("HERE");
      alertError("alerts", "success", "Success:", "Account updated successfully.");
    }
  }
});

/*
Handle any htmx errors that are not swapped by the htmx config.
*/
document.body.addEventListener("htmx:responseError", function(e) {
  const error = JSON.parse(e.detail.xhr.response);
  switch (e.detail.xhr.status) {
    case 400:
      alertError("alerts", "danger", "Error:", error.error);
      break;
    case 409:
      alertError("alerts", "danger", "Conflict:", error.error);
      break;
    case 422:
      alertError("alerts", "danger", "Validation error:", error.error);
      break;
    default:
      throw new Error(`unhandled htmx error: ${error.error}`);
  }
});