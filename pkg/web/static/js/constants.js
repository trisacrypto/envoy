const countries = {
  AF: 'Afghanistan',
  AX: 'Aland Islands',
  AL: 'Albania',
  DZ: 'Algeria',
  AS: 'American Samoa',
  AD: 'Andorra',
  AO: 'Angola',
  AI: 'Anguilla',
  AQ: 'Antarctica',
  AG: 'Antigua And Barbuda',
  AR: 'Argentina',
  AM: 'Armenia',
  AW: 'Aruba',
  AU: 'Australia',
  AT: 'Austria',
  AZ: 'Azerbaijan',
  BS: 'Bahamas',
  BH: 'Bahrain',
  BD: 'Bangladesh',
  BB: 'Barbados',
  BY: 'Belarus',
  BE: 'Belgium',
  BZ: 'Belize',
  BJ: 'Benin',
  BM: 'Bermuda',
  BT: 'Bhutan',
  BO: 'Bolivia',
  BA: 'Bosnia And Herzegovina',
  BW: 'Botswana',
  BV: 'Bouvet Island',
  BR: 'Brazil',
  IO: 'British Indian Ocean Territory',
  BN: 'Brunei Darussalam',
  BG: 'Bulgaria',
  BF: 'Burkina Faso',
  BI: 'Burundi',
  KH: 'Cambodia',
  CM: 'Cameroon',
  CA: 'Canada',
  CV: 'Cape Verde',
  KY: 'Cayman Islands',
  CF: 'Central African Republic',
  TD: 'Chad',
  CL: 'Chile',
  CN: 'China',
  CX: 'Christmas Island',
  CC: 'Cocos (Keeling) Islands',
  CO: 'Colombia',
  KM: 'Comoros',
  CG: 'Congo',
  CD: 'Congo, Democratic Republic',
  CK: 'Cook Islands',
  CR: 'Costa Rica',
  CI: "Cote D'Ivoire",
  HR: 'Croatia',
  CU: 'Cuba',
  CY: 'Cyprus',
  CZ: 'Czech Republic',
  DK: 'Denmark',
  DJ: 'Djibouti',
  DM: 'Dominica',
  DO: 'Dominican Republic',
  EC: 'Ecuador',
  EG: 'Egypt',
  SV: 'El Salvador',
  GQ: 'Equatorial Guinea',
  ER: 'Eritrea',
  EE: 'Estonia',
  ET: 'Ethiopia',
  FK: 'Falkland Islands (Malvinas)',
  FO: 'Faroe Islands',
  FJ: 'Fiji',
  FI: 'Finland',
  FR: 'France',
  GF: 'French Guiana',
  PF: 'French Polynesia',
  TF: 'French Southern Territories',
  GA: 'Gabon',
  GM: 'Gambia',
  GE: 'Georgia',
  DE: 'Germany',
  GH: 'Ghana',
  GI: 'Gibraltar',
  GR: 'Greece',
  GL: 'Greenland',
  GD: 'Grenada',
  GP: 'Guadeloupe',
  GU: 'Guam',
  GT: 'Guatemala',
  GG: 'Guernsey',
  GN: 'Guinea',
  GW: 'Guinea-Bissau',
  GY: 'Guyana',
  HT: 'Haiti',
  HM: 'Heard Island & Mcdonald Islands',
  VA: 'Holy See (Vatican City State)',
  HN: 'Honduras',
  HK: 'Hong Kong',
  HU: 'Hungary',
  IS: 'Iceland',
  IN: 'India',
  ID: 'Indonesia',
  IR: 'Iran, Islamic Republic Of',
  IQ: 'Iraq',
  IE: 'Ireland',
  IM: 'Isle Of Man',
  IL: 'Israel',
  IT: 'Italy',
  JM: 'Jamaica',
  JP: 'Japan',
  JE: 'Jersey',
  JO: 'Jordan',
  KZ: 'Kazakhstan',
  KE: 'Kenya',
  KI: 'Kiribati',
  KR: 'Korea',
  KW: 'Kuwait',
  KG: 'Kyrgyzstan',
  LA: "Lao People's Democratic Republic",
  LV: 'Latvia',
  LB: 'Lebanon',
  LS: 'Lesotho',
  LR: 'Liberia',
  LY: 'Libyan Arab Jamahiriya',
  LI: 'Liechtenstein',
  LT: 'Lithuania',
  LU: 'Luxembourg',
  MO: 'Macao',
  MK: 'Macedonia',
  MG: 'Madagascar',
  MW: 'Malawi',
  MY: 'Malaysia',
  MV: 'Maldives',
  ML: 'Mali',
  MT: 'Malta',
  MH: 'Marshall Islands',
  MQ: 'Martinique',
  MR: 'Mauritania',
  MU: 'Mauritius',
  YT: 'Mayotte',
  MX: 'Mexico',
  FM: 'Micronesia, Federated States Of',
  MD: 'Moldova',
  MC: 'Monaco',
  MN: 'Mongolia',
  ME: 'Montenegro',
  MS: 'Montserrat',
  MA: 'Morocco',
  MZ: 'Mozambique',
  MM: 'Myanmar',
  NA: 'Namibia',
  NR: 'Nauru',
  NP: 'Nepal',
  NL: 'Netherlands',
  AN: 'Netherlands Antilles',
  NC: 'New Caledonia',
  NZ: 'New Zealand',
  NI: 'Nicaragua',
  NE: 'Niger',
  NG: 'Nigeria',
  NU: 'Niue',
  NF: 'Norfolk Island',
  MP: 'Northern Mariana Islands',
  NO: 'Norway',
  OM: 'Oman',
  PK: 'Pakistan',
  PW: 'Palau',
  PS: 'Palestinian Territory, Occupied',
  PA: 'Panama',
  PG: 'Papua New Guinea',
  PY: 'Paraguay',
  PE: 'Peru',
  PH: 'Philippines',
  PN: 'Pitcairn',
  PL: 'Poland',
  PT: 'Portugal',
  PR: 'Puerto Rico',
  QA: 'Qatar',
  RE: 'Reunion',
  RO: 'Romania',
  RU: 'Russian Federation',
  RW: 'Rwanda',
  BL: 'Saint Barthelemy',
  SH: 'Saint Helena',
  KN: 'Saint Kitts And Nevis',
  LC: 'Saint Lucia',
  MF: 'Saint Martin',
  PM: 'Saint Pierre And Miquelon',
  VC: 'Saint Vincent And Grenadines',
  WS: 'Samoa',
  SM: 'San Marino',
  ST: 'Sao Tome And Principe',
  SA: 'Saudi Arabia',
  SN: 'Senegal',
  RS: 'Serbia',
  SC: 'Seychelles',
  SL: 'Sierra Leone',
  SG: 'Singapore',
  SK: 'Slovakia',
  SI: 'Slovenia',
  SB: 'Solomon Islands',
  SO: 'Somalia',
  ZA: 'South Africa',
  GS: 'South Georgia And Sandwich Isl.',
  ES: 'Spain',
  LK: 'Sri Lanka',
  SD: 'Sudan',
  SR: 'Suriname',
  SJ: 'Svalbard And Jan Mayen',
  SZ: 'Swaziland',
  SE: 'Sweden',
  CH: 'Switzerland',
  SY: 'Syrian Arab Republic',
  TW: 'Taiwan',
  TJ: 'Tajikistan',
  TZ: 'Tanzania',
  TH: 'Thailand',
  TL: 'Timor-Leste',
  TG: 'Togo',
  TK: 'Tokelau',
  TO: 'Tonga',
  TT: 'Trinidad And Tobago',
  TN: 'Tunisia',
  TR: 'Turkey',
  TM: 'Turkmenistan',
  TC: 'Turks And Caicos Islands',
  TV: 'Tuvalu',
  UG: 'Uganda',
  UA: 'Ukraine',
  AE: 'United Arab Emirates',
  GB: 'United Kingdom',
  US: 'United States',
  UM: 'United States Outlying Islands',
  UY: 'Uruguay',
  UZ: 'Uzbekistan',
  VU: 'Vanuatu',
  VE: 'Venezuela',
  VN: 'Viet Nam',
  VG: 'Virgin Islands, British',
  VI: 'Virgin Islands, U.S.',
  WF: 'Wallis And Futuna',
  EH: 'Western Sahara',
  YE: 'Yemen',
  ZM: 'Zambia',
  ZW: 'Zimbabwe'
};

