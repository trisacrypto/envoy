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
  { value: '', label: 'Select Type of Identification' },
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

/*
Encodes a JSON object as a base64 encoded JSON string.
*/
export function encode(obj) {
  const textEncoder = new TextEncoder();
  const bytes = textEncoder.encode(JSON.stringify(obj));
  return btoa(String.fromCharCode(...bytes));
}

/*
FormReader extends FormData to filter fields based on a prefix.
*/
class FormReader {
  constructor(form, prefix) {
    // Basic type checking and form data conversion.
    if (!(form instanceof FormData)) {
      if (form instanceof HTMLFormElement) {
        form = new FormData(form);
      } else if (form instanceof FormReader) {
        form = form.form;
      } else {
        throw new Error("cannot construct envelope from non-form data");
      }
    }

    this.form = form;
    this.prefix = prefix || "";
  }

  load() {
    // Load the json:prefix data if it exists on the form, otherwise return null.
    if (this.prefix) {
      const key = `json:${this.prefix}`;
      if (this.form.has(key)) {
        const value = this.form.get(key);
        if (value) {
          return JSON.parse(value);
        }
      }
    }
    return null;
  }

  has(key) {
    const prefixedKey = this.prefix ? `${this.prefix}_${key}` : key;
    return this.form.has(prefixedKey);
  }

  get(key) {
    const prefixedKey = this.prefix ? `${this.prefix}_${key}` : key;
    return this.form.get(prefixedKey);
  }

  entries() {
    // If a prefix is set, only return entries that start with the prefix.
    if (this.prefix) {
      const result = [];
      const prefix = this.prefix + "_";

      for (const [key, value] of this.form.entries()) {
        if (key.startsWith(prefix)) {
          result.push([key.replace(prefix, ""), value]);
        }
      }
      return result;
    }

    // Otherwise, return all entries.
    return this.form.entries();
  }

}

/*
Envelope is a helper class to manage API calls to deeply nested IVMS101 structures.
*/
export class Envelope {

  constructor(form, options) {
    const defaultOptions = {
      prefix: "",
      transactionPrefix: "transaction",
      identityPrefixes: {
        originatorPrefix: "originator",
        beneficiaryPrefix: "beneficiary",
        beneficiaryVASPPrefix: "beneficiaryVASP",
        originatingVASPPrefix: "originatingVASP",
      }
    };

    options = options || {};
    this.options = Object.assign(defaultOptions, options);

    this.sent_at = null;
    this.received_at = null;

    if (form) {
      // Create a transaction and identity object from the form data.
      this.transaction = new Transaction(form, {prefix: this.options.transactionPrefix});
      this.identity = new Identity(form, this.identityPrefixes);

      // Get top level form elements from the form data.
      form = new FormReader(form, this.options.prefix);
      if (form.has("sent_at")) this.sent_at = form.get("sent_at");
      if (form.has("received_at")) this.received_at = form.get("received_at");

      // Add the account numbers to the identity object.
      const originatorAccount = this.transaction.originator();
      if (originatorAccount) {
        this.identity.addOriginatorAccount(originatorAccount);
      }

      const beneficiaryAccount = this.transaction.beneficiary();
      if (beneficiaryAccount) {
        this.identity.addBeneficiaryAccount(beneficiaryAccount);
      }
    } else {

      // Create an empty transaction and identity object.
      this.transaction = new Transaction();
      this.identity = new Identity();
    }
  }

  entries() {
    const result = [];
    result.push(["transaction", this.transaction]);
    result.push(["identity", this.identity]);

    if (this.sent_at) result.push(["sent_at", this.sent_at]);
    if (this.received_at) result.push(["received_at", this.received_at]);
    return result;
  }

  toJSON() {
    return Object.fromEntries(this.entries());
  }
}

/*
Wraps transaction information as part of the envelope payload.
*/
export class Transaction {

  static FIELDS = [
    "txid",
    "originator",
    "beneficiary",
    "amount",
    "network",
    "timestamp",
    "asset_type",
    "tag",
    "extra_json"
  ]

