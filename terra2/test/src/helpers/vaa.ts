import { soliditySha3 } from "web3-utils";

const abi = require("web3-eth-abi");
const elliptic = require("elliptic");

export const TEST_SIGNER_PKS = [
  "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0",
];

export function makeGovernanceVaaPayload(
  governanceChain: number,
  foreignChain: number,
  foreignTokenBridge: string
) {
  return (
    "000000000000000000000000000000000000000000546f6b656e427269646765" +
    abi.encodeParameter("uint16", governanceChain).substring(2 + 62) +
    "0000" +
    abi.encodeParameter("uint16", foreignChain).substring(2 + (64 - 4)) +
    foreignTokenBridge
  );
}

export function makeAttestationVaaPayload(
  chain: number,
  hexlifiedTokenAddress: string,
  decimals: number,
  symbol: string,
  name: string
) {
  return (
    abi.encodeParameter("uint8", 2).substring(2 + 62) +
    hexlifiedTokenAddress +
    abi.encodeParameter("uint16", chain).substring(2 + (64 - 4)) +
    abi.encodeParameter("uint8", decimals).substring(2 + 62) +
    symbol +
    name
  );
}

export function makeTransferVaaPayload(
  payloadType: number,
  amount: string,
  hexlifiedTokenAddress: string,
  encodedTo: string,
  toChain: number,
  relayerFee: string,
  additionalPayload: string | undefined
): string {
  const data =
    abi.encodeParameter("uint8", payloadType).substring(2 + 62) +
    // amount
    abi.encodeParameter("uint256", amount).substring(2) +
    // tokenaddress
    hexlifiedTokenAddress +
    // tokenchain
    // TODO: add tests for non-native tokens too
    "0012" + // we only care about terra2-specific tokens for these tests
    // receiver
    encodedTo +
    // receiving chain
    abi.encodeParameter("uint16", toChain).substring(2 + (64 - 4)) +
    // fee
    abi.encodeParameter("uint256", relayerFee).substring(2);

  // additional payload
  if (additionalPayload === undefined) {
    additionalPayload = "";
  }

  return data + Buffer.from(additionalPayload, "utf8").toString("hex");
}

export function signAndEncodeVaa(
  timestamp: number,
  nonce: number,
  emitterChainId: number,
  emitterAddress: string,
  sequence: number,
  data: string,
  signers: string[],
  guardianSetIndex: number,
  consistencyLevel: number
): string {
  const body: string[] = [
    abi.encodeParameter("uint32", timestamp).substring(2 + (64 - 8)),
    abi.encodeParameter("uint32", nonce).substring(2 + (64 - 8)),
    abi.encodeParameter("uint16", emitterChainId).substring(2 + (64 - 4)),
    emitterAddress,
    abi.encodeParameter("uint64", sequence).substring(2 + (64 - 16)),
    abi.encodeParameter("uint8", consistencyLevel).substring(2 + (64 - 2)),
    data,
  ];

  const hash = soliditySha3(soliditySha3("0x" + body.join(""))!)!;

  let signatures = "";

  for (const i in signers) {
    const ec = new elliptic.ec("secp256k1");
    const key = ec.keyFromPrivate(signers[i]);
    const signature = key.sign(hash.substring(2), { canonical: true });

    const packSig = [
      abi.encodeParameter("uint8", i).substring(2 + (64 - 2)),
      zeroPadBytes(signature.r.toString(16), 32),
      zeroPadBytes(signature.s.toString(16), 32),
      abi
        .encodeParameter("uint8", signature.recoveryParam)
        .substr(2 + (64 - 2)),
    ];

    signatures += packSig.join("");
  }

  const vm = [
    abi.encodeParameter("uint8", 1).substring(2 + (64 - 2)),
    abi.encodeParameter("uint32", guardianSetIndex).substring(2 + (64 - 8)),
    abi.encodeParameter("uint8", signers.length).substring(2 + (64 - 2)),

    signatures,
    body.join(""),
  ].join("");

  return vm;
}

function zeroPadBytes(value: string, length: number) {
  while (value.length < 2 * length) {
    value = "0" + value;
  }
  return value;
}
