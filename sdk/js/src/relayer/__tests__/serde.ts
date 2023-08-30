import { describe, expect, test } from "@jest/globals";
import { ethers } from "ethers";
import { tryNativeToUint8Array } from "../../../";
import {
  CCTPKey,
  KeyType,
  MessageKey,
  packCCTPKey,
  packMessageKey,
  packVaaKey,
  parseCCTPKey,
  parseMessageKey,
  parseVaaKey,
  VaaKey,
} from "../structs";

describe("Wormhole Relayer Serde Tests", () => {
  test("Parse VaaKey", async () => {
    const vaaKey: VaaKey = {
      chainId: 16,
      emitterAddress: Buffer.from(
        tryNativeToUint8Array(
          "0x7FA9385bE102ac3EAc297483Dd6233D62b3e1496",
          "ethereum"
        )
      ),
      sequence: ethers.BigNumber.from(52),
    };
    const messageKey: MessageKey = {
      keyType: KeyType.VAA,
      key: packVaaKey(vaaKey),
    };

    const encodedVaaKey = packVaaKey(vaaKey);
    const encodedMessageKey = packMessageKey(messageKey);

    // console.log(vaaKey, encodedVaaKey, parseVaaKey(Buffer.from(ethers.utils.arrayify(encodedVaaKey))));
    const expectedEncodedVaaKey =
      "0x00100000000000000000000000007fa9385be102ac3eac297483dd6233d62b3e14960000000000000034";
    const expectedEncodedMessageKey =
      "0x0100100000000000000000000000007fa9385be102ac3eac297483dd6233d62b3e14960000000000000034";
    expect(encodedVaaKey).toEqual(expectedEncodedVaaKey);
    expect(parseVaaKey(ethers.utils.arrayify(encodedVaaKey))).toEqual(vaaKey);

    const [parsedMessageKey] = parseMessageKey(
      ethers.utils.arrayify(expectedEncodedMessageKey),
      0
    );
    expect(parsedMessageKey.keyType).toEqual(messageKey.keyType);
    expect(ethers.utils.hexlify(parsedMessageKey.key)).toEqual(
      ethers.utils.hexlify(messageKey.key)
    );
  });

  test("Parse CCTPKey", () => {
    const cctpKey: CCTPKey = {
      domain: 2,
      nonce: ethers.BigNumber.from(123 + 2 ** 13),
    };
    const packedCCTPKey = packCCTPKey(cctpKey);
    const parsedCCTPKey = parseCCTPKey(packedCCTPKey);

    expect(parsedCCTPKey).toEqual(cctpKey);
    expect(packedCCTPKey).toEqual("0x00000002000000000000207b");

    const messageKey = {
      keyType: KeyType.CCTP,
      key: packedCCTPKey,
    };
    const packedMessageKey = packMessageKey(messageKey);
    const [parsedMessageKey] = parseMessageKey(packedMessageKey, 0);

    expect(parsedMessageKey.keyType).toEqual(messageKey.keyType);
    expect(ethers.utils.hexlify(parsedMessageKey.key)).toEqual(
      ethers.utils.hexlify(messageKey.key)
    );
    expect(packedMessageKey).toEqual("0x020000000c00000002000000000000207b");
  });
});
