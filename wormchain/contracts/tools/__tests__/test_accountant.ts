import "dotenv/config";
import { describe, expect, jest, test } from "@jest/globals";
import {
  approveEth,
  attestFromEth,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  ChainId,
  CONTRACTS,
  createWrappedOnEth,
  getEmitterAddressEth,
  getForeignAssetEth,
  getSignedVAAWithRetry,
  hexToUint8Array,
  parseSequenceFromLogEth,
  parseTokenTransferVaa,
  redeemOnEth,
  serialiseVAA,
  sign,
  TokenBridgeTransfer,
  transferFromEth,
  tryNativeToHexString,
  tryNativeToUint8Array,
  uint8ArrayToHex,
  VAA,
} from "@certusone/wormhole-sdk";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { ethers } from "ethers";
import * as devnetConsts from "../devnet-consts.json";
import { parseUnits } from "ethers/lib/utils";
import { CosmWasmClient } from "@cosmjs/cosmwasm-stargate";

jest.setTimeout(120000);

if (process.env.INIT_SIGNERS_KEYS_CSV === "undefined") {
  let msg = `.env is missing. run "make contracts-tools-deps" to fetch.`;
  console.error(msg);
  throw msg;
}

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
const ETH_NODE_URL = ci ? "http://eth-devnet:8545" : "http://localhost:8545";
const BSC_NODE_URL = ci ? "http://eth-devnet2:8545" : "http://localhost:8546";
const ETH_PRIVATE_KEY9 =
  "0xb0057716d5917badaf911b193b12b910811c1497b5bada8d7711f758981c3773";
const ETH_GA_TEST_TOKEN =
  devnetConsts.chains[CHAIN_ID_ETH].addresses.testGA.address;
const DECIMALS = devnetConsts.chains[CHAIN_ID_ETH].addresses.testGA.decimals;
const VAA_SIGNERS = process.env.INIT_SIGNERS_KEYS_CSV.split(",");
const GOVERNANCE_CHAIN = Number(devnetConsts.global.governanceChainId);
const GOVERNANCE_EMITTER = devnetConsts.global.governanceEmitterAddress;
const TENDERMINT_URL = ci ? "http://wormchain:26657" : "http://localhost:26659";
const GA_ADDRESS =
  "wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465";

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

let ethProvider: ethers.providers.JsonRpcProvider;
let ethSigner: ethers.Wallet;
let bscProvider: ethers.providers.JsonRpcProvider;
let bscSigner: ethers.Wallet;
let cosmWasmClient: CosmWasmClient;

beforeAll(async () => {
  // create a signer for Eth
  ethProvider = new ethers.providers.JsonRpcProvider(ETH_NODE_URL);
  ethSigner = new ethers.Wallet(ETH_PRIVATE_KEY9, ethProvider);
  // create a signer for BSC
  bscProvider = new ethers.providers.JsonRpcProvider(BSC_NODE_URL);
  bscSigner = new ethers.Wallet(ETH_PRIVATE_KEY9, bscProvider);
  cosmWasmClient = await CosmWasmClient.connect(TENDERMINT_URL);
});

afterAll(async () => {
  cosmWasmClient.disconnect();
});

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

const fetchGlobalAccountantBalance = async (
  tokenAddress: string,
  chainId: ChainId,
  tokenChain: ChainId
): Promise<BigInt> => {
  try {
    return BigInt(
      await cosmWasmClient.queryContractSmart(GA_ADDRESS, {
        balance: {
          token_address: tokenAddress,
          chain_id: chainId,
          token_chain: tokenChain,
        },
      })
    );
  } catch (e) {
    if (e.message?.includes("accountant::state::account::Balance not found")) {
      // account not created yet
      return BigInt(0);
    }
    throw e;
  }
};

const fetchGlobalAccountantTransferStatus = async (
  emitterChain: ChainId,
  emitterAddress: string,
  sequence: string
): Promise<any> => {
  return await cosmWasmClient.queryContractSmart(GA_ADDRESS, {
    transfer_status: {
      emitter_chain: emitterChain,
      emitter_address: emitterAddress,
      sequence: Number(sequence),
    },
  });
};

