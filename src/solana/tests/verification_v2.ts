import assert from "assert"

import * as anchor from "@coral-xyz/anchor"
import { Keypair, PublicKey } from "@solana/web3.js"
import { toUniversal, UniversalAddress } from "@wormhole-foundation/sdk-definitions"
import { getPublicKey, sign } from "@noble/secp256k1"
import { keccak_256 } from "@noble/hashes/sha3"
import { randomBytes } from "@noble/hashes/utils"

// import { VerificationV2 } from "../target/types/verification_v2.js"

import { guardianAddress, TestingWormholeCore } from "./testing-wormhole-core.js"
import { WormholeContracts, TestsHelper } from "./testing_helpers.js"

const $ = new TestsHelper()

const encodeU16BE = (value: number) => [value >> 8, value & 0xFF]

const encodeU32BE = (value: number) => [
  ...encodeU16BE(value >> 16),
  ...encodeU16BE(value & 0xFFFF),
]

const encodeU64BE = (value: number | bigint) => [
  ...encodeU32BE(Number(BigInt(value) >> BigInt(32))),
  ...encodeU32BE(Number(BigInt(value) & BigInt(0xFFFFFFFF))),
]

export interface VAABody {
  readonly timestamp: number
  readonly nonce: number
  readonly emitterChainId: number
  readonly emitterAddress: Uint8Array
  readonly sequence: number
  readonly consistencyLevel: number
  readonly payload: Uint8Array
}

export const createVAAv2 = (tssIndex: number, body: VAABody, privateKey: Uint8Array) => {
  const vaaBody = new Uint8Array([
    ...encodeU32BE(body.timestamp),
    ...encodeU32BE(body.nonce),
    ...encodeU16BE(body.emitterChainId),
    ...body.emitterAddress,
    ...encodeU64BE(body.sequence),
    body.consistencyLevel,
    ...body.payload,
  ])

  const signature = sign(keccak_256(vaaBody), privateKey)

  const TSS_VAA_VERSION = 0x02

  return new Uint8Array([
    TSS_VAA_VERSION,
    ...encodeU32BE(tssIndex),
    ...signature.toCompactRawBytes(),
    signature.recovery,
    ...vaaBody,
  ])
}

export interface AppendThresholdKeyMessage {
  readonly tssIndex: number
  readonly tssKey: Uint8Array
  readonly expirationDelaySeconds: number
}

export const createAppendThresholdKeyMessage = (tssIndex: number, tssKey: Uint8Array, expirationDelaySeconds: number) => {
  const MODULE_VERIFICATION_V2 = new Uint8Array([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x54, 0x53, 0x53,
  ])

  const ACTION_APPEND_THRESHOLD_KEY = 0x01

  assert(tssKey.length === 20, "TSS key must be 20 bytes")

  return new Uint8Array([
    ...MODULE_VERIFICATION_V2,
    ACTION_APPEND_THRESHOLD_KEY,
    ...encodeU32BE(tssIndex),
    ...tssKey,
    ...encodeU32BE(expirationDelaySeconds),
  ])
}

// ------------------------------------------------------------------------------------------------


//TODO: get implementation for schnorr signing
async function thresholdGuardianSign(x: Uint8Array): Promise<Uint8Array> {
  return x;
}

describe("VerificationV2", function() {
  const coreV1Address = new PublicKey('worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth')
  const guardianSetExpirationTime = 86400
  const fee = 100
  const txSigner = $.keypair.generate()
  let coreV1: TestingWormholeCore<"Devnet">;
  const connection = $.connection
  let payer: Keypair;
  // const coreV2 = anchor.workspace.VerificationV2 as Program<VerificationV2>

  before(async function() {
    payer = anchor.getProvider().wallet?.payer!
    assert(payer, "Payer not found")

    await $.airdrop([
      txSigner.publicKey,
      payer.publicKey,
    ]);

    coreV1 = new TestingWormholeCore(
      txSigner,
      connection,
      WormholeContracts.Network,
      coreV1Address,
      WormholeContracts.addresses,
    );

    const txid = await coreV1.initialize(undefined, guardianSetExpirationTime, fee)
    const tx = await $.getTransaction(txid)
  });

  it("Check correct core v1 setup", async function() {
    const accounts = await connection.getProgramAccounts(coreV1Address)
    assert(accounts.length === 2, "Expected 2 accounts")

    const guardianSetIndex = await coreV1.client.getGuardianSetIndex();
    assert(guardianSetIndex === 0, "Expected guardian set index to be 0")
    const guardianSet = await coreV1.client.getGuardianSet(guardianSetIndex);
    // FIXME? initialize doesn't seem to set the correct expiration time
    // assert.strictEqual(guardianSet.expiry, BigInt(guardianSetExpirationTime), "Guardian set expiration time")

    const queriedFee = await coreV1.client.getMessageFee();
    assert(queriedFee === BigInt(fee), "Expected fee to be 100")

    assert(guardianSet.index === 0, "Expected guardian set index to be 0")
    assert(guardianSet.keys.length === 1, "Expected guardian set keys to have length 1")

    const queriedGuardian = new UniversalAddress(guardianSet.keys[0], "hex");
    const expectedGuardian = toUniversal("Ethereum", guardianAddress)
    assert(queriedGuardian.equals(expectedGuardian), "Expected guardian set keys to be the devnet guardian")
  })

  it("Posts append threshold key VAA successfully", async function(){
    const guardianPrivateKey = randomBytes(32)
    const guardianPublicKey = keccak_256(getPublicKey(guardianPrivateKey)).slice(12)
    const message = createAppendThresholdKeyMessage(
      0,
      guardianPublicKey,
      guardianSetExpirationTime,
    )

    const governanceContract = new UniversalAddress("0000000000000000000000000000000000000000000000000000000000000004", "hex");
    const postedVaaAddress = await coreV1.postVaa(
      payer,
      {
        chain: "Solana",
        emitterAddress: governanceContract,
      },
      message,
    )
  })

});
