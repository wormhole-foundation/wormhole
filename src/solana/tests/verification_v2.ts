import assert from "assert"

import * as anchor from "@coral-xyz/anchor"
import { PublicKey } from "@solana/web3.js"
import { VerificationV2 } from "../target/types/verification_v2.js"

import { MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock"
import * as coreV1 from "@certusone/wormhole-sdk/lib/cjs/solana/wormhole";

import { sign } from "@noble/secp256k1"
import { keccak_256 } from "@noble/hashes/sha3"

import { boilerPlateReduction } from "./testing_helpers.js"
import { Program } from "@coral-xyz/anchor"
import { CONTRACTS } from "@certusone/wormhole-sdk"

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

// Configure the client to use the local cluster.
anchor.setProvider(anchor.AnchorProvider.env())

const connection = anchor.getProvider().connection
const payer = anchor.getProvider().wallet?.payer
assert(payer, "Payer not found")

const mockGuardians = new MockGuardians(0, ["cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0"])

const coreV2 = anchor.workspace.VerificationV2 as Program<VerificationV2>

const {
  requestAirdrop,
  guardianSign,
  postSignedMsgAsVaaOnSolana,
  expectIxToSucceed,
  expectIxToFailWithError,
} = boilerPlateReduction(connection, payer)

const guardianSetExpirationTime = 86400
const fee = 100n
const devnetGuardian = mockGuardians.getPublicKeys()[0]
const initialGuardians = [devnetGuardian]
const coreV1Address = new PublicKey('worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth')

await expectIxToSucceed(
  coreV1.createInitializeInstruction(
    coreV1Address,
    payer.publicKey,
    guardianSetExpirationTime,
    fee,
    initialGuardians,
  )
)

const accounts = await connection.getProgramAccounts(coreV1Address)
assert(accounts.length === 2, "Expected 2 accounts")

const info = await coreV1.getWormholeBridgeData(connection, coreV1Address)
assert(info.guardianSetIndex === 0, "Expected guardian set index to be 0")
assert(info.config.guardianSetExpirationTime === guardianSetExpirationTime, "Expected guardian set expiration time to be 86400")
assert(info.config.fee === fee, "Expected fee to be 100")

const guardianSet = await coreV1.getGuardianSet(connection, coreV1Address, info.guardianSetIndex)
assert(guardianSet.index === 0, "Expected guardian set index to be 0")
assert(guardianSet.keys.length === 1, "Expected guardian set keys to have length 1")
assert(devnetGuardian.equals(guardianSet.keys[0]), "Expected guardian set keys to be the devnet guardian")
