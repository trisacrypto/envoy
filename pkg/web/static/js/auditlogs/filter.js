import { createChoices, createFlatpickr } from '../modules/components.js';
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

    // Grab the list element
    this.list = document.getElementById(this.listID);

    // Add event listeners to the form
    // NOTE: using the form reset event was causing the choices to empty out??
    this.form.addEventListener('submit', this.onSubmit.bind(this));
    this.form.addEventListener('reset', this.onReset.bind(this));

    // Initialize the actor types and resource types choices
    this.actorTypes = createChoices(this.form.querySelector('[name="actorTypes"]'));
    this.resourceTypes = createChoices(this.form.querySelector('[name="resourceTypes"]'));

    // Initialize the before and after datetime pickers
    this.beforePicker = this.form.querySelector('[name="before"]');
    createFlatpickr(this.beforePicker);

    this.afterPicker = this.form.querySelector('[name="after"]');
    createFlatpickr(this.afterPicker);

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

    const before = query.get('before')
    if (before) {
      this.beforePicker._flatpickr.setDate(before)
      nFilters++;
    }

    const after = query.get('after')
    if (after) {
      this.afterPicker._flatpickr.setDate(after)
      nFilters++;
    }

    const actorId = query.get('actor_id')
    if (actorId) {
      this.form.querySelector('[name="actorId"]').value = actorId;
      nFilters++;
    }

    const resourceId = query.get('resource_id')
    if (resourceId) {
      this.form.querySelector('[name="resourceId"]').value = resourceId;
      nFilters++;
    }

    this.updateFilterBadge(nFilters);
  }

  onSubmit(e) {
    e.preventDefault();
    const formData = new FormData(this.form);
    const actorTypes = formData.getAll("actorTypes");
    const resourceTypes = formData.getAll("resourceTypes");
    const before = formData.get("before")
    const after = formData.get("after")
    const actorId = formData.get("actorId")
    const resourceId = formData.get("resourceId")

    this.updateFilterBadge(actorTypes.length + resourceTypes.length + (before ? 1 : 0) + (after ? 1 : 0) + (actorId ? 1 : 0) + (resourceId ? 1 : 0));
    this.filterList(actorTypes, resourceTypes, before, after, actorId, resourceId);
    return false;
  }

  onReset(e) {
    this.updateFilterBadge(0);
    this.filterList(null, null);
  }

  filterList(actorTypes, resourceTypes, before, after, actorId, resourceId) {
    const url = this.list.getAttribute('hx-get');
    const path = urlPath(url);
    const query = urlQuery(url);

    // Remove existing filters
    query.delete('actor_types');
    query.delete('resource_types');
    query.delete('before');
    query.delete('after');
    query.delete('actor_id');
    query.delete('resource_id');

    // Add the specified filters
    if (!actorId && actorTypes) {
      actorTypes.forEach(actorT => query.append('actor_types', actorT));
    }

    if (!resourceId && resourceTypes) {
      resourceTypes.forEach(resourceT => query.append('resource_types', resourceT));
    }

    if (before) {
      query.append('before', before);
    }

    if (after) {
      query.append('after', after);
    }

    if (actorId) {
      query.append('actor_id', actorId);
    }

    if (resourceId) {
      query.append('resource_id', resourceId);
    }

    this.list.setAttribute('hx-get', `${path}?${query.toString()}`);
    htmx.process(this.list);

    this.list.dispatchEvent(new CustomEvent('list-filter', {
      detail: {
        actorTypes: actorTypes,
        resourceTypes: resourceTypes,
        before: before,
        after: after
      },
      bubbles: true,
      cancelable: true
    }));
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