export const countriesArray = Object.entries(countries).map(([value, text]) => ({ text, value }));

export const networks = {
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

export const networksArray = Object.entries(networks).map(([value, text]) => ({ text, value }));

export const IDENTIFIER_TYPE = {
  ADDRESS_TYPE_CODE_MISC: 'Unspecified',
  ADDRESS_TYPE_CODE_HOME: 'Residential',
  ADDRESS_TYPE_CODE_BIZZ: 'Business',
  ADDRESS_TYPE_CODE_GEOG: 'Geographic',
  LEGAL_PERSON_NAME_TYPE_CODE_LEGL: 'Legal',
  LEGAL_PERSON_NAME_TYPE_CODE_SHRT: 'Short',
  LEGAL_PERSON_NAME_TYPE_CODE_TRAD: 'Trading',
  LEGAL_PERSON_NAME_TYPE_CODE_MISC: 'Unspecified',
  NATIONAL_IDENTIFIER_TYPE_CODE_ARNU: 'ARNU',
  NATIONAL_IDENTIFIER_TYPE_CODE_CCPT: 'Passport',
  NATIONAL_IDENTIFIER_TYPE_CODE_RAID: 'RAID',
  NATIONAL_IDENTIFIER_TYPE_CODE_DRLC: "Driver's License",
  NATIONAL_IDENTIFIER_TYPE_CODE_FIIN: 'FIIN',
  NATIONAL_IDENTIFIER_TYPE_CODE_TXID: 'Tax ID',
  NATIONAL_IDENTIFIER_TYPE_CODE_SOCS: 'Social Security',
  NATIONAL_IDENTIFIER_TYPE_CODE_IDCD: 'Identity Card',
  NATIONAL_IDENTIFIER_TYPE_CODE_LEIX: 'LEI',
  NATIONAL_IDENTIFIER_TYPE_CODE_MISC: 'Unspecified',
  NATURAL_PERSON_NAME_TYPE_CODE_ALIA: 'Alias',
  NATURAL_PERSON_NAME_TYPE_CODE_BIRT: 'Birth',
  NATURAL_PERSON_NAME_TYPE_CODE_MAID: 'Maiden',
  NATURAL_PERSON_NAME_TYPE_CODE_LEGL: 'Legal',
  NATURAL_PERSON_NAME_TYPE_CODE_MISC: 'Unspecified',
};

// Create arrays for each identifier type category from the IDENTIFIER_TYPE object.
export const addressTypeArray = Object.entries(IDENTIFIER_TYPE)
  .filter(([key]) => key.includes('ADDRESS_TYPE_CODE'))
  .map(([value, text]) => ({ text, value }));

export const legalPersonNameTypeArray = Object.entries(IDENTIFIER_TYPE)
  .filter(([key]) => key.includes('LEGAL_PERSON_NAME_TYPE_CODE'))
  .map(([value, text]) => ({ text, value }));

export const nationalIdentifierTypeArray = Object.entries(IDENTIFIER_TYPE)
  .filter(([key]) => key.includes('NATIONAL_IDENTIFIER_TYPE_CODE'))
  .map(([value, text]) => ({ text, value }));

export const naturalPersonNtlIdTypeArray = Object.entries(IDENTIFIER_TYPE)
  .filter(([key]) => key.includes('NATIONAL_IDENTIFIER_TYPE_CODE') && !['LEIX', 'TXID', 'RAID'].includes(key.split('_').pop()))
  .map(([value, text]) => ({ text, value }));

export const naturalPersonNameTypeArray = Object.entries(IDENTIFIER_TYPE)
  .filter(([key]) => key.includes('NATURAL_PERSON_NAME_TYPE_CODE'))
  .map(([value, text]) => ({ text, value }));


// Reject error codes
export const REJECT_CODES = {
  REJECTED: "Default rejection - no specified reason for rejecting the transaction",
  UNKNOWN_WALLET_ADDRESS: "VASP does not control the specified wallet address",
  UNKNOWN_IDENTITY: "VASP does not have KYC information for the specified wallet address",
  UNKNOWN_ORIGINATOR: "Specifically, the Originator Account cannot be identified",
  UNKNOWN_BENEFICIARY: "Specifically, the Beneficiary account cannot be identified",
  UNSUPPORTED_CURRENCY: "VASP cannot support the fiat currency or coin described in the transaction",
  EXCEEDED_TRADING_VOLUME: "No longer able to receive more transaction inflows",
  COMPLIANCE_CHECK_FAIL: "An internal compliance check has failed or black listing has occurred",
  NO_COMPLIANCE: "VASP not able to implement travel rule compliance",
  HIGH_RISK: "VASP unwilling to conduct the transaction because of a risk assessment",
  OUT_OF_NETWORK: "Wallet address or transaction is not available on this network",
  UNPARSEABLE_IDENTITY: "Unable to parse identity record",
  UNPARSEABLE_TRANSACTION: "Unable to parse transaction data record",
  MISSING_FIELDS: "There are missing required fields in the transaction data",
  INCOMPLETE_IDENTITY: "The identity record is not complete enough for compliance purposes of the receiving VASPs",
  VALIDATION_ERROR: "There was an error validating a field in the transaction data",
  COMPLIANCE_PERIOD_EXCEEDED: "The review period has exceeded the required compliance timeline",
  CANCELED: "The TRISA exchange was canceled",
  CANCEL_TRANSACTION: "The TRISA exchange was canceled",
  BVRC999: "Request could not be processed by recipient (Alias: Sygna BVRC Rejected Code)",
};

// API Key Permissions
export const API_KEY_PERMISSIONS = [
  "users:manage", "users:view",
	"apikeys:manage", "apikeys:view", "apikeys:revoke",
	"counterparties:manage", "counterparties:view",
	"accounts:manage", "accounts:view",
	"travelrule:manage", "travelrule:delete", "travelrule:view",
	"config:manage", "config:view",
	"pki:manage", "pki:delete", "pki:view",
]