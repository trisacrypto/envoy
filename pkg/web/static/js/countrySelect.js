import { countriesArray } from './constants.js';

// Display a searchable dropdown for countries in the originator section of the send envelope form.
const origCountrySelect = new SlimSelect({
  select: '#orig_countries',
  settings: {
    contentLocation: document.getElementById('orig-countries-content'),
  }
});

// Display a searchable dropdown for countries in the beneficiary section of the send envelope form.
const benfCountrySelect = new SlimSelect({
  select: '#benf_countries',
  settings: {
    contentLocation: document.getElementById('benf-countries-content'),
  }
});

// Set the placeholder and country options for the originator and beneficiary dropdowns.
countriesArray.unshift({ 'placeholder': true, 'text': 'Select a country', 'value': '' });
origCountrySelect.setData(countriesArray);
benfCountrySelect.setData(countriesArray);