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