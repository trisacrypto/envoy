const previewEnvelopeBttn = document.getElementById('preview-envelope-bttn')
const secureEnvelopeForm = document.getElementById('secure-envelope-form')

// Get form values and display in preview modal.
previewEnvelopeBttn?.addEventListener('click', () => {
  const formData = new FormData(secureEnvelopeForm);
  const envelopeData = Object.fromEntries(formData);

  // Originator Info Form Values
  document.getElementById('orig-name').textContent = envelopeData.orig_first_name + ' ' + envelopeData.orig_last_name
  document.getElementById('orig-internal-acct').textContent = envelopeData.customer_identifier
  document.getElementById('orig-addr-one').textContent = envelopeData.address_one

  if (envelopeData.address_two) {
    document.getElementById('orig-addr-two').textContent = envelopeData.address_two
  }

  document.getElementById('orig-addr-three').textContent = envelopeData.city + ' ' + envelopeData.region + ' ' + envelopeData.postal_code
  document.getElementById('orig-country').textContent = envelopeData.country

  // Beneficiary Info Form Values
  document.getElementById('benf-name').textContent = ''
  document.getElementById('benf-vasp-name').textContent = envelopeData.beneficiary_vasp
  document.getElementById('benf-wallet-addr').textContent = envelopeData.wallet_address

  // Virtual Asset Form Values
  document.getElementById('asset-type').textContent = envelopeData.asset_type
  document.getElementById('transfer-amt').textContent = envelopeData.amount
  document.getElementById('transfer-tag').textContent = envelopeData.tag
});

// Display a searchable dropdown for networks.
const networkSelect = new SlimSelect({
  select: '#networks',
});

const networks = {
  "BTC": "Bitcoin",
  "ETH": "Ethereum",
  "AGM": "Argoneum",
  "BCH": "Bitcoin Cash",
  "BTG": "Bitcoin Gold",
  "XBC": "Bitcoinplus",
  "BTX": "BitCore",
  "CHC": "Chaincoin",
  "DASH": "Dash",
  "DOGEC": "DogeCash",
  "DOGE": "Dogecoin",
  "FTC": "Feathercoin",
  "GRS": "Groestlcoin",
  "KOTO": "Koto",
  "LBTC": "Liquid",
  "LTC": "Litecoin",
  "MONA": "Monacoin",
  "POLIS": "Polis",
  "TRC": "Terracoin",
  "UFO": "UFO",
  "XVG": "Verge Currency",
  "VIA": "Viacoin",
  "ZCL": "Zclassic",
  "XZC": "ZCoin"
};
const networksArray = Object.entries(networks).map(([value, text]) => ({ text, value }));
networksArray.unshift({ 'placeholder': true, 'text': 'Select a network', 'value': '' });
networkSelect.setData(networksArray);