describe("Global Accountant Tests", () => {
  test("Metrics and Contract Queries", async () => {
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
        // already attested
      } else {
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
        await ethProvider.send("anvil_mine", ["0x40"]); // 64 blocks should get the above block to `finalized`
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
      const beforeEthBalance = await fetchGlobalAccountantBalance(
        tryNativeToHexString(ETH_GA_TEST_TOKEN, CHAIN_ID_ETH),
        CHAIN_ID_ETH,
        CHAIN_ID_ETH
      );
      const beforeBscBalance = await fetchGlobalAccountantBalance(
        tryNativeToHexString(ETH_GA_TEST_TOKEN, CHAIN_ID_ETH),
        CHAIN_ID_BSC,
        CHAIN_ID_ETH
      );
      const amount = parseUnits("1", DECIMALS);
      // approve the bridge to spend tokens
      await approveEth(
        CONTRACTS.DEVNET.ethereum.token_bridge,
        ETH_GA_TEST_TOKEN,
        ethSigner,
        amount
      );
      // transfer tokens out
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
      await ethProvider.send("anvil_mine", ["0x40"]); // 64 blocks should get the above block to `finalized`
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
      await redeemOnEth(
        CONTRACTS.DEVNET.bsc.token_bridge,
        bscSigner,
        signedVAA
      );
      const afterMetrics = await fetchGlobalAccountantMetrics();
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
      const parsedVAA = parseTokenTransferVaa(signedVAA);
      const transferStatus = await fetchGlobalAccountantTransferStatus(
        CHAIN_ID_ETH,
        emitterAddress,
        sequence
      );
      expect(transferStatus).toMatchObject({
        committed: {
          data: {
            amount: parsedVAA.amount.toString(),
            token_chain: CHAIN_ID_ETH,
            token_address: tryNativeToHexString(
              ETH_GA_TEST_TOKEN,
              CHAIN_ID_ETH
            ),
            recipient_chain: CHAIN_ID_BSC,
          },
        },
      });
      const afterEthBalance = await fetchGlobalAccountantBalance(
        tryNativeToHexString(ETH_GA_TEST_TOKEN, CHAIN_ID_ETH),
        CHAIN_ID_ETH,
        CHAIN_ID_ETH
      );
      expect(afterEthBalance).toBeGreaterThan(beforeEthBalance.valueOf());
      const afterBscBalance = await fetchGlobalAccountantBalance(
        tryNativeToHexString(ETH_GA_TEST_TOKEN, CHAIN_ID_ETH),
        CHAIN_ID_BSC,
        CHAIN_ID_ETH
      );
      expect(afterBscBalance).toBeGreaterThan(beforeBscBalance.valueOf());
    }

    //
    // STEP 2 - send the token back
    //
    {
      const beforeMetrics = await fetchGlobalAccountantMetrics();
      const beforeEthBalance = await fetchGlobalAccountantBalance(
        tryNativeToHexString(ETH_GA_TEST_TOKEN, CHAIN_ID_ETH),
        CHAIN_ID_ETH,
        CHAIN_ID_ETH
      );
      const beforeBscBalance = await fetchGlobalAccountantBalance(
        tryNativeToHexString(ETH_GA_TEST_TOKEN, CHAIN_ID_ETH),
        CHAIN_ID_BSC,
        CHAIN_ID_ETH
      );
      const amount = parseUnits("1", DECIMALS);
      // approve the bridge to spend tokens
      await approveEth(
        CONTRACTS.DEVNET.bsc.token_bridge,
        attestedAddress,
        bscSigner,
        amount
      );
      // transfer tokens out
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
      await bscProvider.send("anvil_mine", ["0x40"]); // 64 blocks should get the above block to `finalized`
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
      await redeemOnEth(
        CONTRACTS.DEVNET.ethereum.token_bridge,
        ethSigner,
        signedVAA
      );
      const afterMetrics = await fetchGlobalAccountantMetrics();
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
      const parsedVAA = parseTokenTransferVaa(signedVAA);
      const transferStatus = await fetchGlobalAccountantTransferStatus(
        CHAIN_ID_BSC,
        emitterAddress,
        sequence
      );
      expect(transferStatus).toMatchObject({
        committed: {
          data: {
            amount: parsedVAA.amount.toString(),
            token_chain: CHAIN_ID_ETH,
            token_address: tryNativeToHexString(
              ETH_GA_TEST_TOKEN,
              CHAIN_ID_ETH
            ),
            recipient_chain: CHAIN_ID_ETH,
          },
        },
      });
      const afterEthBalance = await fetchGlobalAccountantBalance(
        tryNativeToHexString(ETH_GA_TEST_TOKEN, CHAIN_ID_ETH),
        CHAIN_ID_ETH,
        CHAIN_ID_ETH
      );
      expect(afterEthBalance).toBeLessThan(beforeEthBalance.valueOf());
      const afterBscBalance = await fetchGlobalAccountantBalance(
        tryNativeToHexString(ETH_GA_TEST_TOKEN, CHAIN_ID_ETH),
        CHAIN_ID_BSC,
        CHAIN_ID_ETH
      );
      expect(afterBscBalance).toBeLessThan(beforeBscBalance.valueOf());
    }

    //
    // STEP 3a - redeem spoofed tokens
    //
    {
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
      await approveEth(
        CONTRACTS.DEVNET.bsc.token_bridge,
        attestedAddress,
        bscSigner,
        amount
      );
      // transfer tokens out
      const receipt = await transferFromEth(
        CONTRACTS.DEVNET.bsc.token_bridge,
        bscSigner,
        attestedAddress,
        amount,
        CHAIN_ID_ETH,
        tryNativeToUint8Array(await ethSigner.getAddress(), CHAIN_ID_ETH)
      );
      const sequence = parseSequenceFromLogEth(
        receipt,
        CONTRACTS.DEVNET.bsc.core
      );
      await bscProvider.send("anvil_mine", ["0x40"]); // 64 blocks should get the above block to `finalized`
      await sleep(30 * 1000); // give the guardian a few seconds to pick up the transfers and attempt to submit them
      const afterMetrics = await fetchGlobalAccountantMetrics();
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
      // the transfer should fail, because there's an insufficient source balance
      if (VAA_SIGNERS.length > 1) {
        const transferStatus = await fetchGlobalAccountantTransferStatus(
          CHAIN_ID_BSC,
          getEmitterAddressEth(CONTRACTS.DEVNET.bsc.token_bridge),
          sequence
        );
        expect(Object.keys(transferStatus)).toContain("pending");
        expect(Object.keys(transferStatus)).not.toContain("committed");
      } else {
        await expect(
          fetchGlobalAccountantTransferStatus(
            CHAIN_ID_BSC,
            getEmitterAddressEth(CONTRACTS.DEVNET.bsc.token_bridge),
            sequence
          )
        ).rejects.toThrow();
      }
    }
  });
});
