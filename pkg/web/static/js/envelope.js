import { IDENTIFIER_TYPE, countriesArray, nationalIdentifierTypeArray } from './constants.js';

const birthplace = 'birthplace';
const country = 'country';
const nationalIdType = 'idType';

const envelopeDropdowns = [
  { sel: '#og_id_birth_place', options: birthplace },
  { sel: '#bf_id_birth_place', options: birthplace },
  { sel: '#og_id_country', options: country },
  { sel: '#bf_id_country', options: country },
  { sel: '#og_id_type_code', options: nationalIdType },
  { sel: '#bf_id_type_code', options: nationalIdType },
];

envelopeDropdowns.forEach((dropdown) => setSlimSelect(dropdown.sel, dropdown.options));

function setSlimSelect(sel, options) {
  const newDropdown = new SlimSelect({
    select: sel
  });

  if (options === birthplace || options === country) {
    countriesArray.unshift({ 'placeholder': true, 'text': 'Select a country', 'value': '' });
    newDropdown.setData(countriesArray);
  };

  if (options === nationalIdType) {
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
        identification: {},
      },
      beneficiary:{
        identification: {},
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

    for (const key in params) {
      if (key.startsWith('orig_')) {
        data.originator[key.replace('orig_', '')] = params[key];
      };

      if (key.startsWith('og_id_')) {
        data.originator.identification[key.replace('og_id_', '')] = params[key];
      }

      if (key.startsWith('benf_')) {
        data.beneficiary[key.replace('benf_', '')] = params[key];
      };

      if (key.startsWith('bf_id_')) {
        data.beneficiary.identification[key.replace('bf_id_', '')] = params[key];
      };
    };

    // Modify outgoing request data.
    console.log(data);
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

