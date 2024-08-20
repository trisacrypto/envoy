import { IDENTIFIER_TYPE, countriesArray, naturalPersonNtlIdTypeArray, networksArray } from './constants.js';
import { setSuccessToast } from './utils.js';

const network = 'network';
const birthplace = 'birthplace';
const country = 'country';
const nationalIdType = 'idType';

const envelopeDropdowns = [
  { sel: '#networks', options: network },
  { sel: '#orig_countries', options: country },
  { sel: '#benf_countries', options: country },
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

  if (options === network) {
    networksArray.unshift({ 'placeholder': true, 'text': 'Select a network', 'value': '' });
    newDropdown.setData(networksArray);
  };

  if (options === birthplace || options === country) {
    countriesArray.unshift({ 'placeholder': true, 'text': 'Select a country', 'value': '' });
    newDropdown.setData(countriesArray);
  };

  if (options === nationalIdType) {
    naturalPersonNtlIdTypeArray.unshift({ 'placeholder': true, 'text': 'Select national identifier type', 'value': '' });
    newDropdown.setData(naturalPersonNtlIdTypeArray);
  };
};

document.body.addEventListener('htmx:configRequest', (e) => {
  if (e.detail.path === '/v1/transactions/prepare' && e.detail.verb === 'post') {
    const params = e.detail.parameters;

    let data = {
      travel_address: params.travel_address,
      originator: {
        identification: {},
      },
      beneficiary: {
        identification: {},
      },
      // TODO: Add notes to data
      transfer: {
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

  if (e.detail.requestConfig.path === '/v1/transactions/send-prepared' && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    const previewEnvModal = document.getElementById('preview_envelope');
    const secureEnvForm = document.getElementById('secure-envelope-form');
    secureEnvForm.reset();
    
    // Reset the SlimSelect dropdowns after form submission.
    envelopeDropdowns.forEach((dropdown) => {
      const slimSelect = new SlimSelect({
        select: dropdown.sel
      });
      slimSelect.setData([]);
    });

    previewEnvModal.close();
    // Manually reset the screen position to ensure user is at the top of the page.
    window.scrollTo(0, 0);
    setSuccessToast('Success! Secure envelope sent.');
  };

  disableSubmitButton();
});

// Disable submit button to prevent multiple form submissions.
function disableSubmitButton() {
  const previewForm = document.getElementById('preview-form');
  const previewSbmtBtn = document.getElementById('preview-sbmt-btn');
  const previewBtnText = document.getElementById('preview-btn-text');
  previewForm?.addEventListener('submit', () => {
    previewBtnText?.classList.add('hidden');
    previewSbmtBtn.disabled = true;
  });
};