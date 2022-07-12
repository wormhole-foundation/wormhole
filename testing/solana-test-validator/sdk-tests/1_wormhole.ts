import { expect } from "chai";
import * as web3 from "@solana/web3.js";
import {
  MockGuardians,
  MockEthereumEmitter,
  WormholeGovernanceEmitter,
  ethPrivateToPublic,
} from "../../../sdk/js/src/mock";
import { parseVaa, parseGovernanceVaa } from "../../../sdk/js/src/vaa";
import {
  getPostedVaa,
  getGuardianSet,
  createSetFeesInstruction,
  createTransferFeesInstruction,
  createUpgradeGuardianSetInstruction,
  getWormholeInfo,
  createInitializeInstruction,
  getInitializeAccounts,
  getPostMessageAccounts,
  getPostVaaAccounts,
  getSetFeesAccounts,
  getTransferFeesAccounts,
  getUpgradeGuardianSetAccounts,
  getVerifySignatureAccounts,
} from "../../../sdk/js/src/solana/wormhole";
import { postVaa } from "../../../sdk/js/src/solana/sendAndConfirmPostVaa";
import { NodeWallet } from "../../../sdk/js/src/solana/utils";

import {
  CORE_BRIDGE_ADDRESS,
  ETHEREUM_WALLET_BYTES32,
  GOVERNANCE_EMITTER_ADDRESS,
  GUARDIAN_KEYS,
  GUARDIAN_SET_INDEX,
  LOCALHOST,
  TOKEN_BRIDGE_ADDRESS,
} from "./helpers/consts";

