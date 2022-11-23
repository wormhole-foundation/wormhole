import { ethers } from "ethers";
import { CONTRACTS } from "./consts";
import { Implementation__factory } from "../ethers-contracts";
import { parseVaa, GuardianSignature } from "../vaa";
import { hexToUint8Array } from "./array";
import { keccak256 } from "../utils";

const ETHEREUM_CORE_BRIDGE = CONTRACTS["MAINNET"].ethereum.core;

function hex(x: string): string {
  return ethers.utils.hexlify(x, { allowMissingPrefix: true });
}
interface GuardianSetData {
  index: number;
  keys: string[];
  expiry: number;
}

export async function getCurrentGuardianSet(
  provider: ethers.providers.JsonRpcProvider
): Promise<GuardianSetData> {
  let result: GuardianSetData = {
    index: 0,
    keys: [],
    expiry: 0,
  };
  const core = Implementation__factory.connect(ETHEREUM_CORE_BRIDGE, provider);
  const index = await core.getCurrentGuardianSetIndex();
  const guardianSet = await core.getGuardianSet(index);
  result.index = index;
  result.keys = guardianSet[0];
  result.expiry = guardianSet[1];
  return result;
}

/**
 *
 * Takes in a hexstring representation of a signed vaa and a guardian set.
 * Attempts to remove invalid guardian signatures, update total remaining
 * valid signatures, and update the guardian set index
 * @throws if not enough valid signatures remain
 **/

export function repairVaa(
  vaaHex: string,
  guardianSetData: GuardianSetData
): string {
  const guardianSetIndex = guardianSetData.index;
  const currentGuardianSet = guardianSetData.keys;
  const minNumSignatures =
    Math.floor((2.0 * currentGuardianSet.length) / 3.0) + 1;
  const version = vaaHex.slice(0, 2);
  const parsedVaa = parseVaa(hexToUint8Array(vaaHex));
  const numSignatures = parsedVaa.guardianSignatures.length;
  const digest = keccak256(parsedVaa.hash).toString("hex");

  var validSignatures: GuardianSignature[] = [];

  // take each signature, check if valid against hash & current guardian set
  parsedVaa.guardianSignatures.forEach((signature) => {
    try {
      const vaaGuardianPublicKey = ethers.utils.recoverAddress(
        hex(digest),
        hex(signature.signature.toString("hex"))
      );
      const currentIndex = signature.index;
      const currentGuardianPublicKey = currentGuardianSet[currentIndex];

      if (currentGuardianPublicKey === vaaGuardianPublicKey) {
        validSignatures.push(signature);
      }
    } catch (_) {}
  });

  // re-construct vaa with signatures that remain
  const numRepairedSignatures = validSignatures.length;
  if (numRepairedSignatures < minNumSignatures) {
    throw new Error(`There are not enough valid signatures to repair.`);
  }
  const repairedSignatures = validSignatures
    .sort(function (a, b) {
      return a.index - b.index;
    })
    .map((signature) => {
      return `${signature.index
        .toString(16)
        .padStart(2, "0")}${signature.signature.toString("hex")}`;
    })
    .join("");
  const newSignatureBody = `${version}${guardianSetIndex
    .toString(16)
    .padStart(8, "0")}${numRepairedSignatures
    .toString(16)
    .padStart(2, "0")}${repairedSignatures}`;

  const repairedVaa = `${newSignatureBody}${vaaHex.slice(
    12 + numSignatures * 132
  )}`;
  return repairedVaa;
}

/**
 *
 * Takes in a hexstring representation of a signed vaa and an eth provider.
 * Attempts to query eth core contract and retrieve current guardian set.
 * Then attempts to repair the vaa.
 **/

export async function repairVaaWithCurrentGuardianSet(
  vaaHex: string,
  provider: ethers.providers.JsonRpcProvider
): Promise<string> {
  const guardianSetData = await getCurrentGuardianSet(provider);
  return repairVaa(vaaHex, guardianSetData);
}
