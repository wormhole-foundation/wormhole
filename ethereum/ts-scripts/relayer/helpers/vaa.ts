import * as wh from "@wormhole-foundation/sdk";
import { ethers } from "ethers";
import {
  ChainInfo,
  getWormholeRelayerAddress,
  getDeliveryProviderAddress,
  loadGuardianKeys,
  loadGuardianSetIndex,
} from "./env";
import { nativeEvmAddressToHex } from "./utils";
const elliptic = require("elliptic");

const governanceChainId = 1;
const governanceContract =
  "0x0000000000000000000000000000000000000000000000000000000000000004";
//don't use the variable module in global scope in node
const wormholeRelayerModule =
  "0x0000000000000000000000000000000000576f726d686f6c6552656c61796572";


export function createWormholeRelayerUpgradeVAA(
  chain: ChainInfo,
  newAddress: string,
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
      nativeEvmAddressToHex(newAddress)
    ],
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
      nativeEvmAddressToHex(getDeliveryProviderAddress(chain))
    ],
  );

  return encodeAndSignGovernancePayload(payload);
}

export async function createRegisterChainVAA(
  chain: ChainInfo,
): Promise<string> {
  const coreRelayerAddress = await getWormholeRelayerAddress(chain);
  console.log(
    `Creating registration VAA for Wormhole Relayer ${coreRelayerAddress} (chain ${chain.chainId})`,
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
      nativeEvmAddressToHex(coreRelayerAddress)
    ],
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
    ],
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

export function extractChainToBeRegisteredFromRegisterChainVaa(vaa: Buffer): number {
  // Structure of a register chain Vaa
  // version: uint8 <-- should be 1
  // guardianSetIndex: uint32
  // signaturesLength: uint8
  // signatures: bytes66[signaturesLength]
  // timestamp: uint32
  // nonce: uint32
  // emitterChainId: uint16 <-- should be wh governance (solana chain id)
  // emitterContract: bytes32 <-- should be wh governance
  // sequence: uint64
  // consistencyLevel: uint8
  // module: bytes32 <-- should be wormhole relayer
  // action: uint8 <-- should be register chain
  // chain: uint16 <-- should be broadcast
  // emitterChain: uint16 <-- need to extract
  // emitterAddress: bytes 32
  const uint8Size = 1;
  const uint16Size = uint8Size * 2;
  const uint32Size = uint8Size * 4;
  const uint64Size = uint8Size * 8;
  const bytes32Size = 32;
  // Each signature has the guardian index in one byte (uint8) and (r, s, v) tuple in 65 bytes
  const signatureSize = 66;

  const governanceChain = 1;
  const governanceContract =
    "0x0000000000000000000000000000000000000000000000000000000000000004";
  // See WormholeRelayerGovernance.sol
  const GOVERNANCE_ACTION_REGISTER_WORMHOLE_RELAYER_CONTRACT = 1;
  const TARGET_CHAIN_BROADCAST = 0;

  // We'll do some very basic sanity checks
  // We won't verify signatures here
  const version = vaa.readUint8(0);
  if (version !== 1) {
    throw new Error("Unknown VAA version ${version}");
  }

  const signaturesOffset = uint8Size + uint32Size;
  const signaturesLength = vaa.readUint8(signaturesOffset);

  const timestampOffset =
    signaturesOffset + uint8Size + signaturesLength * signatureSize;
  const emitterChainIdOffset = timestampOffset + uint32Size * 2;
  const emitterChainId = vaa.readUint16BE(emitterChainIdOffset);
  const emitterContractOffset = emitterChainIdOffset + uint16Size;
  const emitterContract = vaa.subarray(
    emitterContractOffset,
    emitterContractOffset + bytes32Size,
  );
  if (emitterChainId !== governanceChain) {
    throw new Error(
      `VAA initiated by incorrect chain. Expected chain ${governanceChain} but found chain ${emitterChainId}`,
    );
  }
  if (
    !emitterContract.equals(Buffer.from(governanceContract.substring(2), "hex"))
  ) {
    throw new Error(
      `VAA initiated by incorrect contract. Expected contract ${governanceContract} but found contract ${
        "0x" + emitterContract.toString("hex")
      }`,
    );
  }

  const moduleOffset =
    emitterContractOffset + bytes32Size + uint64Size + uint8Size;
  const moduleBuf = vaa.subarray(moduleOffset, moduleOffset + bytes32Size);
  if (
    !moduleBuf.equals(Buffer.from(wormholeRelayerModule.substring(2), "hex"))
  ) {
    throw new Error(
      `Unexpected governance module ${"0x" + moduleBuf.toString("hex")}`,
    );
  }

  const actionOffset = moduleOffset + bytes32Size;
  const action = vaa.readUint8(actionOffset);
  if (action !== GOVERNANCE_ACTION_REGISTER_WORMHOLE_RELAYER_CONTRACT) {
    throw new Error(
      `Unexpected wormhole relayer governance action id ${action}`,
    );
  }

  const governanceTargetChainOffset = actionOffset + uint8Size;
  const governanceTargetChain = vaa.readUint16BE(governanceTargetChainOffset);
  if (governanceTargetChain !== TARGET_CHAIN_BROADCAST) {
    throw new Error(
      `Expected the register chain VAA to be addressed as a broadcast to all chains but it is addressed to chain ${governanceTargetChain} instead`,
    );
  }

  const chainToRegisterOffset = governanceTargetChainOffset + uint16Size;
  return vaa.readUint16BE(chainToRegisterOffset);
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
