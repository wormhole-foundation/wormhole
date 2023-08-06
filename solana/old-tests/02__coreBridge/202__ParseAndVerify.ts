import {
  MockEmitter,
  MockGuardians,
} from "@certusone/wormhole-sdk/lib/cjs/mock";
import { web3 } from "@coral-xyz/anchor";
import { expect } from "chai";
import { ethers } from "ethers";
import { coreBridge } from "wormhole-solana-sdk";
import {
  CORE_BRIDGE_PROGRAM_ID,
  GUARDIAN_KEYS,
  LOCALHOST,
  airdrop,
  expectIxErr,
  expectIxOk,
} from "../helpers";

const GUARDIAN_SET_INDEX = 6;

describe("Core Bridge: New Parse and Verify VAA", () => {
  const connection = new web3.Connection(LOCALHOST, "processed");

  const payerSigner = web3.Keypair.generate();
  const payer = payerSigner.publicKey;

  const foreignEmitter = new MockEmitter(
    "000000000000000000000000deadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
    2,
    2_202_000
  );
  foreignEmitter.sequence = 6900;

  const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

  const localVariables = new Map<string, any>();

  before("Airdrop Payer", async () => {
    await airdrop(connection, payer);
  });

  describe("Ok", async () => {
    it("Invoke `init_encoded_vaa` with Large VAA Message Payload", async () => {
      const encodedVaaSigner = web3.Keypair.generate();

      // Accounts.
      const accounts = coreBridge.InitEncodedVaaContext.new(
        payerSigner.publicKey,
        encodedVaaSigner.publicKey
      );

      // Data.
      const repeatedMessage = "Someone set up us the bomb. ";

      // NOTE: The largest VAA message payload that can be encoded is 15KB
      // because to generate the message hash for anything larger requires more
      // than 1.4M compute units, which is the current compute limit.
      const messagePayload = Buffer.alloc(15 * 1024, repeatedMessage);

      const nonce = 420;
      const consistencyLevel = 1;
      const timestamp = 12345678;
      const published = foreignEmitter.publishMessage(
        nonce,
        messagePayload,
        consistencyLevel,
        timestamp
      );

      const signedVaa = guardians.addSignatures(
        published,
        [0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 14]
      );

      const dataLength = 46 + signedVaa.length;
      const createIx = await connection
        .getMinimumBalanceForRentExemption(dataLength)
        .then((lamports) =>
          web3.SystemProgram.createAccount({
            fromPubkey: payer,
            newAccountPubkey: encodedVaaSigner.publicKey,
            space: dataLength,
            lamports,
            programId: new web3.PublicKey(CORE_BRIDGE_PROGRAM_ID),
          })
        );
      const initIx = await coreBridge.initEncodedVaaIx(
        connection,
        CORE_BRIDGE_PROGRAM_ID,
        accounts
      );

      await expectIxOk(
        connection,
        [createIx, initIx],
        [payerSigner, encodedVaaSigner]
      );

      const { writeAuthority, encodedVaa } = accounts;

      const encodedVaaData = await coreBridge.EncodedVaa.fromAccountAddress(
        connection,
        CORE_BRIDGE_PROGRAM_ID,
        encodedVaa
      );

      expect(encodedVaaData.status).equals(1); // Writing
      expect(encodedVaaData.writeAuthority.equals(writeAuthority)).is.true;
      expect(encodedVaaData.version).equals(0); // Unset
      expect(encodedVaaData.bytes.equals(Buffer.alloc(signedVaa.length))).is
        .true;

      // Save for later.
      localVariables.set("vaaPubkey", encodedVaa);
      localVariables.set("messagePayload", messagePayload);
      localVariables.set("signedVaa", signedVaa);
    });

    it("Invoke `process_encoded_vaa` To Write VAA", async () => {
      const encodedVaa: web3.PublicKey = localVariables.get("vaaPubkey")!;
      const signedVaa: Buffer = localVariables.get("signedVaa")!;

      let vaaIndex = 0;

      const accounts = coreBridge.ProcessEncodedVaaContext.new(
        payerSigner.publicKey,
        encodedVaa,
        null
      );

      // Break up into chunks. Max chunk size is 990 (due to transaction size).
      const maxChunkSize = 990;
      while (vaaIndex < signedVaa.length) {
        const dataLength = Math.min(signedVaa.length - vaaIndex, maxChunkSize);
        const data = signedVaa.subarray(vaaIndex, vaaIndex + dataLength);

        const ix = await coreBridge.processEncodedVaaIx(
          connection,
          CORE_BRIDGE_PROGRAM_ID,
          accounts,
          { write: { index: vaaIndex, data } }
        );

        await expectIxOk(connection, [ix], [payerSigner]);

        vaaIndex += dataLength;
      }

      const encodedVaaData = await coreBridge.EncodedVaa.fromAccountAddress(
        connection,
        CORE_BRIDGE_PROGRAM_ID,
        encodedVaa
      );

      expect(encodedVaaData.status).equals(1); // Writing
      expect(encodedVaaData.bytes.equals(signedVaa)).is.true;
    });

    it("Invoke `process_encoded_vaa` To Verify Signatures", async () => {
      const encodedVaa: web3.PublicKey = localVariables.get("vaaPubkey")!;

      const guardianSet = coreBridge.GuardianSet.address(
        CORE_BRIDGE_PROGRAM_ID,
        GUARDIAN_SET_INDEX
      );
      const accounts = coreBridge.ProcessEncodedVaaContext.new(
        payerSigner.publicKey,
        encodedVaa,
        guardianSet
      );

      // Current max compute unit limit is 1.4M.
      const computeIx = web3.ComputeBudgetProgram.setComputeUnitLimit({
        units: 375000,
      });

      const verifySigsIx = await coreBridge.processEncodedVaaIx(
        connection,
        CORE_BRIDGE_PROGRAM_ID,
        accounts,
        { verifySignaturesV1: {} }
      );

      await expectIxOk(connection, [computeIx, verifySigsIx], [payerSigner]);

      const encodedVaaData = await coreBridge.EncodedVaa.fromAccountAddress(
        connection,
        CORE_BRIDGE_PROGRAM_ID,
        encodedVaa
      );

      expect(encodedVaaData.status).equals(2); // Verified
      expect(encodedVaaData.version).equals(1); // V1
    });

    it("Cannot Invoke `post_vaa_v1` With Message Payload > 10,145 Bytes", async () => {
      const signedVaa: Buffer = localVariables.get("signedVaa")!;
      const vaa: web3.PublicKey = localVariables.get("vaaPubkey")!;

      const messageHash = Array.from(
        ethers.utils.arrayify(
          ethers.utils.keccak256(signedVaa.subarray(signedVaa[5] * 66 + 6))
        )
      );

      const accounts = coreBridge.PostVaaV1Context.new(
        payerSigner.publicKey,
        vaa,
        coreBridge.PostedVaaV1.address(CORE_BRIDGE_PROGRAM_ID, messageHash)
      );

      const ix = await coreBridge.postVaaV1Ix(
        connection,
        CORE_BRIDGE_PROGRAM_ID,
        accounts,
        { tryOnce: {} }
      );

      await expectIxErr(
        connection,
        [ix],
        [payerSigner],
        "Account data size realloc limited to 10240 in inner instructions"
      );
    });

    it("Invoke `post_vaa_v1` With Max Message Payload Size 10,145 Bytes", async () => {
      const encodedVaaSigner = web3.Keypair.generate();

      // Accounts.
      const initAccounts = coreBridge.InitEncodedVaaContext.new(
        payerSigner.publicKey,
        encodedVaaSigner.publicKey
      );

      const { encodedVaa: vaa } = initAccounts;

      // Data.
      const repeatedMessage = "Goldilocks. ";

      // NOTE: The largest VAA message payload that can be encoded is 15KB
      // because to generate the message hash for anything larger requires more
      // than 1.4M compute units, which is the current compute limit.
      const messagePayload = Buffer.alloc(10145, repeatedMessage);

      const nonce = 420;
      const consistencyLevel = 1;
      const timestamp = 12345678;
      const published = foreignEmitter.publishMessage(
        nonce,
        messagePayload,
        consistencyLevel,
        timestamp
      );

      const signedVaa = guardians.addSignatures(
        published,
        [0, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 15, 16]
      );

      const dataLength = 46 + signedVaa.length;
      const createIx = await connection
        .getMinimumBalanceForRentExemption(dataLength)
        .then((lamports) =>
          web3.SystemProgram.createAccount({
            fromPubkey: payer,
            newAccountPubkey: encodedVaaSigner.publicKey,
            space: dataLength,
            lamports,
            programId: new web3.PublicKey(CORE_BRIDGE_PROGRAM_ID),
          })
        );
      const initIx = await coreBridge.initEncodedVaaIx(
        connection,
        CORE_BRIDGE_PROGRAM_ID,
        initAccounts
      );

      let remainingNumBytes = signedVaa.length;
      let vaaIndex = 0;

      const writeAccounts = coreBridge.ProcessEncodedVaaContext.new(
        payerSigner.publicKey,
        vaa,
        null
      );

      // Because we are putting the first write instruction with the init
      // instructions, we cannot write as many bytes due to other data filling
      // the transaction (signature, create instruction data, etc).
      const firstChunkSize = Math.min(844, remainingNumBytes);
      const firstChunk = signedVaa.subarray(0, firstChunkSize);

      const firstProcessIx = await coreBridge.processEncodedVaaIx(
        connection,
        CORE_BRIDGE_PROGRAM_ID,
        writeAccounts,
        { write: { index: 0, data: firstChunk } }
      );
      vaaIndex += firstChunkSize;
      remainingNumBytes -= firstChunkSize;

      await expectIxOk(
        connection,
        [createIx, initIx, firstProcessIx],
        [payerSigner, encodedVaaSigner]
      );

      // Break up into chunks. Max chunk size is 990 (due to transaction size).
      const maxChunkSize = 990;
      while (remainingNumBytes > 0) {
        const dataLength = Math.min(remainingNumBytes, maxChunkSize);
        const data = signedVaa.subarray(vaaIndex, vaaIndex + dataLength);

        const ix = await coreBridge.processEncodedVaaIx(
          connection,
          CORE_BRIDGE_PROGRAM_ID,
          writeAccounts,
          { write: { index: vaaIndex, data } }
        );

        vaaIndex += dataLength;
        remainingNumBytes -= dataLength;

        await expectIxOk(connection, [ix], [payerSigner]);
      }

      const guardianSet = coreBridge.GuardianSet.address(
        CORE_BRIDGE_PROGRAM_ID,
        GUARDIAN_SET_INDEX
      );
      const verifyAccounts = coreBridge.ProcessEncodedVaaContext.new(
        payerSigner.publicKey,
        vaa,
        guardianSet
      );

      const computeIx = web3.ComputeBudgetProgram.setComputeUnitLimit({
        units: 400000,
      });

      const verifySigsIx = await coreBridge.processEncodedVaaIx(
        connection,
        CORE_BRIDGE_PROGRAM_ID,
        verifyAccounts,
        { verifySignaturesV1: {} }
      );

      const messageHash = Array.from(
        ethers.utils.arrayify(
          ethers.utils.keccak256(signedVaa.subarray(signedVaa[5] * 66 + 6))
        )
      );

      const postAccounts = coreBridge.PostVaaV1Context.new(
        payerSigner.publicKey,
        vaa,
        coreBridge.PostedVaaV1.address(CORE_BRIDGE_PROGRAM_ID, messageHash)
      );

      const postIx = await coreBridge.postVaaV1Ix(
        connection,
        CORE_BRIDGE_PROGRAM_ID,
        postAccounts,
        { tryOnce: {} }
      );

      await expectIxOk(
        connection,
        [computeIx, verifySigsIx, postIx],
        [payerSigner]
      );
    });
  });
});
