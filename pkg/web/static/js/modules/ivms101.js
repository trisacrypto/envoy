/*
Code to manage IVMS101 forms and elements.
*/

import { createChoicesWithArray } from "./components.js";


const ADDRESS_TYPE = [
  { value: '', label: 'Select Type of Address' },
  { value: 'HOME', label: 'Residential' },
  { value: 'BIZZ', label: 'Business' },
  { value: 'GEOG', label: 'Geographic' },
  { value: 'MISC', label: 'Unspecified or Miscellaneous' },
];

export function selectAddressType(elem) {
  const elementOptions = elem.dataset.addressType ? JSON.parse(elem.dataset.addressType) : {};
  return createChoicesWithArray(elem, elementOptions, ADDRESS_TYPE);
}

const NATIONAL_IDENTIFIER_TYPE = [
  { value: '', label: 'Select Type of National Identification' },
  { value: 'ARNU', label: 'Alien Residential Number' },
  { value: 'CCPT', label: 'Passport Number' },
  { value: 'RAID', label: 'Registration Authority ID' },
  { value: 'DRLC', label: "Driver's License Number" },
  { value: 'FIIN', label: 'Foreign Investment Identity Number' },
  { value: 'TXID', label: 'Tax Identification Number' },
  { value: 'SOCS', label: 'Social Security Number' },
  { value: 'IDCD', label: 'Identity Card Number' },
  { value: 'LEIX', label: 'Legal Entity Identifier (LEI)' },
  { value: 'MISC', label: 'Unspecified or Miscellaneous' },
];

export function selectNationalIdentifierType(elem) {
  const elementOptions = elem.dataset.nationalIdentifierType ? JSON.parse(elem.dataset.nationalIdentifierType) : {};
  return createChoicesWithArray(elem, elementOptions, NATIONAL_IDENTIFIER_TYPE);
}