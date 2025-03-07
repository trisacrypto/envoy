import { isRequestFor, checkStatus } from '../htmx/helpers.js';

/*
Add alert with alert-danger to the alerts div (profile-specific implementation).
When the alert is added it is removed after 5 seconds.
*/
function alertError(title, message) {
  const alerts = document.getElementById("alerts");
  alerts.insertAdjacentHTML('beforeend', `
    <div class="alert alert-danger alert-dismissible fade show" role="alert">
        <strong>${title}</strong>: <span>${message}</span>.
        <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
    </div>
  `);

  setTimeout(() => {
    document.querySelector('.alert').remove()
  }, 5000);
}

/*
Handle save profile errors as JSON responses from the backend.
*/
document.body.addEventListener("htmx:responseError", (e) => {
  const error = JSON.parse(e.detail.xhr.response);
  switch (e.detail.xhr.status) {
    case 400:
      alertError("Could not save profile", error.error);
      break;
    case 422:
      alertError("Validation error", error.error);
      break;
    default:
      break;
  }
});

/*
Add a success alert when the profile has been saved and remove it after 5 seconds.
*/
document.body.addEventListener("htmx:afterSettle", (e) => {
  if (isRequestFor(e, "/v1/profile", "put") && checkStatus(e, 200)) {
    // Display success message -- the profile has been saved.
    const alerts = document.getElementById("alerts");
    alerts.insertAdjacentHTML('beforeend', `
      <div class="alert alert-success alert-dismissible fade show" role="alert">
          <h4 class="alert-heading">Profile Saved</h4>
          <p class="mb-1">Note that for some changes to take effect like your gravatar or full name, you must log out and log in again.</p>
          <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
      </div>
    `);

    setTimeout(() => {
      document.querySelector('.alert').remove()
    }, 5000);
  }
});