import { networksArray } from "./constants.js";
import { networkSelect } from "./networkSelect.js";
import { setSuccessToast } from "./utils.js";

const addWalletBttn = document.getElementById('add-wallet-bttn')
const extractWalletRE = /(crypto_address|network)_(\d+)/g;
const newAcctModal = document.getElementById('new_acct_modal')
const walletDiv = document.getElementById('crypto-wallets')

// Modify the crypto wallet addresses to be sent as an array of objects in the request.
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

    // Remove any empty objects from the crypto_addresses array.
    data.crypto_addresses = data.crypto_addresses.filter((obj) => Object.keys(obj).length > 0)

    // Modify the outgoing request with the new data
    e.detail.parameters = data;
  }
});

// Add a new wallet address and network field to the new customer account form modal on click.
addWalletBttn?.addEventListener('click', () => {
  // Generate a random UUID for each new crypto wallet address and network field to ensure a unique ID as wallets are added and deleted. 
  // IDs will only be generated in secure contexts.
  let walletID = self.crypto.randomUUID();
  walletDiv?.insertAdjacentHTML('beforeend', `
  <div class="grid gap-6 my-4 md:grid-cols-2 crypto-wallets">
    <div>
      <label for="crypto_address_${walletID}" class="label-style">Wallet Address</label>
      <input type="text" id="crypto_address_${walletID}" name="crypto_address_${walletID}" class="input-style" />
    </div>
    <div>
      <label for="network_${walletID}" class="label-style">Network</label>
      <div class="flex items-center gap-x-1">
        <select id="network_${walletID}" name="network_${walletID}" class="acct-networks"></select>
        <button type="button" onclick="this.parentNode.parentNode.parentNode.remove()" class="tooltip tooltip-left" data-tip="Delete wallet">
          <i class="fa-solid fa-trash text-xs"><span class="sr-only">Delete wallet</span></i>
        </button>
      </div>
    </div>
  </div>
  `);

  // Create a searchable select dropdown for the network when a new wallet is added.
  const acctNetworks = document.querySelectorAll('.acct-networks')
  // Initialize SlimSelect for each crypto wallet network.
  acctNetworks.forEach((network) => {
    new SlimSelect({
      select: network,
      settings: {
        contentLocation: document.getElementById('new_acct_modal')
      }
    });

    // Get selected network for each additional wallet and set value to ensure the dropdown 
    // does not reset to the default when wallets are added.
    const selectedNetwork = network?.value

    // Set network options and value for each wallet.
    network?.slim?.setData(networksArray);
    network?.slim?.setSelected(selectedNetwork);
  })
})

// Close the new customer account modal and reset the form values on success.
document.body.addEventListener('htmx:afterRequest', (e) => {
  const newAcctForm = 'new-acct-form'
  // Check if the request to register a new customer account was successful.
  if (e.detail.elt.id === newAcctForm && e.detail.requestConfig.verb === 'post' && e.detail.successful) {
    // Close the modal and reset the form values.
    newAcctModal?.close()
    document.getElementById(newAcctForm).reset()
    networkSelect.setSelected({ 'placeholder': true, 'text': 'Select a country', 'value': '' })
    setSuccessToast('Success! A new customer account has been created.');

    // If user added more than 1 wallet, remove the additional wallets.
    while (walletDiv?.children.length > 1) {
      walletDiv?.removeChild(walletDiv.lastChild)
    }
  }
});

// Set the network value in the edit customer account modal form.
document.body.addEventListener('htmx:afterSettle', (e) => {
  const acctID = document.getElementById('acct_id')
  const acctPreviewEP = `/v1/accounts/${acctID?.value}/edit`;

  if (e.detail.requestConfig.path === acctPreviewEP && e.detail.requestConfig.verb === 'get') {
    // Initialize SlimSelect for each crypto wallet network.
    const walletNetworks = document.querySelectorAll('.acct-networks')
    walletNetworks.forEach((network) => {
      new SlimSelect({
        select: network,
        settings: {
          contentLocation: document.getElementById('acct_modal')
        }
      });

      // Get each network value selected by the requester from the hidden input field.
      const networkID = network?.id;
      const selectedNetwork = document.querySelector(`.${networkID}`);

      if (selectedNetwork) {
        const networkValue = selectedNetwork?.value;
        setNetworkData(network, networkValue);
      };
    });
  };
});

// Set the network options and selected value in a SlimSelect dropdown for each network.
function setNetworkData(el, value) {
  // Add network options to the select element.
  el?.slim?.setData(networksArray);
  el?.slim?.setSelected(value);
}