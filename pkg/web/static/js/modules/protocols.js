import { createChoicesWithArray } from "./components.js";

const PROTOCOLS = [
  {"value": "", "label": "Select Protocol"},
  {"value": "sunrise", "label": "Sunrise"},
  {"value": "trp", "label": "TRP"},
];

export function selectProtocol(elem) {
  const elementOptions = elem.dataset.protocols ? JSON.parse(elem.dataset.protocols) : {};
  return createChoicesWithArray(elem, elementOptions, PROTOCOLS);
}