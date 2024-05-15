import { countries } from './countriesList.js';

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

  const countriesArray = Object.entries(countries).map(([value, text]) => ({ text, value }));
  countrySelect.setData(countriesArray);
  countrySelect.setSelected(cpartySelectedCountry);
});