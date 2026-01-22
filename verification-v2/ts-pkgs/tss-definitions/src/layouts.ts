import { chainToChainId, enumItem, Layout, LayoutToType } from '@wormhole-foundation/sdk-base';
import { envelopeLayout, layoutItems } from '@wormhole-foundation/sdk-definitions';
import type { PayloadLiteral, DecomposeLiteral, Payload, ProtocolName, ComposeLiteral } from '@wormhole-foundation/sdk-definitions';


/* VerificationV2 layouts */

export const schnorrSignatureLayout = [
  {name: "r", binary: "bytes", size: 20},
  {name: "s", binary: "bytes", size: 32},
] as const satisfies Layout;

export const headerV2Layout = [
  {name: "version",         binary: "uint",  size: 1, custom: 2, omit: true},
  {name: "schnorrKeyIndex", binary: "uint",  size: 4                       },
  {name: "signature",       binary: "bytes", layout: schnorrSignatureLayout},
] as const satisfies Layout;

export type HeaderV2 = LayoutToType<typeof headerV2Layout>;

export const baseV2Layout = [
  ...headerV2Layout,
  ...envelopeLayout,
] as const satisfies Layout;

type VAAV2Base = LayoutToType<typeof baseV2Layout>;
export type VAAV2Header = LayoutToType<typeof headerV2Layout>;

/**
 * A VAA is a Verifiable Action Assertion, a signed message that contains
 * information about an action that has occurred on a chain.
 *
 * See {@link https://docs.wormhole.com/wormhole/explore-wormhole/vaa | this link} for more.
 *
 */
export interface VAAV2<PL extends PayloadLiteral = PayloadLiteral> extends VAAV2Base {
  readonly protocolName: DecomposeLiteral<PL>[0];
  readonly payloadName: DecomposeLiteral<PL>[1];
  readonly payloadLiteral: PL;
  readonly payload: Payload<PL>;
  //TODO various problems with storing the hash here:
  // 1. On EVM the core bridge actually uses the double keccak-ed hash because of an early oversight
  // 2. As discussed on slack, storing memoized values on an object is a smell too
  //kept as is for now to get something usable out there, but this should receive more thought once
  //  the SDK has matured a little further.
  readonly hash: Uint8Array;
}

//We enforce distribution over union types, e.g. have
//    ProtocolVAAV2<"TokenBridge", "Transfer" | "TransferWithPayload">
//  turned into
//    VAAV2<"TokenBridge:Transfer"> | VAAV2<"TokenBridge:TransferWithPayload">
//  rather than
//    VAAV2<"TokenBridge:Transfer" | "TokenBridge:TransferWithPayload">
//  because while the latter is considered more idiomatic/canonical, it actually interferes with
//  the most natural way to narrow VAAs via querying the payloadName or payloadLiteral.
//  (Thanks for absolutely nothing, Typescript).
//  For example, given the TokenBridge VAAV2 union example:
//  if (vaa.payloadName === "Transfer")
//    typeof vaa //no narrowing - still resolves to the union type when using the latter approach
export type DistributiveVAAV2<PL extends PayloadLiteral> = PL extends PayloadLiteral
  ? VAAV2<PL>
  : never;

/** A  utility type that maps a protocol and payload name to its defined structure */
export type ProtocolVAAV2<PN extends ProtocolName, PayloadName extends string> = ComposeLiteral<
  PN,
  PayloadName,
  PayloadLiteral
> extends infer PL extends PayloadLiteral
  ? DistributiveVAAV2<PL>
  : never;

/** @dev module: TSS */
export const MODULE_VERIFICATION_V2 = Uint8Array.from([
  0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
  0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x54, 0x53, 0x53,
]);

export const appendSchnorrKeyMessageLayout = [
  {name: "module",                 binary: "bytes",           custom: MODULE_VERIFICATION_V2, omit: true},
  {name: "action",                 binary: "uint",  size:  1, custom: 1,                      omit: true},
  {name: "schnorrKeyIndex",        binary: "uint",  size:  4},
  {name: "expectedMssIndex",       binary: "uint",  size:  4},
  {name: "schnorrKey",             binary: "bytes", size: 32},
  {name: "expirationDelaySeconds", binary: "uint",  size:  4},
  {name: "shardDataHash",          binary: "bytes", size: 32},
] as const satisfies Layout;



