import { createList, createPageSizeSelect } from '../modules/components.js';
import Filter from './filter.js';

document.addEventListener("htmx:afterSettle", function (e) {
  const logList = document.getElementById('complianceAuditLogList');
  if (logList) {
    // Initialize List.js
    const list = createList(logList);

    // Initialize Page Size Select
    const pageSizeSelect = document.getElementById('pageSizeSelect');
    createPageSizeSelect(pageSizeSelect, list);
  }

  // Initialize filters
  const filterForm = document.getElementById('filterListForm');
  if (filterForm) {
    new Filter(filterForm);
  }
});
