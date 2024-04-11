const previewEnvelopeBttn = document.getElementById('preview-envelope-bttn')
const secureEnvelopeForm = document.getElementById('secure-envelope-form')

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
        country: params.benf_country,
        crypto_address: params.benf_crypto_address
      },
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

  if (e.detail.path === '/v1/transactions/send' && e.detail.verb === 'post') {
    console.log(e)
   // TODO: Add form to transfer preview and transform data from the params before sending to the server.
  }
});

