// src/send-message.mjs
import { getInitialTestAccountsWallets } from '@aztec/accounts/testing';
import { Contract, createPXEClient, loadContractArtifact, waitForPXE } from '@aztec/aztec.js';
import { readFileSync } from 'fs';
import WormholeJson from "../../../contracts/target/wormhole_contracts-Wormhole.json" assert { type: "json" };

const WormholeJsonContractArtifact = loadContractArtifact(WormholeJson);

const { PXE_URL = 'http://localhost:8090' } = process.env;

async function main() {
  const pxe = createPXEClient(PXE_URL);
  await waitForPXE(pxe);

  console.log(`Connected to PXE at ${PXE_URL}`);

  // Read the deployed contract address from addresses.json
  let addresses;
  try {
    addresses = JSON.parse(readFileSync('addresses.json', 'utf8'));
  } catch (error) {
    console.error("Error reading addresses.json file:", error);
    process.exit(1);
  }
  
  if (!addresses.wormhole) {
    console.error("Wormhole contract address not found in addresses.json");
    process.exit(1);
  }

  console.log("Addresses from addresses.json:", addresses);

  const [ownerWallet] = await getInitialTestAccountsWallets(pxe);

  // Connect to the already deployed contract
  const contract = await Contract.at(addresses.wormhole, WormholeJsonContractArtifact, ownerWallet);
  console.log(`Connected to Wormhole contract at ${addresses.wormhole}`);

  // The message to send
  let message = "Hello World";

  // Convert message to bytes
  let encoder = new TextEncoder();
  let messageBytes = encoder.encode(message);
  
  // Create a padded array (try different sizes - this one is 31 bytes)
  const PAYLOAD_SIZE = 31;
  let paddedBytes = new Array(PAYLOAD_SIZE).fill(0);
  
  // Copy the message bytes into the padded array
  for (let i = 0; i < messageBytes.length && i < PAYLOAD_SIZE; i++) {
    paddedBytes[i] = messageBytes[i];
  }

  let payloads = [];
  for (let i = 0; i < 8; i++) {
    payloads.push(paddedBytes);
  }
  
  console.log(`Sending message: "${messageBytes} 8 times"`);
  console.log(`Padded payload (${paddedBytes.length} bytes):`, payloads);
  
  // Send the message with nonce 100 and consistency level 2
  console.log("Sending transaction...");
  const tx = await contract.methods.publish_message(100, payloads, 1,2).send();
  
  // Wait for the transaction to be mined
  const receipt = await tx.wait();
  console.log(`Transaction sent! Hash: ${receipt.txHash}`);
  
  // Get the block number to query logs
  const blockNumber = await pxe.getBlockNumber();
  
  // Query logs for the transaction
  const logFilter = {
    fromBlock: blockNumber - 1,
    toBlock: blockNumber,
    contractAddress: addresses.token // Filter logs for our contract
  };
  
  const publicLogs = (await pxe.getPublicLogs(logFilter)).logs;
  console.log("Transaction logs:", publicLogs);
}

main().catch((err) => {
  console.error(`Error in message sending script: ${err}`);
  process.exit(1);
});