import { encoding, serializeLayout } from "@wormhole-foundation/sdk-base"
import { headerV2Layout, VAAV2Header } from "../../src"

export const signatureTestMessage100Zeroed = {
  r: encoding.hex.decode("0x41CF8d30EBCc800b655eAD15cC96014d36c4246B"),
  s: encoding.hex.decode("0xfb5fa64887c4a05818b02afa7483e5115f19a93739c4b9ce4e92bae191a2ef4b"),
}

export const invalidSignature = {
  r: encoding.hex.decode("0xE46Df5BEa4597CEF7D346EfF36356A3F0bA33a56"),
  s: encoding.hex.decode("0x1c2d1ca6fd3830e653d6abfc57956f3700059a661d8cabae684ea1bc62294e4c"),
}

const getDeserializedHeaderTestMessage100Zeroed = (schnorrKeyIndex: number): VAAV2Header => ({
  schnorrKeyIndex: schnorrKeyIndex,
  signature: signatureTestMessage100Zeroed,
})

const getDeserializedHeaderTestMessageInvalidSignature = (schnorrKeyIndex: number): VAAV2Header => ({
  schnorrKeyIndex: schnorrKeyIndex,
  signature: invalidSignature,
})

const getHeaderTestMessage100Zeroed = (schnorrKeyIndex: number): Uint8Array =>
  serializeLayout(headerV2Layout, getDeserializedHeaderTestMessage100Zeroed(schnorrKeyIndex))

const getHeaderTestMessageInvalidSignature = (schnorrKeyIndex: number): Uint8Array =>
  serializeLayout(headerV2Layout, getDeserializedHeaderTestMessageInvalidSignature(schnorrKeyIndex))

export const getTestMessage100Zeroed = (schnorrKeyIndex: number) => Uint8Array.from([
  ...getHeaderTestMessage100Zeroed(schnorrKeyIndex),
  ...new Uint8Array(100)
])

export const getTestMessageInvalidSignature = (schnorrKeyIndex: number) => Uint8Array.from([
  ...getHeaderTestMessageInvalidSignature(schnorrKeyIndex),
  ...new Uint8Array(100)
])