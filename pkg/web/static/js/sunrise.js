import { setSuccessToast } from "./utils.js";

document.body.addEventListener('htmx:configRequest', (e) => {
  if (e.detail.path === '/v1/sunrise/send' && e.detail.verb === 'post') {
    const params = e.detail.parameters;

    let data = {
      email: params.email,
      counterparty: params.counterparty,
      originator: {
        identification: {},
      },
      beneficiary: {
      },
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
    };

    // Modify outgoing request data.
    e.detail.parameters = data;
  }
});

document.body.addEventListener('htmx:afterRequest', (e) => {
  if (e.detail.requestConfig.path === '/v1/sunrise/send' && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    setSuccessToast('Success! The sunrise message has been sent.');
  }
});
