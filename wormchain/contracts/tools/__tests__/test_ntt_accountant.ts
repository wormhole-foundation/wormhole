import {
  ChainId,
  ethers_contracts,
  Other,
  Payload,
  VAA,
  serialiseVAA,
  sign,
  CONTRACTS,
  parseSequenceFromLogEth,
  getSignedVAAWithRetry,
} from "@certusone/wormhole-sdk";
import { CosmWasmClient } from "@cosmjs/cosmwasm-stargate";
import { describe, expect, jest, test } from "@jest/globals";
import {
  getWallet,
  getWormchainSigningClient,
} from "@wormhole-foundation/wormchain-sdk";
import { ZERO_FEE } from "@wormhole-foundation/wormchain-sdk/lib/core/consts";
import { toUtf8 } from "cosmwasm";
import "dotenv/config";
import * as devnetConsts from "../devnet-consts.json";
import { ethers } from "ethers";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import {
  GetPriceOptParams,
  getPrice,
  sendToEvm,
} from "@certusone/wormhole-sdk/lib/cjs/relayer";

jest.setTimeout(120000);

if (process.env.INIT_SIGNERS_KEYS_CSV === "undefined") {
  let msg = `.env is missing. run "make contracts-tools-deps" to fetch.`;
  console.error(msg);
  throw msg;
}

/*
 * Goals:
 *   1. Inits
 *      a. Ensure a hub init is saved
 *      b. Ensure a duplicate hub init is rejected
 *      c. Ensure a non-hub init is rejected
 *   2. Registrations
 *      a. Ensure a hub registration to an transceiver without a known hub is rejected
 *      b. Ensure an transceiver registration to a hub is saved
 *      c. Ensure a hub registration to an transceiver with a known hub is saved
 *      d. Ensure an transceiver registration to another transceiver without a known hub is rejected
 *      e. Ensure an transceiver registration from an transceiver without a known hub to a non-hub is rejected
 *      f. Ensure an transceiver registration to another transceiver with a known hub is saved
 *      g. Ensure a duplicate registration is rejected
 *   3. Transfer VAAs
 *      a. Ensure a token can be sent from its hub transceiver
 *      b. Ensure a token decimal shift works as expected
 *      c. Ensure a token can be sent back to its hub transceiver
 *      d. Ensure a token can be sent between non-hub transceivers
 *      e. Ensure a token sent from a source transceiver without a known hub is rejected
 *      f. Ensure a token sent from a source chain without a known transceiver is rejected
 *      g. Ensure a token sent from a source chain without a matching transceiver is rejected
 *      h. Ensure a token sent to a destination chain without a known transceiver is rejected
 *      i. Ensure a token sent to a destination chain without a matching transceiver is rejected
 *      j. Ensure spoofed tokens for more than the outstanding amount rejects successfully
 *      k. Ensure a rogue endpoint cannot complete a transfer
 *   4. Relayers
 *      a. Ensure a relayer registration is saved
 *      b. Ensure a valid NTT transfer works
 *      c. Ensure an invalid NTT transfer rejects
 *      d. Ensure an invalid payload reverts
 *      e. Ensure a non-delivery reverts
 *   5. Observations
 *      a-i. Repeat transfer tests via guardian
 *   6. Validate the guardian metrics for Observations a-i
 *   7. Bonus: Validate the on chain contract state via queries
 */

// Guardian (observation) testing
// ---
// positive tests will use the network of
//   HUB_CHAIN (Eth) - ETH_WALLET_EMITTER
//   SPOKE_CHAIN_A (BSC) - BSC_WALLET_EMITTER
// ---
// negative tests will use the faux network of
//   HUB_CHAIN (Eth) - BSC_WALLET_EMITTER
//   SPOKE_CHAIN_A (BSC) - ETH_WALLET_EMITTER

const ci = !!process.env.CI;

const GUARDIAN_HOST = ci ? "guardian" : "localhost";
const GUARDIAN_RPCS = [`http://${GUARDIAN_HOST}:7071`];
const GUARDIAN_METRICS = `http://${GUARDIAN_HOST}:6060/metrics`;
const VAA_SIGNERS = process.env.INIT_SIGNERS_KEYS_CSV.split(",");
const GOVERNANCE_CHAIN = Number(devnetConsts.global.governanceChainId);
const GOVERNANCE_EMITTER = devnetConsts.global.governanceEmitterAddress;
const TENDERMINT_URL = ci ? "http://wormchain:26657" : "http://localhost:26659";
const NTT_GA_ADDRESS =
  "wormhole17p9rzwnnfxcjp32un9ug7yhhzgtkhvl9jfksztgw5uh69wac2pgshdnj3k";
const ETH_WALLET = devnetConsts.gancheDefaults[11];
const BSC_WALLET = devnetConsts.gancheDefaults[12];
const ETH_WALLET_EMITTER = ETH_WALLET.public
  .substring(2)
  .padStart(64, "0")
  .toLowerCase();
const BSC_WALLET_EMITTER = BSC_WALLET.public
  .substring(2)
  .padStart(64, "0")
  .toLowerCase();
const ETH_NODE_URL = ci ? "http://eth-devnet:8545" : "http://localhost:8545";
const BSC_NODE_URL = ci ? "http://eth-devnet2:8545" : "http://localhost:8546";

const HUB_CHAIN = 2;
const HUB_TRANSCEIVER = `0000000000000000000000000000000000000000000000000000000000000042`;
const SPOKE_CHAIN_A = 4;
const SPOKE_TRANSCEIVER_A = `0000000000000000000000000000000000000000000000000000000000000043`;
const ROGUE_TRANSCEIVER_A = `ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff43`;
const SPOKE_CHAIN_B = 5;
const SPOKE_TRANSCEIVER_B = `0000000000000000000000000000000000000000000000000000000000000044`;
const FAUX_HUB_CHAIN = 420;
const FAUX_HUB_TRANSCEIVER =
  "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef";
const FAUX_SPOKE_CHAIN_A = SPOKE_CHAIN_A;
const FAUX_SPOKE_TRANSCEIVER_A = FAUX_HUB_TRANSCEIVER;
const UNKNOWN_SPOKE_CHAIN = 404;
const UNKNOWN_SPOKE_TRANSCEIVER =
  "beeffacebeeffacebeeffacebeeffacebeeffacebeeffacebeeffacebeefface";
const RELAYER_ADDRESS = ci
  ? "0xb98F46E96cb1F519C333FdFB5CCe0B13E0300ED4"
  : "0xcC680D088586c09c3E0E099a676FA4b6e42467b4";
const RELAYER_EMITTER = ci
  ? "000000000000000000000000b98F46E96cb1F519C333FdFB5CCe0B13E0300ED4"
  : "000000000000000000000000cc680d088586c09c3e0e099a676fa4b6e42467b4";
const dummy32 = `0000000000000000000000000000000000000000000000000000000000001234`;

const host = ci
  ? devnetConsts.chains[3104].tendermintUrlTilt
  : devnetConsts.chains[3104].tendermintUrlLocal;
// NttAccountantTest = wormhole18s5lynnmx37hq4wlrw9gdn68sg2uxp5rwf5k3u
const mnemonic =
  "quality vacuum heart guard buzz spike sight swarm shove special gym robust assume sudden deposit grid alcohol choice devote leader tilt noodle tide penalty";

