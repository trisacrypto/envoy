import { networksArray } from "./constants.js";

const networkContentDiv = document.getElementById('network-content')
// Display a searchable dropdown for networks.
const networkSelect = new SlimSelect({
  select: '#networks',
  settings: {
    contentLocation: networkContentDiv,
  },
});

networksArray.unshift({ 'placeholder': true, 'text': 'Select a network', 'value': '' });
networkSelect.setData(networksArray);