import { countriesArray } from './constants.js';

document.body.addEventListener('htmx:afterRequest', (e) => {
  const addCpartyForm = 'new-cparty-form';
  // Check if the request to add a new counterparty was successful.
  if (e.detail.elt.id === addCpartyForm && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    // Close the add counterparty modal and reset the form.
    document.getElementById('add_cparty_modal').close();
    document.getElementById(addCpartyForm).reset();
    countrySelect.setSelected({ 'placeholder': true, 'text': 'Select a country', 'value': '' });
  }
});

document.body.addEventListener('htmx:afterRequest', (e) => {
  const editCpartyForm = 'edit-cparty-form';
  // Check if the request to update a user added counterparty was successful.
  if (e.detail.elt.id === editCpartyForm && e.detail.requestConfig.verb === 'put' && e.detail.successful) {
    // Close the edit counterparty modal and reset the form.
    document.getElementById('cparty_modal').close();
    document.getElementById(editCpartyForm).reset();
  }
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