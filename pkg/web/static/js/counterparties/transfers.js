import { createList } from "../modules/components.js";

document.addEventListener("htmx:afterSettle", function(e) {
  // Initialize the transfers list.
  const table = document.getElementById("counterpartyTransfersList");
  if (table) createList(table);

  // Initialize the status tooltips.
  const tooltips = document.querySelectorAll("[data-bs-toggle='tooltip']");
  tooltips.forEach(tooltip => {
    new Tooltip(tooltip);
  });
});