  constructor(form, options) {
    const defaultOptions = {
      prefix: "transaction",
    };

    options = options || {};
    this.options = Object.assign(defaultOptions, options);

    if (form) {
      form = new FormReader(form, this.options.prefix);

      // Try loading the transaction data from the form.
      this.data = form.load();

      // If the data is still null, then load it from the form fields.
      if (!this.data) {
        this.data = {};
        for (const field of Transaction.FIELDS) {
          if (form.has(field)) {
            this.data[field] = form.get(field);
            if (field === "amount") {
              this.data[field] = parseFloat(this.data[field]);
            }
          }
        }
      }

    } else {
      this.data = {};
    }
  }

  originator() {
    if (this.data && this.data["originator"]) {
      return this.data["originator"];
    }
    return null;
  }

  beneficiary() {
    if (this.data && this.data["beneficiary"]) {
      return this.data["beneficiary"];
    }
    return null;
  }

  entries() {
    if (this.data) {
      const result = [];
      for (const [key, value] of Object.entries(this.data)) {
        if (key == "amount") {
          result.push(["json:amount", value]);
        } else {
          result.push([key, value]);
        }
      }
      return result;
    }
    return [];
  }

  toJSON() {
    return this.data;
  }
}

/*
An IVMS101 identity payload containing originator, beneficiary, and VASP information.
*/
export class Identity {

  static FIELDS = [
    "originator",
    "beneficiary",
    "originatingVASP",
    "beneficiaryVASP",
    "transferPath",
    "payloadMetadata"
  ]

  constructor(form, options) {
    const defaultOptions = {
      originatorPrefix: "originator",
      beneficiaryPrefix: "beneficiary",
      beneficiaryVASPPrefix: "beneficiaryVASP",
      originatingVASPPrefix: "originatingVASP",
    };

    options = options || {};
    this.options = Object.assign(defaultOptions, options);

    if (form) {
      this.data = {}

      // TODO: handle multiple originators and beneficiaries
      // TODO: handle originators and beneficiaries as companies
      this.data["originator"] = {
        originatorPersons: [
          new NaturalPerson(form, {prefix: this.options.originatorPrefix})
        ]
      };

      this.data["beneficiary"] = {
        beneficiaryPersons: [
          new NaturalPerson(form, {prefix: this.options.beneficiaryPrefix})
        ]
      };

      // Handle the VASPs as companies
      this.data["originatingVASP"] = {
        originatingVASP: new LegalPerson(form, {prefix: this.options.originatingVASPPrefix})
      };

      this.data["beneficiaryVASP"] = {
        beneficiaryVASP: new LegalPerson(form, {prefix: this.options.beneficiaryVASPPrefix})
      };
    } else {
      this.data = {};
    }

  }

  addOriginatorAccount(account) {
    if (!this.data["originator"]) {
      this.data["originator"] = {
        "originatorPersons": [],
        "accountNumber": []
      };
    }

    if (!this.data["originator"]["accountNumber"]) {
      this.data["originator"]["accountNumber"] = [];
    }

    this.data["originator"]["accountNumber"].push(account);
  }

  addBeneficiaryAccount(account) {
    if (!this.data["beneficiary"]) {
      this.data["beneficiary"] = {
        "beneficiaryPersons": [],
        "accountNumber": []
      };
    }

    if (!this.data["beneficiary"]["accountNumber"]) {
      this.data["beneficiary"]["accountNumber"] = [];
    }

    this.data["beneficiary"]["accountNumber"].push(account);
  }

  entries() {
    if (this.data) {
      return Object.entries(this.data);
    }
    return [];
  }

  toJSON() {
    return this.data;
  }
}

export class NaturalPerson {

