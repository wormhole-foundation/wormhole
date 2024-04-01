import { ethers } from "ethers";
import { tryNativeToHexString } from "@certusone/wormhole-sdk";
import {
  ChainInfo,
  getWormholeRelayerAddress,
  getDeliveryProviderAddress,
  loadGuardianKeys,
  loadGuardianSetIndex,
} from "./env";
const elliptic = require("elliptic");

const governanceChainId = 1;
const governanceContract =
  "0x0000000000000000000000000000000000000000000000000000000000000004";
//don't use the variable module in global scope in node
const wormholeRelayerModule =
  "0x0000000000000000000000000000000000576f726d686f6c6552656c61796572";

export function createWormholeRelayerUpgradeVAA(
  chain: ChainInfo,
  newAddress: string
) {
  /*
      bytes32 module;
        uint8 action;
        uint16 chain;
        bytes32 newContract; //listed as address in the struct, but is actually bytes32 inside the VAA
      */

  const payload = ethers.utils.solidityPack(
    ["bytes32", "uint8", "uint16", "bytes32"],
    [
      wormholeRelayerModule,
      2,
      chain.chainId,
      "0x" + tryNativeToHexString(newAddress, "ethereum"),
    ]
  );

  return encodeAndSignGovernancePayload(payload);
}

export function createDefaultDeliveryProviderVAA(chain: ChainInfo) {
  /*
    bytes32 module;
    uint8 action;
    uint16 chain;
    bytes32 newProvider; //Struct in the contract is an address, wire type is a wh format 32
    */

  const payload = ethers.utils.solidityPack(
    ["bytes32", "uint8", "uint16", "bytes32"],
    [
      wormholeRelayerModule,
      3,
      chain.chainId,
      "0x" +
        tryNativeToHexString(getDeliveryProviderAddress(chain), "ethereum"),
    ]
  );

  return encodeAndSignGovernancePayload(payload);
}

export async function createRegisterChainVAA(
  chain: ChainInfo
): Promise<string> {
  const coreRelayerAddress = await getWormholeRelayerAddress(chain);
  console.log(
    `Creating registration VAA for Wormhole Relayer ${coreRelayerAddress} (chain ${chain.chainId})`
  );

  // bytes32 module;
  // uint8 action;
  // uint16 chain; //0
  // uint16 emitterChain;
  // bytes32 emitterAddress;

  const payload = ethers.utils.solidityPack(
    ["bytes32", "uint8", "uint16", "uint16", "bytes32"],
    [
      wormholeRelayerModule,
      1,
      0,
      chain.chainId,
      "0x" + tryNativeToHexString(coreRelayerAddress, "ethereum"),
    ]
  );

  return encodeAndSignGovernancePayload(payload);
}

export function encodeAndSignGovernancePayload(payload: string): string {
  const timestamp = Math.floor(+new Date() / 1000);
  const nonce = 1;
  const sequence = 1;
  const consistencyLevel = 1;

  const encodedVAABody = ethers.utils.solidityPack(
    ["uint32", "uint32", "uint16", "bytes32", "uint64", "uint8", "bytes"],
    [
      timestamp,
      nonce,
      governanceChainId,
      governanceContract,
      sequence,
      consistencyLevel,
      payload,
    ]
  );

  const hash = doubleKeccak256(encodedVAABody);

  const signers = loadGuardianKeys();
  let signatures = "";

  for (const i in signers) {
    // sign the hash
    const ec = new elliptic.ec("secp256k1");
    const key = ec.keyFromPrivate(signers[i]);
    const signature = key.sign(hash.substring(2), { canonical: true });

    // pack the signatures
    const packSig = [
      ethers.utils.solidityPack(["uint8"], [i]).substring(2),
      zeroPadBytes(signature.r.toString(16), 32),
      zeroPadBytes(signature.s.toString(16), 32),
      ethers.utils
        .solidityPack(["uint8"], [signature.recoveryParam])
        .substring(2),
    ];
    signatures += packSig.join("");
  }

  const vm = [
    ethers.utils.solidityPack(["uint8"], [1]).substring(2),
    ethers.utils
      .solidityPack(["uint32"], [loadGuardianSetIndex()])
      .substring(2), // guardianSetIndex
    ethers.utils.solidityPack(["uint8"], [signers.length]).substring(2), // number of signers
    signatures,
    encodedVAABody.substring(2),
  ].join("");

  return "0x" + vm;
}

export function doubleKeccak256(body: ethers.BytesLike) {
  return ethers.utils.keccak256(ethers.utils.keccak256(body));
}

export function zeroPadBytes(value: string, length: number): string {
  while (value.length < 2 * length) {
    value = "0" + value;
  }
  return value;
}
