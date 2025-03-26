/*
EditWallet manages a modal form that can be used to create or edit a crypto address.
*/

import { selectNetwork } from '../modules/networks.js';

class EditWallet {
  constructor(modalSelector) {
    if (!modalSelector) {
      modalSelector = "#editCryptoAddressModal";
    }

    this.modal = document.querySelector(modalSelector);
    if (!this.modal) {
      throw new Error("no modal found, specify the modal selector");
    }

    // Register the event handlers on the modal
    this.modal.addEventListener("show.bs.modal", this.onShowModal);
    this.modal.addEventListener("hidden.bs.modal", this.onHiddenModal);

    // Get the original form to ensure we can reset the form to its original state
    this.form = this.modal.querySelector("form").cloneNode(true);
    this.title = this.modal.querySelector(".modal-title");

    // Initialize the network choices
    selectNetwork(this.modal.querySelector('[name="network"]'));
  }

  onShowModal = (event) => {};

  onHiddenModal = (event) => {
    // Restore the form to its original state
    const form = this.form.cloneNode(true);
    selectNetwork(form.querySelector('[name="network"]'))
    htmx.process(form);

    this.modal.querySelector("form").replaceWith(form);
    this.title.textContent = "Create Crypto Address";
  };

};

export default EditWallet;