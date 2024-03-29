const addWalletBttn = document.getElementById('add-wallet-bttn')

addWalletBttn?.addEventListener('click', () => {
  const walletDiv = document.getElementById('crypto-wallets')
  const walletCount = walletDiv.children.length + 1
  walletDiv.insertAdjacentHTML('beforeend', `
  <div class="grid gap-6 my-4 md:grid-cols-2">
    <div>
      <label for="crypto_address" class="label-style">Wallet Address ${walletCount}</label>
      <input type="text" id="crypto_address" name="crypto_address" class="input-style" />
    </div>
    <div>
      <label for="network" class="label-style">Network</label>
      <div class="flex items-center gap-x-1">
        <input type="text" id="network" name="network" class="input-style" />
        <button type="button" onclick="this.parentNode.parentNode.parentNode.remove()">
          <i class="fa-solid fa-trash"><span class="sr-only">Delete wallet</span></i>
        </button>
      </div>
    </div>
  </div>
  `)
})