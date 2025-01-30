import { countriesArray, addressTypeArray, naturalPersonNtlIdTypeArray, nationalIdentifierTypeArray } from "./constants.js";
import { setSuccessToast } from "./utils.js";

// Set data for SlimSelect elements that appear in the send envelope and send message forms.
const country = 'country';
const natnitc = 'natnitc';
const legnitc = 'legnitc';
const address = 'address';

function initializeSlimSelects()  {
  const elements = document.querySelectorAll('[data-select-type]');
  elements.forEach((element) => setSlimSelect(element));
}

function setSlimSelect(element) {
  // Initialize SlimSelect only for select elements that exist in the DOM.
  if (!element) {
    return
  };

  const selectType = element.getAttribute('data-select-type');
  const newDropdown = new SlimSelect({
    select: '#' + element.getAttribute('id'),
  });

  if (selectType === country) {
    countriesArray.unshift({ 'placeholder': true, 'text': 'Select a country', 'value': '' });
    newDropdown.setData(countriesArray);
  };

  if (selectType === address) {
    addressTypeArray.unshift({ 'placeholder': true, 'text': 'Select address type', 'value': '' });
    newDropdown.setData(addressTypeArray);
  };

  if (selectType === natnitc) {
    naturalPersonNtlIdTypeArray.unshift({ 'placeholder': true, 'text': 'Select national identifier type', 'value': '' });
    newDropdown.setData(naturalPersonNtlIdTypeArray);
  };

  if (selectType === legnitc) {
    nationalIdentifierTypeArray.unshift({ 'placeholder': true, 'text': 'Select national identifier type', 'value': '' });
    newDropdown.setData(nationalIdentifierTypeArray);
  };

  // Set the default value on the dropdown if present.
  const value = element?.getAttribute('value');
  if (value) {
    newDropdown.setSelected(value);
  };
};


// Initialie all dropdown slim selects in DOM
initializeSlimSelects()

// HTMX event listeners
const rejectEP = "/sunrise/reject";
const acceptEP = "/sunrise/accept";

document.body.addEventListener('htmx:configRequest', (e) => {
  if (e.detail.path === rejectEP && e.detail.verb === 'post') {
    // Parse the retry checkbox into a boolean for the POST data
    const isRetryChecked = document.getElementById('retry');
    const retryTransaction = isRetryChecked?.checked;
    e.detail.parameters = {
      ...e.detail.parameters,
      retry: retryTransaction,
    };
  }

  if (e.detail.path === acceptEP && e.detail.verb === 'post') {
    // Convert beneficiary form into JSON envelope.
    e.detail.parameters = parseBeneficiaryForm(e.detail.parameters);
  }
});

// Reset the reject transaction form if the request is successful.
document.body.addEventListener('htmx:afterRequest', (e) => {
  if (e.detail.requestConfig.path === rejectEP && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    const transactionRejectForm = document.getElementById('transaction-reject-form')
    const transactionRejectionModal = document.getElementById('transaction_rejection_modal')
    transactionRejectionModal.close()
    transactionRejectForm.reset()
    setSuccessToast('Thank you! The sunrise message has been rejected.')
  }
});


function parseBeneficiaryForm(params) {
  let data = {
    identity: {
      originator: {},
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
            customerIdentification: "",
            countryOfResidence: "",
          },
        }],
        accountNumber:[],
      },
      originatingVASP: {},
      beneficiaryVASP: {
        beneficiaryVASP: {
          legalPerson: {
            name: {
              nameIdentifier: []
            },
            geographicAddress: [{
              addressLine: []
            }],
            nationalIdentification: {},
            customerNumber: "",
            countryOfRegistration: "",
          },
        },
      },
    },
    transaction: {},
    transfer_state: "accepted",
  };

  const beneficiaryPerson = data.identity.beneficiary.beneficiaryPersons[0].naturalPerson;
  const beneficiaryVASP = data.identity.beneficiaryVASP.beneficiaryVASP.legalPerson;

  for (const key in params) {
    // Remove prefix from the key.
    const newKey = key.split('_').slice(2).join('_');
    const nameIndx = key.split('_')[3];

    switch (true) {
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
      // Set the beneficiary account number from the wallet address
      case key.startsWith('benf_crypto_address_'):
        data.identity.beneficiary.accountNumber.push(params[key]);
        break;
      // Set the beneficiary VASP name identifiers and name identifier type.
      case key.startsWith('id_benf_legalPersonNameIdentifierType_'):
        beneficiaryVASP.name.nameIdentifier.push({ legalPersonName: params[`id_benf_legalPersonName_${nameIndx}`], legalPersonNameIdentifierType: params[`id_benf_legalPersonNameIdentifierType_${nameIndx}`] });
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

  // Manipulate address lines
  for (let i = 0; i < beneficiaryPerson.geographicAddress.length; i++) {
      let addr = beneficiaryPerson.geographicAddress[i];
      beneficiaryPerson.geographicAddress[i] = convertToAddressLines(addr);
  }

  for (let i = 0; i < beneficiaryVASP.geographicAddress.length; i++) {
      let addr = beneficiaryVASP.geographicAddress[i];
      beneficiaryVASP.geographicAddress[i] = convertToAddressLines(addr);
  }

  // Ensure country of residence is set for the beneficiary natural person.
  beneficiaryPerson.countryOfResidence = beneficiaryPerson.nationalIdentification.countryOfIssue;

  return data;
};

function convertToAddressLines(addr) {
  // Make sure city, state, and post_code are defined.
  let line = ""
  for (const key of ["city", "state", "post_code"]) {
    if (addr[key]) {
      if (line !== "") {
        line += ", ";
      }
      line += addr[key];
    }
  }

  // Add city, state, and post_code as an address line
  addr.addressLine.push(line);

  // Empty specified values.
  addr.city = "";
  addr.state = "";
  addr.post_code = "";

  // Filter empty address lines.
  addr.addressLine = addr.addressLine.filter(line => line !== '');
  return addr;
}