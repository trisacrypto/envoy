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
    this.submit = this.modal.querySelector('button[type="submit"]');
    this.baseURL = this.form.getAttribute("hx-post");

    // Initialize the network choices
    this.network = selectNetwork(this.modal.querySelector('[name="network"]'));
  }

  onShowModal = (event) => {
    const button = event.relatedTarget;
    if (button.id === 'createCryptoAddressBtn') {
      return;
    }

    this.title.textContent = "Edit Crypto Address";
    this.submit.textContent = "Save";

    const form = this.modal.querySelector("form");
    form.querySelector('[name="crypto_address"]').value = button.dataset.bsCryptoAddress;
    this.network.setChoiceByValue(button.dataset.bsNetwork);
    form.querySelector('[name="asset_type"]').value = button.dataset.bsAssetType;
    form.querySelector('[name="tag"]').value = button.dataset.bsTag;

    var idfc = document.createElement("input");
    idfc.setAttribute("type", "hidden");
    idfc.setAttribute("name", "id");
    idfc.setAttribute("value", button.dataset.bsCryptoAddressId);
    form.appendChild(idfc);

    form.removeAttribute("hx-post");
    form.setAttribute("hx-put", this.baseURL + "/" + button.dataset.bsCryptoAddressId);
    htmx.process(form);
  };

  onHiddenModal = (event) => {
    // Restore the form to its original state
    const form = this.form.cloneNode(true);
    this.network = selectNetwork(form.querySelector('[name="network"]'))
    htmx.process(form);

    this.modal.querySelector("form").replaceWith(form);
    this.title.textContent = "Add Crypto Address";
    this.submit.textContent = "Create";
  };

};

export default EditWallet;