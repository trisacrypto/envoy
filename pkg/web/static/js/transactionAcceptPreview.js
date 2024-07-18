import { countriesArray, networksArray, legalPersonNameTypeArray, addressTypeArray, nationalIdentifierTypeArray, naturalPersonNtlIdTypeArray, naturalPersonNameTypeArray } from "./constants.js";

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
  // Add placeholder in case the network value is empty. Importing the options with the placeholder 
  // from the constants file resulted in the options not being displayed.
  networksArray.unshift({ 'placeholder': true, 'text': 'Select a network', 'value': '' });
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
  // Add placeholder in case the country value is empty. Importing the options with the placeholder 
  // from the constants file resulted in the options not being displayed.
  countriesArray.unshift({ 'placeholder': true, 'text': 'Select a country', 'value': '' });
  el.slim.setData(countriesArray)
  el.slim.setSelected(value)
}

// Set the identifier type options and selected value in the SlimSelect dropdown for each identifier selection.
function setIdentifierData(el, identifier) {
  const identifierDataID = identifier.dataset.id
  switch (identifierDataID) {
    case 'legal-person-name-type':
      // Add placeholder in case the legal name type value is empty. Importing the options with the placeholder 
      // from the constants file resulted in the options not being displayed.
      legalPersonNameTypeArray.unshift({ 'placeholder': true, 'text': 'Select a name type', 'value': '' });
      el.slim.setData(legalPersonNameTypeArray)
      el.slim.setSelected(identifier.value)
      break;
    case 'address-identifier-type':
      // Add placeholder in case the address type value is empty. Importing the options with the placeholder
      // from the constants file resulted in the options not being displayed.
      addressTypeArray.unshift({ 'placeholder': true, 'text': 'Select an address type', 'value': '' });
      el.slim.setData(addressTypeArray)
      el.slim.setSelected(identifier.value)
      break;
    case 'natural-person-ntl-id-type':
      // Add placeholder in case the natural person identifier type value is empty. Importing the options with the placeholder
      // from the constants file resulted in the options not being displayed.
      naturalPersonNtlIdTypeArray.unshift({ 'placeholder': true, 'text': 'Select an identifier type', 'value': '' });
      el.slim.setData(naturalPersonNtlIdTypeArray)
      el.slim.setSelected(identifier.value)
      break;
    case 'national-identifier-type':
      // Add placeholder in case the national identifier type value is empty. Importing the options with the placeholder 
      // from the constants file resulted in the options not being displayed.
      nationalIdentifierTypeArray.unshift({ 'placeholder': true, 'text': 'Select an identifier type', 'value': '' });
      el.slim.setData(nationalIdentifierTypeArray)
      el.slim.setSelected(identifier.value)
      break;
    case 'natural-person-name-type':
      // Add placeholder in case the natural person name type value is empty. Importing the options with the placeholder
      // from the constants file resulted in the options not being displayed.
      naturalPersonNameTypeArray.unshift({ 'placeholder': true, 'text': 'Select a name type', 'value': '' });
      el.slim.setData(naturalPersonNameTypeArray)
      el.slim.setSelected(identifier.value)
      break;
  }
}

