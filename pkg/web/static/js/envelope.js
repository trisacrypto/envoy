import { IDENTIFIER_TYPE, countriesArray, nationalIdentifierTypeArray } from './constants.js';

const birthplace = 'birthplace';
const country = 'country';
const idType = 'idType';

const envelopeDropdowns = [
  { sel: '#orig_birth_place', type: birthplace },
  { sel: '#benf_birth_place', type: birthplace },
  { sel: '#orig_identification_country', type: country },
  { sel: '#benf_identification_country', type: country },
  { sel: '#orig_type_code', type: idType },
  { sel: '#benf_type_code', type: idType },
];

envelopeDropdowns.forEach((dropdown) => setSlimSelect(dropdown.sel, dropdown.type));

function setSlimSelect(sel, type) {
  const newDropdown = new SlimSelect({
    select: sel
  });

  if (type === 'birthplace' || type === 'country') {
    countriesArray.unshift({ 'placeholder': true, 'text': 'Select a country', 'value': '' });
    newDropdown.setData(countriesArray);
  };

  if (type === 'idType') {
    nationalIdentifierTypeArray.unshift({ 'placeholder': true, 'text': 'Select national identifier type', 'value': '' });
    newDropdown.setData(nationalIdentifierTypeArray);
  };
};

document.body.addEventListener('htmx:configRequest', (e) => {
  if (e.detail.path === '/v1/transactions/prepare' && e.detail.verb === 'post') {
    const params = e.detail.parameters;

    let data = {
      travel_address: params.travel_address,
      originator:{
        first_name: params.orig_first_name,
        last_name: params.orig_last_name,
        customer_id: params.orig_customer_id,
        addr_line_1: params.orig_addr_line_1,
        addr_line_2: params.orig_addr_line_2,
        city: params.orig_city,
        post_code: params.orig_post_code,
        state: params.orig_state,
        country: params.orig_country,
        crypto_address: params.orig_crypto_address
      },
      beneficiary:{
        first_name: params.benf_first_name,
        last_name: params.benf_last_name,
        customer_id: params.benf_customer_id,
        addr_line_1: params.benf_addr_line_1,
        addr_line_2: params.benf_addr_line_2,
        city: params.benf_city,
        state: params.benf_state,
        post_code: params.benf_post_code,
        country: params.benf_country,
        crypto_address: params.benf_crypto_address
      },
      // TODO: Add notes to data
      transfer:{
        amount: parseFloat(params.amount),
        network: params.network,
        asset_type: params.asset_type,
        transaction_id: params.transaction_id,
        tag: params.tag,
      },
    }

    // Modify outgoing request data.
    e.detail.parameters = data;
  }

  if (e.detail.path === '/v1/transactions/send-prepared' && e.detail.verb === 'post') {
    const params = e.detail.parameters;

    // Parse JSON data and remove dump property.
    let data = JSON.parse(params.prepared_payload);
    delete data.dump;

    // Modify outgoing request with parsed JSON data.
    e.detail.parameters = data;
  }
});

// Use human readable identifier types in the transaction preview.
document.body.addEventListener('htmx:afterSettle', (e) => {
  if (e.detail.requestConfig.path === '/v1/transactions/prepare' && e.detail.requestConfig.verb === 'post') {
    const identifierTypes = document.querySelectorAll('.identifier-type');
    identifierTypes.forEach((identifierType) => {
      const identifierCode = identifierType.textContent;
      const readableIdentifierType = IDENTIFIER_TYPE[identifierCode];
      identifierType.textContent = readableIdentifierType || identifierCode;
    });
  }
})

