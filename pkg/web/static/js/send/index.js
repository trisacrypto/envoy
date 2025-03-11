/*
Application code for the send TRISA/TRP forms.
*/

import { selectNetwork } from '../modules/networks.js';
import { selectCountry } from '../modules/countries.js';
import { selectTRISATravelAddress } from '../modules/components.js';
import { selectAddressType, selectNationalIdentifierType } from '../modules/ivms101.js';


// Initialize the network select choices.
document.querySelectorAll('[data-networks]').forEach(elem => {
  selectNetwork(elem);
});

// Initialize the country select choices.
document.querySelectorAll('[data-countries]').forEach(elem => {
  selectCountry(elem);
});

// Initialize the address type select choices.
document.querySelectorAll('[data-address-type]').forEach(elem => {
  selectAddressType(elem);
});

// Initialize the national identifier type select choices.
document.querySelectorAll('[data-national-identifier-type]').forEach(elem => {
  selectNationalIdentifierType(elem);
});

// Initialize the TRISA VASP Selection choices.
const vaspSelect = document.getElementById('trisaTravelAddress');
if (vaspSelect)  {
  selectTRISATravelAddress(vaspSelect);
}

// Handle form submission
document.getElementById('sendTransferForm').addEventListener('submit', function(e) {
  e.preventDefault();
  const form = e.target;
  const formData = new FormData(form);
  const data = Object.fromEntries(formData.entries());
  const json = JSON.stringify(data, null, 2);
  console.log(json);
  return false;
});