import { REJECT_CODES } from "./constants.js"

document.body.addEventListener('htmx:afterSettle', (e) => {
  const idEl = document.getElementById('envelope-id')
  const id = idEl?.value
  if (e.detail.requestConfig.path === `/v1/transactions/${id}/repair` && e.detail.requestConfig.verb === 'get') {
  const errorCode = document.getElementById('repair-error')
    const errorMsg = errorCode.textContent.trim()
    const readableError = REJECT_CODES[errorMsg]
    errorCode.textContent = readableError
  }
})