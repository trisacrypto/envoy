import { countriesArray } from './constants.js';

// Use SlimSelect after the counterparty preview partial has been swapped and settled into the DOM.
document.body.addEventListener('htmx:afterSettle', () => {
  const countrySelect = new SlimSelect({
    select: '#country',
    settings: {
      contentLocation: document.getElementById('country-list'),
    },
  });

  // Get the selected country value.
  const cpartyCountry = document.getElementById('selected-country');
  const cpartySelectedCountry = cpartyCountry.value;

  // Add the country data and set the currently selected country.
  countrySelect.setData(countriesArray);
  countrySelect.setSelected(cpartySelectedCountry);
});