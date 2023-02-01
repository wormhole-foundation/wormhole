import "dotenv/config";
import {
  approveEth,
  attestFromEth,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CONTRACTS,
  createWrappedOnEth,
  getEmitterAddressEth,
  getForeignAssetEth,
  getSignedVAAWithRetry,
  hexToUint8Array,
  parseSequenceFromLogEth,
  redeemOnEth,
  serialiseVAA,
  sign,
  TokenBridgeTransfer,
  transferFromEth,
  tryNativeToUint8Array,
  uint8ArrayToHex,
  VAA,
} from "@certusone/wormhole-sdk";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { ethers } from "ethers";
import * as devnetConsts from "./devnet-consts.json";
import { parseUnits } from "ethers/lib/utils";

if (process.env.INIT_SIGNERS_KEYS_CSV === "undefined") {
  let msg = `.env is missing. run "make contracts-tools-deps" to fetch.`;
  console.error(msg);
  throw msg;
}

// TODO: consider using jest

/*
 * Goals:
 *   1. Ensure a token can be sent from its origin chain
 *   2. Ensure a token can be sent back from a foreign chain
 *   3. Ensure spoofed tokens for more than the outstanding amount rejects successfully
 *   4. Validate the guardian metrics for each of these cases
 *   5. Bonus: Validate the on chain contract state via queries
 */

const ci = !!process.env.CI;

const GUARDIAN_HOST = ci ? "guardian" : "localhost";
const GUARDIAN_RPCS = [`http://${GUARDIAN_HOST}:7071`];
const GUARDIAN_METRICS = `http://${GUARDIAN_HOST}:6060/metrics`;
const ETH_NODE_URL = ci ? "ws://eth-devnet:8545" : "ws://localhost:8545";
const BSC_NODE_URL = ci ? "ws://eth-devnet2:8545" : "ws://localhost:8546";
const ETH_PRIVATE_KEY9 =
  "0xb0057716d5917badaf911b193b12b910811c1497b5bada8d7711f758981c3773";
const ETH_GA_TEST_TOKEN =
  devnetConsts.chains[CHAIN_ID_ETH].addresses.testGA.address;
const DECIMALS = devnetConsts.chains[CHAIN_ID_ETH].addresses.testGA.decimals;
const VAA_SIGNERS = process.env.INIT_SIGNERS_KEYS_CSV.split(",");
const GOVERNANCE_CHAIN = Number(devnetConsts.global.governanceChainId);
const GOVERNANCE_EMITTER = devnetConsts.global.governanceEmitterAddress;

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// Guardian metrics are prometheus data
const fetchGlobalAccountantMetrics = async (): Promise<{
  global_accountant_connection_errors_total: number;
  global_accountant_error_events_received: number;
  global_accountant_events_received: number;
  global_accountant_submit_failures: number;
  global_accountant_total_balance_errors: number;
  global_accountant_total_digest_mismatches: number;
  global_accountant_transfer_vaas_outstanding: number;
  global_accountant_transfer_vaas_submitted: number;
  global_accountant_transfer_vaas_submitted_and_approved: number;
}> =>
  (await (await fetch(GUARDIAN_METRICS)).text())
    .split("\n")
    .filter((m) => m.startsWith("global_accountant"))
    .reduce((p, m) => {
      const [k, v] = m.split(" ");
      p[k] = Number(v);
      return p;
    }, {} as any);

