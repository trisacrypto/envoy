import { networksArray, legalPersonNameTypeArray, addressTypeArray, nationalIdentifierTypeArray, naturalPersonNameTypeArray } from "./constants.js";

document.body.addEventListener('htmx:afterSettle', () => {
  // Initialize a SlimSelect dropdown for the transaction network.
  const transactionNetwork = new SlimSelect({
    select: "#network",
    settings: {
      contentLocation: document.getElementById('network-content')
    }
  })

  // Set the list of options in the network dropdown and display the value selected by the requester.
  const networkEl = document.getElementById('selected-network');
  const networkValue = networkEl?.value;
  transactionNetwork.setData(networksArray);
  transactionNetwork.setSelected(networkValue);

  // Initialize a SlimSelect dropdown for each country selection in the form.
  const countries = document.querySelectorAll('.countries')
  countries.forEach((country) => {
    new SlimSelect({
      select: country,
    })

    // Get each country value selected by the requester from the hidden input field.
    const countryID = country.id
    const selectedCountry = document.querySelector(`.${countryID}`)

    if (selectedCountry) {
      const countryValue = selectedCountry.value
      setCountryData(country, countryValue)
    }

  })

  // Initialize a SlimSelect dropdown for each identifier type selection in the form.
  const identifierTypes = document.querySelectorAll('.identifier-types')
  identifierTypes.forEach((identifier) => {
    new SlimSelect({
      select: identifier,
    })

    const identifierID = identifier.id
    const selectedIdentifier = document.querySelector(`.${identifierID}`)
    if (selectedIdentifier) {
      setIdentifierData(identifier, selectedIdentifier)
    }
  })
})

// Set the country options and selected value in a SlimSelect dropdown for each country selection.
function setCountryData(el, value) {
  el.slim.setData(countriesArray)
  el.slim.setSelected(value)
}

// Set the identifier type options and selected value in the SlimSelect dropdown for each identifier selection.
function setIdentifierData(el, identifier) {
  const identifierDataID = identifier.dataset.id
  switch (identifierDataID) {
    case 'legal-person-name-type':
      el.slim.setData(legalPersonNameTypeArray)
      el.slim.setSelected(identifier.value)
      break;
    case 'address-identifier-type':
      el.slim.setData(addressTypeArray)
      el.slim.setSelected(identifier.value)
      break;
    case 'national-identifier-type':
      el.slim.setData(nationalIdentifierTypeArray)
      el.slim.setSelected(identifier.value)
      break;
    case 'natural-person-name-type':
      el.slim.setData(naturalPersonNameTypeArray)
      el.slim.setSelected(identifier.value)
      break;
  }
}