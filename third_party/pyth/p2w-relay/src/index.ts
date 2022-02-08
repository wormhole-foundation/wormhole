import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import {PythImplementation__factory} from "./ethers-contracts";

import * as http from "http";
import * as net from "net";
import fs from "fs";


import {ethers} from "ethers";

import {getSignedAttestation, parseBatchAttestation, p2w_core, sol_addr2buf} from "@certusone/p2w-sdk";

import {setDefaultWasm, importCoreWasm} from "@certusone/wormhole-sdk/lib/cjs/solana/wasm";

interface NewAttestationsResponse {
    pendingSeqnos: Array<number>,
}


async function readinessProbeRoutine(port: number) {
    let srv = net.createServer();

    return await srv.listen(port);
}

(async () => {

    // p2w-attest exposes an HTTP endpoint that shares the currently pending sequence numbers
    const P2W_ATTESTATIONS_HOST = process.env.P2W_ATTESTATIONS_HOST || "p2w-attest";
    const P2W_ATTESTATIONS_PORT = Number(process.env.P2W_ATTESTATIONS_PORT || "4343");
    const P2W_ATTESTATIONS_POLL_INTERVAL_MS = Number(process.env.P2W_ATTESTATIONS_POLL_INTERVAL_MS || "5000");

    const P2W_SOL_ADDRESS = process.env.P2W_SOL_ADDRESS || "P2WH424242424242424242424242424242424242424";

    const READINESS_PROBE_PORT = Number(process.env.READINESS_PROBE_PORT || "2000");

    const P2W_RELAY_RETRY_COUNT = Number(process.env.P2W_RELAY_RETRY_COUNT || "3");

    // ETH node connection details; Currently, we expect to read BIP44
    // wallet recovery mnemonics from a text file.
    const ETH_NODE_URL = process.env.ETH_NODE_URL || "ws://eth-devnet:8545";
    const ETH_P2W_CONTRACT = process.env.ETH_P2W_CONTRACT || "0xA94B7f0465E98609391C623d0560C5720a3f2D33";
    const ETH_MNEMONIC_FILE = process.env.ETH_MNEMONIC_FILE || "../../../ethereum/devnet_mnemonic.txt";
    const ETH_HD_WALLET_PATH = process.env.ETH_HD_WALLET_PATH || "m/44'/60'/0'/0/0";

    // Public RPC address for use with signed attestation queries
    const GUARDIAN_RPC_HOST_PORT = process.env.GUARDIAN_RPC_HOST_PORT || "http://guardian:7071";

    let readinessProbe = null;

    let seqnoPool: Map<number, number> = new Map();

    console.log(`Polling attestations endpoint every ${P2W_ATTESTATIONS_POLL_INTERVAL_MS / 1000} seconds`);

    setDefaultWasm("node");
    const {parse_vaa} = await importCoreWasm();

    let p2w_eth: any;

    // Connect to ETH
    try {
	let provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
	let mnemonic: string = fs.readFileSync(ETH_MNEMONIC_FILE).toString("utf-8").trim();
	let wallet = ethers.Wallet.fromMnemonic(mnemonic, ETH_HD_WALLET_PATH);
	console.log(`Using ETH wallet pubkey: ${wallet.publicKey}`);
	let signer = new ethers.Wallet(wallet.privateKey, provider);
	let balance = await signer.getBalance();
	console.log(`Account balance is ${balance}`);
	let factory = new PythImplementation__factory(signer);
	p2w_eth = factory.attach(ETH_P2W_CONTRACT);
    }
    catch(e) {
	console.error(`Error: Could not instantiate ETH contract:`, e);
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
		console.error("Could not reach attestations endpoint", res);
	    } else {
		let chunks: string[] = [];
		res.setEncoding("utf-8");

		res.on('data', (chunk) => {
		    chunks.push(chunk);
		});

		res.on('end', () => {
		    let body = chunks.join('');

		    let response: NewAttestationsResponse = JSON.parse(body);

		    console.log(`Got ${response.pendingSeqnos.length} new seqnos: ${response.pendingSeqnos}`);

		    for (let seqno of response.pendingSeqnos) {
			seqnoPool.set(seqno, 0);
		    }
		});
	    }
	}).on('error', (e) => {
	    console.error(`Got error: ${e.message}`);
	});

	console.log("Processing seqnos:", seqnoPool);
	for (let poolEntry of seqnoPool) {

	    let seqno = poolEntry[0];
	    let attempts = poolEntry[1];

	    if (attempts >= P2W_RELAY_RETRY_COUNT) {
		console.warn(`[seqno ${poolEntry}] Exceeded retry count, removing from list`);
		seqnoPool.delete(seqno);
		continue;
	    }

	    let vaaResponse: any;
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
		console.error(`[seqno ${poolEntry}] Error: Could not call getSignedAttestation:`, e);

		seqnoPool.set(seqno, attempts + 1);

		continue;
	    }

	    console.log(`[seqno ${poolEntry}] Price attestation VAA bytes:\n`, vaaResponse.vaaBytes);

	    let parsedVaa = parse_vaa(vaaResponse.vaaBytes);

	    console.log(`[seqno ${poolEntry}] Parsed VAA:\n`, parsedVaa);

	    let parsedAttestations = await parseBatchAttestation(parsedVaa.payload);

	    console.log(`[seqno ${poolEntry}] Parsed ${parsedAttestations.length} price attestations:\n`, parsedAttestations);

	    // try {
	    // 	let tx = await p2w_eth.attestPrice(vaaResponse.vaaBytes, {gasLimit: 1000000});
	    // 	let retval = await tx.wait();
	    // 	console.log(`[seqno ${poolEntry}] attestPrice() output:\n`, retval);
	    // } catch(e) {
	    // 	console.error(`[seqno ${poolEntry}, {parsedAttestations.length} symbols] Error: Could not call attestPrice() on ETH:`, e);

	    // 	seqnoPool.set(seqno, attempts + 1);

	    // 	continue;
	    // }

	    console.warn("TODO: implement relayer ETH call");

	    // for (let att of parsedAttestations) {

	    // 	let product_id = att.product_id;
	    // 	let price_type = att.price_type == "Price" ? 1 : 0;
	    // 	let latest_attestation: any;
	    // 	try {
	    // 	    let p2w = await p2w_core();

	    // 	    console.log(`Looking up latestAttestation for `, product_id, price_type);

	    // 	    latest_attestation = await p2w_eth.latestAttestation(product_id, price_type);
	    // 	} catch(e) {
	    // 	    console.error(`[seqno ${poolEntry}] Error: Could not call latestAttestation() on ETH:`, e);

	    // 	    seqnoPool.set(seqno, attempts + 1);

	    // 	    continue;
	    // 	}

	    // 	console.log(`[seqno ${poolEntry}] Latest price type ${price_type} attestation of ${product_id} is ${latest_attestation}`);
	    // }

	    if (!readinessProbe) {
		console.log(`[seqno ${poolEntry}] Attestation successful. Starting readiness probe.`);
		readinessProbe = readinessProbeRoutine(READINESS_PROBE_PORT);
	    }

	    seqnoPool.delete(seqno); // Everything went well, seqno no longer pending.
	}

	await new Promise(f => {setTimeout(f, P2W_ATTESTATIONS_POLL_INTERVAL_MS);});
    }

})();