let client: any;
let signer: string;
let ethProvider: ethers.providers.JsonRpcProvider;
let ethSigner: ethers.Wallet;
let fauxEthSigner: ethers.Wallet;
let bscProvider: ethers.providers.JsonRpcProvider;
let bscSigner: ethers.Wallet;
let fauxBscSigner: ethers.Wallet;
let cosmWasmClient: CosmWasmClient;

beforeAll(async () => {
  const wallet = await getWallet(mnemonic);
  client = await getWormchainSigningClient(host, wallet);
  const signers = await wallet.getAccounts();
  signer = signers[0].address;
  // create a signer for Eth
  ethProvider = new ethers.providers.JsonRpcProvider(ETH_NODE_URL);
  ethSigner = new ethers.Wallet(ETH_WALLET.private, ethProvider);
  fauxEthSigner = new ethers.Wallet(BSC_WALLET.private, ethProvider);
  // create a signer for BSC
  bscProvider = new ethers.providers.JsonRpcProvider(BSC_NODE_URL);
  bscSigner = new ethers.Wallet(BSC_WALLET.private, bscProvider);
  fauxBscSigner = new ethers.Wallet(ETH_WALLET.private, bscProvider);
  cosmWasmClient = await CosmWasmClient.connect(TENDERMINT_URL);
});

afterAll(async () => {
  cosmWasmClient.disconnect();
});

type GuardianMetrics = {
  global_accountant_connection_errors_total: number;
  global_accountant_error_events_received: number;
  global_accountant_events_received: number;
  global_accountant_submit_failures: number;
  global_accountant_total_balance_errors: number;
  global_accountant_total_digest_mismatches: number;
  global_accountant_transfer_vaas_outstanding: number;
  global_accountant_transfer_vaas_submitted: number;
  global_accountant_transfer_vaas_submitted_and_approved: number;
};

// Guardian metrics are prometheus data
const fetchGlobalAccountantMetrics = async (): Promise<GuardianMetrics> =>
  (await (await fetch(GUARDIAN_METRICS)).text())
    .split("\n")
    .filter((m) => m.startsWith("global_accountant"))
    .reduce((p, m) => {
      const [k, v] = m.split(" ");
      p[k] = Number(v);
      return p;
    }, {} as any);

async function waitForMetricsChange(
  failurePredicate: (GuardianMetrics) => boolean,
  retryTimeout: number = 1000,
  retryAttempts: number = 30
) {
  let passed = false;
  let attempts = 0;
  while (!passed) {
    attempts++;
    await new Promise((resolve) => setTimeout(resolve, retryTimeout));
    let afterMetrics;
    try {
      afterMetrics = await fetchGlobalAccountantMetrics();
    } catch (e) {
      continue;
    }
    if (failurePredicate(afterMetrics)) {
      if (retryAttempts !== undefined && attempts > retryAttempts) {
        throw new Error("Expected metrics change did not occur");
      }
    } else {
      return;
    }
  }
}