describe("Wormhole (Core Bridge)", () => {
  const connection = new web3.Connection(LOCALHOST, "processed");

  const wallet = new NodeWallet(web3.Keypair.generate());

  // for signing wormhole messages
  const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

  const localVariables: any = {};

  before("Airdrop SOL", async () => {
    await connection
      .requestAirdrop(wallet.key(), 1000 * web3.LAMPORTS_PER_SOL)
      .then(async (signature) => connection.confirmTransaction(signature));
  });

  describe("Accounts", () => {
    // for generating governance wormhole messages
    const governance = new WormholeGovernanceEmitter(
      GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex")
    );

    // hijacking the ethereum token bridge address for our fake emitter
    const ethereumWormhole = new MockEthereumEmitter(ETHEREUM_WALLET_BYTES32);

    const payer = new web3.PublicKey(
      "6sbzC1eH4FTujJXWj51eQe25cYvr4xfXbJ1vAj7j2k5J"
    );

    it("Instruction 1: Initialize", () => {
      const accounts = getInitializeAccounts(CORE_BRIDGE_ADDRESS, payer);

      // verify accounts
      expect(accounts.bridge.toString()).to.equal(
        "FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP"
      );
      expect(accounts.guardianSet.toString()).to.equal(
        "6MxkvoEwgB9EqQRLNhvYaPGhfcLtBtpBqdQugr3AZUgD"
      );
      expect(accounts.feeCollector.toString()).to.equal(
        "GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).to.be.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).to.be.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
    });

    it("Instruction 2: Post Message", () => {
      const message = web3.Keypair.generate();

      const accounts = getPostMessageAccounts(
        CORE_BRIDGE_ADDRESS,
        payer,
        TOKEN_BRIDGE_ADDRESS,
        message.publicKey
      );

      // verify accounts
      expect(accounts.bridge.toString()).to.equal(
        "FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP"
      );
      expect(accounts.message.equals(message.publicKey)).to.be.true;
      expect(accounts.emitter.toString()).to.equal(
        "ENG1wQ7CQKH8ibAJ1hSLmJgL9Ucg6DRDbj752ZAfidLA"
      );
      expect(accounts.sequence.toString()).to.equal(
        "7F4RNrCkBJxs1uidvF96iPieZ8upkEnc8NdpHoJ8YjxH"
      );
      expect(accounts.feeCollector.toString()).to.equal(
        "GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).to.be.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).to.be.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
    });

    it("Instruction 3: Post VAA", () => {
      const message = Buffer.from("All your base are belong to us.");
      const nonce = 0;
      const consistencyLevel = 15;
      const timestamp = 12345678;
      const published = ethereumWormhole.publishMessage(
        nonce,
        message,
        consistencyLevel,
        timestamp
      );
      const signedVaa = guardians.addSignatures(published, [0]);

      const signatureSet = web3.Keypair.generate();
      const accounts = getPostVaaAccounts(
        CORE_BRIDGE_ADDRESS,
        payer,
        signatureSet.publicKey,
        signedVaa
      );

      // verify accounts
      expect(accounts.guardianSet.toString()).to.equal(
        "6MxkvoEwgB9EqQRLNhvYaPGhfcLtBtpBqdQugr3AZUgD"
      );
      expect(accounts.bridge.toString()).to.equal(
        "FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP"
      );
      expect(accounts.signatureSet.equals(signatureSet.publicKey)).to.be.true;
      expect(accounts.vaa.toString()).to.equal(
        "smp3N82nif213rmZYz3s8m5S69ts1ZSiHcZVDDMnGqZ"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).to.be.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).to.be.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
    });

    it("Instruction 4: Set Fees", () => {
      const timestamp = 23456789;
      const newFeeAmount = 42069n;
      const message = governance.publishSetMessageFee(
        timestamp,
        1,
        newFeeAmount
      );
      const signedVaa = guardians.addSignatures(message, [0]);

      const accounts = getSetFeesAccounts(
        CORE_BRIDGE_ADDRESS,
        payer,
        signedVaa
      );

      // verify accounts
      expect(accounts.bridge.toString()).to.equal(
        "FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP"
      );
      expect(accounts.vaa.toString()).to.equal(
        "CHUdr8ajDxfLqqYrDCad2k4js4xb21MZU4foPF4xiTMr"
      );
      expect(accounts.claim.toString()).to.equal(
        "47ZotAwbp8GGQmZVq3kLQ4f9yBHscCJBPV2KJXMaxisB"
      );
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
    });

    it("Instruction 5: Transfer Fees", () => {
      const timestamp = 34567890;
      const chain = 1;
      const amount = 0n;
      const recipient = payer;
      const message = governance.publishTransferFees(
        timestamp,
        chain,
        amount,
        recipient.toBuffer()
      );
      const signedVaa = guardians.addSignatures(message, [0]);

      const accounts = getTransferFeesAccounts(
        CORE_BRIDGE_ADDRESS,
        payer,
        recipient,
        signedVaa
      );

      // verify accounts
      expect(accounts.bridge.toString()).to.equal(
        "FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP"
      );
      expect(accounts.vaa.toString()).to.equal(
        "Fm1NGCiz1bWNLvJyKrB3V1Y5TXS5NuVECbqRECVrS8Mh"
      );
      expect(accounts.claim.toString()).to.equal(
        "AnconEKKTRjhicZXkTczB9Sfqz8yXSokq75KpERoQe7h"
      );
      expect(accounts.feeCollector.toString()).to.equal(
        "GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).to.be.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
    });

    it("Instruction 6: Upgrade Contract", () => {
      const timestamp = 45678901;
      // TODO
    });

    it("Instruction 7: Upgrade Guardian Set", () => {
      const timestamp = 56789012;
      const newGuardianSetIndex = guardians.setIndex + 1;
      const newGuardianSet = guardians.getPublicKeys();
      const message = governance.publishGuardianSetUpgrade(
        timestamp,
        newGuardianSetIndex,
        newGuardianSet
      );
      const signedVaa = guardians.addSignatures(message, [0]);

      const accounts = getUpgradeGuardianSetAccounts(
        CORE_BRIDGE_ADDRESS,
        payer,
        signedVaa
      );

      // verify accounts
      expect(accounts.bridge.toString()).to.equal(
        "FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP"
      );
      expect(accounts.vaa.toString()).to.equal(
        "7keWEVKwLSNrXLpUguvPpCvBejfnkdVZb9aYJn4TfKM7"
      );
      expect(accounts.claim.toString()).to.equal(
        "61CMENz2PV9BTswhzMCEaLZFpSXQbeshjCAfeEAYHfGf"
      );
      expect(accounts.guardianSetOld.toString()).to.equal(
        "6MxkvoEwgB9EqQRLNhvYaPGhfcLtBtpBqdQugr3AZUgD"
      );
      expect(accounts.guardianSetNew.toString()).to.equal(
        "C45UtUx2ihfZCTVXSxX71G7vhZ61r2BS7TwC8Zc7CZmQ"
      );
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
    });

    it("Instruction 8: Verify Signatures", () => {
      const timestamp = 67890123;
      const message = Buffer.from("All your base are belong to us.");
      const nonce = 0;
      const consistencyLevel = 15;
      const published = ethereumWormhole.publishMessage(
        nonce,
        message,
        consistencyLevel,
        timestamp
      );
      const signedVaa = guardians.addSignatures(published, [0]);

      const signatureSet = web3.Keypair.generate();
      const accounts = getVerifySignatureAccounts(
        CORE_BRIDGE_ADDRESS,
        payer,
        signatureSet.publicKey,
        signedVaa
      );

      // verify accounts
      expect(accounts.guardianSet.toString()).to.equal(
        "6MxkvoEwgB9EqQRLNhvYaPGhfcLtBtpBqdQugr3AZUgD"
      );
      expect(accounts.signatureSet.equals(signatureSet.publicKey)).to.be.true;
      expect(accounts.instructions.equals(web3.SYSVAR_INSTRUCTIONS_PUBKEY)).to
        .be.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).to.be.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
    });
  });

  describe("Wormhole Program Interaction", () => {
    // for generating governance wormhole messages
    const governance = new WormholeGovernanceEmitter(
      GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex")
    );

    // hijacking the ethereum token bridge address for our fake emitter
    const ethereumWormhole = new MockEthereumEmitter(ETHEREUM_WALLET_BYTES32);

    describe("Setup Wormhole Program", () => {
      it("Initialize", async () => {
        const guardianSetExpirationTime = 86400;
        const fee = 100n;
        const initialGuardians = guardians.getPublicKeys().slice(0, 1);

        guardians.getPublicKeys().slice();
        const initializeTx = await web3.sendAndConfirmTransaction(
          connection,
          new web3.Transaction().add(
            createInitializeInstruction(
              CORE_BRIDGE_ADDRESS,
              wallet.key(),
              guardianSetExpirationTime,
              fee,
              initialGuardians
            )
          ),
          [wallet.signer()]
        );
        // console.log(`initializeTx: ${initializeTx}`);

        // verify data
        const info = await getWormholeInfo(connection, CORE_BRIDGE_ADDRESS);
        expect(info.guardianSetIndex).to.equal(GUARDIAN_SET_INDEX);
        expect(info.config.guardianSetExpirationTime).to.equal(
          guardianSetExpirationTime
        );
        expect(info.config.fee).to.equal(fee);

        const guardianSet = await getGuardianSet(
          connection,
          CORE_BRIDGE_ADDRESS,
          0
        );
        expect(guardianSet.index).to.equal(GUARDIAN_SET_INDEX);
        expect(guardianSet.expirationTime).to.equal(0);
        expect(guardianSet.keys).has.length(1);
        expect(
          Buffer.compare(initialGuardians[0], guardianSet.keys.at(0)!)
        ).to.equal(0);
      });
    });

    describe("Post VAA with One Guardian", () => {
      it("Verify Guardian Signature and Post Message", async () => {
        const message = Buffer.from("All your base are belong to us.");
        const nonce = 0;
        const consistencyLevel = 15;
        const timestamp = 12345678;
        const published = ethereumWormhole.publishMessage(
          nonce,
          message,
          consistencyLevel,
          timestamp
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
          // console.log(`verifySignatures: ${verifyTx}`);
        }
        // console.log(`postVaa:          ${postTx}`);

        // verify data
        const parsed = parseVaa(signedVaa);
        const messageData = await getPostedVaa(
          connection,
          CORE_BRIDGE_ADDRESS,
          parsed.hash
        ).then((postedVaa) => postedVaa.message);

        expect(messageData.consistencyLevel).to.equal(consistencyLevel);
        expect(messageData.consistencyLevel).to.equal(parsed.consistencyLevel);
        expect(
          Buffer.compare(messageData.emitterAddress, parsed.emitterAddress)
        ).to.equal(0);
        expect(messageData.emitterChain).to.equal(parsed.emitterChain);
        expect(messageData.nonce).to.equal(nonce);
        expect(messageData.nonce).to.equal(parsed.nonce);
        expect(Buffer.compare(messageData.payload, message)).to.equal(0);
        expect(Buffer.compare(messageData.payload, parsed.payload)).to.equal(0);
        expect(messageData.sequence).to.equal(parsed.sequence);
        expect(messageData.vaaTime).to.equal(timestamp);
        expect(messageData.vaaTime).to.equal(parsed.timestamp);
        expect(messageData.vaaVersion).to.equal(parsed.version);
      });

      // it("Post Message Unreliable", () => {
      //   // jk
      // });
    });

    describe("Governance", () => {
      it("Set Fees to Arbitrary Amount", async () => {
        const previousFee = await getWormholeInfo(
          connection,
          CORE_BRIDGE_ADDRESS
        ).then((info) => info.config.fee);

        const timestamp = 1;
        const newFeeAmount = previousFee + BigInt(69420);
        const message = governance.publishSetMessageFee(
          timestamp,
          1,
          newFeeAmount
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
          // console.log(`verifySignatures: ${verifyTx}`);
        }
        // console.log(`postVaa:          ${postTx}`);

        const setFeeTx = await web3.sendAndConfirmTransaction(
          connection,
          new web3.Transaction().add(
            createSetFeesInstruction(
              CORE_BRIDGE_ADDRESS,
              wallet.key(),
              signedVaa
            )
          ),
          [wallet.signer()]
        );
        // console.log(`setFee:           ${setFeeTx}`);

        const currentFee = await getWormholeInfo(
          connection,
          CORE_BRIDGE_ADDRESS
        ).then((info) => info.config.fee);
        expect(currentFee).to.equal(newFeeAmount);
      });

      // this test is a little silly because we will not have had anyone using
      // the core bridge where someone will have paid fees. So we just demonstrate
      // that the instruction works by sending 0 lamports to an arbitrary recipient
      it("Transfer Fees to Recipient", async () => {
        const recipient = web3.Keypair.generate().publicKey;
        //const balanceBefore = await connection.getBalance(recipient);

        const timestamp = 2;
        const chain = 1;
        const amount = 0n;
        const message = governance.publishTransferFees(
          timestamp,
          chain,
          amount,
          recipient.toBuffer()
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
          // console.log(`verifySignatures: ${verifyTx}`);
        }
        // console.log(`postVaa:          ${postTx}`);

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
        // console.log(`transferFee:      ${transferFeeTx}`);

        //const balanceAfter = await connection.getBalance(recipient);
      });

      it("Upgrade Contract", () => {
        // TODO: need to write bpf to buffer and verify upgrade_contract instruction
      });

      it("Upgrade Guardian Set to 19 Guardians", async () => {
        const timestamp = 3;
        const newGuardianSetIndex = guardians.setIndex + 1;
        const newGuardianSet = guardians.getPublicKeys();
        const message = governance.publishGuardianSetUpgrade(
          timestamp,
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
          // console.log(`verifySignatures:   ${verifyTx}`);
        }
        // console.log(`postVaa:            ${postTx}`);

        const parsed = parseGovernanceVaa(signedVaa);
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
        // console.log(`upgradeGuardianSet: ${upgradeTx}`);

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
    });

    describe("Post VAA with 19 Guardians", () => {
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
          // console.log(`verifySignatures: ${verifyTx}`);
        }
        // console.log(`postVaa:          ${postTx}`);

        // verify data
        const parsed = parseVaa(signedVaa);
        const messageData = await getPostedVaa(
          connection,
          CORE_BRIDGE_ADDRESS,
          parsed.hash
        ).then((postedVaa) => postedVaa.message);
        expect(messageData.consistencyLevel).to.equal(consistencyLevel);
        expect(messageData.consistencyLevel).to.equal(parsed.consistencyLevel);
        expect(
          Buffer.compare(messageData.emitterAddress, parsed.emitterAddress)
        ).to.equal(0);
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
          // console.log(`verifySignatures: ${verifyTx}`);
        }
        // console.log(`postVaa:          ${postTx}`);

        // verify data
        const parsed = parseVaa(signedVaa);
        const messageData = await getPostedVaa(
          connection,
          CORE_BRIDGE_ADDRESS,
          parsed.hash
        ).then((postedVaa) => postedVaa.message);
        expect(messageData.consistencyLevel).to.equal(consistencyLevel);
        expect(messageData.consistencyLevel).to.equal(parsed.consistencyLevel);
        expect(
          Buffer.compare(messageData.emitterAddress, parsed.emitterAddress)
        ).to.equal(0);
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
  });
});
