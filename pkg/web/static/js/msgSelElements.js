import { countriesArray, naturalPersonNtlIdTypeArray, networksArray } from './constants.js';

// Set data for SlimSelect elements that appear in the send envelope and send message forms.

const network = 'network';
const country = 'country';
const nationalIdType = 'idType';

const envelopeDropdowns = [
  { sel: '#networks', options: network },
  { sel: '#orig_countries', options: country },
  { sel: '#benf_countries', options: country },
  { sel: '#og_id_country', options: country },
  { sel: '#bf_id_country', options: country },
  { sel: '#og_id_type_code', options: nationalIdType },
  { sel: '#bf_id_type_code', options: nationalIdType },
];

envelopeDropdowns.forEach((dropdown) => setSlimSelect(dropdown.sel, dropdown.options));

function setSlimSelect(sel, options) {
  const element = document.querySelector(sel)
  const value = element?.getAttribute('value');

   // Initialize SlimSelect only for select elements that exist in the DOM.
  if (!element) {
    return
  };

  const newDropdown = new SlimSelect({
    select: sel
  });

  if (options === network) {
    networksArray.unshift({ 'placeholder': true, 'text': 'Select a network', 'value': '' });
    newDropdown.setData(networksArray);
  };

  if (options === country) {
    countriesArray.unshift({ 'placeholder': true, 'text': 'Select a country', 'value': '' });
    newDropdown.setData(countriesArray);
  };

  if (options === nationalIdType) {
    naturalPersonNtlIdTypeArray.unshift({ 'placeholder': true, 'text': 'Select national identifier type', 'value': '' });
    newDropdown.setData(naturalPersonNtlIdTypeArray);
  };

  // Set the default value on the dropdown if present.
  if (value) {
    newDropdown.setSelected(value);
  };
};