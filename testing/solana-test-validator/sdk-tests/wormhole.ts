import { expect } from "chai";
import { readFileSync } from "fs";
import * as web3 from "@solana/web3.js";
import {
  MockGuardians,
  MockEthereumEmitter,
  WormholeGovernanceEmitter,
} from "../../../sdk/js/src/utils/mock";
import { parseVaa } from "../../../sdk/js/src/vaa/wormhole";

import {
  CORE_BRIDGE_ADDRESS,
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  GOVERNANCE_EMITTER_ADDRESS,
  GUARDIAN_KEYS,
  GUARDIAN_SET_INDEX,
  LOCALHOST,
} from "./helpers/consts";
import {
  getPostedVaa,
  getGuardianSet,
  createSetFeesInstruction,
  createTransferFeesInstruction,
  createUpgradeGuardianSetInstruction,
  getBridgeInfo,
  feeCollectorKey,
  createBridgeFeeTransferInstruction,
} from "../../../sdk/js/src/solana/wormhole";
import { postVaa } from "../../../sdk/js/src/solana/sendAndConfirmPostVaa";
import { NodeWallet } from "../../../sdk/js/src/solana/utils";

describe("Wormhole (Core Bridge)", () => {
  const connection = new web3.Connection(LOCALHOST);

  const wallet = new NodeWallet(web3.Keypair.generate());

  // for signing wormhole messages
  const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

  // for generating governance wormhole messages
  const governance = new WormholeGovernanceEmitter(GOVERNANCE_EMITTER_ADDRESS);

  // hijacking the ethereum token bridge address for our fake emitter
  const ethereumWormhole = new MockEthereumEmitter(
    ETHEREUM_TOKEN_BRIDGE_ADDRESS
  );

  before("Airdrop SOL", async () => {
    await connection
      .requestAirdrop(wallet.key(), 1000 * web3.LAMPORTS_PER_SOL)
      .then(async (signature) => {
        await connection.confirmTransaction(signature);
        return signature;
      });
  });

  describe("Instruction 1: Post Message", () => {
    // TODO: add mock implementation contract and test that it can use the post_message instruction
  });

  describe("Instructions 2 and 7: Post VAA and Verify Signatures", () => {
    it("Verify Guardian Signature and Post Message", async () => {
      const message = Buffer.from("All your base are belong to us.");
      const nonce = 0;
      const consistencyLevel = 15;
      const published = ethereumWormhole.publishMessage(
        nonce,
        message,
        consistencyLevel
      );
      const signedVaa = guardians.addSignatures(published, [0]);
      // console.log(`signedVaa: ${signedVaa.toString("base64")}`);

      const txSignatures = await postVaa(
        connection,
        wallet.signTransaction,
        CORE_BRIDGE_ADDRESS,
        wallet.key(),
        signedVaa
      ).then((results) => results.map((result) => result.signature));
      const postTx = txSignatures.pop()!;
      for (const verifyTx of txSignatures) {
        console.log(`verifySignatures: ${verifyTx}`);
      }
      console.log(`postVaa:          ${postTx}`);

      // verify data
      const parsed = parseVaa(signedVaa);
      const messageData = await getPostedVaa(
        connection,
        CORE_BRIDGE_ADDRESS,
        parsed.hash
      ).then((postedVaa) => postedVaa.message);

      expect(messageData.consistencyLevel).to.equal(consistencyLevel);
      expect(messageData.consistencyLevel).to.equal(parsed.consistencyLevel);
      expect(messageData.emitterAddress.toString("hex")).to.equal(
        parsed.emitterAddress
      );
      expect(messageData.emitterChain).to.equal(parsed.emitterChain);
      expect(messageData.nonce).to.equal(nonce);
      expect(messageData.nonce).to.equal(parsed.nonce);
      expect(Buffer.compare(messageData.payload, message)).to.equal(0);
      expect(Buffer.compare(messageData.payload, parsed.payload)).to.equal(0);
      expect(messageData.sequence).to.equal(parsed.sequence);
      expect(messageData.vaaTime).to.equal(parsed.timestamp);
      expect(messageData.vaaVersion).to.equal(parsed.version);
    });
  });

  describe("Instruction 3: Set Fees", () => {
    it("Set Fees to Arbitrary Amount", async () => {
      const previousFee = await getBridgeInfo(
        connection,
        CORE_BRIDGE_ADDRESS
      ).then((info) => info.config.fee);

      const newFeeAmount = previousFee + BigInt(69420);
      const message = governance.publishSetMessageFee(1, newFeeAmount);
      const signedVaa = guardians.addSignatures(message, [0]);
      // console.log(`signedVaa: ${signedVaa.toString("base64")}`);

      const txSignatures = await postVaa(
        connection,
        wallet.signTransaction,
        CORE_BRIDGE_ADDRESS,
        wallet.key(),
        signedVaa
      ).then((results) => results.map((result) => result.signature));
      const postTx = txSignatures.pop()!;
      for (const verifyTx of txSignatures) {
        console.log(`verifySignatures: ${verifyTx}`);
      }
      console.log(`postVaa:          ${postTx}`);

      const setFeeTx = await web3.sendAndConfirmTransaction(
        connection,
        new web3.Transaction().add(
          createSetFeesInstruction(CORE_BRIDGE_ADDRESS, wallet.key(), signedVaa)
        ),
        [wallet.signer()]
      );
      console.log(`setFee:           ${setFeeTx}`);

      const currentFee = await getBridgeInfo(
        connection,
        CORE_BRIDGE_ADDRESS
      ).then((info) => info.config.fee);
      expect(currentFee).to.equal(newFeeAmount);
    });
  });

  describe("Instruction 4: Transfer Fees", () => {
    // this test is a little silly because we will not have had anyone using
    // the core bridge where someone will have paid fees. So we just demonstrate
    // that the instruction works by sending 0 lamports to an arbitrary recipient
    it("Transfer Fees to Recipient", async () => {
      const recipient = web3.Keypair.generate().publicKey;
      //const balanceBefore = await connection.getBalance(recipient);

      const message = governance.publishTransferFees(1, 0n, recipient);
      const signedVaa = guardians.addSignatures(message, [0]);
      // console.log(`signedVaa: ${signedVaa.toString("base64")}`);

      const txSignatures = await postVaa(
        connection,
        wallet.signTransaction,
        CORE_BRIDGE_ADDRESS,
        wallet.key(),
        signedVaa
      ).then((results) => results.map((result) => result.signature));
      const postTx = txSignatures.pop()!;
      for (const verifyTx of txSignatures) {
        console.log(`verifySignatures: ${verifyTx}`);
      }
      console.log(`postVaa:          ${postTx}`);

      const transferFeeTx = await web3.sendAndConfirmTransaction(
        connection,
        new web3.Transaction().add(
          createTransferFeesInstruction(
            CORE_BRIDGE_ADDRESS,
            wallet.key(),
            recipient,
            signedVaa
          )
        ),
        [wallet.signer()]
      );
      console.log(`transferFee:      ${transferFeeTx}`);

      //const balanceAfter = await connection.getBalance(recipient);
    });
  });

  describe("Instruction 5: Upgrade Contract", () => {
    // TODO: need to write bpf to buffer and verify upgrade_contract instruction
  });

  describe("Instruction 6: Upgrade Guardian Set", () => {
    it("Upgrade Guardian Set to 19 Guardians", async () => {
      const newGuardianSetIndex = guardians.setIndex + 1;
      const newGuardianSet = guardians.getPublicKeys();
      const message = governance.publishGuardianSetUpgrade(
        newGuardianSetIndex,
        newGuardianSet
      );
      const signedVaa = guardians.addSignatures(message, [0]);
      // console.log(`signedVaa: ${signedVaa.toString("base64")}`);

      const txSignatures = await postVaa(
        connection,
        wallet.signTransaction,
        CORE_BRIDGE_ADDRESS,
        wallet.key(),
        signedVaa
      ).then((results) => results.map((result) => result.signature));
      const postTx = txSignatures.pop()!;
      for (const verifyTx of txSignatures) {
        console.log(`verifySignatures:   ${verifyTx}`);
      }
      console.log(`postVaa:            ${postTx}`);

      const parsed = parseVaa(signedVaa);
      const upgradeTx = await web3.sendAndConfirmTransaction(
        connection,
        new web3.Transaction().add(
          createUpgradeGuardianSetInstruction(
            CORE_BRIDGE_ADDRESS,
            wallet.key(),
            parsed
          )
        ),
        [wallet.signer()]
      );
      console.log(`upgradeGuardianSet: ${upgradeTx}`);

      // update guardian's set index now and verify upgrade
      guardians.updateGuardianSetIndex(newGuardianSetIndex);

      const guardianSetData = await getGuardianSet(
        connection,
        CORE_BRIDGE_ADDRESS,
        newGuardianSetIndex
      );
      expect(guardianSetData.index).to.equal(newGuardianSetIndex);
      expect(guardianSetData.creationTime).to.equal(parsed.timestamp);
      for (let i = 0; i < newGuardianSet.length; ++i) {
        const key = guardianSetData.keys.at(i)!;
        const expectedKey = newGuardianSet.at(i)!;
        expect(Buffer.compare(key, expectedKey)).to.equal(0);
      }
    });

    it("Post VAA Signed with 13 Guardians", async () => {
      const message = Buffer.from("All your base are belong to us.");
      const nonce = 0;
      const consistencyLevel = 15;
      const published = ethereumWormhole.publishMessage(
        nonce,
        message,
        consistencyLevel
      );
      const signedVaa = guardians.addSignatures(
        published,
        [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18]
      );
      // console.log(`signedVaa: ${signedVaa.toString("base64")}`);

      const txSignatures = await postVaa(
        connection,
        wallet.signTransaction,
        CORE_BRIDGE_ADDRESS,
        wallet.key(),
        signedVaa
      ).then((results) => results.map((result) => result.signature));
      const postTx = txSignatures.pop()!;
      for (const verifyTx of txSignatures) {
        console.log(`verifySignatures: ${verifyTx}`);
      }
      console.log(`postVaa:          ${postTx}`);

      // verify data
      const parsed = parseVaa(signedVaa);
      const messageData = await getPostedVaa(
        connection,
        CORE_BRIDGE_ADDRESS,
        parsed.hash
      ).then((postedVaa) => postedVaa.message);
      expect(messageData.consistencyLevel).to.equal(consistencyLevel);
      expect(messageData.consistencyLevel).to.equal(parsed.consistencyLevel);
      expect(messageData.emitterAddress.toString("hex")).to.equal(
        parsed.emitterAddress
      );
      expect(messageData.emitterChain).to.equal(parsed.emitterChain);
      expect(messageData.nonce).to.equal(nonce);
      expect(messageData.nonce).to.equal(parsed.nonce);
      expect(Buffer.compare(messageData.payload, message)).to.equal(0);
      expect(Buffer.compare(messageData.payload, parsed.payload)).to.equal(0);
      expect(messageData.sequence).to.equal(parsed.sequence);
      expect(messageData.vaaTime).to.equal(parsed.timestamp);
      expect(messageData.vaaVersion).to.equal(parsed.version);
    });

    it("Post VAA Signed with 19 Guardians", async () => {
      const message = Buffer.from("All your base are belong to us.");
      const nonce = 0;
      const consistencyLevel = 15;
      const published = ethereumWormhole.publishMessage(
        nonce,
        message,
        consistencyLevel
      );
      const signedVaa = guardians.addSignatures(published, [
        ...Array(19).keys(),
      ]);
      // console.log(`signedVaa: ${signedVaa.toString("base64")}`);

      const txSignatures = await postVaa(
        connection,
        wallet.signTransaction,
        CORE_BRIDGE_ADDRESS,
        wallet.key(),
        signedVaa
      ).then((results) => results.map((result) => result.signature));
      const postTx = txSignatures.pop()!;
      for (const verifyTx of txSignatures) {
        console.log(`verifySignatures: ${verifyTx}`);
      }
      console.log(`postVaa:          ${postTx}`);

      // verify data
      const parsed = parseVaa(signedVaa);
      const messageData = await getPostedVaa(
        connection,
        CORE_BRIDGE_ADDRESS,
        parsed.hash
      ).then((postedVaa) => postedVaa.message);
      expect(messageData.consistencyLevel).to.equal(consistencyLevel);
      expect(messageData.consistencyLevel).to.equal(parsed.consistencyLevel);
      expect(messageData.emitterAddress.toString("hex")).to.equal(
        parsed.emitterAddress
      );
      expect(messageData.emitterChain).to.equal(parsed.emitterChain);
      expect(messageData.nonce).to.equal(nonce);
      expect(messageData.nonce).to.equal(parsed.nonce);
      expect(Buffer.compare(messageData.payload, message)).to.equal(0);
      expect(Buffer.compare(messageData.payload, parsed.payload)).to.equal(0);
      expect(messageData.sequence).to.equal(parsed.sequence);
      expect(messageData.vaaTime).to.equal(parsed.timestamp);
      expect(messageData.vaaVersion).to.equal(parsed.version);
    });
  });

  describe("Instruction 8: Post Message Unreliable", () => {
    // lol
  });
});
