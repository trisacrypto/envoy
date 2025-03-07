// Initialize a List.js component on the specified element using the DashKit default
// options. Used after an HTMX request settles to ensure the component is loaded.
// The list is returned so that it can be used for programmatic interaction.
export function createList(elem) {
  const listAlert = elem.querySelector('.list-alert');
  const listAlertCount = elem.querySelector('.list-alert-count');
  const listAlertClose = elem.querySelector('.list-alert .btn-close');
  const listCheckboxes = elem.querySelectorAll('.list-checkbox');
  const listCheckboxAll = elem.querySelector('.list-checkbox-all');
  const listPagination = elem.querySelectorAll('.list-pagination');
  const listPaginationPrev = elem.querySelector('.list-pagination-prev');
  const listPaginationNext = elem.querySelector('.list-pagination-next');
  const listOptions = elem.dataset.list && JSON.parse(elem.dataset.list);

  const defaultOptions = {
    listClass: 'list',
    searchClass: 'list-search',
    sortClass: 'list-sort',
  };

  // Merge options
  const options = Object.assign(defaultOptions, listOptions);

  // Initialize the list object
  const list = new List(elem, options);

  // Pagination
  if (listPagination) {
    [].forEach.call(listPagination, function (pagination) {
      pagination.addEventListener('click', function (e) {
        e.preventDefault();
      });
    });
  }

  // Pagination (next)
  if (listPaginationNext) {
    listPaginationNext.addEventListener('click', function (e) {
      e.preventDefault();

      const nextItem = parseInt(list.i) + parseInt(list.page);

      if (nextItem <= list.size()) {
        list.show(nextItem, list.page);
        list.update()
      }
    });
  }

  // Pagination (prev)
  if (listPaginationPrev) {
    listPaginationPrev.addEventListener('click', function (e) {
      e.preventDefault();

      const prevItem = parseInt(list.i) - parseInt(list.page);

      if (prevItem > 0) {
        list.show(prevItem, list.page);
        list.update()
      }
    });
  }

  // TODO: handle checkboxes if necessary.
  return list;
}

// Initialize a Choices.js component on the specified element using the DashKit default
// options. Used after an HTMX request settles to ensure the component is loaded.
export function createChoices(elem) {
  const elementOptions = elem.dataset.choices ? JSON.parse(elem.dataset.choices) : {};
  const defaultOptions = {
    classNames: {
      containerInner: elem.className,
      input: 'form-control',
      inputCloned: 'form-control-sm',
      listDropdown: 'dropdown-menu',
      itemChoice: 'dropdown-item',
      activeState: 'show',
      selectedState: 'active',
    },
    allowHTML: false,
    shouldSort: false,
    callbackOnCreateTemplates: function (template) {
      return {
        choice: ({ classNames }, data) => {
          const classes = `${classNames.item} ${classNames.itemChoice} ${data.disabled ? classNames.itemDisabled : classNames.itemSelectable}`;
          const disabled = data.disabled ? 'data-choice-disabled aria-disabled="true"' : 'data-choice-selectable';
          const role = data.groupId > 0 ? 'role="treeitem"' : 'role="option"';
          const selectText = this.config.itemSelectText;

          const label =
            data.customProperties && data.customProperties.avatarSrc
              ? `
            <div class="avatar avatar-xs me-3">
              <img class="avatar-img rounded-circle" src="${data.customProperties.avatarSrc}" alt="${data.label}" >
            </div> ${data.label}
          `
              : data.label;

          return template(`
            <div class="${classes}" data-select-text="${selectText}" data-choice ${disabled} data-id="${data.id}" data-value="${data.value}" ${role}>
              ${label}
            </div>
          `);
        },
      };
    },
  };

  const options = {
    ...elementOptions,
    ...defaultOptions,
  };

  return new Choices(elem, options);
}

// Initializes a page size select element for a List.js component.
export function createPageSizeSelect(elem, list) {
  // Initialize Choices.
  createChoices(elem);

  // Change the page when page size is selected.
  elem.addEventListener('change', function(e) {
    list.page = parseInt(e.target.value);
    list.show(1, list.page);
    list.update()
  });
}

/*
This function activates copy and paste buttons that are on text inputs.
*/
export function activateCopyButtons() {
  const btns = document.querySelectorAll('[data-clipboard-target]');
  btns.forEach(btn => {
    btn.addEventListener('click', function() {
      const target = btn.dataset.clipboardTarget;
      const value = document.querySelector(target).value;
      navigator.clipboard.writeText(value)
        .then(() => {
          btn.classList.add('btn-success');
          btn.classList.remove('btn-outline-secondary');
          btn.innerHTML = '<i class="fe fe-clipboard"></i>';
        })
        .catch(() => {
          btn.classList.add('btn-danger');
          btn.classList.remove('btn-outline-secondary');
          btn.innerHTML = '<i class="fe fe-x-octagon"></i>';
        })
        .finally(() => {
          setTimeout(() => {
            btn.classList.remove('btn-success', 'btn-danger');
            btn.classList.add('btn-outline-secondary');
            btn.innerHTML = '<i class="fe fe-copy"></i>';
          }, 500);
        });
    });
  });
}