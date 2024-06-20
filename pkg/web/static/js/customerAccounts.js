import { networksArray } from "./constants.js";

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

// Count the number of wallet addresses to ensure a unique number as crypto wallets or added and deleted.
// Start the counter at 1 to avoid a conflict with the first crypto wallet.
let walletIdCounter = 1;

// Add a new wallet address and network field to the new customer account form modal on click.
addWalletBttn?.addEventListener('click', () => {
  const walletCount = walletIdCounter;
  walletDiv?.insertAdjacentHTML('beforeend', `
  <div class="grid gap-6 my-4 md:grid-cols-2 crypto-wallets">
    <div>
      <label for="crypto_address_${walletCount}" class="label-style">Wallet Address</label>
      <input type="text" id="crypto_address_${walletCount}" name="crypto_address_${walletCount}" class="input-style" />
    </div>
    <div>
      <label for="network_${walletCount}" class="label-style">Network</label>
      <div class="flex items-center gap-x-1">
        <select id="network_${walletCount}" name="network_${walletCount}" class="acct-networks"></select>
        <button type="button" onclick="this.parentNode.parentNode.parentNode.remove()" class="tooltip tooltip-left" data-tip="Delete wallet">
          <i class="fa-solid fa-trash text-xs"><span class="sr-only">Delete wallet</span></i>
        </button>
      </div>
    </div>
  </div>
  `);

  // Increment the wallet counter for the next wallet address.
  walletIdCounter++;

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

    // Get the selected network value for each additional wallet and set it as the selected value
    // to ensure the dropdown does not reset the value when a new wallet is added.
    const selectedValue = network?.value

    // Add network options to each additional wallet.
    network?.slim?.setData(networksArray);
    network?.slim?.setSelected(selectedValue);
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