/* Solana core program layouts */

const versionItem = <const N extends number>(custom: N) =>
  ({ name: 'version', binary: 'uint', size: 1, custom, omit: true }) as const;

const consistencyLevelItem = { name: 'consistencyLevel', binary: 'uint', size: 1 } as const;

const timestampItem = { name: 'timestamp', binary: 'uint', size: 4, endianness: 'little' } as const;

const nonceAndSequenceLayout = [
  { name: 'nonce', binary: 'uint', size: 4, endianness: 'little' },
  { name: 'sequence', binary: 'uint', size: 8, endianness: 'little' },
] as const satisfies Layout;

const emitterAddressAndPayloadLayout = [
  { name: 'emitterAddress', ...layoutItems.universalAddressItem },
  { name: 'payload', binary: 'bytes', lengthSize: 4, lengthEndianness: 'little' },
] as const satisfies Layout;

// From here: https://github.com/wormhole-foundation/wormhole/blob/7bd40b595e22c5512dfaa2ed8e6d7441df743a69/solana/programs/core-bridge/src/legacy/state/posted_message_v1/mod.rs#L17-L35
const messageStatusItem = {
  name: 'messageStatus',
  ...enumItem([
    ['Published', 0],
    ['Writing', 1],
    ['ReadyForPublishing', 2],
  ]),
} as const;

// From here: https://github.com/wormhole-foundation/wormhole/blob/7bd40b595e22c5512dfaa2ed8e6d7441df743a69/solana/programs/core-bridge/src/legacy/state/posted_message_v1/mod.rs#L39-L73
// reuses unused fields (that were only used for VAAs) from here: https://github.com/wormhole-foundation/wormhole/blob/7247a0fc0c96ab9493b8d0b886a7a54ee2a8fcce/solana/bridge/program/src/accounts/posted_message.rs#L46-L76
// hence these fields will only have sensible values when parsing posted messages by the solana core bridge rewrite
const postedMessageV1Layout = [
  versionItem(0),
  consistencyLevelItem,
  { name: 'emitterAuthority', ...layoutItems.universalAddressItem },
  messageStatusItem,
  { name: 'unusedGap', binary: 'uint', size: 3, custom: 0, omit: true },
  timestampItem,
  ...nonceAndSequenceLayout,
  {
    name: 'emitterChain',
    binary: 'uint',
    size: 2,
    endianness: 'little',
    custom: { from: chainToChainId('Solana'), to: 'Solana' },
  },
  ...emitterAddressAndPayloadLayout,
] as const satisfies Layout;

//from here: https://github.com/wormhole-foundation/wormhole/blob/7bd40b595e22c5512dfaa2ed8e6d7441df743a69/solana/programs/core-bridge/src/legacy/state/posted_vaa_v1.rs#L12-L43
//reuses unused fields (that were only used for posted messages) from here: https://github.com/wormhole-foundation/wormhole/blob/7247a0fc0c96ab9493b8d0b886a7a54ee2a8fcce/solana/bridge/program/src/accounts/posted_message.rs#L46-L76
//hence these fields will only have sensible values when parsing posted messages by the solana core bridge rewrite
const postedVaaV1Layout = [
  versionItem(1),
  consistencyLevelItem,
  timestampItem,
  { name: 'signatureSet', ...layoutItems.universalAddressItem },
  { name: 'guardianSetIndex', binary: 'uint', size: 4, endianness: 'little' },
  ...nonceAndSequenceLayout,
  { name: 'emitterChain', ...layoutItems.chainItem(), endianness: 'little' },
  ...emitterAddressAndPayloadLayout,
] as const satisfies Layout;

export const coreV1AccountDataLayout = {
  binary: 'switch',
  idSize: 3,
  idTag: 'discriminator',
  layouts: [
    //numeric values are ascii->number encoding of strings
    [[0x6d7367, 'msg'], postedMessageV1Layout],
    [[0x6d7375, 'msu'], postedMessageV1Layout],
    [[0x766161, 'vaa'], postedVaaV1Layout],
  ],
} as const satisfies Layout;