  constructor(form, options) {
    const defaultOptions = {
      prefix: "",
    };

    options = options || {};
    this.options = Object.assign(defaultOptions, options);

    if (form) {
      form = new FormReader(form, this.options.prefix);

      // Try loading the data from the form.
      this.data = form.load();

      // If the data is still null, then load it from the form fields.
      if (!this.data) {
        this.data = {
          "name": {
            "nameIdentifier": []
          },
          "nationalIdentification": {},
          "geographicAddress": [],
          "dateAndPlaceOfBirth": {},
        };
      }

      form.entries().forEach(([key, value]) => {
        // If we don't have a value, then skip it.
        if (!value) return;

        // Identify the object to update; handling nested data structures
        let obj = this.data;

        if (key.startsWith("name_nameIdentifier_")) {
          key = key.replace("name_nameIdentifier_", "");
          const idx = parseInt(key.split("_", 1)[0]);
          key = key.replace(idx + "_", "");

          if (obj.name.nameIdentifier.length <= idx) {
            obj.name.nameIdentifier.push({});
          }

          obj = obj.name.nameIdentifier[idx];
        }

        if (key.startsWith("nationalIdentification_")) {
          key = key.replace("nationalIdentification_", "");
          obj = obj.nationalIdentification;
        }

        if (key.startsWith("dateAndPlaceOfBirth_")) {
          key = key.replace("dateAndPlaceOfBirth_", "");
          obj = obj.dateAndPlaceOfBirth;
        }

        // NOTE: geographicAddress needs to be handled last because it might create a
        // key that is an index and doesn't have the startsWith method.
        if (key.startsWith("geographicAddress_")) {
          key = key.replace("geographicAddress_", "");
          const idx = parseInt(key.split("_", 1)[0]);
          key = key.replace(idx + "_", "");

          if (obj.geographicAddress.length <= idx) {
            obj.geographicAddress.push({"addressLine": ["", "", ""]});
          }

          obj = obj.geographicAddress[idx];
          if (key.startsWith("addressLine_")) {
            key = parseInt(key.replace("addressLine_", ""));
            obj = obj.addressLine;
          }
        }

        obj[key] = value;
      });

    } else {
      this.data = {};
    }
  }

  entries() {
    return [["naturalPerson", this.data]];
  }

  toJSON() {
    return {
      "naturalPerson": this.data
    };
  }
}

export class LegalPerson {

  constructor(form, options) {
    const defaultOptions = {
      prefix: "",
    };

    options = options || {};
    this.options = Object.assign(defaultOptions, options);

    if (form) {
      form = new FormReader(form, this.options.prefix);

      // Try loading the data from the form.
      this.data = form.load();

      // If the data is still null, then load it from the form fields.
      if (!this.data) {
        this.data = {
          "name": {
            "nameIdentifier": []
          },
          "nationalIdentification": {},
          "geographicAddress": [],
        };
      }

      form.entries().forEach(([key, value]) => {
        // If we don't have a value, then skip it.
        if (!value) return;

        // Identify the object to update; handling nested data structures
        let obj = this.data;

        if (key.startsWith("name_nameIdentifier_")) {
          key = key.replace("name_nameIdentifier_", "");
          const idx = parseInt(key.split("_", 1)[0]);
          key = key.replace(idx + "_", "");

          if (obj.name.nameIdentifier.length <= idx) {
            obj.name.nameIdentifier.push({});
          }

          obj = obj.name.nameIdentifier[idx];
        }

        if (key.startsWith("nationalIdentification_")) {
          key = key.replace("nationalIdentification_", "");
          obj = obj.nationalIdentification;
        }

        // NOTE: geographicAddress needs to be handled last because it might create a
        // key that is an index and doesn't have the startsWith method.
        if (key.startsWith("geographicAddress_")) {
          key = key.replace("geographicAddress_", "");
          const idx = parseInt(key.split("_", 1)[0]);
          key = key.replace(idx + "_", "");

          if (obj.geographicAddress.length <= idx) {
            obj.geographicAddress.push({"addressLine": ["", "", ""]});
          }

          obj = obj.geographicAddress[idx];
          if (key.startsWith("addressLine_")) {
            key = parseInt(key.replace("addressLine_", ""));
            obj = obj.addressLine;
          }
        }

        obj[key] = value;
      });

    } else {
      this.data = {};
    }
  }

  entries() {
    return [["legalPerson", this.data]];
  }

  toJSON() {
    return {
      "legalPerson": this.data
    };
  }
}