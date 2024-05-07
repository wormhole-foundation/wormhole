// The edit-vaa command allows the user to create, update or sign a VAA. It queries the core contract on Ethereum
// to get the guardian set. It can take signature data from wormscan or (in the case of testnet or devnet) it can
// take a guardian secret as input.
//
// Sign a VAA using signatures from wormscan:
//   worm edit-vaa -n mainnet --vaa $VAA --wormscanurl https://api.wormholescan.io/api/v1/observations/1/0000000000000000000000000000000000000000000000000000000000000004/651169458827220885
//
// Create the same VAA from scratch:
//   worm edit-vaa -n mainnet \
//     --ec 1 --ea 0x0000000000000000000000000000000000000000000000000000000000000004 \
//     --gsi 3 --sequence 651169458827220885 --nonce 2166843495 --cl 32 \
//     --payload 000000000000000000000000000000436972636c65496e746567726174696f6e020002000600000000000000000000000009fb06a271faff70a651047395aaeb6265265f1300000001 \
//     --wormscanurl https://api.wormholescan.io/api/v1/observations/1/0000000000000000000000000000000000000000000000000000000000000004/651169458827220885
//
// Sign a VAA using the testnet guardian key:
//   worm edit-vaa --vaa $VAA --gs $TESTNET_GUARDIAN_SECRET
//

import { Implementation__factory } from "@certusone/wormhole-sdk/lib/esm/ethers-contracts";
import { CONTRACTS } from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { Other } from "@certusone/wormhole-sdk/lib/esm/vaa";
import axios from "axios";
import { ethers } from "ethers";
import yargs from "yargs";
import { NETWORK_OPTIONS, NETWORKS } from "../consts";
import { assertNetwork, Network } from "../utils";
import { parse, Payload, serialiseVAA, sign, Signature, VAA } from "../vaa";

export const command = "edit-vaa";
export const desc = "Edits or generates a VAA";
export const builder = (y: typeof yargs) =>
  y
    .option("vaa", {
      alias: "v",
      describe: "vaa in hex format",
      type: "string",
      demandOption: true,
    })
    .option("network", NETWORK_OPTIONS)
    .option("guardian-set-index", {
      alias: "gsi",
      describe: "guardian set index",
      type: "number",
    })
    .option("signatures", {
      alias: "sigs",
      describe: "comma separated list of signatures",
      type: "string",
    })
    .option("wormscanurl", {
      alias: "wsu",
      describe: "url to wormscan entry for the vaa that includes signatures",
      type: "string",
    })
    .option("wormscan", {
      alias: "ws",
      describe:
        "if specified, will query the wormscan entry for the vaa to get the signatures",
      type: "boolean",
    })
    .option("emitter-chain-id", {
      alias: "ec",
      describe: "emitter chain id to be used in the vaa",
      type: "number",
      demandOption: false,
    })
    .option("emitter-address", {
      alias: "ea",
      describe: "emitter address to be used in the vaa",
      type: "string",
    })
    .option("nonce", {
      alias: "no",
      describe: "nonce to be used in the vaa",
      type: "number",
    })
    .option("sequence", {
      alias: "seq",
      describe: "sequence number to be used in the vaa",
      type: "string",
    })
    .option("consistency-level", {
      alias: "cl",
      describe: "consistency level to be used in the vaa",
      type: "number",
    })
    .option("timestamp", {
      alias: "ts",
      describe: "timestamp to be used in the vaa in unix seconds",
      type: "number",
    })
    .option("payload", {
      alias: "p",
      describe: "payload in hex format",
      type: "string",
    })
    .option("guardian-secret", {
      alias: "gs",
      describe: "Guardian's secret key",
      type: "string",
    });
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  const network = argv.network.toUpperCase();
  assertNetwork(network);

  let numSigs = 0;
  if (argv.signatures) {
    numSigs += 1;
  }

  if (argv.wormscan) {
    numSigs += 1;
  }

  if (argv.wormscanurl) {
    numSigs += 1;
  }

  if (argv["guardian-secret"]) {
    numSigs += 1;
  }

  if (numSigs > 1) {
    throw new Error(
      `may only specify one of "--signatures", "--wormscan", "--wormscanurl" or "--guardian-secret"`
    );
  }

  let vaa: VAA<Payload | Other>;
  if (argv.vaa) {
    let buf: Buffer;
    try {
      buf = Buffer.from(String(argv.vaa), "hex");
      if (buf.length == 0) {
        throw Error("Couldn't parse VAA as hex");
      }
    } catch (e) {
      buf = Buffer.from(String(argv.vaa), "base64");
      if (buf.length == 0) {
        throw Error("Couldn't parse VAA as base64 or hex");
      }
    }
    vaa = parse(buf);
  } else {
    vaa = {
      version: 1,
      guardianSetIndex: 0,
      signatures: [],
      timestamp: 0,
      nonce: 0,
      emitterChain: 0,
      emitterAddress: "0x0",
      sequence: BigInt(Math.floor(Math.random() * 100000000)),
      consistencyLevel: 0,
      payload: {
        type: "Other",
        hex: `00`,
      },
    };
  }

  if (argv["guardian-set-index"]) {
    vaa.guardianSetIndex = Number(argv["guardian-set-index"]);
  }

  if (argv.signatures) {
    vaa.signatures = argv.signatures.split(",").map((s, i) => ({
      signature: s,
      guardianSetIndex: i,
    }));
  } else if (argv.wormscan) {
    const wormscanurl =
      "https://api.wormholescan.io/api/v1/observations/" +
      vaa.emitterChain.toString() +
      "/" +
      vaa.emitterAddress.replace(/^(0x)/, "") +
      "/" +
      vaa.sequence.toString();
    const wormscanData = await axios.get(wormscanurl);
    const guardianSet = await getGuardianSet(network, vaa.guardianSetIndex);
    vaa.signatures = await getSigsFromWormscanData(
      wormscanData.data,
      guardianSet
    );
  } else if (argv.wormscanurl) {
    const wormscanData = await axios.get(argv.wormscanurl);
    const guardianSet = await getGuardianSet(network, vaa.guardianSetIndex);
    vaa.signatures = await getSigsFromWormscanData(
      wormscanData.data,
      guardianSet
    );
  } else if (argv["guardian-secret"]) {
    vaa.guardianSetIndex = 0;
    vaa.signatures = sign([argv["guardian-secret"]], vaa as VAA<Payload>);
  }

  if (argv["emitter-chain-id"]) {
    vaa.emitterChain = argv["emitter-chain-id"];
  }

  if (argv["emitter-address"]) {
    vaa.emitterAddress = argv["emitter-address"];
  }

  if (argv.nonce) {
    vaa.nonce = argv.nonce;
  }

  if (argv.sequence) {
    vaa.sequence = BigInt(argv.sequence);
  }

  if (argv["consistency-level"]) {
    vaa.consistencyLevel = argv["consistency-level"];
  }

  if (argv.timestamp) {
    vaa.timestamp = argv.timestamp;
  }

  if (argv["payload"]) {
    vaa.payload = {
      type: "Other",
      hex: argv["payload"],
    };
  }

  console.log(serialiseVAA(vaa as unknown as VAA<Payload>));
};

