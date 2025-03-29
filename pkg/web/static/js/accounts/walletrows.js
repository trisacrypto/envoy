/*
WalletRows manages a form that can insert one or more crypto wallet addresses using
a template described by the selector.
*/

import { selectNetwork } from '../modules/networks.js';

class WalletRows {

  constructor(selector) {
    this.selector = selector;
    if (!this.selector) {
      this.selector = ".crypto_addresses";
    }

    // Get the template for each row using the selector
    let row = document.querySelector(this.selector);
    if (!row) {
      throw new Error("no wallet rows found, specify at least one row with given selector");
    }

    this.template = row.cloneNode(true);
    this.parent = row.parentNode;

    // Initialize the rows and event handlers
    this.init();

    // Ensure the network select field is initialized
    row.querySelectorAll('[data-networks]').forEach(elem => {
      selectNetwork(elem);
    });
  }

  init = () => {
    const rows = document.querySelectorAll(this.selector);
    for (let i = 0; i < rows.length; i++) {
      const isLast = (i === rows.length - 1);
      this.initRow(rows[i], i, isLast);
    }
  }

  initRow = (row, idx, isLast) => {
    // Handle the action button
    const action = row.querySelector("button[type='button']");
    const newAction = action.cloneNode(true);
    if (isLast) {
      newAction.classList.remove("btn-outline-danger");
      newAction.classList.add("btn-outline-success");

      const icon = newAction.querySelector("i");
      icon.classList.remove("fe-trash");
      icon.classList.add("fe-plus");

      newAction.addEventListener("click", this.appendWalletRow);
    } else {
      newAction.classList.remove("btn-outline-success");
      newAction.classList.add("btn-outline-danger");

      const icon = newAction.querySelector("i");
      icon.classList.remove("fe-plus");
      icon.classList.add("fe-trash");

      newAction.addEventListener("click", this.removeWalletRow(idx));
    }

    action.replaceWith(newAction);
  }

  appendWalletRow = (e) => {
    const newRow = this.template.cloneNode(true);

    // Ensure the network select field is initialized
    newRow.querySelectorAll('[data-networks]').forEach(elem => {
      selectNetwork(elem);
    });

    this.parent.appendChild(newRow);
    this.init();
  }

  removeWalletRow = (idx) => {
    const self = this;
    return (e) => {
      const row = document.querySelectorAll(self.selector)[idx];
      if (row)  {
        row.remove();
        self.init();
      }
    }
  }

  reset = (e) => {
    // Delete all existing rows from the parent, then append a row.
    const rows = document.querySelectorAll(this.selector);
    for (let i = 0; i < rows.length; i++) {
      rows[i].remove();
    }

    this.appendWalletRow();
  }
}

export default WalletRows;