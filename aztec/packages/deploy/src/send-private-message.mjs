// src/send-message.mjs
import { getInitialTestAccountsWallets } from '@aztec/accounts/testing';
import { Contract, createPXEClient, loadContractArtifact, waitForPXE } from '@aztec/aztec.js';
import { readFileSync, writeFileSync } from 'fs';
import WormholeJson from "../../../contracts/target/wormhole_contracts-Wormhole.json" assert { type: "json" };
import { TokenContract } from '@aztec/noir-contracts.js/Token'; 

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
    
    if (!addresses.wormhole ||  !addresses.token) {
        console.error("Wormhole or token contract address not found in addresses.json");
        process.exit(1);
    }

    console.log("Addresses from addresses.json:", addresses);

    const [ownerWallet, receiverWallet] = await getInitialTestAccountsWallets(pxe);
    const ownerAddress = ownerWallet.getAddress();

    // Connect to the already deployed contract
    const contract = await Contract.at(addresses.wormhole, WormholeJsonContractArtifact, ownerWallet);
    console.log(`Connected to Wormhole contract at ${addresses.wormhole}`);
    
    const token = await TokenContract.at(addresses.token, ownerWallet);
    console.log(`Connected to Token contract at ${addresses.token}`);
    
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
    
    console.log("Sending transaction...");

    const msg_fee = 3n;
    // get nonce and increment it
    const nonce_file_data = JSON.parse(readFileSync('nonce.json', 'utf8'));

    // Safe BigInt handling
    const current_nonce = nonce_file_data.token_nonce
        ? BigInt(nonce_file_data.token_nonce)
        : 0n;

    const token_nonce = current_nonce + 1n;

    const new_nonce_data = { token_nonce: token_nonce.toString() };
    
    writeFileSync('nonce.json', JSON.stringify(new_nonce_data, null, 2));  
    console.log(`Using token nonce: ${token_nonce}`);

    console.log(`Taking payment in private...`);
    const privateAction = token.methods.transfer_in_private(
        ownerAddress,
        receiverWallet.getAddress(),
        msg_fee,
        token_nonce
    );
    console.log(`${ownerAddress} is transferring ${msg_fee} tokens to ${receiverWallet.getAddress()} in private`);

    const initWitness = await ownerWallet.createAuthWit({ 
        caller: contract.address, 
        action: privateAction 
    });

    console.log(`Generated private authwit`);

    const tx = contract.methods.publish_message_in_private(100, payloads, msg_fee, 2, ownerAddress, token_nonce).send({ authWitnesses: [initWitness] });
    
    // Wait for the transaction to be mined
    const receipt = await tx.wait();
    console.log(`Transaction sent! Hash: ${receipt.txHash}`);
    
    const sampleLogFilter = {
        fromBlock: 0,
        toBlock: 190,
        contractAddress: '0x081a143b80470311c64f8fd1b67a074e2aa312bf5e22e6ebe0b17c5b3b44470b'
    };

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
  console.error(`Error in message sending script: ${err}`);
  process.exit(1);
});