const fetchGlobalAccountantBalance = async (
  chainId: ChainId,
  tokenChain: ChainId,
  tokenAddress: string
): Promise<BigInt> => {
  try {
    return BigInt(
      await cosmWasmClient.queryContractSmart(NTT_GA_ADDRESS, {
        balance: {
          chain_id: chainId,
          token_chain: tokenChain,
          token_address: tokenAddress,
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
  return await cosmWasmClient.queryContractSmart(NTT_GA_ADDRESS, {
    transfer_status: {
      emitter_chain: emitterChain,
      emitter_address: emitterAddress,
      sequence: Number(sequence),
    },
  });
};

const makeVAA = (
  emitterChain: number,
  emitterAddress: string,
  payload: string
) => {
  let vaa: VAA<Other> = {
    version: 1,
    guardianSetIndex: 0,
    signatures: [],
    timestamp: 0,
    nonce: 0,
    emitterChain: emitterChain,
    emitterAddress: emitterAddress,
    sequence: BigInt(Math.floor(Math.random() * 100000000)),
    consistencyLevel: 1,
    payload: {
      type: "Other",
      hex: payload,
    },
  };
  vaa.signatures = sign(VAA_SIGNERS, vaa as unknown as VAA<Payload>);
  return vaa;
};

const submitVAA = async (vaa: VAA<Other>) => {
  const msg = client.wasm.msgExecuteContract({
    sender: signer,
    contract: NTT_GA_ADDRESS,
    msg: toUtf8(
      JSON.stringify({
        submit_vaas: {
          vaas: [
            Buffer.from(
              serialiseVAA(vaa as unknown as VAA<Payload>),
              "hex"
            ).toString("base64"),
          ],
        },
      })
    ),
    funds: [],
  });
  const result = await client.signAndBroadcast(signer, [msg], {
    ...ZERO_FEE,
    gas: "10000000",
  });
  return result;
};

const chainToHex = (chainId: number) => chainId.toString(16).padStart(4, "0");

const mockTransferPayload = (
  decimals: number,
  amount: number,
  toChain: number
) => {
  const seq = dummy32;
  const d = decimals.toString(16).padStart(2, "0");
  const a = amount.toString(16).padStart(16, "0");
  const payload = `994E5454${d}${a}${dummy32}${dummy32}${chainToHex(toChain)}`;
  const payloadLen = (payload.length / 2).toString(16).padStart(4, "0");
  const msg = `${seq}${dummy32}${payloadLen}${payload}`;
  const msgLen = (msg.length / 2).toString(16).padStart(4, "0");
  return `9945FF10${dummy32}${dummy32}${msgLen}${msg}0000000000000000`;
};

const mockDeliveryPayload = (sender: string, payload: string) => {
  const payloadLen = (payload.length / 2).toString(16).padStart(8, "0");
  return `01${chainToHex(
    0
  )}${dummy32}${payloadLen}${payload}${dummy32}${dummy32}00000000${chainToHex(
    0
  )}${dummy32}${dummy32}${dummy32}${sender}00`;
};

describe("NTT Global Accountant Tests", () => {
  describe("1. Inits", () => {
    test("a. Ensure a hub init is saved", async () => {
      const vaa = makeVAA(
        HUB_CHAIN,
        HUB_TRANSCEIVER,
        "9c23bd3b000000000000000000000000bb807f76cda53b1b4256e1b6f33bb46be36508e3000000000000000000000000002a68f967bfa230780a385175d0c86ae4048d309612"
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(0);
      const response = await cosmWasmClient.queryContractSmart(NTT_GA_ADDRESS, {
        all_transceiver_hubs: {},
      });
      const hub = response.hubs.find(
        (entry) =>
          entry.key[0] === HUB_CHAIN && entry.key[1] === HUB_TRANSCEIVER
      );
      expect(hub).toBeDefined();
      expect(hub.data).toStrictEqual([HUB_CHAIN, HUB_TRANSCEIVER]);
      // check replay protection
      {
        const result = await submitVAA(vaa);
        expect(result.code).toEqual(5);
        expect(result.rawLog).toMatch("message already processed");
      }
    });
    test("b. Ensure a duplicate hub init is rejected", async () => {
      const vaa = makeVAA(
        HUB_CHAIN,
        HUB_TRANSCEIVER,
        "9c23bd3b000000000000000000000000bb807f76cda53b1b4256e1b6f33bb46be36508e3000000000000000000000000002a68f967bfa230780a385175d0c86ae4048d309612"
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(5);
      expect(result.rawLog).toMatch("hub entry already exists");
    });
    test("c. Ensure a non-hub init is rejected", async () => {
      const vaa = makeVAA(
        SPOKE_CHAIN_A,
        SPOKE_TRANSCEIVER_A,
        "9c23bd3b0000000000000000000000001fc14f21b27579f4f23578731cd361cca8aa39f701000000000000000000000000eb502b1d35e975321b21cce0e8890d20a7eb289d12"
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(5);
      expect(result.rawLog).toMatch("ignoring non-locking NTT initialization");
    });
  });
  describe("2. Registrations", () => {
    test("a. Ensure a hub registration to an transceiver without a known hub is rejected", async () => {
      const vaa = makeVAA(
        HUB_CHAIN,
        HUB_TRANSCEIVER,
        `18fc67c2${chainToHex(SPOKE_CHAIN_A)}${SPOKE_TRANSCEIVER_A}`
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(5);
      expect(result.rawLog).toMatch("no registered hub");
    });
    test("b. Ensure an transceiver registration to a hub is saved", async () => {
      const vaa = makeVAA(
        SPOKE_CHAIN_A,
        SPOKE_TRANSCEIVER_A,
        `18fc67c2${chainToHex(HUB_CHAIN)}${HUB_TRANSCEIVER}`
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(0);
      const response = await cosmWasmClient.queryContractSmart(NTT_GA_ADDRESS, {
        all_transceiver_peers: {},
      });
      const peer = response.peers.find(
        (entry) =>
          entry.key[0] === SPOKE_CHAIN_A &&
          entry.key[1] === SPOKE_TRANSCEIVER_A &&
          entry.key[2] === HUB_CHAIN
      );
      expect(peer).toBeDefined();
      expect(peer.data).toStrictEqual(HUB_TRANSCEIVER);
      // check replay protection
      {
        const result = await submitVAA(vaa);
        expect(result.code).toEqual(5);
        expect(result.rawLog).toMatch("message already processed");
      }
    });
    test("c. Ensure a hub registration to an transceiver with a known hub is saved", async () => {
      const vaa = makeVAA(
        HUB_CHAIN,
        HUB_TRANSCEIVER,
        `18fc67c2${chainToHex(SPOKE_CHAIN_A)}${SPOKE_TRANSCEIVER_A}`
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(0);
      const response = await cosmWasmClient.queryContractSmart(NTT_GA_ADDRESS, {
        all_transceiver_peers: {},
      });
      const peer = response.peers.find(
        (entry) =>
          entry.key[0] === HUB_CHAIN &&
          entry.key[1] === HUB_TRANSCEIVER &&
          entry.key[2] === SPOKE_CHAIN_A
      );
      expect(peer).toBeDefined();
      expect(peer.data).toStrictEqual(SPOKE_TRANSCEIVER_A);
    });
    test("d. Ensure an transceiver registration to another transceiver without a known hub is rejected", async () => {
      const vaa = makeVAA(
        SPOKE_CHAIN_A,
        SPOKE_TRANSCEIVER_A,
        `18fc67c2${chainToHex(SPOKE_CHAIN_B)}${SPOKE_TRANSCEIVER_B}`
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(5);
      expect(result.rawLog).toMatch("no registered hub");
    });
    test("e. Ensure an transceiver registration from an transceiver without a known hub to a non-hub is rejected", async () => {
      const vaa = makeVAA(
        SPOKE_CHAIN_B,
        SPOKE_TRANSCEIVER_B,
        `18fc67c2${chainToHex(SPOKE_CHAIN_A)}${SPOKE_TRANSCEIVER_A}`
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(5);
      expect(result.rawLog).toMatch(
        "ignoring attempt to register peer before hub"
      );
    });
    test("f. Ensure an transceiver registration to another transceiver with a known hub is saved", async () => {
      {
        const vaa = makeVAA(
          SPOKE_CHAIN_B,
          SPOKE_TRANSCEIVER_B,
          `18fc67c2${chainToHex(HUB_CHAIN)}${HUB_TRANSCEIVER}`
        );
        const result = await submitVAA(vaa);
        expect(result.code).toEqual(0);
        const response = await cosmWasmClient.queryContractSmart(
          NTT_GA_ADDRESS,
          {
            all_transceiver_peers: {},
          }
        );
        const peer = response.peers.find(
          (entry) =>
            entry.key[0] === SPOKE_CHAIN_B &&
            entry.key[1] === SPOKE_TRANSCEIVER_B &&
            entry.key[2] === HUB_CHAIN
        );
        expect(peer).toBeDefined();
        expect(peer.data).toStrictEqual(HUB_TRANSCEIVER);
      }
      {
        const vaa = makeVAA(
          SPOKE_CHAIN_A,
          SPOKE_TRANSCEIVER_A,
          `18fc67c2${chainToHex(SPOKE_CHAIN_B)}${SPOKE_TRANSCEIVER_B}`
        );
        const result = await submitVAA(vaa);
        expect(result.code).toEqual(0);
        const response = await cosmWasmClient.queryContractSmart(
          NTT_GA_ADDRESS,
          {
            all_transceiver_peers: {},
          }
        );
        const peer = response.peers.find(
          (entry) =>
            entry.key[0] === SPOKE_CHAIN_A &&
            entry.key[1] === SPOKE_TRANSCEIVER_A &&
            entry.key[2] === SPOKE_CHAIN_B
        );
        expect(peer).toBeDefined();
        expect(peer.data).toStrictEqual(SPOKE_TRANSCEIVER_B);
      }
    });
    test("g. Ensure a duplicate registration is rejected", async () => {
      {
        const vaa = makeVAA(
          SPOKE_CHAIN_B,
          SPOKE_TRANSCEIVER_B,
          `18fc67c2${chainToHex(SPOKE_CHAIN_A)}${SPOKE_TRANSCEIVER_A}`
        );
        const result = await submitVAA(vaa);
        expect(result.code).toEqual(0);
      }
      {
        const vaa = makeVAA(
          SPOKE_CHAIN_B,
          SPOKE_TRANSCEIVER_B,
          `18fc67c2${chainToHex(SPOKE_CHAIN_A)}${SPOKE_TRANSCEIVER_A}`
        );
        const result = await submitVAA(vaa);
        expect(result.code).toEqual(5);
        expect(result.rawLog).toMatch(
          "peer entry for this chain already exists"
        );
      }
    });
    test("h. Ensure a registration that would mismatch hubs is rejected", async () => {
      {
        // set faux hub
        const vaa = makeVAA(
          FAUX_HUB_CHAIN,
          FAUX_HUB_TRANSCEIVER,
          "9c23bd3b000000000000000000000000bb807f76cda53b1b4256e1b6f33bb46be36508e3000000000000000000000000002a68f967bfa230780a385175d0c86ae4048d309612"
        );
        const result = await submitVAA(vaa);
        expect(result.code).toEqual(0);
      }
      {
        // set attempt to register legit spoke with it
        const vaa = makeVAA(
          FAUX_HUB_CHAIN,
          FAUX_HUB_TRANSCEIVER,
          `18fc67c2${chainToHex(SPOKE_CHAIN_A)}${SPOKE_TRANSCEIVER_A}`
        );
        const result = await submitVAA(vaa);
        expect(result.code).toEqual(5);
        expect(result.rawLog).toMatch("peer hub does not match");
      }
    });
  });
  describe("3. Transfer VAAs", () => {
    test("a. Ensure a token can be sent from its hub transceiver", async () => {
      const vaa = makeVAA(
        HUB_CHAIN,
        HUB_TRANSCEIVER,
        mockTransferPayload(8, 10, SPOKE_CHAIN_A)
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(0);
      // check replay protection
      {
        const result = await submitVAA(vaa);
        expect(result.code).toEqual(5);
        expect(result.rawLog).toMatch("message already processed");
      }
    });
    test("b. Ensure a token decimal shift works as expected", async () => {
      const vaa = makeVAA(
        SPOKE_CHAIN_A,
        SPOKE_TRANSCEIVER_A,
        mockTransferPayload(6, 1, HUB_CHAIN)
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(5);
      expect(result.rawLog).toMatch(
        "insufficient balance in source account: Overflow: Cannot Sub with 10 and 100"
      );
    });
    test("c. Ensure a token can be sent back to its hub transceiver", async () => {
      const vaa = makeVAA(
        SPOKE_CHAIN_A,
        SPOKE_TRANSCEIVER_A,
        mockTransferPayload(8, 1, HUB_CHAIN)
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(0);
    });
    test("d. Ensure a token can be sent between non-hub transceivers", async () => {
      const vaa = makeVAA(
        SPOKE_CHAIN_A,
        SPOKE_TRANSCEIVER_A,
        mockTransferPayload(8, 1, SPOKE_CHAIN_B)
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(0);
    });
    test("e. Ensure a token sent from a source transceiver without a known hub is rejected", async () => {
      const vaa = makeVAA(
        UNKNOWN_SPOKE_CHAIN,
        UNKNOWN_SPOKE_TRANSCEIVER,
        mockTransferPayload(8, 1, HUB_CHAIN)
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(5);
      expect(result.rawLog).toMatch("no registered hub");
    });
    test("f. Ensure a token sent from a source chain without a known transceiver is rejected", async () => {
      const vaa = makeVAA(
        FAUX_HUB_CHAIN,
        FAUX_HUB_TRANSCEIVER,
        mockTransferPayload(8, 1, HUB_CHAIN)
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(5);
      expect(result.rawLog).toMatch(
        "no registered source peer for chain Ethereum"
      );
    });
    test("g. Ensure a token sent from a source chain without a matching transceiver is rejected", async () => {
      {
        // set faux spoke registration to hub but not vice-versa
        {
          const vaa = makeVAA(
            FAUX_SPOKE_CHAIN_A,
            FAUX_SPOKE_TRANSCEIVER_A,
            `18fc67c2${chainToHex(FAUX_HUB_CHAIN)}${FAUX_HUB_TRANSCEIVER}`
          );
          const result = await submitVAA(vaa);
          expect(result.code).toEqual(0);
        }
      }
      const vaa = makeVAA(
        FAUX_SPOKE_CHAIN_A,
        FAUX_SPOKE_TRANSCEIVER_A,
        mockTransferPayload(8, 1, FAUX_HUB_CHAIN)
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(5);
      expect(result.rawLog).toMatch(
        "no registered destination peer for chain Bsc"
      );
    });
    test("h. Ensure a token sent to a destination chain without a known transceiver is rejected", async () => {
      const vaa = makeVAA(
        HUB_CHAIN,
        HUB_TRANSCEIVER,
        mockTransferPayload(8, 1, UNKNOWN_SPOKE_CHAIN)
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(5);
      expect(result.rawLog).toMatch("no registered source peer for chain");
    });
    test("i. Ensure a token sent to a destination chain without a matching transceiver is rejected", async () => {
      const vaa = makeVAA(
        FAUX_HUB_CHAIN,
        FAUX_HUB_TRANSCEIVER,
        mockTransferPayload(8, 1, HUB_CHAIN)
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(5);
      expect(result.rawLog).toMatch(
        "no registered source peer for chain Ethereum"
      );
    });
    test("j. Ensure spoofed tokens for more than the outstanding amount rejects successfully", async () => {
      const vaa = makeVAA(
        SPOKE_CHAIN_A,
        SPOKE_TRANSCEIVER_A,
        mockTransferPayload(8, 9000, HUB_CHAIN)
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(5);
      expect(result.rawLog).toMatch(
        "insufficient balance in source account: Overflow: Cannot Sub"
      );
    });
    test("k. Ensure a rogue endpoint cannot complete a transfer", async () => {
      {
        // set faux spoke registration to legit hub but not vice-versa
        {
          const vaa = makeVAA(
            SPOKE_CHAIN_A,
            ROGUE_TRANSCEIVER_A,
            `18fc67c2${chainToHex(HUB_CHAIN)}${HUB_TRANSCEIVER}`
          );
          const result = await submitVAA(vaa);
          expect(result.code).toEqual(0);
        }
      }
      const vaa = makeVAA(
        SPOKE_CHAIN_A,
        ROGUE_TRANSCEIVER_A,
        mockTransferPayload(8, 1, HUB_CHAIN)
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(5);
      expect(result.rawLog).toMatch("peers are not cross-registered");
    });
  });
  describe("4. Relayers", () => {
    test("a. Ensure a relayer registration is saved", async () => {
      const relayerEmitterAsBase64 = Buffer.from(
        RELAYER_EMITTER,
        "hex"
      ).toString("base64");
      // register eth
      const vaa = makeVAA(
        GOVERNANCE_CHAIN,
        GOVERNANCE_EMITTER,
        `0000000000000000000000000000000000576f726d686f6c6552656c617965720100000002${RELAYER_EMITTER}`
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(0);
      const response = await cosmWasmClient.queryContractSmart(NTT_GA_ADDRESS, {
        relayer_chain_registration: {
          chain: 2,
        },
      });
      expect(response.address).toEqual(relayerEmitterAsBase64);
      // check replay protection
      {
        const result = await submitVAA(vaa);
        expect(result.code).toEqual(5);
        expect(result.rawLog).toMatch("message already processed");
      }
      {
        // register bsc
        const vaa = makeVAA(
          GOVERNANCE_CHAIN,
          GOVERNANCE_EMITTER,
          `0000000000000000000000000000000000576f726d686f6c6552656c617965720100000004${RELAYER_EMITTER}`
        );
        const result = await submitVAA(vaa);
        expect(result.code).toEqual(0);
        const response = await cosmWasmClient.queryContractSmart(
          NTT_GA_ADDRESS,
          {
            relayer_chain_registration: {
              chain: 4,
            },
          }
        );
        expect(response.address).toEqual(relayerEmitterAsBase64);
      }
    });
    test("b. Ensure a valid NTT transfer works", async () => {
      const vaa = makeVAA(
        HUB_CHAIN,
        RELAYER_EMITTER,
        mockDeliveryPayload(
          HUB_TRANSCEIVER,
          mockTransferPayload(8, 1, SPOKE_CHAIN_A)
        )
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(0);
    });
    test("c. Ensure an invalid NTT transfer rejects", async () => {
      {
        const vaa = makeVAA(
          HUB_CHAIN,
          RELAYER_EMITTER,
          mockDeliveryPayload(
            UNKNOWN_SPOKE_TRANSCEIVER,
            mockTransferPayload(8, 1, SPOKE_CHAIN_A)
          )
        );
        const result = await submitVAA(vaa);
        expect(result.code).toEqual(5);
        expect(result.rawLog).toMatch("no registered hub");
      }
      {
        const vaa = makeVAA(
          SPOKE_CHAIN_A,
          RELAYER_EMITTER,
          mockDeliveryPayload(
            SPOKE_TRANSCEIVER_A,
            mockTransferPayload(8, 9999, HUB_CHAIN)
          )
        );
        const result = await submitVAA(vaa);
        expect(result.code).toEqual(5);
        expect(result.rawLog).toMatch(
          "insufficient balance in source account: Overflow: Cannot Sub"
        );
      }
    });
    test("d. Ensure an invalid payload reverts", async () => {
      const vaa = makeVAA(
        HUB_CHAIN,
        RELAYER_EMITTER,
        mockDeliveryPayload(dummy32, dummy32)
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(5);
      expect(result.rawLog).toMatch("unsupported NTT action");
    });
    test("e. Ensure a non-delivery reverts", async () => {
      const vaa = makeVAA(
        HUB_CHAIN,
        RELAYER_EMITTER,
        mockTransferPayload(8, 1, SPOKE_CHAIN_A)
      );
      const result = await submitVAA(vaa);
      expect(result.code).toEqual(5);
      expect(result.rawLog).toMatch("PayloadMismatch");
    });
  });
  describe("5. Observations", () => {
    test("setup", async () => {
      {
        // init the locking hub
        const vaa = makeVAA(
          HUB_CHAIN,
          ETH_WALLET_EMITTER,
          "9c23bd3b000000000000000000000000bb807f76cda53b1b4256e1b6f33bb46be36508e3000000000000000000000000002a68f967bfa230780a385175d0c86ae4048d309612"
        );
        const result = await submitVAA(vaa);
        expect(result.code).toEqual(0);
        const response = await cosmWasmClient.queryContractSmart(
          NTT_GA_ADDRESS,
          {
            all_transceiver_hubs: {},
          }
        );
        const hub = response.hubs.find(
          (entry) =>
            entry.key[0] === HUB_CHAIN && entry.key[1] === ETH_WALLET_EMITTER
        );
        expect(hub).toBeDefined();
        expect(hub.data).toStrictEqual([HUB_CHAIN, ETH_WALLET_EMITTER]);
      }
      {
        // register the spokes with the hub
        {
          const vaa = makeVAA(
            SPOKE_CHAIN_A,
            BSC_WALLET_EMITTER,
            `18fc67c2${chainToHex(HUB_CHAIN)}${ETH_WALLET_EMITTER}`
          );
          const result = await submitVAA(vaa);
          expect(result.code).toEqual(0);
          const response = await cosmWasmClient.queryContractSmart(
            NTT_GA_ADDRESS,
            {
              all_transceiver_peers: {},
            }
          );
          const peer = response.peers.find(
            (entry) =>
              entry.key[0] === SPOKE_CHAIN_A &&
              entry.key[1] === BSC_WALLET_EMITTER &&
              entry.key[2] === HUB_CHAIN
          );
          expect(peer).toBeDefined();
          expect(peer.data).toStrictEqual(ETH_WALLET_EMITTER);
        }
        {
          const vaa = makeVAA(
            SPOKE_CHAIN_B,
            ETH_WALLET_EMITTER,
            `18fc67c2${chainToHex(HUB_CHAIN)}${ETH_WALLET_EMITTER}`
          );
          const result = await submitVAA(vaa);
          expect(result.code).toEqual(0);
          const response = await cosmWasmClient.queryContractSmart(
            NTT_GA_ADDRESS,
            {
              all_transceiver_peers: {},
            }
          );
          const peer = response.peers.find(
            (entry) =>
              entry.key[0] === SPOKE_CHAIN_B &&
              entry.key[1] === ETH_WALLET_EMITTER &&
              entry.key[2] === HUB_CHAIN
          );
          expect(peer).toBeDefined();
          expect(peer.data).toStrictEqual(ETH_WALLET_EMITTER);
        }
      }
      {
        // register the hub with the spoke
        {
          const vaa = makeVAA(
            HUB_CHAIN,
            ETH_WALLET_EMITTER,
            `18fc67c2${chainToHex(SPOKE_CHAIN_A)}${BSC_WALLET_EMITTER}`
          );
          const result = await submitVAA(vaa);
          expect(result.code).toEqual(0);
          const response = await cosmWasmClient.queryContractSmart(
            NTT_GA_ADDRESS,
            {
              all_transceiver_peers: {},
            }
          );
          const peer = response.peers.find(
            (entry) =>
              entry.key[0] === HUB_CHAIN &&
              entry.key[1] === ETH_WALLET_EMITTER &&
              entry.key[2] === SPOKE_CHAIN_A
          );
          expect(peer).toBeDefined();
          expect(peer.data).toStrictEqual(BSC_WALLET_EMITTER);
        }
        {
          const vaa = makeVAA(
            HUB_CHAIN,
            ETH_WALLET_EMITTER,
            `18fc67c2${chainToHex(SPOKE_CHAIN_B)}${ETH_WALLET_EMITTER}`
          );
          const result = await submitVAA(vaa);
          expect(result.code).toEqual(0);
          const response = await cosmWasmClient.queryContractSmart(
            NTT_GA_ADDRESS,
            {
              all_transceiver_peers: {},
            }
          );
          const peer = response.peers.find(
            (entry) =>
              entry.key[0] === HUB_CHAIN &&
              entry.key[1] === ETH_WALLET_EMITTER &&
              entry.key[2] === SPOKE_CHAIN_B
          );
          expect(peer).toBeDefined();
          expect(peer.data).toStrictEqual(ETH_WALLET_EMITTER);
        }
      }
      {
        // register the spokes with each other
        {
          const vaa = makeVAA(
            SPOKE_CHAIN_A,
            BSC_WALLET_EMITTER,
            `18fc67c2${chainToHex(SPOKE_CHAIN_B)}${ETH_WALLET_EMITTER}`
          );
          const result = await submitVAA(vaa);
          expect(result.code).toEqual(0);
          const response = await cosmWasmClient.queryContractSmart(
            NTT_GA_ADDRESS,
            {
              all_transceiver_peers: {},
            }
          );
          const peer = response.peers.find(
            (entry) =>
              entry.key[0] === SPOKE_CHAIN_A &&
              entry.key[1] === BSC_WALLET_EMITTER &&
              entry.key[2] === SPOKE_CHAIN_B
          );
          expect(peer).toBeDefined();
          expect(peer.data).toStrictEqual(ETH_WALLET_EMITTER);
        }
        {
          const vaa = makeVAA(
            SPOKE_CHAIN_B,
            ETH_WALLET_EMITTER,
            `18fc67c2${chainToHex(SPOKE_CHAIN_A)}${BSC_WALLET_EMITTER}`
          );
          const result = await submitVAA(vaa);
          expect(result.code).toEqual(0);
          const response = await cosmWasmClient.queryContractSmart(
            NTT_GA_ADDRESS,
            {
              all_transceiver_peers: {},
            }
          );
          const peer = response.peers.find(
            (entry) =>
              entry.key[0] === SPOKE_CHAIN_B &&
              entry.key[1] === ETH_WALLET_EMITTER &&
              entry.key[2] === SPOKE_CHAIN_A
          );
          expect(peer).toBeDefined();
          expect(peer.data).toStrictEqual(BSC_WALLET_EMITTER);
        }
      }
    });
    test("a. Ensure a token can be sent from its hub transceiver", async () => {
      const beforeMetrics = await fetchGlobalAccountantMetrics();
      const beforeEthBalance = await fetchGlobalAccountantBalance(
        HUB_CHAIN,
        HUB_CHAIN,
        ETH_WALLET_EMITTER
      );
      const beforeBscBalance = await fetchGlobalAccountantBalance(
        SPOKE_CHAIN_A,
        HUB_CHAIN,
        ETH_WALLET_EMITTER
      );
      const core = ethers_contracts.Implementation__factory.connect(
        CONTRACTS.DEVNET.ethereum.core,
        ethSigner
      );
      const tx = await core.publishMessage(
        42,
        `0x${mockTransferPayload(8, 10, SPOKE_CHAIN_A)}`,
        200
      );
      const receipt = await tx.wait();
      // get the sequence from the logs (needed to fetch the vaa)
      const sequence = parseSequenceFromLogEth(
        receipt,
        CONTRACTS.DEVNET.ethereum.core
      );
      // poll until the guardian(s) witness and sign the vaa
      const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
        GUARDIAN_RPCS,
        HUB_CHAIN,
        ETH_WALLET_EMITTER,
        sequence,
        {
          transport: NodeHttpTransport(),
        },
        1000,
        30
      );
      await waitForMetricsChange(
        (afterMetrics) =>
          afterMetrics.global_accountant_events_received <=
            beforeMetrics.global_accountant_events_received ||
          afterMetrics.global_accountant_transfer_vaas_submitted <=
            beforeMetrics.global_accountant_transfer_vaas_submitted ||
          afterMetrics.global_accountant_transfer_vaas_submitted_and_approved <=
            beforeMetrics.global_accountant_transfer_vaas_submitted_and_approved
      );
      const transferStatus = await fetchGlobalAccountantTransferStatus(
        HUB_CHAIN,
        ETH_WALLET_EMITTER,
        sequence
      );
      expect(transferStatus).toMatchObject({
        committed: {
          data: {
            amount: "10",
            token_chain: HUB_CHAIN,
            token_address: ETH_WALLET_EMITTER,
            recipient_chain: SPOKE_CHAIN_A,
          },
        },
      });
      const afterEthBalance = await fetchGlobalAccountantBalance(
        HUB_CHAIN,
        HUB_CHAIN,
        ETH_WALLET_EMITTER
      );
      expect(afterEthBalance).toBeGreaterThan(beforeEthBalance.valueOf());
      const afterBscBalance = await fetchGlobalAccountantBalance(
        SPOKE_CHAIN_A,
        HUB_CHAIN,
        ETH_WALLET_EMITTER
      );
      expect(afterBscBalance).toBeGreaterThan(beforeBscBalance.valueOf());
      // check replay protection
      {
        const msg = client.wasm.msgExecuteContract({
          sender: signer,
          contract: NTT_GA_ADDRESS,
          msg: toUtf8(
            JSON.stringify({
              submit_vaas: {
                vaas: [Buffer.from(signedVAA).toString("base64")],
              },
            })
          ),
          funds: [],
        });
        const result = await client.signAndBroadcast(signer, [msg], {
          ...ZERO_FEE,
          gas: "10000000",
        });
        expect(result.code).toEqual(5);
        expect(result.rawLog).toMatch("message already processed");
      }
    });
    test("b. Ensure a token decimal shift works as expected", async () => {
      const beforeMetrics = await fetchGlobalAccountantMetrics();
      const core = ethers_contracts.Implementation__factory.connect(
        CONTRACTS.DEVNET.bsc.core,
        bscSigner
      );
      const tx = await core.publishMessage(
        42,
        `0x${mockTransferPayload(6, 1, HUB_CHAIN)}`,
        200
      );
      const receipt = await tx.wait();
      // get the sequence from the logs (needed to fetch the vaa)
      const sequence = parseSequenceFromLogEth(
        receipt,
        CONTRACTS.DEVNET.bsc.core
      );
      await waitForMetricsChange(
        (afterMetrics) =>
          afterMetrics.global_accountant_error_events_received <=
            beforeMetrics.global_accountant_error_events_received ||
          afterMetrics.global_accountant_transfer_vaas_submitted <=
            beforeMetrics.global_accountant_transfer_vaas_submitted ||
          afterMetrics.global_accountant_total_balance_errors <=
            beforeMetrics.global_accountant_total_balance_errors
      );
      // the transfer should fail, because there's an insufficient source balance
      if (VAA_SIGNERS.length > 1) {
        const transferStatus = await fetchGlobalAccountantTransferStatus(
          SPOKE_CHAIN_A,
          BSC_WALLET_EMITTER,
          sequence
        );
        expect(Object.keys(transferStatus)).toContain("pending");
        expect(Object.keys(transferStatus)).not.toContain("committed");
      } else {
        await expect(
          fetchGlobalAccountantTransferStatus(
            SPOKE_CHAIN_A,
            BSC_WALLET_EMITTER,
            sequence
          )
        ).rejects.toThrow();
      }
    });
    test("c. Ensure a token can be sent back to its hub transceiver", async () => {
      const beforeMetrics = await fetchGlobalAccountantMetrics();
      const beforeEthBalance = await fetchGlobalAccountantBalance(
        HUB_CHAIN,
        HUB_CHAIN,
        ETH_WALLET_EMITTER
      );
      const beforeBscBalance = await fetchGlobalAccountantBalance(
        SPOKE_CHAIN_A,
        HUB_CHAIN,
        ETH_WALLET_EMITTER
      );
      const core = ethers_contracts.Implementation__factory.connect(
        CONTRACTS.DEVNET.bsc.core,
        bscSigner
      );
      const tx = await core.publishMessage(
        42,
        `0x${mockTransferPayload(8, 1, HUB_CHAIN)}`,
        200
      );
      const receipt = await tx.wait();
      // get the sequence from the logs (needed to fetch the vaa)
      const sequence = parseSequenceFromLogEth(
        receipt,
        CONTRACTS.DEVNET.bsc.core
      );
      // poll until the guardian(s) witness and sign the vaa
      await getSignedVAAWithRetry(
        GUARDIAN_RPCS,
        SPOKE_CHAIN_A,
        BSC_WALLET_EMITTER,
        sequence,
        {
          transport: NodeHttpTransport(),
        },
        1000,
        30
      );
      await waitForMetricsChange(
        (afterMetrics) =>
          afterMetrics.global_accountant_events_received <=
            beforeMetrics.global_accountant_events_received ||
          afterMetrics.global_accountant_transfer_vaas_submitted <=
            beforeMetrics.global_accountant_transfer_vaas_submitted ||
          afterMetrics.global_accountant_transfer_vaas_submitted_and_approved <=
            beforeMetrics.global_accountant_transfer_vaas_submitted_and_approved
      );
      const transferStatus = await fetchGlobalAccountantTransferStatus(
        SPOKE_CHAIN_A,
        BSC_WALLET_EMITTER,
        sequence
      );
      expect(transferStatus).toMatchObject({
        committed: {
          data: {
            amount: "1",
            token_chain: HUB_CHAIN,
            token_address: ETH_WALLET_EMITTER,
            recipient_chain: HUB_CHAIN,
          },
        },
      });
      const afterEthBalance = await fetchGlobalAccountantBalance(
        HUB_CHAIN,
        HUB_CHAIN,
        ETH_WALLET_EMITTER
      );
      expect(afterEthBalance).toBeLessThan(beforeEthBalance.valueOf());
      const afterBscBalance = await fetchGlobalAccountantBalance(
        SPOKE_CHAIN_A,
        HUB_CHAIN,
        ETH_WALLET_EMITTER
      );
      expect(afterBscBalance).toBeLessThan(beforeBscBalance.valueOf());
    });
    test("d. Ensure a token can be sent between non-hub transceivers", async () => {
      const beforeMetrics = await fetchGlobalAccountantMetrics();
      const beforeBscBalance = await fetchGlobalAccountantBalance(
        SPOKE_CHAIN_A,
        HUB_CHAIN,
        ETH_WALLET_EMITTER
      );
      const beforePolygonBalance = await fetchGlobalAccountantBalance(
        SPOKE_CHAIN_B,
        HUB_CHAIN,
        ETH_WALLET_EMITTER
      );
      const core = ethers_contracts.Implementation__factory.connect(
        CONTRACTS.DEVNET.bsc.core,
        bscSigner
      );
      const tx = await core.publishMessage(
        42,
        `0x${mockTransferPayload(8, 1, SPOKE_CHAIN_B)}`,
        200
      );
      const receipt = await tx.wait();
      // get the sequence from the logs (needed to fetch the vaa)
      const sequence = parseSequenceFromLogEth(
        receipt,
        CONTRACTS.DEVNET.bsc.core
      );
      // poll until the guardian(s) witness and sign the vaa
      await getSignedVAAWithRetry(
        GUARDIAN_RPCS,
        SPOKE_CHAIN_A,
        BSC_WALLET_EMITTER,
        sequence,
        {
          transport: NodeHttpTransport(),
        },
        1000,
        30
      );
      await waitForMetricsChange(
        (afterMetrics) =>
          afterMetrics.global_accountant_events_received <=
            beforeMetrics.global_accountant_events_received ||
          afterMetrics.global_accountant_transfer_vaas_submitted <=
            beforeMetrics.global_accountant_transfer_vaas_submitted ||
          afterMetrics.global_accountant_transfer_vaas_submitted_and_approved <=
            beforeMetrics.global_accountant_transfer_vaas_submitted_and_approved
      );
      const transferStatus = await fetchGlobalAccountantTransferStatus(
        SPOKE_CHAIN_A,
        BSC_WALLET_EMITTER,
        sequence
      );
      expect(transferStatus).toMatchObject({
        committed: {
          data: {
            amount: "1",
            token_chain: HUB_CHAIN,
            token_address: ETH_WALLET_EMITTER,
            recipient_chain: SPOKE_CHAIN_B,
          },
        },
      });
      const afterBscBalance = await fetchGlobalAccountantBalance(
        SPOKE_CHAIN_A,
        HUB_CHAIN,
        ETH_WALLET_EMITTER
      );
      expect(afterBscBalance).toBeLessThan(beforeBscBalance.valueOf());
      const afterPolygonBalance = await fetchGlobalAccountantBalance(
        SPOKE_CHAIN_B,
        HUB_CHAIN,
        ETH_WALLET_EMITTER
      );
      expect(afterPolygonBalance).toBeGreaterThan(
        beforePolygonBalance.valueOf()
      );
    });
    test("e. Ensure a token sent from a source transceiver without a known hub is rejected", async () => {
      const beforeMetrics = await fetchGlobalAccountantMetrics();
      const core = ethers_contracts.Implementation__factory.connect(
        CONTRACTS.DEVNET.ethereum.core,
        fauxEthSigner
      );
      const tx = await core.publishMessage(
        42,
        `0x${mockTransferPayload(8, 1, SPOKE_CHAIN_A)}`,
        200
      );
      const receipt = await tx.wait();
      // get the sequence from the logs (needed to fetch the vaa)
      const sequence = parseSequenceFromLogEth(
        receipt,
        CONTRACTS.DEVNET.ethereum.core
      );
      await waitForMetricsChange(
        (afterMetrics) =>
          afterMetrics.global_accountant_error_events_received <=
            beforeMetrics.global_accountant_error_events_received ||
          afterMetrics.global_accountant_transfer_vaas_submitted <=
            beforeMetrics.global_accountant_transfer_vaas_submitted
      );
      // the transfer should fail, because there's an insufficient source balance
      await expect(
        fetchGlobalAccountantTransferStatus(
          HUB_CHAIN,
          BSC_WALLET_EMITTER,
          sequence
        )
      ).rejects.toThrow();
    });
    test("f. Ensure a token sent from a source chain without a known transceiver is rejected", async () => {
      {
        // init the locking hub
        const vaa = makeVAA(
          HUB_CHAIN,
          BSC_WALLET_EMITTER,
          "9c23bd3b000000000000000000000000bb807f76cda53b1b4256e1b6f33bb46be36508e3000000000000000000000000002a68f967bfa230780a385175d0c86ae4048d309612"
        );
        const result = await submitVAA(vaa);
        expect(result.code).toEqual(0);
      }
      const beforeMetrics = await fetchGlobalAccountantMetrics();
      const core = ethers_contracts.Implementation__factory.connect(
        CONTRACTS.DEVNET.ethereum.core,
        fauxEthSigner
      );
      const tx = await core.publishMessage(
        42,
        `0x${mockTransferPayload(8, 1, SPOKE_CHAIN_A)}`,
        200
      );
      const receipt = await tx.wait();
      // get the sequence from the logs (needed to fetch the vaa)
      const sequence = parseSequenceFromLogEth(
        receipt,
        CONTRACTS.DEVNET.ethereum.core
      );
      await waitForMetricsChange(
        (afterMetrics) =>
          afterMetrics.global_accountant_error_events_received <=
            beforeMetrics.global_accountant_error_events_received ||
          afterMetrics.global_accountant_transfer_vaas_submitted <=
            beforeMetrics.global_accountant_transfer_vaas_submitted
      );
      // the transfer should fail, because there's an insufficient source balance
      await expect(
        fetchGlobalAccountantTransferStatus(
          HUB_CHAIN,
          BSC_WALLET_EMITTER,
          sequence
        )
      ).rejects.toThrow();
    });
    test("g. Ensure a token sent from a source chain without a matching transceiver is rejected", async () => {
      {
        // set faux spoke registration to hub but not vice-versa
        const vaa = makeVAA(
          FAUX_SPOKE_CHAIN_A,
          ETH_WALLET_EMITTER,
          `18fc67c2${chainToHex(HUB_CHAIN)}${BSC_WALLET_EMITTER}`
        );
        const result = await submitVAA(vaa);
        expect(result.code).toEqual(0);
      }
      const beforeMetrics = await fetchGlobalAccountantMetrics();
      const core = ethers_contracts.Implementation__factory.connect(
        CONTRACTS.DEVNET.bsc.core,
        fauxBscSigner
      );
      const tx = await core.publishMessage(
        42,
        `0x${mockTransferPayload(8, 0, HUB_CHAIN)}`,
        200
      );
      const receipt = await tx.wait();
      // get the sequence from the logs (needed to fetch the vaa)
      const sequence = parseSequenceFromLogEth(
        receipt,
        CONTRACTS.DEVNET.bsc.core
      );
      await waitForMetricsChange(
        (afterMetrics) =>
          afterMetrics.global_accountant_error_events_received <=
            beforeMetrics.global_accountant_error_events_received ||
          afterMetrics.global_accountant_transfer_vaas_submitted <=
            beforeMetrics.global_accountant_transfer_vaas_submitted
      );
      // the transfer should fail, because there's an insufficient source balance
      await expect(
        fetchGlobalAccountantTransferStatus(
          FAUX_SPOKE_CHAIN_A,
          ETH_WALLET_EMITTER,
          sequence
        )
      ).rejects.toThrow();
    });
    test("h. Ensure a token sent to a destination chain without a known transceiver is rejected", async () => {
      const beforeMetrics = await fetchGlobalAccountantMetrics();
      const core = ethers_contracts.Implementation__factory.connect(
        CONTRACTS.DEVNET.ethereum.core,
        ethSigner
      );
      const tx = await core.publishMessage(
        42,
        `0x${mockTransferPayload(8, 1, UNKNOWN_SPOKE_CHAIN)}`,
        200
      );
      const receipt = await tx.wait();
      // get the sequence from the logs (needed to fetch the vaa)
      const sequence = parseSequenceFromLogEth(
        receipt,
        CONTRACTS.DEVNET.ethereum.core
      );
      await waitForMetricsChange(
        (afterMetrics) =>
          afterMetrics.global_accountant_error_events_received <=
            beforeMetrics.global_accountant_error_events_received ||
          afterMetrics.global_accountant_transfer_vaas_submitted <=
            beforeMetrics.global_accountant_transfer_vaas_submitted
      );
      // the transfer should fail, because there's an insufficient source balance
      await expect(
        fetchGlobalAccountantTransferStatus(
          HUB_CHAIN,
          ETH_WALLET_EMITTER,
          sequence
        )
      ).rejects.toThrow();
    });
    // test i. is the same case as h.
    test("j. Ensure spoofed tokens for more than the outstanding amount rejects successfully", async () => {
      const beforeMetrics = await fetchGlobalAccountantMetrics();
      const core = ethers_contracts.Implementation__factory.connect(
        CONTRACTS.DEVNET.bsc.core,
        bscSigner
      );
      const tx = await core.publishMessage(
        42,
        `0x${mockTransferPayload(8, 9000, HUB_CHAIN)}`,
        200
      );
      const receipt = await tx.wait();
      // get the sequence from the logs (needed to fetch the vaa)
      const sequence = parseSequenceFromLogEth(
        receipt,
        CONTRACTS.DEVNET.bsc.core
      );
      await waitForMetricsChange(
        (afterMetrics) =>
          afterMetrics.global_accountant_error_events_received <=
            beforeMetrics.global_accountant_error_events_received ||
          afterMetrics.global_accountant_transfer_vaas_submitted <=
            beforeMetrics.global_accountant_transfer_vaas_submitted ||
          afterMetrics.global_accountant_total_balance_errors <=
            beforeMetrics.global_accountant_total_balance_errors
      );
      // the transfer should fail, because there's an insufficient source balance
      if (VAA_SIGNERS.length > 1) {
        const transferStatus = await fetchGlobalAccountantTransferStatus(
          SPOKE_CHAIN_A,
          BSC_WALLET_EMITTER,
          sequence
        );
        expect(Object.keys(transferStatus)).toContain("pending");
        expect(Object.keys(transferStatus)).not.toContain("committed");
      } else {
        await expect(
          fetchGlobalAccountantTransferStatus(
            SPOKE_CHAIN_A,
            BSC_WALLET_EMITTER,
            sequence
          )
        ).rejects.toThrow();
      }
    });
    test("k. Relayed message gets accounted", async () => {
      const beforeMetrics = await fetchGlobalAccountantMetrics();
      const beforeEthBalance = await fetchGlobalAccountantBalance(
        HUB_CHAIN,
        HUB_CHAIN,
        ETH_WALLET_EMITTER
      );
      const beforeBscBalance = await fetchGlobalAccountantBalance(
        SPOKE_CHAIN_A,
        HUB_CHAIN,
        ETH_WALLET_EMITTER
      );
      const relayerOptionalParameters: GetPriceOptParams = {
        environment: "DEVNET",
        wormholeRelayerAddress: RELAYER_ADDRESS,
        sourceChainProvider: ethProvider,
      };
      const value = await getPrice(
        "ethereum",
        "bsc",
        0,
        relayerOptionalParameters
      );
      const tx = await sendToEvm(
        ethSigner,
        "ethereum",
        "bsc",
        BSC_WALLET.public,
        `0x${mockTransferPayload(8, 10, SPOKE_CHAIN_A)}`,
        0,
        { value },
        { ...relayerOptionalParameters, consistencyLevel: 200 }
      );
      const receipt = await tx.wait();
      // get the sequence from the logs (needed to fetch the vaa)
      const sequence = parseSequenceFromLogEth(
        receipt,
        CONTRACTS.DEVNET.ethereum.core
      );
      // poll until the guardian(s) witness and sign the vaa
      const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
        GUARDIAN_RPCS,
        HUB_CHAIN,
        RELAYER_EMITTER,
        sequence,
        {
          transport: NodeHttpTransport(),
        },
        1000,
        30
      );
      await waitForMetricsChange(
        (afterMetrics) =>
          afterMetrics.global_accountant_events_received <=
            beforeMetrics.global_accountant_events_received ||
          afterMetrics.global_accountant_transfer_vaas_submitted <=
            beforeMetrics.global_accountant_transfer_vaas_submitted ||
          afterMetrics.global_accountant_transfer_vaas_submitted_and_approved <=
            beforeMetrics.global_accountant_transfer_vaas_submitted_and_approved
      );
      const transferStatus = await fetchGlobalAccountantTransferStatus(
        HUB_CHAIN,
        RELAYER_EMITTER,
        sequence
      );
      expect(transferStatus).toMatchObject({
        committed: {
          data: {
            amount: "10",
            token_chain: HUB_CHAIN,
            token_address: ETH_WALLET_EMITTER,
            recipient_chain: SPOKE_CHAIN_A,
          },
        },
      });
      const afterEthBalance = await fetchGlobalAccountantBalance(
        HUB_CHAIN,
        HUB_CHAIN,
        ETH_WALLET_EMITTER
      );
      expect(afterEthBalance).toBeGreaterThan(beforeEthBalance.valueOf());
      const afterBscBalance = await fetchGlobalAccountantBalance(
        SPOKE_CHAIN_A,
        HUB_CHAIN,
        ETH_WALLET_EMITTER
      );
      expect(afterBscBalance).toBeGreaterThan(beforeBscBalance.valueOf());
      // check replay protection
      {
        const msg = client.wasm.msgExecuteContract({
          sender: signer,
          contract: NTT_GA_ADDRESS,
          msg: toUtf8(
            JSON.stringify({
              submit_vaas: {
                vaas: [Buffer.from(signedVAA).toString("base64")],
              },
            })
          ),
          funds: [],
        });
        const result = await client.signAndBroadcast(signer, [msg], {
          ...ZERO_FEE,
          gas: "10000000",
        });
        expect(result.code).toEqual(5);
        expect(result.rawLog).toMatch("message already processed");
      }
    });
  });
});