// Modify parameters sent in the body of a request via the htmx configRequest.
const idEl = document.getElementById('envelope-id')
const id = idEl?.value
document.body.addEventListener('htmx:configRequest', (e) => {
  const transactionSendEP = `/v1/transactions/${id}/send`;
  if (e.detail.path === transactionSendEP && e.detail.verb === 'post') {
    const params = e.detail.parameters;

    let data = {
      identity: {
        originator: {
          originator_persons: [{
            natural_person: {
              name: {
                name_identifiers: [{}]
              },
              geographic_addresses: [{
                address_line: []
              }],
              national_identification: {},
              date_and_place_of_birth:{},
              account_numbers: []
            },
          }],
        },
        beneficiary: {
          beneficiary_persons: [{
            natural_person: {
              name: {
                name_identifiers: [{}]
              },
              geographic_addresses: [{
                address_line: []
              }],
              national_identification: {},
              date_and_place_of_birth:{},
              account_numbers: []
            },
          }]
        },
        originating_vasp: {
          originating_vasp: {
            legal_person: {
              name: {
                name_identifiers: [{}]
              },
              geographic_addresses: [{
                address_line: []
              }],
              national_identification: {}
            },
          }
        },
        beneficiary_vasp: {
          beneficiary_vasp: {
            legal_person: {
              name: {
                name_identifiers: [{}]
              },
              geographic_addresses: [{
                address_line: []
              }],
              national_identification: {}
            },
          }
        },
      },
      transaction: {}
    }

    const originatorPerson = data.identity.originator.originator_persons[0].natural_person;
    const beneficiaryPerson = data.identity.beneficiary.beneficiary_persons[0].natural_person;
    const originatingVASP = data.identity.originating_vasp.originating_vasp.legal_person;
    const beneficiaryVASP = data.identity.beneficiary_vasp.beneficiary_vasp.legal_person;

    for (const key in params) {
      // Remove prefix from the key.
      const newKey = key.split('_').slice(2).join('_');

      switch (true) {
        // Set the transaction details.
        case key.startsWith('env_transaction_'):
          data.transaction[newKey] = params[key];
          break;
        // Set the originator name identifiers and name identifier type.
        case key.startsWith('id_og_'):
          originatorPerson.name.name_identifiers[0][newKey] = params[key];
          break;
        // Set the originator date and place of birth.
        case key.startsWith('originator_birth_'):
          originatorPerson.date_and_place_of_birth[newKey] = params[key];
          break;
        // Set the originator address line.
        case key.startsWith('address_og_'):
          originatorPerson.geographic_addresses[0].address_line.push(params[key]);
          break;
        // Set the originator country and address type.
        case key.startsWith('addr_og_'):
          originatorPerson.geographic_addresses[0][newKey] = params[key];
          break;
        // Set details for the originator natural person that's not a name identifier or geographic address.
        case key.startsWith('np_og_'):
          originatorPerson[newKey] = params[key];
          break;
        // Set national identification for the originator.
        case key.startsWith('originator_id_'):
          originatorPerson.national_identification[newKey] = params[key];
          break;
        // Set the originator account number.
        case key.startsWith('acct_og_'):
          originatorPerson.account_numbers.push(params[key]);
          break;
        // Set the beneficiary name identifiers and name identifier type.
        case key.startsWith('id_bf_'):
          beneficiaryPerson.name.name_identifiers[0][newKey] = params[key];
          break;
        // Set the beneficiary date and place of birth.
        case key.startsWith('beneficiary_birth_'):
          beneficiaryPerson.date_and_place_of_birth[newKey] = params[key];
          break;
        // Set the beneficiary address line.
        case key.startsWith('address_bf_'):
          beneficiaryPerson.geographic_addresses[0].address_line.push(params[key]);
          break;
        // Set the beneficiary country and address type.
        case key.startsWith('addr_bf_'):
          beneficiaryPerson.geographic_addresses[0][newKey] = params[key];
          break;
        // Set details for the beneficiary natural person that's not a name identifier or geographic address.
        case key.startsWith('np_bf_'):
          beneficiaryPerson[newKey] = params[key];
          break;
        // Set national identification for the beneficiary.
        case key.startsWith('beneficiary_id_'):
          beneficiaryPerson.national_identification[newKey] = params[key];
          break;
        // Set the beneficiary account number.
        case key.startsWith('acct_bf_'):
          beneficiaryPerson.account_numbers.push(params[key]);
          break;
        // Set the originating VASP name identifiers and name identifier type.
        case key.startsWith('id_orig'):
          originatingVASP.name.name_identifiers[0][newKey] = params[key];
          break;
        // Set the originating VASP address line.
        case key.startsWith('address_orig'):
          originatingVASP.geographic_addresses[0].address_line.push(params[key]);
          break;
        // Set the originating VASP country and address type.
        case key.startsWith('addr_orig'):
          originatingVASP.geographic_addresses[0][newKey] = params[key];
          break;
        // Set the originating VASP national identification.
        case key.startsWith('nat_orig'):
          originatingVASP.national_identification[newKey] = params[key];
          break;
        // Set the originating VASP country of registration.
        case key.startsWith('ctry_orig'):
          originatingVASP[newKey] = params[key];
          break;
        // Set the beneficiary VASP name identifiers and name identifier type.
        case key.startsWith('id_benf'):
          beneficiaryVASP.name.name_identifiers[0][newKey] = params[key];
          break;
        // Set the beneficiary VASP address line.
        case key.startsWith('address_benf'):
          beneficiaryVASP.geographic_addresses[0].address_line.push(params[key]);
          break;
        // Set the beneficiary VASP country and address type.
        case key.startsWith('addr_benf'):
          beneficiaryVASP.geographic_addresses[0][newKey] = params[key];
          break;
        // Set the beneficiary VASP national identification.
        case key.startsWith('nat_benf'):
          beneficiaryVASP.national_identification[newKey] = params[key];
          break;
        // Set the beneficiary VASP country of registration.
        case key.startsWith('ctry_benf'):
          beneficiaryVASP[newKey] = params[key];
          break;
      };
    };

    // Convert transaction amount to a float. If conversion fails, set amount to the original value.
    const amount = parseFloat(data.transaction.amount);
    data.transaction.amount = isNaN(amount) ? data.transaction.amount : amount;
    e.detail.parameters = data;
  };
});