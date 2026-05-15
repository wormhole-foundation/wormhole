import { RequestPrefix } from "@wormhole-foundation/sdk-definitions";
import type { DeriveType, Layout } from "binary-layout";

// Binary layout for the VAA v1 request in https://github.com/wormholelabs-xyz/example-messaging-executor?tab=readme-ov-file#vaa-v1-request
// As of this writing, it is not yet available in https://github.com/wormhole-foundation/wormhole-sdk-ts/tree/main/core/definitions/src/protocols/executor
// On-chain on SVM, one could use https://github.com/wormholelabs-xyz/example-messaging-executor/blob/2061262868ed420e911a54ef619dd9b00949beb1/svm/modules/executor-requests/src/lib.rs#L7

export const vaaV1RequestLayout = [
  { name: "chain", binary: "uint", size: 2 },
  { name: "address", binary: "bytes", size: 32 },
  { name: "sequence", binary: "uint", size: 8 },
] as const satisfies Layout;

export type VAAv1Request = DeriveType<typeof vaaV1RequestLayout>;

export const requestLayout = [
  {
    name: "request",
    binary: "switch",
    idSize: 4,
    idTag: "prefix",
    layouts: [[[0x45525631, RequestPrefix.ERV1], vaaV1RequestLayout]],
  },
] as const satisfies Layout;

export type RequestLayout = DeriveType<typeof requestLayout>;
