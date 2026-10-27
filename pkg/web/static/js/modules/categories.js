import { createChoicesWithArray } from "./components.js";

const BUSINESS_CATEGORIES = [
  {"value": "", "label": "Select Business Category"},
  {"value": "UNKNOWN_ENTITY", "label": "Unknown Entity"},
  {"value": "PRIVATE_ORGANIZATION", "label": "Private Organization"},
  {"value": "GOVERNMENT_ENTITY", "label": "Government Entity"},
  {"value": "BUSINESS_ENTITY", "label": "Business Entity"},
  {"value": "NON_COMMERCIAL_ENTITY", "label": "Non-Commercial Entity"},
  {"value": "NON_VASP_ENTITY", "label": "Non-VASP Entity"}
]

const VASP_CATEGORIES = [
  {"value": "Exchange", "label": "Exchange"},
  {"value": "DEX", "label": "DEX"},
  {"value": "P2P", "label": "P2P"},
  {"value": "Kiosk", "label": "Kiosk"},
  {"value": "Custodian", "label": "Custodian"},
  {"value": "OTC", "label": "OTC"},
  {"value": "Fund", "label": "Fund"},
  {"value": "Project", "label": "Project"},
  {"value": "Gambling", "label": "Gambling"},
  {"value": "Miner", "label": "Miner"},
  {"value": "Mixer", "label": "Mixer"},
  {"value": "Individual", "label": "Individual"},
  {"value": "Other", "label": "Other"},
  {"value": "Unknown", "label": "Unknown"}
]

export function selectBusinessCategory(elem) {
  const elementOptions = elem.dataset.businessCategories ? JSON.parse(elem.dataset.businessCategories) : {};
  return createChoicesWithArray(elem, elementOptions, BUSINESS_CATEGORIES);
}

export function selectVASPCategory(elem) {
  const elementOptions = elem.dataset.vaspCategories ? JSON.parse(elem.dataset.vaspCategories) : {};
  return createChoicesWithArray(elem, elementOptions, VASP_CATEGORIES);
}