import {
  ChainId,
  Other,
  Payload,
  VAA,
  serialiseVAA,
  sign,
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

jest.setTimeout(60000);

if (process.env.INIT_SIGNERS_KEYS_CSV === "undefined") {
  let msg = `.env is missing. run "make contracts-tools-deps" to fetch.`;
  console.error(msg);
  throw msg;
}

// for now, this test only submits locally signed VAAs directly to the contract
// once the guardian is working with the accountant contract, the observation tests should be implemented

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
 *   3. Observations
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
 *   4. Transfer VAAs
 *      a-i. Repeat Observation tests
 *   5. Relayers
 *      a. Ensure a relayer registration is saved
 *      b. Ensure a valid NTT transfer works
 *      c. Ensure an invalid NTT transfer rejects
 *      d. Ensure an invalid payload reverts
 *      e. Ensure a non-delivery reverts
 *   6. Validate the guardian metrics for Observations a-i
 *   7. Bonus: Validate the on chain contract state via queries
 */

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

const HUB_CHAIN = 2;
const HUB_TRANSCEIVER = `0000000000000000000000000000000000000000000000000000000000000042`;
const SPOKE_CHAIN_A = 4;
const SPOKE_TRANSCEIVER_A = `0000000000000000000000000000000000000000000000000000000000000043`;
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
const RELAYER_EMITTER =
  "00000000000000000000000053855d4b64e9a3cf59a84bc768ada716b5536bc5";
const dummy32 = `0000000000000000000000000000000000000000000000000000000000001234`;

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

const host = ci
  ? devnetConsts.chains[3104].tendermintUrlTilt
  : devnetConsts.chains[3104].tendermintUrlLocal;
// TODO: have a mnemonic dedicated for this test
const mnemonic =
  devnetConsts.chains[3104].accounts.wormchainNodeOfGuardian0.mnemonic;

let client: any;
let signer: string;
let cosmWasmClient: CosmWasmClient;

beforeAll(async () => {
  const wallet = await getWallet(mnemonic);
  client = await getWormchainSigningClient(host, wallet);
  const signers = await wallet.getAccounts();
  signer = signers[0].address;
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
      await cosmWasmClient.queryContractSmart(NTT_GA_ADDRESS, {
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
  const seq = `0`.padStart(16, "0");
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

describe("Global Accountant Tests", () => {
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
  describe("4. Transfer VAAs", () => {
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
  });
  describe("5. Relayers", () => {
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
});
