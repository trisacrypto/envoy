import { networksArray } from "./constants.js";

document.body.addEventListener('htmx:afterSettle', () => {
  const transactionNetwork = new SlimSelect({
    select: "#transaction-networks",
    settings: {
      contentLocation: document.getElementById('network-content')
    }
  })

  // Set the list of options in the network dropdown and display the value selected by the requester.
  const networkEl = document.getElementById('selected-network');
  const networkValue = networkEl.value;
  transactionNetwork.setData(networksArray);
  transactionNetwork.setSelected(networkValue);
})