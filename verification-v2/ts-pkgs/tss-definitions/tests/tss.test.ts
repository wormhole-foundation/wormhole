import { describe, it, expect } from 'vitest';
import { encoding } from "@wormhole-foundation/sdk-base";
import {
  serialize,
  deserialize,
  VAAV2,
} from "../src/index.js";
import { signatureTestMessage100Zeroed } from './utils/vaaV2.js';

// Invalid VAA
const fakeVaaV2 = "0x020000000241cf8d30ebcc800b655ead15cc96014d36c4246bfb5fa64887c4a05818b02afa7483e5115f19a93739c4b9ce4e92bae191a2ef4b00000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000";

describe("VAA V2 serialization", function () {
  // TODO Add more tests when we have valid vaa v2
  it("Should deserialize a VAA V2", function () {
    const testKeyIndex = 2;
    const deserialized = deserialize(fakeVaaV2) as VAAV2;
    expect(deserialized.schnorrKeyIndex).toBe(testKeyIndex);
    expect(deserialized.signature.r).toEqual(signatureTestMessage100Zeroed.r);
    expect(deserialized.signature.s).toEqual(signatureTestMessage100Zeroed.s);
    const serialized = `0x${encoding.hex.encode(serialize(deserialized))}`;
    expect(serialized).toBe(fakeVaaV2);
  });
});

