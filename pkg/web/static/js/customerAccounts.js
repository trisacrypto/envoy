const addWalletBttn = document.getElementById('add-wallet-bttn')
const extractWalletRE = /(crypto_address|network)_(\d+)/g;

document.body.addEventListener("htmx:configRequest", (e) => {
  // Check if this is a POST request for the accounts form.
  if (e.detail.path == "/v1/accounts" && e.detail.verb == "post") {
    // Modify the POST data to ensure the wallet addresses are collected correctly
    const params = e.detail.parameters;
    let data = {
      crypto_addresses: []
    };

    // Add all parameters to the data except the crypto_address and network information.
    // TODO: it would be better to sort keys rather than use a while to extend the
    // crypto addresses array until there are enough objects to populate it.
    for (const [key, value] of Object.entries(params)) {
      if (key.startsWith("crypto_address") || key.startsWith("network")) {
        const matches = key.matchAll(extractWalletRE)
        for (const [_, key, idxs] of matches) {
          const idx = parseInt(idxs);
          while (data.crypto_addresses.length < idx+1) {
            data.crypto_addresses.push({});
          }

          data.crypto_addresses[idx][key] = value;
        }
      } else {
        data[key] = value;
      }
    }

    // Modify the outgoing request with the new data
    e.detail.parameters = data;
  }
});


addWalletBttn?.addEventListener('click', () => {
  const walletDiv = document.getElementById('crypto-wallets')
  const walletCount = walletDiv.children.length + 1
  walletDiv.insertAdjacentHTML('beforeend', `
  <div class="grid gap-6 my-4 md:grid-cols-2 crypto-wallets">
    <div>
      <label for="crypto_address_${walletCount}" class="label-style">Wallet Address ${walletCount}</label>
      <input type="text" id="crypto_address_${walletCount}" name="crypto_address_${walletCount}" class="input-style" />
    </div>
    <div>
      <label for="network_${walletCount}" class="label-style">Network</label>
      <div class="flex items-center gap-x-1">
        <input type="text" id="network_${walletCount}" name="network_${walletCount}" class="input-style" />
        <button type="button" onclick="this.parentNode.parentNode.parentNode.remove()">
          <i class="fa-solid fa-trash"><span class="sr-only">Delete wallet</span></i>
        </button>
      </div>
    </div>
  </div>
  `)
})

document.body.addEventListener('htmx:afterRequest', (e) => {
  const newAcctForm = 'new-acct-form'
  // Check if the request to register a new customer account was successful.
  if (e.detail.elt.id === newAcctForm && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    // Close the modal and reset the form values.
    document.getElementById('new_acct_modal').close()
    document.getElementById(newAcctForm).reset()
  }
});