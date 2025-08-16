import { createChoices } from '../modules/components.js';
import { urlPath, urlQuery } from '../htmx/helpers.js';

class Filters {
  listID = "auditlogs";

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

    // Initialize the actor types and resource types choices
    this.actorTypes = createChoices(this.form.querySelector('[name="actorTypes"]'));
    this.resourceTypes = createChoices(this.form.querySelector('[name="resourceTypes"]'));

    // Initialize the filters as set by the hx-get attribute
    let nFilters = 0;
    const query = urlQuery(this.list.getAttribute('hx-get'));
    query.getAll('actor_types').forEach(actorT => {
      this.actorTypes.setChoiceByValue(actorT);
      nFilters++;
    });
    query.getAll('resource_types').forEach(resourceT => {
      this.resourceTypes.setChoiceByValue(resourceT);
      nFilters++;
    });

    this.updateFilterBadge(nFilters);
  }

  onSubmit(e) {
    e.preventDefault();
    const formData = new FormData(this.form);
    const actorTypes = formData.getAll("actorTypes");
    const resourceTypes = formData.getAll("resourceTypes");

    this.updateFilterBadge(actorTypes.length + resourceTypes.length);
    this.filterList(actorTypes, resourceTypes);
    return false;
  }

  onReset(e) {
    this.updateFilterBadge(0);
    this.filterList(null, null);
  }

  filterList(actorTypes, resourceTypes) {
    const url = this.list.getAttribute('hx-get');
    const path = urlPath(url);
    const query = urlQuery(url);

    // Remove existing filters
    query.delete('actor_types');
    query.delete('resource_types');

    // Add the specified filters
    if (actorTypes) {
      actorTypes.forEach(actorT => query.append('actor_types', actorT));
    }

    if (resourceTypes) {
      resourceTypes.forEach(resourceT => query.append('resource_types', resourceT));
    }

    this.list.setAttribute('hx-get', `${path}?${query.toString()}`);
    htmx.process(this.list);

    this.list.dispatchEvent(new CustomEvent('list-filter', { detail: { actorTypes: actorTypes, resourceTypes: resourceTypes }, bubbles: true, cancelable: true }));
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
