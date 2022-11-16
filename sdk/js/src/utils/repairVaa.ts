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

export async function getGuardianSets(
  provider: ethers.providers.JsonRpcProvider
) {
  let result: any = {};
  const core = Implementation__factory.connect(ETHEREUM_CORE_BRIDGE, provider);
  (result.address = ETHEREUM_CORE_BRIDGE),
    (result.currentGuardianSetIndex = await core.getCurrentGuardianSetIndex());
  result.guardianSet = {};
  for (let i of Array(result.currentGuardianSetIndex + 1).keys()) {
    let guardian_set = await core.getGuardianSet(i);
    result.guardianSet[i] = { keys: guardian_set[0], expiry: guardian_set[1] };
  }
  return result;
}

export async function repairVaa(vaaHex: string, rpcUrl: string) {
  const provider = new ethers.providers.JsonRpcProvider(rpcUrl);
  const guardianSetsInfo = await getGuardianSets(provider);
  const currentGuardianSetIndex = guardianSetsInfo.currentGuardianSetIndex;
  const guardianSets = guardianSetsInfo?.guardianSet;
  const guardianSetIndexes = Object.keys(guardianSets);
  const vaaGuardianSetIndex = parseInt(vaaHex.slice(2, 10));
  if (vaaGuardianSetIndex === parseInt(currentGuardianSetIndex)) {
    console.log("Vaa has current guardian set index. no repair needed");
    return vaaHex;
  } else if (guardianSetIndexes.includes(vaaGuardianSetIndex.toString())) {
    console.log(
      `Vaa is using old GS index=${vaaGuardianSetIndex}. Will attempt to update to current GS Index=${currentGuardianSetIndex}...\n`
    );

    const currentGuardianSet = guardianSets[currentGuardianSetIndex].keys;
    const minNumSignatures =
      Math.floor((2.0 * currentGuardianSet.length) / 3.0) + 1;
    // console.log(`Current GS: #${currentGuardianSetIndex}`, currentGuardianSet);
    const numSignatures = parseInt(vaaHex.slice(10, 12), 16);
    const version = vaaHex.slice(0, 2);
    console.log(
      `There are ${numSignatures} signatures found in vaa. There are ${currentGuardianSet.length} Guardians. We need at least ${minNumSignatures}.\n`
    );
    try {
      const parsed_vaa = parseVaa(hexToUint8Array(vaaHex));

      const parsedVaaSignatureLength = parsed_vaa.guardianSignatures.length;
      if (parsedVaaSignatureLength !== numSignatures) {
        console.error("# of parsed signatures does not match signature length");
      }
      const digest = keccak256(parsed_vaa.hash).toString("hex");

      var validSignatures: GuardianSignature[] = [];
      // take each signature, check if valid against hash & current guardian set
      // if valid, keep
      // if invalid, discard
      parsed_vaa.guardianSignatures.forEach((signature) => {
        try {
          const vaaGuardianPublicKey = ethers.utils.recoverAddress(
            hex(digest),
            hex(signature.signature.toString("hex"))
          );

          const currentIndex = signature.index;
          const currentGuardianPublicKey = currentGuardianSet[currentIndex];

          if (currentGuardianPublicKey === vaaGuardianPublicKey) {
            console.log(
              `found a match for gs index=${currentIndex}, public key=${currentGuardianPublicKey.toString(
                "hex"
              )}`
            );
            validSignatures.push(signature);
          } else {
            console.error(
              `did not find a match for gs index=${currentIndex}, vaaGuardianPublicKey=${vaaGuardianPublicKey}, current public key=${currentGuardianPublicKey}`
            );
          }
        } catch (err) {
          console.error(
            `could not recover address for gs index=${
              signature.index
            }, public key=${signature.signature.toString("hex")}`
          );
        }
      });

      // re-construct vaa with signatures that remain
      const numRepairedSignatures = validSignatures.length;
      if (numRepairedSignatures < minNumSignatures) {
        console.error(
          `There are ${numRepairedSignatures} signatures remaining. Not enough to repair.`
        );
        return vaaHex;
      } else {
        console.log(
          `There are ${numRepairedSignatures} signatures remaining. We have enough to repair...\n`
        );
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
        const newSignatureBody = `${version}${currentGuardianSetIndex
          .toString(16)
          .padStart(8, "0")}${numRepairedSignatures
          .toString(16)
          .padStart(2, "0")}${repairedSignatures}`;

        const repairedVaa = `${newSignatureBody}${vaaHex.slice(
          12 + numSignatures * 132
        )}`;
        return repairedVaa;
      }
    } catch (e) {
      console.error("Could not parse vaa");
      return vaaHex;
    }
  } else {
    console.log(`could not find vaa GS index=${vaaGuardianSetIndex}`);
    return vaaHex;
  }
}