// getGuardianSet queries the core contract on Ethereum for the guardian set and returns it.
const getGuardianSet = async (
  network: Network,
  guardianSetIndex: number
): Promise<string[]> => {
  let n = NETWORKS[network].ethereum;
  let contract_address = CONTRACTS[network].ethereum.core;
  if (contract_address === undefined) {
    throw Error(`Unknown core contract on ${network} for ethereum`);
  }

  const provider = new ethers.providers.JsonRpcProvider(n.rpc);
  const contract = Implementation__factory.connect(contract_address, provider);
  const result = await contract.getGuardianSet(guardianSetIndex);
  return result[0];
};

// getSigsFromWormscanData reads the guardian address / signature pairs from the wormscan data
// and generates an array of signature objects. It then sorts them into order by address.
const getSigsFromWormscanData = (
  wormscanData: any,
  guardianSet: string[]
): Signature[] => {
  let sigs: Signature[] = [];
  for (let data in wormscanData) {
    let guardianAddr = wormscanData[data].guardianAddr;
    let gsi = -1;
    for (let idx = 0; idx < guardianSet.length; idx++) {
      if (guardianSet[idx] === guardianAddr) {
        gsi = idx;
        break;
      }
    }
    if (gsi < 0) {
      console.warn(
        "Failed to look up guardian address " + guardianAddr + ". Skipping."
      );
      continue;
    }
    let sig: Signature = {
      guardianSetIndex: gsi,
      signature: Buffer.from(wormscanData[data].signature, "base64").toString(
        "hex"
      ),
    };

    sigs.push(sig);
  }

  return sigs.sort((s1, s2) => {
    if (s1.guardianSetIndex > s2.guardianSetIndex) {
      return 1;
    }

    if (s1.guardianSetIndex < s2.guardianSetIndex) {
      return -1;
    }

    return 0;
  });
};
