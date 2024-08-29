import { countriesArray } from './constants.js';
import { setSuccessToast } from './utils.js';

document.body.addEventListener('htmx:afterRequest', (e) => {
  const addCpartyForm = document.getElementById('new-cparty-form');
  const cpartyModal = document.getElementById('add_cparty_modal');
  // Check if the request to add a new counterparty was successful.
  if (e.detail.requestConfig.path === '/v1/counterparties' && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    // Close the add counterparty modal and reset the form.
    cpartyModal.close();
    addCpartyForm.reset();
    countrySelect.setSelected({ 'placeholder': true, 'text': 'Select a country', 'value': '' });

    setSuccessToast('Success! A new counterparty VASP has been created.');
  }
});

document.body.addEventListener('htmx:afterRequest', (e) => {
  // Close the edit counterparty modal after a successful request.
  if ( e.detail.requestConfig.verb === 'put' && e.detail.successful) {
    document.getElementById('cparty_modal').close();
  };
});

// Use SlimSelect to create a searchable select dropdown for countries in the add counterparty modal form.
const countrySelect = new SlimSelect({
  select: '#countries',
  settings: {
    contentLocation: document.getElementById('country-content'),
  },
});
countriesArray.unshift({ 'placeholder': true, 'text': 'Select a country', 'value': '' });
countrySelect.setData(countriesArray);

// Set the country value in the edit counterparty modal form.
document.body.addEventListener('htmx:afterSettle', (e) => {
  const cpartyID = document.getElementById('cparty_id');
  const cpartyPreviewEP = `/v1/counterparties/${cpartyID?.value}/edit`;

  if (e.detail.requestConfig.path === cpartyPreviewEP && e.detail.requestConfig.verb === 'get') {
    // Initialize SlimSelect for the country dropdown in the edit counterparty modal form.
    const countrySelect = new SlimSelect({
      select: '#country',
      settings: {
        contentLocation: document.getElementById('cparty_modal'),
      },
    });

    // Get the selected country value.
    const cpartyCountry = document.getElementById('selected-country');
    const cpartySelectedCountry = cpartyCountry.value;

    // Add the country data and set the currently selected country.
    countrySelect.setData(countriesArray);
    countrySelect.setSelected(cpartySelectedCountry);
  };
});