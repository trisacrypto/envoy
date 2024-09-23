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
    if (countryID !== '') {
      const selectedCountry = document.querySelector(`.${countryID}`)
      if (selectedCountry) {
        const countryValue = selectedCountry.value
        setCountryData(country, countryValue)
      }
    }
  })

  // Initialize a SlimSelect dropdown for each identifier type selection in the form.
  const identifierTypes = document.querySelectorAll('.identifier-types')
  identifierTypes.forEach((identifier) => {
    new SlimSelect({
      select: identifier,
    })

    const identifierID = identifier.id
    if (identifierID !== '') {
      const selectedIdentifier = document.querySelector(`.${identifierID}`)
      if (selectedIdentifier) {
        setIdentifierData(identifier, selectedIdentifier)
      }
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
  // Get the identifier type and update to match ivms101 code.
  const identifierValue = identifier.value.split('_').slice(-1)[0]
  switch (identifierDataID) {
    case 'legal-person-name-type':
      // Add placeholder in case the legal name type value is empty. Importing the options with the placeholder
      // from the constants file resulted in the options not being displayed.
      legalPersonNameTypeArray.unshift({ 'placeholder': true, 'text': 'Select a name type', 'value': '' });
      el.slim.setData(legalPersonNameTypeArray)
      el.slim.setSelected(identifierValue)
      break;
    case 'address-identifier-type':
      // Add placeholder in case the address type value is empty. Importing the options with the placeholder
      // from the constants file resulted in the options not being displayed.
      addressTypeArray.unshift({ 'placeholder': true, 'text': 'Select an address type', 'value': '' });
      el.slim.setData(addressTypeArray)
      el.slim.setSelected(identifierValue)
      break;
    case 'natural-person-ntl-id-type':
      // Add placeholder in case the natural person identifier type value is empty. Importing the options with the placeholder
      // from the constants file resulted in the options not being displayed.
      naturalPersonNtlIdTypeArray.unshift({ 'placeholder': true, 'text': 'Select an identifier type', 'value': '' });
      el.slim.setData(naturalPersonNtlIdTypeArray)
      el.slim.setSelected(identifierValue)
      break;
    case 'national-identifier-type':
      // Add placeholder in case the national identifier type value is empty. Importing the options with the placeholder
      // from the constants file resulted in the options not being displayed.
      nationalIdentifierTypeArray.unshift({ 'placeholder': true, 'text': 'Select an identifier type', 'value': '' });
      el.slim.setData(nationalIdentifierTypeArray)
      el.slim.setSelected(identifierValue)
      break;
    case 'natural-person-name-type':
      // Add placeholder in case the natural person name type value is empty. Importing the options with the placeholder
      // from the constants file resulted in the options not being displayed.
      naturalPersonNameTypeArray.unshift({ 'placeholder': true, 'text': 'Select a name type', 'value': '' });
      el.slim.setData(naturalPersonNameTypeArray)
      el.slim.setSelected(identifierValue)
      break;
  }
}

// Modify parameters sent in the body of a request via the htmx configRequest.
const idEl = document.getElementById('envelope-id')
const id = idEl?.value
document.body.addEventListener('htmx:configRequest', (e) => {
  const transactionAcceptEP = `/v1/transactions/${id}/accept`;
  const repairTransactionEP = `/v1/transactions/${id}/repair`;
  if (e.detail.path === transactionAcceptEP && e.detail.verb === 'post' || e.detail.path === repairTransactionEP && e.detail.verb === 'post') {
    const params = e.detail.parameters;
    console.log(params)

    let data = {
      identity: {
        originator: {
          originatorPersons: [{
            naturalPerson: {
              name: {
                nameIdentifier: [{}]
              },
              geographicAddress: [{
                addressLine: []
              }],
              nationalIdentification: {},
              dateAndPlaceOfBirth:{},
            },
          }],
        },
        beneficiary: {
          beneficiaryPersons: [{
            naturalPerson: {
              name: {
                nameIdentifier: [{}]
              },
              geographicAddress: [{
                addressLine: []
              }],
              nationalIdentification: {},
              dateAndPlaceOfBirth:{},
            },
          }]
        },
        originatingVASP: {
          originatingVASP: {
            legalPerson: {
              name: {
                nameIdentifier: []
              },
              geographicAddress: [{
                addressLine: []
              }],
              nationalIdentification: {}
            },
          }
        },
        beneficiaryVASP: {
          beneficiaryVASP: {
            legalPerson: {
              name: {
                nameIdentifier: []
              },
              geographicAddress: [{
                addressLine: []
              }],
              nationalIdentification: {}
            },
          }
        },
      },
      transaction: {},
    }

    // If request is for the transaction accept endpoint set the transfer state to accepted.
    if (e.detail.path === transactionAcceptEP) {
      data.transfer_state = 'accepted';
    }

    const originatorPerson = data.identity.originator.originatorPersons[0].naturalPerson;
    const beneficiaryPerson = data.identity.beneficiary.beneficiaryPersons[0].naturalPerson;
    const originatingVASP = data.identity.originatingVASP.originatingVASP.legalPerson;
    const beneficiaryVASP = data.identity.beneficiaryVASP.beneficiaryVASP.legalPerson;

    for (const key in params) {
      // Remove prefix from the key.
      const newKey = key.split('_').slice(2).join('_');
      const indx = key.split('_')[3];

      switch (true) {
        // Set the transaction details.
        case key.startsWith('env_transaction_'):
          data.transaction[newKey] = params[key];
          break;
        // Set the originator name identifiers and name identifier type.
        case key.startsWith('id_og_'):
          originatorPerson.name.nameIdentifier[0][newKey] = params[key];
          break;
        // Set the originator date and place of birth.
        case key.startsWith('originator_birth_'):
          originatorPerson.dateAndPlaceOfBirth[newKey] = params[key];
          break;
        // Set the originator address line.
        case key.startsWith('address_og_'):
          originatorPerson.geographicAddress[0].addressLine.push(params[key]);
          break;
        // Set the originator country and address type.
        case key.startsWith('addr_og_'):
          originatorPerson.geographicAddress[0][newKey] = params[key];
          break;
        // Set details for the originator natural person that's not a name identifier or geographic address.
        case key.startsWith('np_og_'):
          originatorPerson[newKey] = params[key];
          break;
        // Set national identification for the originator.
        case key.startsWith('originator_id_'):
          originatorPerson.nationalIdentification[newKey] = params[key];
          break;
        // Set the beneficiary name identifiers and name identifier type.
        case key.startsWith('id_bf_'):
          beneficiaryPerson.name.nameIdentifier[0][newKey] = params[key];
          break;
        // Set the beneficiary date and place of birth.
        case key.startsWith('beneficiary_birth_'):
          beneficiaryPerson.dateAndPlaceOfBirth[newKey] = params[key];
          break;
        // Set the beneficiary address line.
        case key.startsWith('address_bf_'):
          beneficiaryPerson.geographicAddress[0].addressLine.push(params[key]);
          break;
        // Set the beneficiary country and address type.
        case key.startsWith('addr_bf_'):
          beneficiaryPerson.geographicAddress[0][newKey] = params[key];
          break;
        // Set details for the beneficiary natural person that's not a name identifier or geographic address.
        case key.startsWith('np_bf_'):
          beneficiaryPerson[newKey] = params[key];
          break;
        // Set national identification for the beneficiary.
        case key.startsWith('beneficiary_id_'):
          beneficiaryPerson.nationalIdentification[newKey] = params[key];
          break;
        // Set the originating VASP name identifiers and name identifier type.
        case key.startsWith('id_orig_legalPersonNameIdentifierType_'):
          originatingVASP.name.nameIdentifier.push({ legalPersonName: params[`id_orig_legalPersonName_${indx}`], legalPersonNameIdentifierType: params[`id_orig_legalPersonNameIdentifierType_${indx}`] });
          break;
        // Set the originating VASP address line.
        case key.startsWith('address_orig'):
          originatingVASP.geographicAddress[0].addressLine.push(params[key]);
          break;
        // Set the originating VASP country and address type.
        case key.startsWith('addr_orig'):
          originatingVASP.geographicAddress[0][newKey] = params[key];
          break;
        // Set the originating VASP national identification.
        case key.startsWith('nat_orig'):
          originatingVASP.nationalIdentification[newKey] = params[key];
          break;
        // Set the originating VASP country of registration.
        case key.startsWith('ctry_orig'):
          originatingVASP[newKey] = params[key];
          break;
        // Set the beneficiary VASP name identifiers and name identifier type.
        case key.startsWith('id_benf_legalPersonNameIdentifierType_'):          
          beneficiaryVASP.name.nameIdentifier.push({ legalPersonName: params[`id_benf_legalPersonName_${indx}`], legalPersonNameIdentifierType: params[`id_benf_legalPersonNameIdentifierType_${indx}`] });
          break;
        // Set the beneficiary VASP address line.
        case key.startsWith('address_benf'):
          beneficiaryVASP.geographicAddress[0].addressLine.push(params[key]);
          break;
        // Set the beneficiary VASP country and address type.
        case key.startsWith('addr_benf'):
          beneficiaryVASP.geographicAddress[0][newKey] = params[key];
          break;
        // Set the beneficiary VASP national identification.
        case key.startsWith('nat_benf'):
          beneficiaryVASP.nationalIdentification[newKey] = params[key];
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