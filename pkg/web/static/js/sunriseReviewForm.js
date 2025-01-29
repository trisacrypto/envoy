import { countriesArray, addressTypeArray, naturalPersonNtlIdTypeArray, nationalIdentifierTypeArray } from "./constants.js";

// Set data for SlimSelect elements that appear in the send envelope and send message forms.

const country = 'country';
const natnitc = 'natnitc';
const legnitc = 'legnitc';
const address = 'address';

function initializeSlimSelects()  {
  const elements = document.querySelectorAll('[data-select-type]');
  elements.forEach((element) => setSlimSelect(element));
}

function setSlimSelect(element) {
  // Initialize SlimSelect only for select elements that exist in the DOM.
  if (!element) {
    return
  };

  const selectType = element.getAttribute('data-select-type');
  const newDropdown = new SlimSelect({
    select: '#' + element.getAttribute('id'),
  });

  if (selectType === country) {
    countriesArray.unshift({ 'placeholder': true, 'text': 'Select a country', 'value': '' });
    newDropdown.setData(countriesArray);
  };

  if (selectType === address) {
    addressTypeArray.unshift({ 'placeholder': true, 'text': 'Select address type', 'value': '' });
    newDropdown.setData(addressTypeArray);
  };

  if (selectType === natnitc) {
    naturalPersonNtlIdTypeArray.unshift({ 'placeholder': true, 'text': 'Select national identifier type', 'value': '' });
    newDropdown.setData(naturalPersonNtlIdTypeArray);
  };

  if (selectType === legnitc) {
    nationalIdentifierTypeArray.unshift({ 'placeholder': true, 'text': 'Select national identifier type', 'value': '' });
    newDropdown.setData(nationalIdentifierTypeArray);
  };

  // Set the default value on the dropdown if present.
  const value = element?.getAttribute('value');
  if (value) {
    newDropdown.setSelected(value);
  };
};

// Initialie all dropdown slim selects in DOM
initializeSlimSelects()