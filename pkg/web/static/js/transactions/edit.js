/*
Application code for sending an accept/reject message on the transfer detail page.
*/

import Alerts from '../modules/alerts.js';


// Initialize the alerts component.
const alerts = new Alerts("#alerts");


/*
Handle any htmx errors that are not swapped by the htmx config.
*/
document.body.addEventListener("htmx:responseError", function(e) {
    const error = JSON.parse(e.detail.xhr.response);
    alerts.danger("Error:", error.error);
});