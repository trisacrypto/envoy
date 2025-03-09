import { createChoices } from '../modules/components.js';
import { urlPath, urlQuery } from '../htmx/helpers.js';

class Filters {
  listID = "transactions";

  constructor(elem) {
    switch (typeof elem) {
      case 'string':
        this.form = document.querySelector(elem);
        break;
      case 'object':
        this.form = elem;
        break;
      default:
        throw new Error('Invalid element type');
    }

    // Grab the required elements
    this.list = document.getElementById(this.listID);

    // Add event listeners to the form
    // NOTE: using the form reset event was causing the choices to empty out??
    this.form.addEventListener('submit', this.onSubmit.bind(this));
    this.form.addEventListener('reset', this.onReset.bind(this));

    // Initialize the status and asset choices
    this.status = createChoices(this.form.querySelector('[name="status"]'));
    this.assets = createChoices(this.form.querySelector('[name="asset"]'));

    // Initialize the filters as set by the hx-get attribute
    let nFilters = 0;
    const query = urlQuery(this.list.getAttribute('hx-get'));
    query.getAll('status').forEach(status => {
      this.status.setChoiceByValue(status);
      nFilters++;
    });
    query.getAll('asset').forEach(status => {
      this.assets.setChoiceByValue(status);
      nFilters++;
    });

    this.updateFilterBadge(nFilters);
  }

  onSubmit(e) {
    e.preventDefault();
    const formData = new FormData(this.form);
    const status = formData.getAll("status");
    const asset = formData.getAll("asset");

    this.updateFilterBadge(status.length + asset.length);
    this.filterList(status, asset);
    return false;
  }

  onReset(e) {
    this.updateFilterBadge(0);
    this.filterList(null, null);
  }

  filterList(status, asset) {
    const url = this.list.getAttribute('hx-get');
    const path = urlPath(url);
    const query = urlQuery(url);

    // Remove existing filters
    query.delete('status');
    query.delete('asset');

    // Add the specified filters
    if (status) {
      status.forEach(status => query.append('status', status));
    }

    if (asset) {
      asset.forEach(asset => query.append('asset', asset));
    }

    this.list.setAttribute('hx-get', `${path}?${query.toString()}`);
    htmx.process(this.list);

    this.list.dispatchEvent(new CustomEvent('list-filter', {detail: {status: status, asset: asset}, bubbles: true, cancelable: true}));
  }

  updateFilterBadge(count) {
    const badge = document.getElementById('numFiltersBadge');
    if (badge) {
      if (count === 0) {
        badge.classList.add('bg-secondary');
        badge.classList.remove('bg-primary');
      } else {
        badge.classList.remove('bg-secondary');
        badge.classList.add('bg-primary');
      }

      badge.textContent = count;
    }
  }
}

export default Filters;