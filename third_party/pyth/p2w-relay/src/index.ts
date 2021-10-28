import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import {PythImplementation__factory} from "./ethers-contracts";

import * as http from "http";
import * as net from "net";
import fs from "fs";


import {ethers} from "ethers";

import {getSignedAttestation, parseAttestation, p2w_core, sol_addr2buf} from "@certusone/p2w-sdk";

interface AttestationState {
    pendingSeqnos: Array<number>,
}

async function readinessProbeRoutine(port: number) {
    let srv = net.createServer();

    return await srv.listen(port);
}

(async () => {

    const P2W_ATTESTATIONS_HOST = process.env.P2W_ATTESTATIONS_HOST || "p2w-attest";
    const P2W_ATTESTATIONS_PORT = Number(process.env.P2W_ATTESTATIONS_PORT || "4343");
    const P2W_SOL_ADDRESS = process.env.P2W_SOL_ADDRESS || "P2WH424242424242424242424242424242424242424";

    const READINESS_PROBE_PORT = Number(process.env.READINESS_PROBE_PORT || "2000");
    const POLL_INTERVAL_MS = 5000;

    const ETH_NODE_URL = process.env.ETH_NODE_URL || "ws://eth-devnet:8545";
    const ETH_P2W_CONTRACT = process.env.ETH_P2W_CONTRACT || "0xA94B7f0465E98609391C623d0560C5720a3f2D33";
    const ETH_MNEMONIC_FILE = process.env.ETH_MNEMONIC_FILE || "../../../ethereum/devnet_mnemonic.txt";
    const ETH_HD_WALLET_PATH = process.env.ETH_HD_WALLET_PATH || "m/44'/60'/0'/0/0";

    const GUARDIAN_RPC_HOST_PORT = process.env.GUARDIAN_RPC_HOST_PORT || "http://guardian:7071";

    let readinessProbe = null;

    let seqnoPool: Set<number> = new Set(); // Seqnos we are yet to process

    console.log(`Polling attestations endpoint every ${POLL_INTERVAL_MS / 1000} seconds`);

    const wormhole = await import("@certusone/wormhole-sdk/lib/solana/core/bridge");

    let p2w_eth;

    // Connect to ETH
    try {
	let provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
	let mnemonic: string = fs.readFileSync(ETH_MNEMONIC_FILE).toString("utf-8").trim();
	console.log(`Using ETH devnet mnemonic: ${mnemonic}`);
	let wallet = ethers.Wallet.fromMnemonic(mnemonic, ETH_HD_WALLET_PATH);
	console.log(`Using ETH wallet pubkey: ${wallet.publicKey}`);
	console.log(`Using ETH wallet privkey: ${wallet.privateKey}`);
	let signer = new ethers.Wallet(wallet.privateKey, provider);
	let balance = await signer.getBalance();
	console.log(`Account balance is ${balance}`);
	let factory = new PythImplementation__factory(signer);
	p2w_eth = factory.attach(ETH_P2W_CONTRACT);
    }
    catch(e) {
	console.log(`Error: Could not instantiate ETH contract:`, e);
	throw e;
    }

    while (true) {
	http.get({
	    hostname: P2W_ATTESTATIONS_HOST,
	    port: P2W_ATTESTATIONS_PORT,
	    path: "/",
	    agent: false
	}, (res) => {
	    if (res.statusCode != 200) {
		console.log("Could not reach attestations endpoint", res);
	    } else {
		let chunks: string[] = [];
		res.setEncoding("utf-8");

		res.on('data', (chunk) => {
		    chunks.push(chunk);
		});

		res.on('end', () => {
		    let body = chunks.join('');

		    let state: AttestationState = JSON.parse(body);

		    console.log(`Got ${state.pendingSeqnos.length} new seqnos: ${state.pendingSeqnos}`);

		    for (let seqno of state.pendingSeqnos) {
			seqnoPool.add(seqno);
		    }
		});
	    }
	}).on('error', (e) => {
	    console.error(`Got error: ${e.message}`);
	});

	console.log("Processing seqnos:", seqnoPool);
	for (let seqno of seqnoPool) {

	    let vaaResponse;
	    try {
		vaaResponse = await getSignedAttestation(
		    GUARDIAN_RPC_HOST_PORT,
		    P2W_SOL_ADDRESS,
		    seqno,
		    {
			transport: NodeHttpTransport()
		    }
		);
	    }
	    catch(e) {
		console.log(`[seqno ${seqno}] Error: Could not call getSignedAttestation:`, e);

		continue;
	    }

	    console.log(`[seqno ${seqno}] Price attestation VAA bytes:\n`, vaaResponse.vaaBytes);

	    seqnoPool.delete(seqno); // We don't care to retry beyond this point

	    let parsedVaa = wormhole.parse_vaa(vaaResponse.vaaBytes);

	    console.log(`[seqno ${seqno}] Parsed VAA:\n`, parsedVaa);

	    let parsedAttestation = await parseAttestation(parsedVaa.payload);

	    console.log(`[seqno ${seqno}] Parsed price attestation:\n`, parsedAttestation);

	    try {
		let tx = await p2w_eth.attestPrice(vaaResponse.vaaBytes, {gasLimit: 1000000});
		let retval = await tx.wait();
		console.log(`[seqno ${seqno}] attestPrice() output:\n`, retval);
	    } catch(e) {
		console.log(`[seqno ${seqno}] Error: Could not call attestPrice() on ETH:`, e);

		continue;
	    }

	    let product_id = parsedAttestation.product_id;
	    let price_type = parsedAttestation.price_type == "Price" ? 1 : 0;
	    let latest_attestation;
	    try {
		let p2w = await p2w_core();
		let emitter_sol = sol_addr2buf(p2w.get_emitter_address(P2W_SOL_ADDRESS));
		let emitter_eth = await p2w_eth.pyth2WormholeEmitter();

		console.log(`Looking up latestAttestation for `, product_id, price_type);

		latest_attestation = await p2w_eth.latestAttestation(product_id, price_type);
	    } catch(e) {
		console.log(`[seqno ${seqno}] Error: Could not call latestAttestation() on ETH:`, e);
		continue;
	    }

	    console.log(`[seqno ${seqno}] Latest price type ${price_type} attestation of ${product_id} is ${latest_attestation}`);
	    if (!readinessProbe) {
		console.log(`[seqno ${seqno}] Attestation successful. Starting readiness probe.`);
		readinessProbe = readinessProbeRoutine(READINESS_PROBE_PORT);
	    }
	}

	await new Promise(f => {setTimeout(f, POLL_INTERVAL_MS);});
    }

})();
