// Ensure the accept type for all HTMX requests is HTML partials.
document.body.addEventListener('htmx:configRequest', (e) => {
  e.detail.headers['Accept'] = 'text/html'
});

// Ensure that all 500 errors redirect to the error page.
document.body.addEventListener('htmx:responseError', (e) => {
  if (e.detail.xhr.status === 500) {
    window.location.href = '/error';
  }
});

function createList(elem) {
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
      }
    });
  }

  // TODO: handle checkboxes if necessary.
  return list;
}

function createChoices(elem) {
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

  new Choices(elem, options);
}