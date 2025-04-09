// src/deploy.mjs
import { getInitialTestAccountsWallets } from '@aztec/accounts/testing';
import { Contract, createPXEClient, loadContractArtifact, waitForPXE } from '@aztec/aztec.js';
import { ExtendedPublicLog } from '@aztec/stdlib/logs';
import WormholeJson from "../../../contracts/target/aztec-Wormhole.json" assert { type: "json" };

import { writeFileSync } from 'fs';

const WormholeJsonContractArtifact = loadContractArtifact(WormholeJson);

const { PXE_URL = 'http://localhost:8080' } = process.env;

async function main() {
  const pxe = createPXEClient(PXE_URL);
  await waitForPXE(pxe);

  const [ownerWallet] = await getInitialTestAccountsWallets(pxe);
  const ownerAddress = ownerWallet.getAddress();

  const token = await Contract.deploy(ownerWallet, WormholeJsonContractArtifact, [ownerAddress, '0x254cd5788032e1cab39f51d63adbac4bf73e97c9b309b692ffb568903be9998a', '0x254cd5788032e1cab39f51d63adbac4bf73e97c9b309b692ffb568903be9998a', 18])
    .send()
    .deployed();

  console.log(`Token deployed at ${token.address.toString()}`);

  const addresses = { token: token.address.toString() };
  writeFileSync('addresses.json', JSON.stringify(addresses, null, 2));

  const contract = await Contract.at(token.address, WormholeJsonContractArtifact, ownerWallet);

  // The message to convert
  let message = "Hello I am stavros vlach";

  // Using TextEncoder (modern approach)
  let encoder = new TextEncoder();
  let bytes = encoder.encode(message);

  const _tx = await contract.methods.publishMessage(100,bytes, 2).send().wait();

  const sampleLogFilter = {
    txHash: '0x100ebe8cfa848587397b272a40426223004c5ee3838d22652c33e10c7fe7d1f7',
    fromBlock: 160,
    toBlock: 190,
    contractAddress: '0x081a143b80470311c64f8fd1b67a074e2aa312bf5e22e6ebe0b17c5b3b44470b'
  };

  console.log(_tx);

  const logs = await pxe.getPublicLogs(sampleLogFilter);

  console.log(logs.logs[0]);

const fromBlock = await pxe.getBlockNumber();
const logFilter = {
  fromBlock,
  toBlock: fromBlock + 1,
};
const publicLogs = (await pxe.getPublicLogs(logFilter)).logs;

console.log(publicLogs);

}

main().catch((err) => {
  console.error(`Error in deployment script: ${err}`);
  process.exit(1);
});