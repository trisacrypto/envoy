import { createChoices } from '../modules/components.js';

class Filters {
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

    // Add event listeners to the form
    // NOTE: using the form reset event was causing the choices to empty out??
    this.form.addEventListener('submit', this.onSubmit.bind(this));
    this.form.addEventListener('reset', this.onReset.bind(this));

    // Initialize the status and asset choices
    this.status = createChoices(this.form.querySelector('[name="status"]'));
    this.assets = createChoices(this.form.querySelector('[name="asset"]'));
  }

  onSubmit(e) {
    e.preventDefault();
    const status = [];
    const assets = [];

    this.updateFilterBadge(3);
    return false;
  }

  onReset(e) {
    this.updateFilterBadge(0);
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