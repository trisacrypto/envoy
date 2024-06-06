import { networksArray } from "./constants.js";

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
  const networkValue = networkEl.value;
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
      setSlimData(country, countryValue)
    }

  })
})

// Set the country options and selected value in a SlimSelect dropdown for each country selection.
function setSlimData(el, value) {
  el.slim.setData(countriesArray)
  el.slim.setSelected(value)
}
