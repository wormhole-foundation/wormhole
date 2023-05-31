import {
  MockEmitter,
  MockGuardians,
} from "@certusone/wormhole-sdk/lib/cjs/mock";
import { BN, web3 } from "@coral-xyz/anchor";
import { expect } from "chai";
import { coreBridge } from "wormhole-solana-sdk";
import {
  CORE_BRIDGE_PROGRAM_ID,
  GUARDIAN_KEYS,
  LOCALHOST,
  airdrop,
  expectIxErr,
  expectIxOk,
  verifySignaturesAndPostVaa,
} from "../helpers";
import { ethers } from "ethers";

const GUARDIAN_SET_INDEX = 3;
const EMITTER_SEQUENCE = 2_102_000;

// Only a subset of those that are invalidated.
const INVALID_SIGNATURE_SETS = [
  "18eK1799CaNMGCUnnCt1Kq2uwKkax6T2WmtrDsZuVFQ",
  "2g6NCUUPaD6AxdHPQMVLpjpAvBfKMek6dDiGUe2A6T33",
  "3hYV5968hNzbqUfcvnQ6v9D5h32hEwGJn19c47N3unNj",
  "76eEyhaEKs4mesjiQiu8bghvwDHNxJW3EfcpbNC78y1z",
  /* ... */
  "E8qKJMwzBCiHCHUmBEcL631kN5CjfsHNx24osFLfHg69",
  "EtMw1nQ4AQaH53RjYz3pRk12rrqWjcYjPDETphYJzmCX",
  "EVNwqfgkUnJoMqBqiHgDfa3TLZPQocX1hpcbAXbpcSLv",
  "FixSiDfTxvoy5Zgjp5KdFU8U23ChwCxPWY3WTkmMW2fU",
];

const SIGNATURE_SET_NO_SIGNATURES: [string, number] = [
  "5Ng9FbCGL2teGdZHVU2xmJiCzofwudeey9EGWCaY4hAT",
  2_102_002,
];

const VALID_SIGNATURE_SETS: [string, number][] = [
  ["8uishP1LzwrNY1EUGwN9hDDyCJNq9KYYiuLdEXR7z8wp", 2_102_003],
  ["FavNDP9raM38ut8GpSwPZPfaQhsHf9ssR3iuwVJq7uDY", 2_102_004],
  ["9FUxeAD7bGDxKtv6oQmaWpEFAFVigRKr9PMGupoVb7vP", 2_102_005],
  ["Cy4QhUDcWfHiXKwNyR9Ua4NmoQ1WkdZsBdt9vNHyvD6K", 2_102_006],
  ["QhjQmkJsJNJPuk2dBdRuLsiAyUy2sVDoqRfgHheSPtd", 2_102_007],
  ["HeYQadUm66isrG9mggNV4snUwMRN1MC2y7yLXCs4c4vo", 2_102_008],
  ["Ap4LoByWRoyQew1VmThFB97BqxqVqaTyYmL5YAzk8YvU", 2_102_009],
  ["5zH7Qq9bdrcJDsJLjZJ63s5BjnftTvHuqe8c3eErHUrW", 2_102_010],
  ["DCiw4352zkToCNNBap5rz9wruLYoWbLCXMcZp2rTuF4P", 2_102_011],
];

const HURRDURR: [string, number][] = [
  ["5Ng9FbCGL2teGdZHVU2xmJiCzofwudeey9EGWCaY4hAT", 0],
  ["8uishP1LzwrNY1EUGwN9hDDyCJNq9KYYiuLdEXR7z8wp", EMITTER_SEQUENCE + 2],
  ["FavNDP9raM38ut8GpSwPZPfaQhsHf9ssR3iuwVJq7uDY", EMITTER_SEQUENCE + 3],
  ["9FUxeAD7bGDxKtv6oQmaWpEFAFVigRKr9PMGupoVb7vP", EMITTER_SEQUENCE + 4],
  ["Cy4QhUDcWfHiXKwNyR9Ua4NmoQ1WkdZsBdt9vNHyvD6K", EMITTER_SEQUENCE + 5],
  ["QhjQmkJsJNJPuk2dBdRuLsiAyUy2sVDoqRfgHheSPtd", EMITTER_SEQUENCE + 6],
  ["HeYQadUm66isrG9mggNV4snUwMRN1MC2y7yLXCs4c4vo", EMITTER_SEQUENCE + 7],
  ["Ap4LoByWRoyQew1VmThFB97BqxqVqaTyYmL5YAzk8YvU", EMITTER_SEQUENCE + 8],
  ["5zH7Qq9bdrcJDsJLjZJ63s5BjnftTvHuqe8c3eErHUrW", EMITTER_SEQUENCE + 9],
  ["DCiw4352zkToCNNBap5rz9wruLYoWbLCXMcZp2rTuF4P", EMITTER_SEQUENCE + 10],
];

