/*
Code to manage IVMS101 forms and elements.
*/

import { choicesDefaultOptions } from "./components.js";


const ADDRESS_TYPE = [
  { value: '', label: 'Select Type of Address' },
  { value: 'HOME', label: 'Residential' },
  { value: 'BIZZ', label: 'Business' },
  { value: 'GEOG', label: 'Geographic' },
  { value: 'MISC', label: 'Unspecified or Miscellaneous' },
];

export function selectAddressType(elem) {
  const elementOptions = elem.dataset.addressType ? JSON.parse(elem.dataset.addressType) : {};
  elementOptions.choices = ADDRESS_TYPE;

  const options = {
    ...elementOptions,
    ...choicesDefaultOptions(elem),
  };

  return new Choices(elem, options);
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
  elementOptions.choices = NATIONAL_IDENTIFIER_TYPE;

  const options = {
    ...elementOptions,
    ...choicesDefaultOptions(elem),
  };

  return new Choices(elem, options);
}