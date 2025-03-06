import { createList, createPageSizeSelect } from '../modules/components.js';

document.addEventListener("htmx:afterSettle", function(e) {
  // Initialize List.js
  const cpList = document.getElementById('accountList');
  const list = createList(cpList);

  // Initialize Page Size Select
  const pageSizeSelect = document.getElementById('pageSizeSelect');
  createPageSizeSelect(pageSizeSelect, list);
});