const EXPECTED_TIMESTAMP = 23456789;
const EXPECTED_NONCE = 12345;
const EXPECTED_PAYLOAD = Buffer.from("Are you looking at me?");
const EXPECTED_CONSISTENCY_LEVEL = 200;

describe("Core Bridge: Legacy Verify Signatures and Post VAA", () => {
  const connection = new web3.Connection(LOCALHOST, "processed");

  const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

  const foreignEmitter = new MockEmitter(
    "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
    2,
    EMITTER_SEQUENCE
  );

  const payerSigner = web3.Keypair.generate();
  const payer = payerSigner.publicKey;

  const localVariables = new Map<string, any>();

  before("Airdrop Payer", async () => {
    await airdrop(connection, payer);
  });

  describe("Ok", async () => {
    it("Invoke `legacy_verify_signatures` and `legacy_post_vaa`", async () => {
      const timestamp = 12345678;
      const nonce = 420;
      const payload = Buffer.from("Someone set us up the bomb.");
      const consistencyLevel = 1;
      const published = foreignEmitter.publishMessage(
        nonce,
        payload,
        consistencyLevel,
        timestamp
      );
      const signedVaa = guardians.addSignatures(
        published,
        [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]
      );

      // Verify and Post
      await verifySignaturesAndPostVaa(connection, payerSigner, signedVaa);

      // TODO: Check Posted VAA account.
    });

    it("Invoke `legacy_post_vaa` Without Unnecessary Accounts and Arguments", async () => {
      const [signatureSetKey, expectedSequence] = VALID_SIGNATURE_SETS[0];
      const signatureSet = new web3.PublicKey(signatureSetKey);

      const { sigVerifySuccesses, messageHash, guardianSetIndex } =
        await coreBridge.SignatureSet.fromAccountAddress(
          connection,
          signatureSet
        );

      // Verify that this signature set reached quorum.
      const guardianSetData = await coreBridge.GuardianSet.fromPda(
        connection,
        CORE_BRIDGE_PROGRAM_ID,
        guardianSetIndex
      );
      const numVerified = sigVerifySuccesses
        .map((v) => Number(v))
        .reduce((prev, curr) => prev + curr);
      expect(numVerified).is.greaterThanOrEqual(
        (2 * guardianSetData.keys.length) / 3
      );

      const accounts = coreBridge.LegacyPostVaaContext.new(
        CORE_BRIDGE_PROGRAM_ID,
        guardianSetIndex,
        signatureSet,
        messageHash,
        payer,
        { bridge: false, clock: false, rent: false }
      );

      expect(accounts._bridge).is.null;
      expect(accounts._clock).is.null;
      expect(accounts._rent).is.null;

      const args = {
        version: undefined,
        guardianSetIndex: undefined,
        timestamp: EXPECTED_TIMESTAMP,
        nonce: EXPECTED_NONCE,
        emitterChain: foreignEmitter.chain,
        emitterAddress: Array.from(foreignEmitter.address),
        sequence: new BN(expectedSequence),
        finality: EXPECTED_CONSISTENCY_LEVEL,
        payload: EXPECTED_PAYLOAD,
      };

      const ix = coreBridge.legacyPostVaaIx(
        CORE_BRIDGE_PROGRAM_ID,
        accounts,
        args
      );

      await expectIxOk(connection, [ix], [payerSigner]);

      // const signers = [0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 14];
      // let i = 0;
      // for (const [signatureSet, _] of HURRDURR) {
      //   ++i;
      //   const published = foreignEmitter.publishMessage(
      //     EXPECTED_NONCE,
      //     EXPECTED_PAYLOAD,
      //     EXPECTED_CONSISTENCY_LEVEL,
      //     EXPECTED_TIMESTAMP
      //   );
      //   console.log("published", signatureSet, published.toString("hex"));
      //   const messageHash = Buffer.from(
      //     ethers.utils.arrayify(ethers.utils.keccak256(published))
      //   );
      //   console.log("messageHash?", messageHash.toString("hex"));
      //   const accountData = await coreBridge.SignatureSet.fromAccountAddress(
      //     connection,
      //     new web3.PublicKey(signatureSet)
      //   );
      //   console.log("accountData?", accountData);
      // const accountData = Buffer.alloc(4 + 19 + 32 + 4);
      // accountData.writeUInt32LE(19, 0);
      // if (i > 1) {
      //   for (let i = 0; i < 19; ++i) {
      //     if (signers.includes(i)) {
      //       accountData.writeUInt8(1, 4 + i);
      //     }
      //   }
      // }
      // accountData.write(messageHash.toString("hex"), 4 + 19, "hex");
      // accountData.writeUInt32LE(GUARDIAN_SET_INDEX, 4 + 19 + 32);
      // console.log(
      //   "and....?",
      //   accountData.subarray(23, 23 + 32).toString("hex")
      // );
      // const allData = {
      //   pubkey: signatureSet,
      //   account: {
      //     lamports: 1301520,
      //     data: [accountData.toString("base64"), "base64"],
      //     owner: CORE_BRIDGE_PROGRAM_ID,
      //     executable: false,
      //     rentEpoch: 0,
      //   },
      // };
      // const fs = require("fs");
      // fs.writeFileSync(
      //   `${__dirname}/../test-accounts/valid_signature_set_${i
      //     .toString()
      //     .padStart(2, "0")}.json`,
      //   JSON.stringify(allData, null, 2)
      // );
      //}
    });

    it("Cannot Invoke `legacy_post_vaa` With Invalid Signature Set Pubkeys", async () => {
      // TODO
      INVALID_SIGNATURE_SETS;
    });

    it("Cannot Invoke `legacy_post_vaa` Without Quorum", async () => {
      const [signatureSetKey, expectedSequence] = SIGNATURE_SET_NO_SIGNATURES;
      const signatureSet = new web3.PublicKey(signatureSetKey);

      const { sigVerifySuccesses, messageHash, guardianSetIndex } =
        await coreBridge.SignatureSet.fromAccountAddress(
          connection,
          signatureSet
        );

      // Verify that this signature set reached quorum.
      const guardianSetData = await coreBridge.GuardianSet.fromPda(
        connection,
        CORE_BRIDGE_PROGRAM_ID,
        guardianSetIndex
      );
      const numVerified = sigVerifySuccesses
        .map((v) => Number(v))
        .reduce((prev, curr) => prev + curr);
      expect(numVerified).is.lessThan((2 * guardianSetData.keys.length) / 3);

      const args = {
        timestamp: EXPECTED_TIMESTAMP,
        nonce: EXPECTED_NONCE,
        emitterChain: foreignEmitter.chain,
        emitterAddress: Array.from(foreignEmitter.address),
        sequence: new BN(expectedSequence),
        finality: EXPECTED_CONSISTENCY_LEVEL,
        payload: EXPECTED_PAYLOAD,
      };

      const ix = coreBridge.LegacyPostVaaContext.instruction(
        CORE_BRIDGE_PROGRAM_ID,
        guardianSetIndex,
        signatureSet,
        messageHash,
        payer,
        args
      );

      await expectIxErr(connection, [ix], [payerSigner], "NoQuorum");
    });

    it("Cannot Invoke `legacy_post_vaa` With Incorrect Body Data", async () => {
      const [signatureSetKey, expectedSequence] = VALID_SIGNATURE_SETS[1];
      const signatureSet = new web3.PublicKey(signatureSetKey);

      const { messageHash, guardianSetIndex } =
        await coreBridge.SignatureSet.fromAccountAddress(
          connection,
          signatureSet
        );

      const incorrectSequence = expectedSequence + 1;
      const args = {
        timestamp: EXPECTED_TIMESTAMP,
        nonce: EXPECTED_NONCE,
        emitterChain: foreignEmitter.chain,
        emitterAddress: Array.from(foreignEmitter.address),
        sequence: new BN(incorrectSequence),
        finality: EXPECTED_CONSISTENCY_LEVEL,
        payload: EXPECTED_PAYLOAD,
      };

      const ix = coreBridge.LegacyPostVaaContext.instruction(
        CORE_BRIDGE_PROGRAM_ID,
        guardianSetIndex,
        signatureSet,
        messageHash,
        payer,
        args
      );

      await expectIxErr(connection, [ix], [payerSigner], "InvalidMessageHash");
    });
  });
});
