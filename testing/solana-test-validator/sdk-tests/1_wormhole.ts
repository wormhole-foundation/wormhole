import { expect } from "chai";
import * as web3 from "@solana/web3.js";
import {
  MockGuardians,
  MockEthereumEmitter,
  GovernanceEmitter,
} from "../../../sdk/js/src/mock";
import { parseVaa, parseGovernanceVaa } from "../../../sdk/js/src/vaa";
import {
  getPostedVaa,
  getGuardianSet,
  createSetFeesInstruction,
  createTransferFeesInstruction,
  createUpgradeGuardianSetInstruction,
  getWormholeBridgeData,
  getInitializeAccounts,
  getPostMessageAccounts,
  getPostVaaAccounts,
  getSetFeesAccounts,
  getTransferFeesAccounts,
  getUpgradeGuardianSetAccounts,
  getVerifySignatureAccounts,
  getUpgradeContractAccounts,
  getSignatureSetData,
} from "../../../sdk/js/src/solana/wormhole";
import { postVaa } from "../../../sdk/js/src/solana/sendAndConfirmPostVaa";
import {
  BpfLoaderUpgradeable,
  NodeWallet,
  getPostMessageCpiAccounts,
} from "../../../sdk/js/src/solana";

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
    const governance = new GovernanceEmitter(
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
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.bridge.toString()).to.equal(
        "DNN2VhmrGTGj6QVnPz4NVfsiSk64cRHzKBLP5kUaQrf8"
      );
      expect(accounts.guardianSet.toString()).to.equal(
        "BJmSHooX4QJCTE4bn5G2Pv6in1nLGyWvL3jxWmT5Avdm"
      );
      expect(accounts.feeCollector.toString()).to.equal(
        "Cxt3Uka7X8vyHYjU6szcuYVPPFyg1fAtoeVy7eyzPjGV"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).is.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
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
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.bridge.toString()).to.equal(
        "DNN2VhmrGTGj6QVnPz4NVfsiSk64cRHzKBLP5kUaQrf8"
      );
      expect(accounts.message.equals(message.publicKey)).is.true;
      expect(accounts.emitter.toString()).to.equal(
        "Ard2Zy4HckbJS2bL7y4361wbKSUH68JZqYBura5d4xtw"
      );
      expect(accounts.sequence.toString()).to.equal(
        "Gdeob8iLpTN4Fc8BEgRdFUWikdUsvrv9Rfc1rNQWy4b7"
      );
      expect(accounts.feeCollector.toString()).to.equal(
        "Cxt3Uka7X8vyHYjU6szcuYVPPFyg1fAtoeVy7eyzPjGV"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).is.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
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
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.guardianSet.toString()).to.equal(
        "BJmSHooX4QJCTE4bn5G2Pv6in1nLGyWvL3jxWmT5Avdm"
      );
      expect(accounts.bridge.toString()).to.equal(
        "DNN2VhmrGTGj6QVnPz4NVfsiSk64cRHzKBLP5kUaQrf8"
      );
      expect(accounts.signatureSet.equals(signatureSet.publicKey)).is.true;
      expect(accounts.vaa.toString()).to.equal(
        "5UfHDKqHwQnMtHjnqfpZJxmAeCyMWD7kYEPcfeKQwvRY"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).is.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
    });

    it("Instruction 4: Set Fees", () => {
      const timestamp = 23456789;
      const newFeeAmount = 42069n;
      const message = governance.publishWormholeSetMessageFee(
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
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.bridge.toString()).to.equal(
        "DNN2VhmrGTGj6QVnPz4NVfsiSk64cRHzKBLP5kUaQrf8"
      );
      expect(accounts.vaa.toString()).to.equal(
        "Bfon9eTUxC8t9PfryWKChbcVnYGUu7nzTXNaY16DNWEM"
      );
      expect(accounts.claim.toString()).to.equal(
        "BgMQaDvs4m9B2NMPbvjRw3VedZ3nnR3E77Cd5cJ8EjV9"
      );
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
    });

    it("Instruction 5: Transfer Fees", () => {
      const timestamp = 34567890;
      const chain = 1;
      const amount = 0n;
      const recipient = payer;
      const message = governance.publishWormholeTransferFees(
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
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.bridge.toString()).to.equal(
        "DNN2VhmrGTGj6QVnPz4NVfsiSk64cRHzKBLP5kUaQrf8"
      );
      expect(accounts.vaa.toString()).to.equal(
        "CTGU5aCPifz9JXveMKjXYqXNPPTx6n49XDEzffxhth15"
      );
      expect(accounts.claim.toString()).to.equal(
        "9cR8vkCTqSctEREA8yjayLbaDr8FGjnuQFF5ABNVQzzR"
      );
      expect(accounts.feeCollector.toString()).to.equal(
        "Cxt3Uka7X8vyHYjU6szcuYVPPFyg1fAtoeVy7eyzPjGV"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
    });

    it("Instruction 6: Upgrade Contract", () => {
      const timestamp = 45678901;
      const chain = 1;
      const implementation = new web3.PublicKey(
        "2B5wMnErS8oKWV1wPTNQQhM1WLyxee2obtBMDtsYeHgA"
      );
      const message = governance.publishWormholeUpgradeContract(
        timestamp,
        chain,
        implementation.toString()
      );
      const signedVaa = guardians.addSignatures(message, [0]);

      const accounts = getUpgradeContractAccounts(
        CORE_BRIDGE_ADDRESS,
        payer,
        signedVaa
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.bridge.toString()).to.equal(
        "DNN2VhmrGTGj6QVnPz4NVfsiSk64cRHzKBLP5kUaQrf8"
      );
      expect(accounts.vaa.toString()).to.equal(
        "8Paf1ZasFS8EoJJfaZASChsHS77pm6LvFUypAbRmPYhZ"
      );
      expect(accounts.claim.toString()).to.equal(
        "3mVQfmBT2g933Bm3yTVtd5Yz4PQkLPMnTgrmkpShAeXM"
      );
      expect(accounts.upgradeAuthority.toString()).to.equal(
        "2Esys2cab9dkWeApHewy7nqx6tNUWKGtFchyhRpzGmR6"
      );
      expect(accounts.spill.equals(payer)).is.true;
      expect(accounts.implementation.equals(implementation)).is.true;
      expect(accounts.programData.toString()).to.equal(
        "Bi88esKkELCqVAYUFjREwnWjeK4RUWecHv7VxZQtVj4f"
      );
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).is.true;
      expect(
        accounts.bpfLoaderUpgradeable.equals(BpfLoaderUpgradeable.programId)
      ).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
    });

    it("Instruction 7: Upgrade Guardian Set", () => {
      const timestamp = 56789012;
      const newGuardianSetIndex = guardians.setIndex + 1;
      const newGuardianSet = guardians.getPublicKeys();
      const message = governance.publishWormholeGuardianSetUpgrade(
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
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.bridge.toString()).to.equal(
        "DNN2VhmrGTGj6QVnPz4NVfsiSk64cRHzKBLP5kUaQrf8"
      );
      expect(accounts.vaa.toString()).to.equal(
        "DeisQv7bpLMenGLiqWGmkdVnyXrfbMmttf9iVXsWsSDg"
      );
      expect(accounts.claim.toString()).to.equal(
        "3up9EcEUXnxkiBfdxBTfK4FuJagHXbzHFbn7uXLWwqt4"
      );
      expect(accounts.guardianSetOld.toString()).to.equal(
        "BJmSHooX4QJCTE4bn5G2Pv6in1nLGyWvL3jxWmT5Avdm"
      );
      expect(accounts.guardianSetNew.toString()).to.equal(
        "mp6ZSV2cM3B8YHuySMsn4yJzpjPG2YLMvqtfGtEMxRX"
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
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.guardianSet.toString()).to.equal(
        "BJmSHooX4QJCTE4bn5G2Pv6in1nLGyWvL3jxWmT5Avdm"
      );
      expect(accounts.signatureSet.equals(signatureSet.publicKey)).is.true;
      expect(accounts.instructions.equals(web3.SYSVAR_INSTRUCTIONS_PUBKEY)).to
        .be.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
    });
  });

  describe("CPI Accounts", () => {
    const payer = new web3.PublicKey(
      "6sbzC1eH4FTujJXWj51eQe25cYvr4xfXbJ1vAj7j2k5J"
    );

    // mock program integrating wormhole
    const cpiProgramId = new web3.PublicKey(
      "pFCBP4bhqdSsrWUVTgqhPsLrfEdChBK17vgFM7TxjxQ"
    );

    it("getPostMessageCpiAccounts", () => {
      const message = web3.Keypair.generate();

      const accounts = getPostMessageCpiAccounts(
        cpiProgramId,
        CORE_BRIDGE_ADDRESS,
        payer,
        message.publicKey
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.wormholeBridge.toString()).to.equal(
        "DNN2VhmrGTGj6QVnPz4NVfsiSk64cRHzKBLP5kUaQrf8"
      );
      expect(accounts.wormholeMessage.equals(message.publicKey)).is.true;
      expect(accounts.wormholeEmitter.toString()).to.equal(
        "Ernk5wzhwTPJDbmTNnELqhxW5J85CH45qJSTsGkKpGYK"
      );
      expect(accounts.wormholeSequence.toString()).to.equal(
        "5w3YWnJUVbuDvBpymrsu6oecpwY17n82Nw9b9qXZ1z6m"
      );
      expect(accounts.wormholeFeeCollector.toString()).to.equal(
        "Cxt3Uka7X8vyHYjU6szcuYVPPFyg1fAtoeVy7eyzPjGV"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).is.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
    });
  });

  describe("Wormhole Program Interaction", () => {
    // for generating governance wormhole messages
    const governance = new GovernanceEmitter(
      GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
      10
    );

    // hijacking the ethereum token bridge address for our fake emitter
    const ethereumWormhole = new MockEthereumEmitter(ETHEREUM_WALLET_BYTES32);

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
        const signingGuardians = [0];
        const signedVaa = guardians.addSignatures(published, signingGuardians);
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

        const signatureSetData = await getSignatureSetData(
          connection,
          messageData.vaaSignatureAccount
        );
        const signed = signatureSetData.signatures;
        expect(signed).has.length(1);
        expect(signed.filter((x) => !x)).has.length(0);
        for (const i of signingGuardians) {
          expect(signed[i]).is.true;
        }
        expect(Buffer.compare(signatureSetData.hash, parsed.hash)).to.equal(0);
        expect(signatureSetData.guardianSetIndex).to.equal(guardians.setIndex);
      });

      // it("Post Message Unreliable", () => {
      //   // jk
      // });
    });

    describe("Governance", () => {
      it("Set Fees to Arbitrary Amount", async () => {
        const previousFee = await getWormholeBridgeData(
          connection,
          CORE_BRIDGE_ADDRESS
        ).then((info) => info.config.fee);

        const timestamp = 1;
        const newFeeAmount = previousFee + BigInt(69420);
        const message = governance.publishWormholeSetMessageFee(
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
        // console.log(`setFeeTx:         ${setFeeTx}`);

        const currentFee = await getWormholeBridgeData(
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
        const message = governance.publishWormholeTransferFees(
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
        // console.log(`transferFeeTx:    ${transferFeeTx}`);

        //const balanceAfter = await connection.getBalance(recipient);
      });

      it("Upgrade Guardian Set to 19 Guardians", async () => {
        const timestamp = 3;
        const newGuardianSetIndex = guardians.setIndex + 1;
        const newGuardianSet = guardians.getPublicKeys();
        const message = governance.publishWormholeGuardianSetUpgrade(
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
        const signingGuardians = [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18];
        const signedVaa = guardians.addSignatures(published, signingGuardians);
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

        const signatureSetData = await getSignatureSetData(
          connection,
          messageData.vaaSignatureAccount
        );
        const signed = signatureSetData.signatures;
        expect(signed).has.length(guardians.signers.length);
        expect(signed.filter((x) => !x)).has.length(
          19 - signingGuardians.length
        );
        for (const i of signingGuardians) {
          expect(signed[i]).is.true;
        }
        expect(Buffer.compare(signatureSetData.hash, parsed.hash)).to.equal(0);
        expect(signatureSetData.guardianSetIndex).to.equal(guardians.setIndex);
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
        const signingGuardians = [...Array(19).keys()];
        const signedVaa = guardians.addSignatures(published, signingGuardians);
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

        const signatureSetData = await getSignatureSetData(
          connection,
          messageData.vaaSignatureAccount
        );
        const signed = signatureSetData.signatures;
        expect(signed).has.length(guardians.signers.length);
        expect(signed.filter((x) => !x)).has.length(
          19 - signingGuardians.length
        );
        for (const i of signingGuardians) {
          expect(signed[i]).is.true;
        }
        expect(Buffer.compare(signatureSetData.hash, parsed.hash)).to.equal(0);
        expect(signatureSetData.guardianSetIndex).to.equal(guardians.setIndex);
      });
    });
  });
});
