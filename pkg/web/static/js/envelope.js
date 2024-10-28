import { setSuccessToast } from './utils.js';

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

document.body.addEventListener('htmx:afterSettle', (e) => {
  // Reset secure envelope form after successful submission.
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