import { getInitialTestAccountsWallets } from '@aztec/accounts/testing';
import { createPXEClient, waitForPXE } from '@aztec/aztec.js';
import { TokenContract } from '@aztec/noir-contracts.js/Token'; 

import { writeFileSync } from 'fs';

const { PXE_URL = 'http://localhost:8090' } = process.env;

// Call `aztec-nargo compile` to compile the contract
// Call `aztec codegen ./src -o src/artifacts/` to generate the contract artifacts

// Run first ``` aztec start --sandbox ```
// then run this script with ``` node deploy.mjs ```


// Following: https://docs.aztec.network/developers/tutorials/codealong/js_tutorials/aztecjs-getting-started#set-up-the-project
async function deployToken(
  adminWallet,
  initialAdminBalance,
) {
  const contract = await TokenContract.deploy(
    adminWallet,
    adminWallet.getAddress(),
    "ProverToken",
    "PTZK",
    18
  )
    .send()
    .deployed();

  if (initialAdminBalance > 0n) {
    // Minter is minting to herself so contract as minter is the same as contract as recipient
    await mintTokensToPublic(
      contract,
      adminWallet,
      adminWallet.getAddress(),
      initialAdminBalance
    );
  }

  return contract;
}

async function mintTokensToPublic(
  token, // TokenContract
  minterWallet, 
  recipient,
  amount
) {
  const tokenAsMinter = await TokenContract.at(token.address, minterWallet);
  await tokenAsMinter.methods
    .mint_to_public(recipient, amount)
    .send()
    .wait();
}

async function mintTokensToPrivate(
  token, // TokenContract
  minterWallet, 
  recipient,
  amount
) {
  const tokenAsMinter = await TokenContract.at(token.address, minterWallet);
  await tokenAsMinter.methods
    .mint_to_private(minterWallet.getAddress(), recipient, amount)
    .send()
    .wait();
}


async function main() {
  const pxe = createPXEClient(PXE_URL);
  await waitForPXE(pxe);

  console.log(`Connected to PXE at ${PXE_URL}`);

  const [ownerWallet, receiverWallet] = await getInitialTestAccountsWallets(pxe);
  const ownerAddress = ownerWallet.getAddress();

  console.log(`Owner address: ${ownerAddress}`);
  console.log(`Receiver address: ${receiverWallet.getAddress()}`);

  let token = await deployToken(ownerWallet, 5000n);
  console.log(`Deployed token contract at ${token.address}`)
  
  const address = { token_address: token.address.toString() };
  writeFileSync('token_address.json', JSON.stringify(address, null, 2));
}

main().catch((err) => {
  console.error(`Error in deployment script: ${err}`);
  process.exit(1);
});