(async () => {
  //
  // PREAMBLE
  //

  // create a signer for Eth
  const ethProvider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
  const ethSigner = new ethers.Wallet(ETH_PRIVATE_KEY9, ethProvider);
  // create a signer for BSC
  const bscProvider = new ethers.providers.WebSocketProvider(BSC_NODE_URL);
  const bscSigner = new ethers.Wallet(ETH_PRIVATE_KEY9, bscProvider);

  let attestedAddress = "";

  //
  // STEP 0 - attest the token
  //
  {
    attestedAddress = await getForeignAssetEth(
      CONTRACTS.DEVNET.bsc.token_bridge,
      bscProvider,
      CHAIN_ID_ETH,
      tryNativeToUint8Array(ETH_GA_TEST_TOKEN, CHAIN_ID_ETH)
    );
    if (attestedAddress && attestedAddress !== ethers.constants.AddressZero) {
      console.log("already attested");
    } else {
      console.log("attesting...");
      // attest the test token
      const receipt = await attestFromEth(
        CONTRACTS.DEVNET.ethereum.token_bridge,
        ethSigner,
        ETH_GA_TEST_TOKEN
      );
      // get the sequence from the logs (needed to fetch the vaa)
      const sequence = parseSequenceFromLogEth(
        receipt,
        CONTRACTS.DEVNET.ethereum.core
      );
      const emitterAddress = getEmitterAddressEth(
        CONTRACTS.DEVNET.ethereum.token_bridge
      );
      console.log(`fetching vaa ${sequence}...`);
      // poll until the guardian(s) witness and sign the vaa
      const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
        GUARDIAN_RPCS,
        CHAIN_ID_ETH,
        emitterAddress,
        sequence,
        {
          transport: NodeHttpTransport(),
        }
      );
      console.log("creating...");
      await createWrappedOnEth(
        CONTRACTS.DEVNET.bsc.token_bridge,
        bscSigner,
        signedVAA
      );
      attestedAddress = await getForeignAssetEth(
        CONTRACTS.DEVNET.bsc.token_bridge,
        bscProvider,
        CHAIN_ID_ETH,
        tryNativeToUint8Array(ETH_GA_TEST_TOKEN, CHAIN_ID_ETH)
      );
    }
  }

  //
  // STEP 1 - send the token out
  //
  {
    const beforeMetrics = await fetchGlobalAccountantMetrics();
    const amount = parseUnits("1", DECIMALS);
    // approve the bridge to spend tokens
    console.log("approving...");
    await approveEth(
      CONTRACTS.DEVNET.ethereum.token_bridge,
      ETH_GA_TEST_TOKEN,
      ethSigner,
      amount
    );
    // transfer tokens out
    console.log("transferring...");
    const receipt = await transferFromEth(
      CONTRACTS.DEVNET.ethereum.token_bridge,
      ethSigner,
      ETH_GA_TEST_TOKEN,
      amount,
      CHAIN_ID_BSC,
      tryNativeToUint8Array(await bscSigner.getAddress(), CHAIN_ID_BSC)
    );
    // get the sequence from the logs (needed to fetch the vaa)
    const sequence = parseSequenceFromLogEth(
      receipt,
      CONTRACTS.DEVNET.ethereum.core
    );
    const emitterAddress = getEmitterAddressEth(
      CONTRACTS.DEVNET.ethereum.token_bridge
    );
    console.log(`fetching vaa ${sequence}...`);
    // poll until the guardian(s) witness and sign the vaa
    const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
      GUARDIAN_RPCS,
      CHAIN_ID_ETH,
      emitterAddress,
      sequence,
      {
        transport: NodeHttpTransport(),
      }
    );
    console.log("redeeming...");
    await redeemOnEth(CONTRACTS.DEVNET.bsc.token_bridge, bscSigner, signedVAA);
    const afterMetrics = await fetchGlobalAccountantMetrics();
    console.log(
      "approved b/a:",
      beforeMetrics.global_accountant_transfer_vaas_submitted_and_approved,
      afterMetrics.global_accountant_transfer_vaas_submitted_and_approved
    );
    if (
      afterMetrics.global_accountant_events_received <=
        beforeMetrics.global_accountant_events_received ||
      afterMetrics.global_accountant_transfer_vaas_submitted <=
        beforeMetrics.global_accountant_transfer_vaas_submitted ||
      afterMetrics.global_accountant_transfer_vaas_submitted_and_approved <=
        beforeMetrics.global_accountant_transfer_vaas_submitted_and_approved
    ) {
      throw new Error("Expected metrics change did not occur");
    }
  }

  //
  // STEP 2 - send the token back
  //
  {
    const beforeMetrics = await fetchGlobalAccountantMetrics();
    const amount = parseUnits("1", DECIMALS);
    // approve the bridge to spend tokens
    console.log("approving...");
    await approveEth(
      CONTRACTS.DEVNET.bsc.token_bridge,
      attestedAddress,
      bscSigner,
      amount
    );
    // transfer tokens out
    console.log("transferring...");
    const receipt = await transferFromEth(
      CONTRACTS.DEVNET.bsc.token_bridge,
      bscSigner,
      attestedAddress,
      amount,
      CHAIN_ID_ETH,
      tryNativeToUint8Array(await ethSigner.getAddress(), CHAIN_ID_ETH)
    );
    // get the sequence from the logs (needed to fetch the vaa)
    const sequence = parseSequenceFromLogEth(
      receipt,
      CONTRACTS.DEVNET.bsc.core
    );
    const emitterAddress = getEmitterAddressEth(
      CONTRACTS.DEVNET.bsc.token_bridge
    );
    console.log(`fetching vaa ${sequence}...`);
    // poll until the guardian(s) witness and sign the vaa
    const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
      GUARDIAN_RPCS,
      CHAIN_ID_BSC,
      emitterAddress,
      sequence,
      {
        transport: NodeHttpTransport(),
      }
    );
    console.log("redeeming...");
    await redeemOnEth(
      CONTRACTS.DEVNET.ethereum.token_bridge,
      ethSigner,
      signedVAA
    );
    const afterMetrics = await fetchGlobalAccountantMetrics();
    console.log(
      "approved b/a:",
      beforeMetrics.global_accountant_transfer_vaas_submitted_and_approved,
      afterMetrics.global_accountant_transfer_vaas_submitted_and_approved
    );
    if (
      afterMetrics.global_accountant_events_received <=
        beforeMetrics.global_accountant_events_received ||
      afterMetrics.global_accountant_transfer_vaas_submitted <=
        beforeMetrics.global_accountant_transfer_vaas_submitted ||
      afterMetrics.global_accountant_transfer_vaas_submitted_and_approved <=
        beforeMetrics.global_accountant_transfer_vaas_submitted_and_approved
    ) {
      throw new Error("Expected metrics change did not occur");
    }
  }

  //
  // STEP 3a - redeem spoofed tokens
  //
  {
    console.log("redeeming spoofed tokens");
    let vaa: VAA<TokenBridgeTransfer> = {
      version: 1,
      guardianSetIndex: 0,
      signatures: [],
      timestamp: 0,
      nonce: 0,
      emitterChain: CHAIN_ID_ETH,
      emitterAddress: getEmitterAddressEth(
        CONTRACTS.DEVNET.ethereum.token_bridge
      ),
      sequence: BigInt(979999116 + Math.floor(Math.random() * 100000000)),
      consistencyLevel: 0,
      payload: {
        module: "TokenBridge",
        type: "Transfer",
        tokenChain: CHAIN_ID_ETH,
        tokenAddress: uint8ArrayToHex(
          tryNativeToUint8Array(ETH_GA_TEST_TOKEN, CHAIN_ID_ETH)
        ),
        amount: parseUnits("9000", DECIMALS).toBigInt(),
        toAddress: uint8ArrayToHex(
          tryNativeToUint8Array(await bscSigner.getAddress(), CHAIN_ID_BSC)
        ),
        chain: CHAIN_ID_BSC,
        fee: BigInt(0),
      },
    };
    vaa.signatures = sign(VAA_SIGNERS, vaa);
    await redeemOnEth(
      CONTRACTS.DEVNET.bsc.token_bridge,
      bscSigner,
      hexToUint8Array(serialiseVAA(vaa))
    );
  }

  //
  // STEP 3b - send the spoofed tokens back
  //
  {
    const beforeMetrics = await fetchGlobalAccountantMetrics();
    const amount = parseUnits("9000", DECIMALS);
    // approve the bridge to spend tokens
    console.log("approving...");
    await approveEth(
      CONTRACTS.DEVNET.bsc.token_bridge,
      attestedAddress,
      bscSigner,
      amount
    );
    // transfer tokens out
    console.log("transferring...");
    const receipt = await transferFromEth(
      CONTRACTS.DEVNET.bsc.token_bridge,
      bscSigner,
      attestedAddress,
      amount,
      CHAIN_ID_ETH,
      tryNativeToUint8Array(await ethSigner.getAddress(), CHAIN_ID_ETH)
    );
    console.log("waiting 30s to fetch metrics...");
    await sleep(30 * 1000); // give the guardian a few seconds to pick up the transfers and attempt to submit them
    const afterMetrics = await fetchGlobalAccountantMetrics();
    console.log(
      "balance errors b/a:",
      beforeMetrics.global_accountant_total_balance_errors,
      afterMetrics.global_accountant_total_balance_errors
    );
    if (
      afterMetrics.global_accountant_error_events_received <=
        beforeMetrics.global_accountant_error_events_received ||
      afterMetrics.global_accountant_transfer_vaas_submitted <=
        beforeMetrics.global_accountant_transfer_vaas_submitted ||
      afterMetrics.global_accountant_total_balance_errors <=
        beforeMetrics.global_accountant_total_balance_errors
    ) {
      throw new Error("Expected metrics change did not occur");
    }
  }

  ethProvider.destroy();
  bscProvider.destroy();
  console.log("success!");
})();
