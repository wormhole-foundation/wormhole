import { expect } from "chai";
import * as web3 from "@solana/web3.js";
import {
  Metadata,
  PROGRAM_ID as TOKEN_METADATA_PROGRAM_ID,
} from "@metaplex-foundation/mpl-token-metadata";
import {
  createMint,
  getAccount,
  getAssociatedTokenAddressSync,
  getMint,
  getOrCreateAssociatedTokenAccount,
  mintTo,
  NATIVE_MINT,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import {
  MockGuardians,
  GovernanceEmitter,
  MockEthereumTokenBridge,
} from "../../../sdk/js/src/mock";
import {
  createApproveAuthoritySignerInstruction,
  createAttestTokenInstruction,
  createCompleteTransferNativeInstruction,
  createCompleteTransferWrappedInstruction,
  createCreateWrappedInstruction,
  createRegisterChainInstruction,
  createTransferNativeInstruction,
  createTransferNativeWithPayloadInstruction,
  createTransferWrappedInstruction,
  createTransferWrappedWithPayloadInstruction,
  deriveCustodyKey,
  deriveEndpointKey,
  deriveMintAuthorityKey,
  deriveRedeemerAccountKey,
  deriveTokenMetadataKey,
  deriveWrappedMintKey,
  getAttestTokenAccounts,
  getCompleteTransferNativeAccounts,
  getCompleteTransferWrappedAccounts,
  getCreateWrappedAccounts,
  getEndpointRegistration,
  getInitializeAccounts,
  getRegisterChainAccounts,
  getTransferNativeAccounts,
  getTransferNativeWithPayloadAccounts,
  getTransferWrappedAccounts,
  getTransferWrappedWithPayloadAccounts,
  getUpgradeContractAccounts,
  getWrappedMeta,
} from "../../../sdk/js/src/solana/tokenBridge";
import { postVaa } from "../../../sdk/js/src/solana/sendAndConfirmPostVaa";
import {
  BpfLoaderUpgradeable,
  getCompleteTransferNativeWithPayloadCpiAccounts,
  getCompleteTransferWrappedWithPayloadCpiAccounts,
  getTransferNativeWithPayloadCpiAccounts,
  getTransferWrappedWithPayloadCpiAccounts,
  NodeWallet,
  signSendAndConfirmTransaction,
} from "../../../sdk/js/src/solana";
import {
  deriveWormholeEmitterKey,
  getPostedMessage,
  getPostedVaa,
  getProgramSequenceTracker,
} from "../../../sdk/js/src/solana/wormhole";
import {
  parseGovernanceVaa,
  parseAttestMetaVaa,
  parseAttestMetaPayload,
  parseTokenBridgeRegisterChainVaa,
  parseTokenTransferPayload,
  parseVaa,
} from "../../../sdk/js/src/vaa";

import {
  CORE_BRIDGE_ADDRESS,
  TOKEN_BRIDGE_ADDRESS,
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  GOVERNANCE_EMITTER_ADDRESS,
  GUARDIAN_KEYS,
  GUARDIAN_SET_INDEX,
  LOCALHOST,
  WETH_ADDRESS,
  DEADBEEF_ADDRESS,
  DEADBEEF_METADATA_ADDRESS,
  DEADBEEF_MINT_ADDRESS,
} from "./helpers/consts";
import { ethAddressToBuffer, now } from "./helpers/utils";
import {
  getForeignAssetSolana,
  getIsWrappedAssetSolana,
  getOriginalAssetSolana,
} from "../../../sdk/js/src/token_bridge";
import { ChainId } from "../../../sdk/js/src";
import {
  transferNativeSol,
  redeemAndUnwrapOnSolana,
} from "../../../sdk/js/src/token_bridge";

describe("Token Bridge", () => {
  const connection = new web3.Connection(LOCALHOST, "processed");

  const wallet = new NodeWallet(web3.Keypair.generate());

  // for signing wormhole messages
  const guardians = new MockGuardians(GUARDIAN_SET_INDEX + 1, GUARDIAN_KEYS);

  const localVariables: any = {};

  before("Airdrop SOL", async () => {
    await connection
      .requestAirdrop(wallet.key(), 1000 * web3.LAMPORTS_PER_SOL)
      .then(async (signature) => connection.confirmTransaction(signature));
  });

  before("Create Mint", async () => {
    localVariables.mint = await createMint(
      connection,
      wallet.signer(),
      wallet.key(),
      null,
      9
    );

    localVariables.mintAta = await getOrCreateAssociatedTokenAccount(
      connection,
      wallet.signer(),
      localVariables.mint,
      wallet.key()
    ).then((account) => account.address);

    const mintToTx = await mintTo(
      connection,
      wallet.signer(),
      localVariables.mint,
      localVariables.mintAta,
      wallet.key(),
      1000 * web3.LAMPORTS_PER_SOL
    );
  });

  before("Create Mint with Metadata", async () => {
    // TODO
  });

  describe("Accounts", () => {
    // for generating governance wormhole messages
    const governance = new GovernanceEmitter(
      GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex")
    );

    // token bridge on Ethereum
    const ethereumTokenBridge = new MockEthereumTokenBridge(
      ETHEREUM_TOKEN_BRIDGE_ADDRESS
    );

    const payer = new web3.PublicKey(
      "6sbzC1eH4FTujJXWj51eQe25cYvr4xfXbJ1vAj7j2k5J"
    );

    it("Instruction 0:  Initialize", () => {
      const accounts = getInitializeAccounts(TOKEN_BRIDGE_ADDRESS, payer);

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.config.toString()).to.equal(
        "GnQ6fGttTRnJpAJuy2XEg5TLgEMtbyU4HDJnBWmojsTv"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
    });

    it("Instruction 1:  Attest Token", () => {
      const mint = NATIVE_MINT;
      const message = web3.Keypair.generate();
      const accounts = getAttestTokenAccounts(
        TOKEN_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        mint,
        message.publicKey
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.config.toString()).to.equal(
        "GnQ6fGttTRnJpAJuy2XEg5TLgEMtbyU4HDJnBWmojsTv"
      );
      expect(accounts.mint.equals(mint)).is.true;
      expect(accounts.wrappedMeta.toString()).to.equal(
        "8xGY7Bx9cWocPYpKRe3sjCYTdm3YchFJFHmJC5FenW6B"
      );
      expect(accounts.splMetadata.toString()).to.equal(
        "6dM4TqWyWJsbx7obrdLcviBkTafD5E8av61zfU6jq57X"
      );
      expect(accounts.wormholeBridge.toString()).to.equal(
        "DNN2VhmrGTGj6QVnPz4NVfsiSk64cRHzKBLP5kUaQrf8"
      );
      expect(accounts.wormholeEmitter.toString()).to.equal(
        "Ard2Zy4HckbJS2bL7y4361wbKSUH68JZqYBura5d4xtw"
      );
      expect(accounts.wormholeSequence.toString()).to.equal(
        "Gdeob8iLpTN4Fc8BEgRdFUWikdUsvrv9Rfc1rNQWy4b7"
      );
      expect(accounts.wormholeFeeCollector.toString()).to.equal(
        "Cxt3Uka7X8vyHYjU6szcuYVPPFyg1fAtoeVy7eyzPjGV"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).is.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });

    it("Instruction 2:  Complete Native", () => {
      const mint = NATIVE_MINT;
      const mintAta = getAssociatedTokenAddressSync(mint, payer);

      const amountEncoded = 42069n;

      const fee = 0n;
      const nonce = 420;
      const timestamp = 23456789;
      const message = ethereumTokenBridge.publishTransferTokens(
        mint.toBuffer().toString("hex"),
        1,
        amountEncoded,
        1,
        mintAta.toBuffer().toString("hex"),
        fee,
        nonce,
        timestamp
      );

      const signedVaa = guardians.addSignatures(
        message,
        [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18]
      );

      const accounts = getCompleteTransferNativeAccounts(
        TOKEN_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        signedVaa
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.config.toString()).to.equal(
        "GnQ6fGttTRnJpAJuy2XEg5TLgEMtbyU4HDJnBWmojsTv"
      );
      expect(accounts.vaa.toString()).to.equal(
        "GMBEenvtgYkQrHZNkx3kYbdYEUYN1GFRGBFN172bk3cN"
      );
      expect(accounts.claim.toString()).to.equal(
        "HzjTihvhEx7BbKnB2KHATNBwGFCEm2nnMG6c4Pwx6pPE"
      );
      expect(accounts.endpoint.toString()).to.equal(
        "4CgrjMnDneBjBBEyXtcikLTbAWpHAD1cwn8W1sSSCLru"
      );
      expect(accounts.to.equals(mintAta)).is.true;
      expect(accounts.toFees.equals(mintAta)).is.true;
      expect(accounts.custody.toString()).to.equal(
        "2Aczo4H847TNDsPradsVXQUZFquJ37ZoHhRmJ2MAqtiM"
      );
      expect(accounts.mint.equals(mint)).is.true;
      expect(accounts.custodySigner.toString()).to.equal(
        "Eb8xqkMEZYeTnDse4BgWiHVByeUj3JDpgbuz98pWdgPE"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).is.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });

    it("Instruction 3:  Complete Wrapped", () => {
      const tokenAddress = ethAddressToBuffer(WETH_ADDRESS);
      const tokenChain = ethereumTokenBridge.chain;
      const mint = deriveWrappedMintKey(
        TOKEN_BRIDGE_ADDRESS,
        tokenChain,
        tokenAddress
      );
      const mintAta = getAssociatedTokenAddressSync(mint, payer);

      const amount = 4206942069n;
      const recipientChain = 1;
      const fee = 0n;
      const nonce = 420;
      const timestamp = 34567890;
      const message = ethereumTokenBridge.publishTransferTokens(
        tokenAddress.toString("hex"),
        tokenChain,
        amount,
        recipientChain,
        mintAta.toBuffer().toString("hex"),
        fee,
        nonce,
        timestamp
      );

      const signedVaa = guardians.addSignatures(
        message,
        [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18]
      );

      const accounts = getCompleteTransferWrappedAccounts(
        TOKEN_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        signedVaa
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.config.toString()).to.equal(
        "GnQ6fGttTRnJpAJuy2XEg5TLgEMtbyU4HDJnBWmojsTv"
      );
      expect(accounts.vaa.toString()).to.equal(
        "7NQnr9aG4xQCp9AjnF37L3CrBbM6GJ2gN98JeFjn7nnf"
      );
      expect(accounts.claim.toString()).to.equal(
        "7Ae57QxvZMwCrknoDWpeaMTLbMP3LBeCJee6KaLEwxP6"
      );
      expect(accounts.endpoint.toString()).to.equal(
        "4CgrjMnDneBjBBEyXtcikLTbAWpHAD1cwn8W1sSSCLru"
      );
      expect(accounts.to.equals(mintAta)).is.true;
      expect(accounts.toFees.equals(mintAta)).is.true;
      expect(accounts.mint.equals(mint)).is.true;
      expect(accounts.wrappedMeta.toString()).to.equal(
        "AWUK8RTEBvUNAWLz1VfagK3rnvJ9oLDZPBJEBCzpjqj7"
      );
      expect(accounts.mintAuthority.toString()).to.equal(
        "J2mhpFfGCwHtUjmeQGhJSa2yk5h3egRoSd1AUhaKx2WG"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).is.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });

    it("Instruction 4:  Transfer Wrapped", () => {
      const tokenAddress = ethAddressToBuffer(WETH_ADDRESS);
      const tokenChain = ethereumTokenBridge.chain;
      const mint = deriveWrappedMintKey(
        TOKEN_BRIDGE_ADDRESS,
        tokenChain,
        tokenAddress
      );
      const mintAta = getAssociatedTokenAddressSync(mint, payer);

      const message = web3.Keypair.generate();
      const accounts = getTransferWrappedAccounts(
        TOKEN_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        message.publicKey,
        mintAta,
        payer,
        tokenChain,
        tokenAddress
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.config.toString()).to.equal(
        "GnQ6fGttTRnJpAJuy2XEg5TLgEMtbyU4HDJnBWmojsTv"
      );
      expect(accounts.from.equals(mintAta)).is.true;
      expect(accounts.fromOwner.equals(payer)).is.true;
      expect(accounts.mint.equals(mint)).is.true;
      expect(accounts.wrappedMeta.toString()).to.equal(
        "AWUK8RTEBvUNAWLz1VfagK3rnvJ9oLDZPBJEBCzpjqj7"
      );
      expect(accounts.authoritySigner.toString()).to.equal(
        "FDYbeBnX3rZnWM1jE6vTSxjxYdeGryxZtirXmzW71FTH"
      );
      expect(accounts.wormholeBridge.toString()).to.equal(
        "DNN2VhmrGTGj6QVnPz4NVfsiSk64cRHzKBLP5kUaQrf8"
      );
      expect(accounts.wormholeEmitter.toString()).to.equal(
        "Ard2Zy4HckbJS2bL7y4361wbKSUH68JZqYBura5d4xtw"
      );
      expect(accounts.wormholeSequence.toString()).to.equal(
        "Gdeob8iLpTN4Fc8BEgRdFUWikdUsvrv9Rfc1rNQWy4b7"
      );
      expect(accounts.wormholeFeeCollector.toString()).to.equal(
        "Cxt3Uka7X8vyHYjU6szcuYVPPFyg1fAtoeVy7eyzPjGV"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).is.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).is.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });

    it("Instruction 5:  Transfer Native", () => {
      const mint = NATIVE_MINT;
      const mintAta = getAssociatedTokenAddressSync(mint, payer);
      const message = web3.Keypair.generate();
      const accounts = getTransferNativeAccounts(
        TOKEN_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        message.publicKey,
        mintAta,
        mint
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.config.toString()).to.equal(
        "GnQ6fGttTRnJpAJuy2XEg5TLgEMtbyU4HDJnBWmojsTv"
      );
      expect(accounts.from.equals(mintAta)).is.true;
      expect(accounts.mint.equals(mint)).is.true;
      expect(accounts.custody.toString()).to.equal(
        "2Aczo4H847TNDsPradsVXQUZFquJ37ZoHhRmJ2MAqtiM"
      );
      expect(accounts.authoritySigner.toString()).to.equal(
        "FDYbeBnX3rZnWM1jE6vTSxjxYdeGryxZtirXmzW71FTH"
      );
      expect(accounts.custodySigner.toString()).to.equal(
        "Eb8xqkMEZYeTnDse4BgWiHVByeUj3JDpgbuz98pWdgPE"
      );
      expect(accounts.wormholeBridge.toString()).to.equal(
        "DNN2VhmrGTGj6QVnPz4NVfsiSk64cRHzKBLP5kUaQrf8"
      );
      expect(accounts.wormholeEmitter.toString()).to.equal(
        "Ard2Zy4HckbJS2bL7y4361wbKSUH68JZqYBura5d4xtw"
      );
      expect(accounts.wormholeSequence.toString()).to.equal(
        "Gdeob8iLpTN4Fc8BEgRdFUWikdUsvrv9Rfc1rNQWy4b7"
      );
      expect(accounts.wormholeFeeCollector.toString()).to.equal(
        "Cxt3Uka7X8vyHYjU6szcuYVPPFyg1fAtoeVy7eyzPjGV"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).is.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).is.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });

    it("Instruction 6:  Register Chain", () => {
      const timestamp = 45678901;
      const message = governance.publishTokenBridgeRegisterChain(
        timestamp,
        2,
        ETHEREUM_TOKEN_BRIDGE_ADDRESS
      );
      const signedVaa = guardians.addSignatures(
        message,
        [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18]
      );

      const accounts = getRegisterChainAccounts(
        TOKEN_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        signedVaa
      );

      // verify accounts
      const parsed = parseGovernanceVaa(signedVaa);
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.config.toString()).to.equal(
        "GnQ6fGttTRnJpAJuy2XEg5TLgEMtbyU4HDJnBWmojsTv"
      );
      expect(accounts.endpoint.toString()).to.equal(
        "4CgrjMnDneBjBBEyXtcikLTbAWpHAD1cwn8W1sSSCLru"
      );
      expect(accounts.vaa.toString()).to.equal(
        "92pVby4LJPSyQxSSHLYv3EdpqWjH5bBoLGBJAeQkunf8"
      );
      expect(accounts.claim.toString()).to.equal(
        "J5LWxMcXo1xmdZq57VD4wrUgvw5taizQ9QEPHooHTwJv"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });

    it("Instruction 7:  Create Wrapped", () => {
      const tokenAddress = WETH_ADDRESS;
      const decimals = 18;
      const symbol = "WETH";
      const name = "Wrapped ETH";
      const nonce = 420;
      const message = ethereumTokenBridge.publishAttestMeta(
        tokenAddress,
        decimals,
        symbol,
        name,
        nonce
      );
      const signedVaa = guardians.addSignatures(
        message,
        [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18]
      );
      const accounts = getCreateWrappedAccounts(
        TOKEN_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        signedVaa
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.config.toString()).to.equal(
        "GnQ6fGttTRnJpAJuy2XEg5TLgEMtbyU4HDJnBWmojsTv"
      );
      expect(accounts.endpoint.toString()).to.equal(
        "4CgrjMnDneBjBBEyXtcikLTbAWpHAD1cwn8W1sSSCLru"
      );
      expect(accounts.vaa.toString()).to.equal(
        "4NDyWDtRvfEdi48a9JgYG28m919hrcdW8gNgRg3jwU99"
      );
      expect(accounts.claim.toString()).to.equal(
        "4dyk94hhqektDX9wUBCL1ZkyQC1Xn3QaTSAdJeZzbTcJ"
      );
      expect(accounts.mint.toString()).to.equal(
        "3tUXFuBNWzZZ8p2xNx5UoWCH664M2KHdDAWrdZAD1VQ3"
      );
      expect(accounts.wrappedMeta.toString()).to.equal(
        "AWUK8RTEBvUNAWLz1VfagK3rnvJ9oLDZPBJEBCzpjqj7"
      );
      expect(accounts.splMetadata.toString()).to.equal(
        "46nJp6UehY8XpgNsSZFTamdXcwiSEQpRGvbBCt2KvVUf"
      );
      expect(accounts.mintAuthority.toString()).to.equal(
        "J2mhpFfGCwHtUjmeQGhJSa2yk5h3egRoSd1AUhaKx2WG"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).is.true;
      expect(accounts.splMetadataProgram.equals(TOKEN_METADATA_PROGRAM_ID)).is
        .true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });

    it("Instruction 8:  Upgrade Contract", () => {
      const timestamp = 56789012;
      const chain = 1;
      const implementation = new web3.PublicKey(
        "2B5wMnErS8oKWV1wPTNQQhM1WLyxee2obtBMDtsYeHgA"
      );
      const message = governance.publishTokenBridgeUpgradeContract(
        timestamp,
        chain,
        implementation.toString()
      );
      const signedVaa = guardians.addSignatures(message, [0]);

      const accounts = getUpgradeContractAccounts(
        TOKEN_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        signedVaa
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.vaa.toString()).to.equal(
        "HM2U2HEfbjrkvYLvry8Sqfmd5cCVxq6RjdLUUrNW2ELR"
      );
      expect(accounts.claim.toString()).to.equal(
        "9jDWWzAosaD6EWH9SMFT3ZwJnDZTeGcCdMU8H5Ba7dpx"
      );
      expect(accounts.upgradeAuthority.toString()).to.equal(
        "B2LFmpNCkfBFpoorLy4BghGbZyi5sdRsbjxSSASpjUoA"
      );
      expect(accounts.spill.equals(payer)).is.true;
      expect(accounts.implementation.equals(implementation)).is.true;
      expect(accounts.programData.toString()).to.equal(
        "3zHkdon6x9fUVqjxu6fCgdp3qMxLFZz59pj1H2NtnbGe"
      );
      expect(accounts.tokenBridgeProgram.equals(TOKEN_BRIDGE_ADDRESS)).to.be
        .true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).is.true;
      expect(
        accounts.bpfLoaderUpgradeable.equals(BpfLoaderUpgradeable.programId)
      ).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
    });

    it("Instruction 11: Transfer Wrapped With Payload", () => {
      const tokenAddress = ethAddressToBuffer(WETH_ADDRESS);
      const tokenChain = ethereumTokenBridge.chain;
      const mint = deriveWrappedMintKey(
        TOKEN_BRIDGE_ADDRESS,
        tokenChain,
        tokenAddress
      );
      const mintAta = getAssociatedTokenAddressSync(mint, payer);

      const message = web3.Keypair.generate();
      const accounts = getTransferWrappedWithPayloadAccounts(
        TOKEN_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        message.publicKey,
        mintAta,
        payer,
        tokenChain,
        tokenAddress
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.config.toString()).to.equal(
        "GnQ6fGttTRnJpAJuy2XEg5TLgEMtbyU4HDJnBWmojsTv"
      );
      expect(accounts.from.equals(mintAta)).is.true;
      expect(accounts.fromOwner.equals(payer)).is.true;
      expect(accounts.mint.equals(mint)).is.true;
      expect(accounts.wrappedMeta.toString()).to.equal(
        "AWUK8RTEBvUNAWLz1VfagK3rnvJ9oLDZPBJEBCzpjqj7"
      );
      expect(accounts.authoritySigner.toString()).to.equal(
        "FDYbeBnX3rZnWM1jE6vTSxjxYdeGryxZtirXmzW71FTH"
      );
      expect(accounts.wormholeBridge.toString()).to.equal(
        "DNN2VhmrGTGj6QVnPz4NVfsiSk64cRHzKBLP5kUaQrf8"
      );
      expect(accounts.wormholeEmitter.toString()).to.equal(
        "Ard2Zy4HckbJS2bL7y4361wbKSUH68JZqYBura5d4xtw"
      );
      expect(accounts.wormholeSequence.toString()).to.equal(
        "Gdeob8iLpTN4Fc8BEgRdFUWikdUsvrv9Rfc1rNQWy4b7"
      );
      expect(accounts.wormholeFeeCollector.toString()).to.equal(
        "Cxt3Uka7X8vyHYjU6szcuYVPPFyg1fAtoeVy7eyzPjGV"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).is.true;
      expect(accounts.sender.equals(payer)).is.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).is.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });

    it("Instruction 12: Transfer Native With Payload", () => {
      const mint = NATIVE_MINT;
      const mintAta = getAssociatedTokenAddressSync(mint, payer);
      const message = web3.Keypair.generate();
      const accounts = getTransferNativeWithPayloadAccounts(
        TOKEN_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        message.publicKey,
        mintAta,
        mint
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.config.toString()).to.equal(
        "GnQ6fGttTRnJpAJuy2XEg5TLgEMtbyU4HDJnBWmojsTv"
      );
      expect(accounts.from.equals(mintAta)).is.true;
      expect(accounts.custody.toString()).to.equal(
        "2Aczo4H847TNDsPradsVXQUZFquJ37ZoHhRmJ2MAqtiM"
      );
      expect(accounts.authoritySigner.toString()).to.equal(
        "FDYbeBnX3rZnWM1jE6vTSxjxYdeGryxZtirXmzW71FTH"
      );
      expect(accounts.custodySigner.toString()).to.equal(
        "Eb8xqkMEZYeTnDse4BgWiHVByeUj3JDpgbuz98pWdgPE"
      );
      expect(accounts.wormholeBridge.toString()).to.equal(
        "DNN2VhmrGTGj6QVnPz4NVfsiSk64cRHzKBLP5kUaQrf8"
      );
      expect(accounts.wormholeEmitter.toString()).to.equal(
        "Ard2Zy4HckbJS2bL7y4361wbKSUH68JZqYBura5d4xtw"
      );
      expect(accounts.wormholeSequence.toString()).to.equal(
        "Gdeob8iLpTN4Fc8BEgRdFUWikdUsvrv9Rfc1rNQWy4b7"
      );
      expect(accounts.wormholeFeeCollector.toString()).to.equal(
        "Cxt3Uka7X8vyHYjU6szcuYVPPFyg1fAtoeVy7eyzPjGV"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).is.true;
      expect(accounts.sender.equals(payer)).is.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).is.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });
  });

  describe("CPI Accounts", () => {
    // token bridge on Ethereum
    const ethereumTokenBridge = new MockEthereumTokenBridge(
      ETHEREUM_TOKEN_BRIDGE_ADDRESS
    );

    const payer = new web3.PublicKey(
      "6sbzC1eH4FTujJXWj51eQe25cYvr4xfXbJ1vAj7j2k5J"
    );

    // mock program integrating token bridge
    const cpiProgramId = new web3.PublicKey(
      "pFCBP4bhqdSsrWUVTgqhPsLrfEdChBK17vgFM7TxjxQ"
    );

    it("getCompleteTransferNativeWithPayloadCpiAccounts", () => {
      const mint = NATIVE_MINT;
      const redeemer = deriveRedeemerAccountKey(cpiProgramId);
      const mintAta = getAssociatedTokenAddressSync(mint, redeemer, true);

      const amountEncoded = 42069n;

      const nonce = 420;
      const timestamp = 23456789;
      const message = ethereumTokenBridge.publishTransferTokensWithPayload(
        mint.toBuffer().toString("hex"),
        1,
        amountEncoded,
        1,
        cpiProgramId.toBuffer().toString("hex"),
        Buffer.alloc(32, 0),
        Buffer.from("All your base are belong to us"),
        nonce,
        timestamp
      );
      expect(message[51]).to.equal(3);

      const signedVaa = guardians.addSignatures(
        message,
        [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18]
      );

      const accounts = getCompleteTransferNativeWithPayloadCpiAccounts(
        TOKEN_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        signedVaa,
        mintAta
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.tokenBridgeConfig.toString()).to.equal(
        "GnQ6fGttTRnJpAJuy2XEg5TLgEMtbyU4HDJnBWmojsTv"
      );
      expect(accounts.vaa.toString()).to.equal(
        "GtiCPc4mxBVsrPQVgYnuVUzhuvh24A54KaDZhcP4mhDa"
      );
      expect(accounts.tokenBridgeClaim.toString()).to.equal(
        "HzjTihvhEx7BbKnB2KHATNBwGFCEm2nnMG6c4Pwx6pPE"
      );
      expect(accounts.tokenBridgeForeignEndpoint.toString()).to.equal(
        "4CgrjMnDneBjBBEyXtcikLTbAWpHAD1cwn8W1sSSCLru"
      );
      expect(accounts.toTokenAccount.equals(mintAta)).is.true;
      expect(accounts.tokenBridgeRedeemer.toString()).to.equal(
        "A2SNTmahH9ryK2PupNMfKibPPaMtcfYBSX4WjZchhatX"
      );
      expect(accounts.toFeesTokenAccount.equals(mintAta)).is.true;
      expect(accounts.tokenBridgeCustody.toString()).to.equal(
        "2Aczo4H847TNDsPradsVXQUZFquJ37ZoHhRmJ2MAqtiM"
      );
      expect(accounts.mint.equals(mint)).is.true;
      expect(accounts.tokenBridgeCustodySigner.toString()).to.equal(
        "Eb8xqkMEZYeTnDse4BgWiHVByeUj3JDpgbuz98pWdgPE"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).is.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });

    it("getCompleteTransferWrappedWithPayloadCpiAccounts", () => {
      const tokenAddress = ethAddressToBuffer(WETH_ADDRESS);
      const tokenChain = ethereumTokenBridge.chain;
      const mint = deriveWrappedMintKey(
        TOKEN_BRIDGE_ADDRESS,
        tokenChain,
        tokenAddress
      );
      const redeemer = deriveRedeemerAccountKey(cpiProgramId);
      const mintAta = getAssociatedTokenAddressSync(mint, redeemer, true);

      const amount = 4206942069n;
      const recipientChain = 1;
      const nonce = 420;
      const timestamp = 34567890;
      const message = ethereumTokenBridge.publishTransferTokensWithPayload(
        tokenAddress.toString("hex"),
        tokenChain,
        amount,
        recipientChain,
        cpiProgramId.toBuffer().toString("hex"),
        Buffer.alloc(32, 0),
        Buffer.from("All your base are belong to us"),
        nonce,
        timestamp
      );
      expect(message[51]).to.equal(3);

      const signedVaa = guardians.addSignatures(
        message,
        [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18]
      );

      const accounts = getCompleteTransferWrappedWithPayloadCpiAccounts(
        TOKEN_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        signedVaa,
        mintAta
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.tokenBridgeConfig.toString()).to.equal(
        "GnQ6fGttTRnJpAJuy2XEg5TLgEMtbyU4HDJnBWmojsTv"
      );
      expect(accounts.vaa.toString()).to.equal(
        "9nFMaAfuXmE4FdJe8koZ4ScvYcJ5znoJPDgT29aVZM1x"
      );
      expect(accounts.tokenBridgeClaim.toString()).to.equal(
        "7Ae57QxvZMwCrknoDWpeaMTLbMP3LBeCJee6KaLEwxP6"
      );
      expect(accounts.tokenBridgeForeignEndpoint.toString()).to.equal(
        "4CgrjMnDneBjBBEyXtcikLTbAWpHAD1cwn8W1sSSCLru"
      );
      expect(accounts.toTokenAccount.equals(mintAta)).is.true;
      expect(accounts.tokenBridgeRedeemer.toString()).to.equal(
        "A2SNTmahH9ryK2PupNMfKibPPaMtcfYBSX4WjZchhatX"
      );
      expect(accounts.toFeesTokenAccount.equals(mintAta)).is.true;
      expect(accounts.tokenBridgeWrappedMint.equals(mint)).is.true;
      expect(accounts.tokenBridgeWrappedMeta.toString()).to.equal(
        "AWUK8RTEBvUNAWLz1VfagK3rnvJ9oLDZPBJEBCzpjqj7"
      );
      expect(accounts.tokenBridgeMintAuthority.toString()).to.equal(
        "J2mhpFfGCwHtUjmeQGhJSa2yk5h3egRoSd1AUhaKx2WG"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).is.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });

    it("getTransferWrappedWithPayloadCpiAccounts", () => {
      const tokenAddress = ethAddressToBuffer(WETH_ADDRESS);
      const tokenChain = ethereumTokenBridge.chain;
      const mint = deriveWrappedMintKey(
        TOKEN_BRIDGE_ADDRESS,
        tokenChain,
        tokenAddress
      );
      const mintAta = getAssociatedTokenAddressSync(mint, cpiProgramId, true);

      const message = web3.Keypair.generate();
      const accounts = getTransferWrappedWithPayloadCpiAccounts(
        cpiProgramId,
        TOKEN_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        message.publicKey,
        mintAta,
        tokenChain,
        tokenAddress
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.tokenBridgeConfig.toString()).to.equal(
        "GnQ6fGttTRnJpAJuy2XEg5TLgEMtbyU4HDJnBWmojsTv"
      );
      expect(accounts.fromTokenAccount.equals(mintAta)).is.true;
      expect(accounts.fromTokenAccountOwner.equals(cpiProgramId)).is.true;
      expect(accounts.tokenBridgeWrappedMint.equals(mint)).is.true;
      expect(accounts.tokenBridgeWrappedMeta.toString()).to.equal(
        "AWUK8RTEBvUNAWLz1VfagK3rnvJ9oLDZPBJEBCzpjqj7"
      );
      expect(accounts.tokenBridgeAuthoritySigner.toString()).to.equal(
        "FDYbeBnX3rZnWM1jE6vTSxjxYdeGryxZtirXmzW71FTH"
      );
      expect(accounts.wormholeBridge.toString()).to.equal(
        "DNN2VhmrGTGj6QVnPz4NVfsiSk64cRHzKBLP5kUaQrf8"
      );
      expect(accounts.tokenBridgeEmitter.toString()).to.equal(
        "Ard2Zy4HckbJS2bL7y4361wbKSUH68JZqYBura5d4xtw"
      );
      expect(accounts.tokenBridgeSequence.toString()).to.equal(
        "Gdeob8iLpTN4Fc8BEgRdFUWikdUsvrv9Rfc1rNQWy4b7"
      );
      expect(accounts.wormholeFeeCollector.toString()).to.equal(
        "Cxt3Uka7X8vyHYjU6szcuYVPPFyg1fAtoeVy7eyzPjGV"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).is.true;
      expect(accounts.tokenBridgeSender.toString()).to.equal(
        "7r3GbMGbRRp3cbPRPv9v5GBktxGpDmK5LBnvjDVxsEDN"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).is.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });

    it("getTransferNativeWithPayloadCpiAccounts", () => {
      const mint = NATIVE_MINT;
      const mintAta = getAssociatedTokenAddressSync(mint, cpiProgramId, true);
      const message = web3.Keypair.generate();
      const accounts = getTransferNativeWithPayloadCpiAccounts(
        cpiProgramId,
        TOKEN_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        message.publicKey,
        mintAta,
        mint
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.tokenBridgeConfig.toString()).to.equal(
        "GnQ6fGttTRnJpAJuy2XEg5TLgEMtbyU4HDJnBWmojsTv"
      );
      expect(accounts.fromTokenAccount.equals(mintAta)).is.true;
      expect(accounts.tokenBridgeCustody.toString()).to.equal(
        "2Aczo4H847TNDsPradsVXQUZFquJ37ZoHhRmJ2MAqtiM"
      );
      expect(accounts.tokenBridgeAuthoritySigner.toString()).to.equal(
        "FDYbeBnX3rZnWM1jE6vTSxjxYdeGryxZtirXmzW71FTH"
      );
      expect(accounts.tokenBridgeCustodySigner.toString()).to.equal(
        "Eb8xqkMEZYeTnDse4BgWiHVByeUj3JDpgbuz98pWdgPE"
      );
      expect(accounts.wormholeBridge.toString()).to.equal(
        "DNN2VhmrGTGj6QVnPz4NVfsiSk64cRHzKBLP5kUaQrf8"
      );
      expect(accounts.tokenBridgeEmitter.toString()).to.equal(
        "Ard2Zy4HckbJS2bL7y4361wbKSUH68JZqYBura5d4xtw"
      );
      expect(accounts.tokenBridgeSequence.toString()).to.equal(
        "Gdeob8iLpTN4Fc8BEgRdFUWikdUsvrv9Rfc1rNQWy4b7"
      );
      expect(accounts.wormholeFeeCollector.toString()).to.equal(
        "Cxt3Uka7X8vyHYjU6szcuYVPPFyg1fAtoeVy7eyzPjGV"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).is.true;
      expect(accounts.tokenBridgeSender.toString()).to.equal(
        "7r3GbMGbRRp3cbPRPv9v5GBktxGpDmK5LBnvjDVxsEDN"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).is.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });
  });

  describe("Token Bridge Program Interaction", () => {
    // for generating governance wormhole messages
    const governance = new GovernanceEmitter(
      GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
      20
    );

    // token bridge on Ethereum
    const ethereumTokenBridge = new MockEthereumTokenBridge(
      ETHEREUM_TOKEN_BRIDGE_ADDRESS
    );

    describe("Setup Token Bridge", () => {
      it("Register Ethereum Token Bridge", async () => {
        const timestamp = now();
        const message = governance.publishTokenBridgeRegisterChain(
          timestamp,
          2,
          ETHEREUM_TOKEN_BRIDGE_ADDRESS
        );
        const signedVaa = guardians.addSignatures(
          message,
          [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18]
        );

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

        const registerChainTx = await web3.sendAndConfirmTransaction(
          connection,
          new web3.Transaction().add(
            createRegisterChainInstruction(
              TOKEN_BRIDGE_ADDRESS,
              CORE_BRIDGE_ADDRESS,
              wallet.key(),
              signedVaa
            )
          ),
          [wallet.signer()]
        );
        // console.log(`registerChainTx: ${registerChainTx}`);

        // verify data
        const parsed = parseTokenBridgeRegisterChainVaa(signedVaa);
        const endpoint = deriveEndpointKey(
          TOKEN_BRIDGE_ADDRESS,
          parsed.foreignChain,
          parsed.foreignAddress
        );
        const endpointRegistration = await getEndpointRegistration(
          connection,
          endpoint
        );
        expect(endpointRegistration.chain).to.equal(2);

        const expectedEmitter = ethAddressToBuffer(
          ETHEREUM_TOKEN_BRIDGE_ADDRESS
        );
        expect(
          Buffer.compare(endpointRegistration.contract, expectedEmitter)
        ).to.equal(0);
      });
    });

    describe("Native Token Handling", () => {
      it("Attest Mint Without Metadata", async () => {
        const mint = localVariables.mint;
        const message = web3.Keypair.generate();
        const nonce = 69;

        const attestTokenTx = await web3.sendAndConfirmTransaction(
          connection,
          new web3.Transaction().add(
            createAttestTokenInstruction(
              TOKEN_BRIDGE_ADDRESS,
              CORE_BRIDGE_ADDRESS,
              wallet.key(),
              mint,
              message.publicKey,
              nonce
            )
          ),
          [wallet.signer(), message]
        );
        // console.log(`attestTokenTx: ${attestTokenTx}`);

        // verify data
        const messageData = await getPostedMessage(
          connection,
          message.publicKey
        ).then((posted) => posted.message);

        expect(messageData.consistencyLevel).to.equal(32);
        expect(
          Buffer.compare(
            messageData.emitterAddress,
            deriveWormholeEmitterKey(TOKEN_BRIDGE_ADDRESS).toBuffer()
          )
        ).to.equal(0);
        expect(messageData.emitterChain).to.equal(1);
        expect(messageData.nonce).to.equal(nonce);
        expect(messageData.sequence).to.equal(0n);
        expect(messageData.vaaTime).to.equal(0);
        expect(messageData.vaaSignatureAccount.equals(web3.PublicKey.default))
          .is.true;
        expect(messageData.vaaVersion).to.equal(0);

        const assetMeta = parseAttestMetaPayload(messageData.payload);
        expect(assetMeta.payloadType).to.equal(2);
        expect(
          Buffer.compare(assetMeta.tokenAddress, mint.toBuffer())
        ).to.equal(0);
        expect(assetMeta.tokenChain).to.equal(1);
        expect(assetMeta.decimals).to.equal(9);
        expect(assetMeta.symbol).to.equal("");
        expect(assetMeta.name).to.equal("");

        const sequenceTracker = await getProgramSequenceTracker(
          connection,
          TOKEN_BRIDGE_ADDRESS,
          CORE_BRIDGE_ADDRESS
        );
        expect(sequenceTracker.value()).to.equal(messageData.sequence + 1n);
      });

      //   it("Attest Mint With Metadata", async () => {
      //     // TODO
      //   });

      it("Send Token", async () => {
        const mint = localVariables.mint;
        const mintAta = localVariables.mintAta;
        const custodyAccount = deriveCustodyKey(TOKEN_BRIDGE_ADDRESS, mint);

        const walletBalanceBefore = await getAccount(connection, mintAta).then(
          (account) => account.amount
        );
        const custodyBalanceBefore = 0n;

        const nonce = 69;
        const amount = BigInt(420 * web3.LAMPORTS_PER_SOL);
        const fee = 0n;
        const targetAddress = Buffer.alloc(32, "deadbeef", "hex");
        const targetChain = 2;

        const approveIx = createApproveAuthoritySignerInstruction(
          TOKEN_BRIDGE_ADDRESS,
          mintAta,
          wallet.key(),
          amount
        );

        const message = web3.Keypair.generate();
        const transferNativeIx = createTransferNativeInstruction(
          TOKEN_BRIDGE_ADDRESS,
          CORE_BRIDGE_ADDRESS,
          wallet.key(),
          message.publicKey,
          mintAta,
          mint,
          nonce,
          amount,
          fee,
          targetAddress,
          targetChain
        );

        const approveAndTransferTx = await web3.sendAndConfirmTransaction(
          connection,
          new web3.Transaction().add(approveIx, transferNativeIx),
          [wallet.signer(), message]
        );
        // console.log(`approveAndTransferTx: ${approveAndTransferTx}`);

        const walletBalanceAfter = await getAccount(connection, mintAta).then(
          (account) => account.amount
        );
        const custodyBalanceAfter = await getAccount(
          connection,
          custodyAccount
        ).then((account) => account.amount);

        // check balance changes
        expect(walletBalanceBefore - walletBalanceAfter).to.equal(amount);
        expect(custodyBalanceAfter - custodyBalanceBefore).to.equal(amount);

        // verify data
        const messageData = await getPostedMessage(
          connection,
          message.publicKey
        ).then((posted) => posted.message);

        expect(messageData.consistencyLevel).to.equal(32);
        expect(
          Buffer.compare(
            messageData.emitterAddress,
            deriveWormholeEmitterKey(TOKEN_BRIDGE_ADDRESS).toBuffer()
          )
        ).to.equal(0);
        expect(messageData.emitterChain).to.equal(1);
        expect(messageData.nonce).to.equal(nonce);
        expect(messageData.sequence).to.equal(1n);
        expect(messageData.vaaTime).to.equal(0);
        expect(messageData.vaaSignatureAccount.equals(web3.PublicKey.default))
          .is.true;
        expect(messageData.vaaVersion).to.equal(0);

        const tokenTransfer = parseTokenTransferPayload(messageData.payload);
        expect(tokenTransfer.payloadType).to.equal(1);
        const mintInfo = await getMint(connection, mint);
        expect(mintInfo.decimals).greaterThan(8);
        // decimals will be 8 on Ethereum token bridge
        const amountEncoded =
          amount / BigInt(Math.pow(10, mintInfo.decimals - 8));
        expect(tokenTransfer.amount).to.equal(amountEncoded);
        expect(tokenTransfer.fee).is.not.null;
        expect(tokenTransfer.fee).to.equal(fee);
        expect(tokenTransfer.fromAddress).is.null;
        expect(Buffer.compare(tokenTransfer.to, targetAddress)).to.equal(0);
        expect(tokenTransfer.toChain).to.equal(targetChain);
        expect(
          Buffer.compare(tokenTransfer.tokenAddress, mint.toBuffer())
        ).to.equal(0);
        expect(tokenTransfer.tokenChain).to.equal(1);

        const sequenceTracker = await getProgramSequenceTracker(
          connection,
          TOKEN_BRIDGE_ADDRESS,
          CORE_BRIDGE_ADDRESS
        );
        expect(sequenceTracker.value()).to.equal(messageData.sequence + 1n);
      });

      it("Receive Token", async () => {
        const mint = localVariables.mint;
        const mintAta = localVariables.mintAta;
        const custodyAccount = deriveCustodyKey(TOKEN_BRIDGE_ADDRESS, mint);

        const walletBalanceBefore = await getAccount(connection, mintAta).then(
          (account) => account.amount
        );
        const custodyBalanceBefore = await getAccount(
          connection,
          custodyAccount
        ).then((account) => account.amount);

        const amount = 420n * BigInt(web3.LAMPORTS_PER_SOL);

        const mintInfo = await getMint(connection, mint);
        expect(mintInfo.decimals).greaterThan(8);
        // decimals will be 8 on Ethereum token bridge
        const amountEncoded =
          amount / BigInt(Math.pow(10, mintInfo.decimals - 8));

        const tokenChain = 1;
        const recipientChain = 1;
        const fee = 0n;
        const nonce = 420;
        const message = ethereumTokenBridge.publishTransferTokens(
          mint.toBuffer().toString("hex"),
          tokenChain,
          amountEncoded,
          recipientChain,
          mintAta.toBuffer().toString("hex"),
          fee,
          nonce
        );

        const signedVaa = guardians.addSignatures(
          message,
          [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18]
        );

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

        const completeNativeTransferIx =
          createCompleteTransferNativeInstruction(
            TOKEN_BRIDGE_ADDRESS,
            CORE_BRIDGE_ADDRESS,
            wallet.key(),
            signedVaa
          );

        const completeNativeTransferTx = await web3.sendAndConfirmTransaction(
          connection,
          new web3.Transaction().add(completeNativeTransferIx),
          [wallet.signer()]
        );
        // console.log(`completeNativeTransferTx: ${completeNativeTransferTx}`);

        const walletBalanceAfter = await getAccount(connection, mintAta).then(
          (account) => account.amount
        );
        const custodyBalanceAfter = await getAccount(
          connection,
          custodyAccount
        ).then((account) => account.amount);

        // check balance changes
        expect(walletBalanceAfter - walletBalanceBefore).to.equal(amount);
        expect(custodyBalanceBefore - custodyBalanceAfter).to.equal(amount);

        // verify data
        const messageData = await getPostedVaa(
          connection,
          CORE_BRIDGE_ADDRESS,
          parseVaa(signedVaa).hash
        ).then((posted) => posted.message);

        expect(messageData.consistencyLevel).to.equal(
          ethereumTokenBridge.consistencyLevel
        );
        expect(
          Buffer.compare(
            messageData.emitterAddress,
            ethAddressToBuffer(ETHEREUM_TOKEN_BRIDGE_ADDRESS)
          )
        ).to.equal(0);
        expect(messageData.emitterChain).to.equal(ethereumTokenBridge.chain);
        expect(messageData.nonce).to.equal(nonce);
        expect(messageData.sequence).to.equal(1n);
        expect(messageData.vaaTime).to.equal(0);
        expect(messageData.vaaVersion).to.equal(1);
        expect(
          Buffer.compare(parseVaa(signedVaa).payload, messageData.payload)
        ).to.equal(0);

        const tokenTransfer = parseTokenTransferPayload(messageData.payload);
        expect(tokenTransfer.payloadType).to.equal(1);
        expect(tokenTransfer.amount).to.equal(amountEncoded);
        expect(tokenTransfer.fee).is.not.null;
        expect(tokenTransfer.fee).to.equal(fee);
        expect(tokenTransfer.fromAddress).is.null;
        expect(Buffer.compare(tokenTransfer.to, mintAta.toBuffer())).to.equal(
          0
        );
        expect(tokenTransfer.toChain).to.equal(recipientChain);
        expect(
          Buffer.compare(tokenTransfer.tokenAddress, mint.toBuffer())
        ).to.equal(0);
        expect(tokenTransfer.tokenChain).to.equal(tokenChain);
      });

      it("Send Token With Payload", async () => {
        const mint = localVariables.mint;
        const mintAta = localVariables.mintAta;
        const custodyAccount = deriveCustodyKey(TOKEN_BRIDGE_ADDRESS, mint);

        const walletBalanceBefore = await getAccount(connection, mintAta).then(
          (account) => account.amount
        );
        const custodyBalanceBefore = await getAccount(
          connection,
          custodyAccount
        ).then((account) => account.amount);

        const nonce = 420;
        const amount = BigInt(69 * web3.LAMPORTS_PER_SOL);
        const targetAddress = Buffer.alloc(32, "deadbeef", "hex");
        const targetChain = 2;

        const approveIx = createApproveAuthoritySignerInstruction(
          TOKEN_BRIDGE_ADDRESS,
          mintAta,
          wallet.key(),
          amount
        );

        const message = web3.Keypair.generate();
        const transferPayload = Buffer.from("All your base are belong to us");
        const transferNativeIx = createTransferNativeWithPayloadInstruction(
          TOKEN_BRIDGE_ADDRESS,
          CORE_BRIDGE_ADDRESS,
          wallet.key(),
          message.publicKey,
          mintAta,
          mint,
          nonce,
          amount,
          targetAddress,
          targetChain,
          transferPayload
        );

        const approveAndTransferTx = await web3.sendAndConfirmTransaction(
          connection,
          new web3.Transaction().add(approveIx, transferNativeIx),
          [wallet.signer(), message]
        );
        // console.log(`approveAndTransferTx: ${approveAndTransferTx}`);

        const walletBalanceAfter = await getAccount(connection, mintAta).then(
          (account) => account.amount
        );
        const custodyBalanceAfter = await getAccount(
          connection,
          custodyAccount
        ).then((account) => account.amount);

        // check balance changes
        expect(walletBalanceBefore - walletBalanceAfter).to.equal(amount);
        expect(custodyBalanceAfter - custodyBalanceBefore).to.equal(amount);

        // verify data
        const messageData = await getPostedMessage(
          connection,
          message.publicKey
        ).then((posted) => posted.message);

        expect(messageData.consistencyLevel).to.equal(32);
        expect(
          Buffer.compare(
            messageData.emitterAddress,
            deriveWormholeEmitterKey(TOKEN_BRIDGE_ADDRESS).toBuffer()
          )
        ).to.equal(0);
        expect(messageData.emitterChain).to.equal(1);
        expect(messageData.nonce).to.equal(nonce);
        expect(messageData.sequence).to.equal(2n);
        expect(messageData.vaaTime).to.equal(0);
        expect(messageData.vaaSignatureAccount.equals(web3.PublicKey.default))
          .is.true;
        expect(messageData.vaaVersion).to.equal(0);

        const tokenTransfer = parseTokenTransferPayload(messageData.payload);
        expect(tokenTransfer.payloadType).to.equal(3);
        const mintInfo = await getMint(connection, mint);
        expect(mintInfo.decimals).greaterThan(8);
        // decimals will be 8 on Ethereum token bridge
        const amountEncoded =
          amount / BigInt(Math.pow(10, mintInfo.decimals - 8));
        expect(tokenTransfer.amount).to.equal(amountEncoded);
        expect(tokenTransfer.fee).is.null;
        expect(tokenTransfer.fromAddress).is.not.null;
        expect(
          new web3.PublicKey(tokenTransfer.fromAddress!).equals(wallet.key())
        ).is.true;
        expect(Buffer.compare(tokenTransfer.to, targetAddress)).to.equal(0);
        expect(tokenTransfer.toChain).to.equal(targetChain);
        expect(
          Buffer.compare(tokenTransfer.tokenAddress, mint.toBuffer())
        ).to.equal(0);
        expect(tokenTransfer.tokenChain).to.equal(1);
        expect(
          Buffer.compare(tokenTransfer.tokenTransferPayload, transferPayload)
        ).to.equal(0);

        const sequenceTracker = await getProgramSequenceTracker(
          connection,
          TOKEN_BRIDGE_ADDRESS,
          CORE_BRIDGE_ADDRESS
        );
        expect(sequenceTracker.value()).to.equal(messageData.sequence + 1n);
      });
    });

    describe("Token Bridge Wrapped Token Handling", () => {
      it("Create Wrapped with Metadata", async () => {
        const tokenAddress = WETH_ADDRESS;
        const decimals = 18;
        const symbol = "WETH";
        const name = "Wrapped ETH";
        const nonce = 420;
        const message = ethereumTokenBridge.publishAttestMeta(
          tokenAddress,
          decimals,
          symbol,
          name,
          nonce
        );
        const signedVaa = guardians.addSignatures(
          message,
          [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18]
        );

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

        const createWrappedIx = createCreateWrappedInstruction(
          TOKEN_BRIDGE_ADDRESS,
          CORE_BRIDGE_ADDRESS,
          wallet.key(),
          signedVaa
        );

        const createWrappedTx = await web3.sendAndConfirmTransaction(
          connection,
          new web3.Transaction().add(createWrappedIx),
          [wallet.signer()]
        );
        // console.log(`createWrappedTx: ${createWrappedTx}`);

        // verify data
        const parsed = parseAttestMetaVaa(signedVaa);
        const messageData = await getPostedVaa(
          connection,
          CORE_BRIDGE_ADDRESS,
          parsed.hash
        ).then((posted) => posted.message);

        expect(messageData.consistencyLevel).to.equal(
          ethereumTokenBridge.consistencyLevel
        );
        const expectedEmitter = ethAddressToBuffer(
          ETHEREUM_TOKEN_BRIDGE_ADDRESS
        );
        expect(
          Buffer.compare(messageData.emitterAddress, expectedEmitter)
        ).to.equal(0);
        expect(messageData.emitterChain).to.equal(ethereumTokenBridge.chain);
        expect(messageData.nonce).to.equal(nonce);
        expect(messageData.sequence).to.equal(2n);
        expect(messageData.vaaTime).to.equal(0);
        expect(messageData.vaaVersion).to.equal(1);
        expect(Buffer.compare(parsed.payload, messageData.payload)).to.equal(0);

        const assetMeta = parseAttestMetaPayload(messageData.payload);
        expect(assetMeta.payloadType).to.equal(2);
        const expectedTokenAddress = ethAddressToBuffer(tokenAddress);
        expect(
          Buffer.compare(assetMeta.tokenAddress, expectedTokenAddress)
        ).to.equal(0);
        expect(assetMeta.tokenChain).to.equal(ethereumTokenBridge.chain);
        expect(assetMeta.decimals).to.equal(decimals);
        expect(assetMeta.symbol).to.equal(symbol);
        expect(assetMeta.name).to.equal(name);

        // check wrapped mint
        const mint = deriveWrappedMintKey(
          TOKEN_BRIDGE_ADDRESS,
          assetMeta.tokenChain,
          assetMeta.tokenAddress
        );
        const mintInfo = await getMint(connection, mint);
        expect(mintInfo.decimals).to.equal(8);
        expect(mintInfo.mintAuthority).is.not.null;
        expect(
          mintInfo.mintAuthority?.equals(
            deriveMintAuthorityKey(TOKEN_BRIDGE_ADDRESS)
          )
        ).is.true;
        expect(mintInfo.supply).to.equal(0n);

        // check wrapped meta
        const wrappedMeta = await getWrappedMeta(
          connection,
          TOKEN_BRIDGE_ADDRESS,
          mint
        );
        expect(wrappedMeta.chain).to.equal(ethereumTokenBridge.chain);
        expect(
          Buffer.compare(wrappedMeta.tokenAddress, expectedTokenAddress)
        ).to.equal(0);
        expect(wrappedMeta.originalDecimals).to.equal(decimals);

        // check metadata
        const expectedName = `${name} (Wormhole)`.padEnd(32, "\0");
        const metadata = await Metadata.fromAccountAddress(
          connection,
          deriveTokenMetadataKey(mint)
        );
        expect(metadata.data.symbol.toString()).equals(symbol.padEnd(10, "\0"));
        expect(metadata.data.name.toString()).equals(expectedName);
        localVariables.oldName = expectedName;
      });

      it("Update (Create) Wrapped with New Metadata", async () => {
        const tokenAddress = WETH_ADDRESS;
        const oldName: string = localVariables.oldName;

        const mint = deriveWrappedMintKey(
          TOKEN_BRIDGE_ADDRESS,
          ethereumTokenBridge.chain,
          tokenAddress
        );

        // check existing metadata
        {
          const metadata = await Metadata.fromAccountAddress(
            connection,
            deriveTokenMetadataKey(mint)
          );
          expect(metadata.data.name.toString()).equals(oldName);
        }

        const decimals = 18;
        const symbol = "WETH";
        const name = "Wrapped Ether";
        const nonce = 420;
        const message = ethereumTokenBridge.publishAttestMeta(
          tokenAddress,
          decimals,
          symbol,
          name,
          nonce
        );
        const signedVaa = guardians.addSignatures(
          message,
          [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18]
        );

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

        const createWrappedIx = createCreateWrappedInstruction(
          TOKEN_BRIDGE_ADDRESS,
          CORE_BRIDGE_ADDRESS,
          wallet.key(),
          signedVaa
        );

        const createWrappedTx = await web3.sendAndConfirmTransaction(
          connection,
          new web3.Transaction().add(createWrappedIx),
          [wallet.signer()]
        );
        // console.log(`createWrappedTx: ${createWrappedTx}`);

        // verify data
        const parsed = parseAttestMetaVaa(signedVaa);
        const messageData = await getPostedVaa(
          connection,
          CORE_BRIDGE_ADDRESS,
          parsed.hash
        ).then((posted) => posted.message);

        expect(messageData.consistencyLevel).to.equal(
          ethereumTokenBridge.consistencyLevel
        );
        const expectedEmitter = ethAddressToBuffer(
          ETHEREUM_TOKEN_BRIDGE_ADDRESS
        );
        expect(
          Buffer.compare(messageData.emitterAddress, expectedEmitter)
        ).to.equal(0);
        expect(messageData.emitterChain).to.equal(ethereumTokenBridge.chain);
        expect(messageData.nonce).to.equal(nonce);
        expect(messageData.sequence).to.equal(3n);
        expect(messageData.vaaTime).to.equal(0);
        expect(messageData.vaaVersion).to.equal(1);
        expect(Buffer.compare(parsed.payload, messageData.payload)).to.equal(0);

        const assetMeta = parseAttestMetaPayload(messageData.payload);
        expect(assetMeta.payloadType).to.equal(2);
        const expectedTokenAddress = ethAddressToBuffer(tokenAddress);
        expect(
          Buffer.compare(assetMeta.tokenAddress, expectedTokenAddress)
        ).to.equal(0);
        expect(assetMeta.tokenChain).to.equal(ethereumTokenBridge.chain);
        expect(assetMeta.decimals).to.equal(decimals);
        expect(assetMeta.symbol).to.equal(symbol);
        expect(assetMeta.name).to.equal(name);

        // check wrapped mint
        const mintInfo = await getMint(connection, mint);
        expect(mintInfo.decimals).to.equal(8);
        expect(mintInfo.mintAuthority).is.not.null;
        expect(
          mintInfo.mintAuthority?.equals(
            deriveMintAuthorityKey(TOKEN_BRIDGE_ADDRESS)
          )
        ).is.true;
        expect(mintInfo.supply).to.equal(0n);

        // check wrapped meta
        const wrappedMeta = await getWrappedMeta(
          connection,
          TOKEN_BRIDGE_ADDRESS,
          mint
        );
        expect(wrappedMeta.chain).to.equal(ethereumTokenBridge.chain);
        expect(
          Buffer.compare(wrappedMeta.tokenAddress, expectedTokenAddress)
        ).to.equal(0);
        expect(wrappedMeta.originalDecimals).to.equal(decimals);

        // check metadata
        const metadata = await Metadata.fromAccountAddress(
          connection,
          deriveTokenMetadataKey(mint)
        );
        expect(metadata.data.name.toString()).not.equals(oldName);

        expect(metadata.data.symbol.toString()).equals(symbol.padEnd(10, "\0"));
        expect(metadata.data.name.toString()).equals(
          `${name} (Wormhole)`.padEnd(32, "\0")
        );
      });

      it("Update (Create) Wrapped with New Metadata for V1 Metadata Account", async () => {
        const tokenAddress = DEADBEEF_ADDRESS;
        const oldExpectedName = "Dead Beef (Wormhole)".padEnd(32, "\0");

        // fetch previously created metadata account
        // check wrapped mint
        {
          const mint = deriveWrappedMintKey(
            TOKEN_BRIDGE_ADDRESS,
            ethereumTokenBridge.chain,
            tokenAddress
          );
          expect(mint.toString()).equals(DEADBEEF_MINT_ADDRESS);

          const metadataKey = deriveTokenMetadataKey(mint);
          expect(metadataKey.toString()).equals(DEADBEEF_METADATA_ADDRESS);

          const metadata = await Metadata.fromAccountAddress(
            connection,
            metadataKey
          );
          expect(metadata.data.name.toString()).equals(oldExpectedName);
        }

        const decimals = 18;
        const symbol = "BEEF";
        const name = "Dead Beef Modified";
        const nonce = 420;
        const message = ethereumTokenBridge.publishAttestMeta(
          tokenAddress,
          decimals,
          symbol,
          name,
          nonce
        );
        const signedVaa = guardians.addSignatures(
          message,
          [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18]
        );

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

        const createWrappedIx = createCreateWrappedInstruction(
          TOKEN_BRIDGE_ADDRESS,
          CORE_BRIDGE_ADDRESS,
          wallet.key(),
          signedVaa
        );

        const createWrappedTx = await web3.sendAndConfirmTransaction(
          connection,
          new web3.Transaction().add(createWrappedIx),
          [wallet.signer()]
        );
        // console.log(`createWrappedTx: ${createWrappedTx}`);

        // verify data
        const parsed = parseAttestMetaVaa(signedVaa);
        const messageData = await getPostedVaa(
          connection,
          CORE_BRIDGE_ADDRESS,
          parsed.hash
        ).then((posted) => posted.message);

        expect(messageData.consistencyLevel).to.equal(
          ethereumTokenBridge.consistencyLevel
        );
        const expectedEmitter = ethAddressToBuffer(
          ETHEREUM_TOKEN_BRIDGE_ADDRESS
        );
        expect(
          Buffer.compare(messageData.emitterAddress, expectedEmitter)
        ).to.equal(0);
        expect(messageData.emitterChain).to.equal(ethereumTokenBridge.chain);
        expect(messageData.nonce).to.equal(nonce);
        expect(messageData.sequence).to.equal(4n);
        expect(messageData.vaaTime).to.equal(0);
        expect(messageData.vaaVersion).to.equal(1);
        expect(Buffer.compare(parsed.payload, messageData.payload)).to.equal(0);

        const assetMeta = parseAttestMetaPayload(messageData.payload);
        expect(assetMeta.payloadType).to.equal(2);
        const expectedTokenAddress = ethAddressToBuffer(tokenAddress);
        expect(
          Buffer.compare(assetMeta.tokenAddress, expectedTokenAddress)
        ).to.equal(0);
        expect(assetMeta.tokenChain).to.equal(ethereumTokenBridge.chain);
        expect(assetMeta.decimals).to.equal(decimals);
        expect(assetMeta.symbol).to.equal(symbol);
        expect(assetMeta.name).to.equal(name);

        // check wrapped mint
        const mint = deriveWrappedMintKey(
          TOKEN_BRIDGE_ADDRESS,
          assetMeta.tokenChain,
          assetMeta.tokenAddress
        );
        const mintInfo = await getMint(connection, mint);
        expect(mintInfo.decimals).to.equal(8);
        expect(mintInfo.mintAuthority).is.not.null;
        expect(
          mintInfo.mintAuthority?.equals(
            deriveMintAuthorityKey(TOKEN_BRIDGE_ADDRESS)
          )
        ).is.true;
        expect(mintInfo.supply).to.equal(0n);

        // check metadata
        const metadata = await Metadata.fromAccountAddress(
          connection,
          deriveTokenMetadataKey(mint)
        );
        expect(metadata.data.name.toString()).not.equals(oldExpectedName);
        expect(metadata.data.name.toString()).equals(
          `${name} (Wormhole)`.padEnd(32, "\0")
        );
      });

      it("Receive Token", async () => {
        const tokenAddress = ethAddressToBuffer(WETH_ADDRESS);
        const tokenChain = ethereumTokenBridge.chain;
        const mint = deriveWrappedMintKey(
          TOKEN_BRIDGE_ADDRESS,
          tokenChain,
          tokenAddress
        );
        const mintAta = await getOrCreateAssociatedTokenAccount(
          connection,
          wallet.signer(),
          mint,
          wallet.key()
        ).then((account) => account.address);

        const walletBalanceBefore = await getAccount(connection, mintAta).then(
          (account) => account.amount
        );
        const supplyBefore = await getMint(connection, mint).then(
          (info) => info.supply
        );

        const amount = 2n * 4206942069n;
        const recipientChain = 1;
        const fee = 0n;
        const nonce = 420;
        const message = ethereumTokenBridge.publishTransferTokens(
          tokenAddress.toString("hex"),
          tokenChain,
          amount,
          recipientChain,
          mintAta.toBuffer().toString("hex"),
          fee,
          nonce
        );

        const signedVaa = guardians.addSignatures(
          message,
          [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18]
        );

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

        const completeTransferWrappedIx =
          createCompleteTransferWrappedInstruction(
            TOKEN_BRIDGE_ADDRESS,
            CORE_BRIDGE_ADDRESS,
            wallet.key(),
            signedVaa
          );

        const completeWrappedTransferTx = await web3.sendAndConfirmTransaction(
          connection,
          new web3.Transaction().add(completeTransferWrappedIx),
          [wallet.signer()]
        );
        // console.log(`completeWrappedTransferTx: ${completeWrappedTransferTx}`);

        const walletBalanceAfter = await getAccount(connection, mintAta).then(
          (account) => account.amount
        );
        const supplyAfter = await getMint(connection, mint).then(
          (info) => info.supply
        );

        // check balance and supply changes
        expect(walletBalanceAfter - walletBalanceBefore).to.equal(amount);
        expect(supplyAfter - supplyBefore).to.equal(amount);

        // verify data
        const parsed = parseVaa(signedVaa);
        const messageData = await getPostedVaa(
          connection,
          CORE_BRIDGE_ADDRESS,
          parsed.hash
        ).then((posted) => posted.message);

        expect(messageData.consistencyLevel).to.equal(
          ethereumTokenBridge.consistencyLevel
        );
        expect(
          Buffer.compare(
            messageData.emitterAddress,
            ethAddressToBuffer(ETHEREUM_TOKEN_BRIDGE_ADDRESS)
          )
        ).to.equal(0);
        expect(messageData.emitterChain).to.equal(ethereumTokenBridge.chain);
        expect(messageData.nonce).to.equal(nonce);
        expect(messageData.sequence).to.equal(5n);
        expect(messageData.vaaTime).to.equal(0);
        expect(messageData.vaaVersion).to.equal(1);
        expect(
          Buffer.compare(parseVaa(signedVaa).payload, messageData.payload)
        ).to.equal(0);

        const tokenTransfer = parseTokenTransferPayload(messageData.payload);
        expect(tokenTransfer.payloadType).to.equal(1);
        expect(tokenTransfer.amount).to.equal(amount);
        expect(tokenTransfer.fee).is.not.null;
        expect(tokenTransfer.fee).to.equal(fee);
        expect(tokenTransfer.fromAddress).is.null;
        expect(Buffer.compare(tokenTransfer.to, mintAta.toBuffer())).to.equal(
          0
        );
        expect(tokenTransfer.toChain).to.equal(recipientChain);
        expect(
          Buffer.compare(tokenTransfer.tokenAddress, tokenAddress)
        ).to.equal(0);
        expect(tokenTransfer.tokenChain).to.equal(tokenChain);
      });

      it("Send Token", async () => {
        const tokenAddress = ethAddressToBuffer(WETH_ADDRESS);
        const tokenChain = ethereumTokenBridge.chain;
        const mint = deriveWrappedMintKey(
          TOKEN_BRIDGE_ADDRESS,
          tokenChain,
          tokenAddress
        );
        const mintAta = getAssociatedTokenAddressSync(mint, wallet.key());

        const walletBalanceBefore = await getAccount(connection, mintAta).then(
          (account) => account.amount
        );
        const supplyBefore = await getMint(connection, mint).then(
          (info) => info.supply
        );

        const nonce = 69;
        const amount = 4206942069n;
        const fee = 0n;
        const targetAddress = Buffer.alloc(32, "deadbeef", "hex");
        const targetChain = 2;

        const approveIx = createApproveAuthoritySignerInstruction(
          TOKEN_BRIDGE_ADDRESS,
          mintAta,
          wallet.key(),
          amount
        );

        const message = web3.Keypair.generate();
        const transferNativeIx = createTransferWrappedInstruction(
          TOKEN_BRIDGE_ADDRESS,
          CORE_BRIDGE_ADDRESS,
          wallet.key(),
          message.publicKey,
          mintAta,
          wallet.key(),
          tokenChain,
          tokenAddress,
          nonce,
          amount,
          fee,
          targetAddress,
          targetChain
        );

        const approveAndTransferTx = await web3.sendAndConfirmTransaction(
          connection,
          new web3.Transaction().add(approveIx, transferNativeIx),
          [wallet.signer(), message]
        );
        // console.log(`approveAndTransferTx: ${approveAndTransferTx}`);

        const walletBalanceAfter = await getAccount(connection, mintAta).then(
          (account) => account.amount
        );
        const supplyAfter = await getMint(connection, mint).then(
          (info) => info.supply
        );

        // check balance changes
        expect(walletBalanceBefore - walletBalanceAfter).to.equal(amount);
        expect(supplyBefore - supplyAfter).to.equal(amount);

        // verify data
        const messageData = await getPostedMessage(
          connection,
          message.publicKey
        ).then((posted) => posted.message);

        expect(messageData.consistencyLevel).to.equal(32);
        expect(
          Buffer.compare(
            messageData.emitterAddress,
            deriveWormholeEmitterKey(TOKEN_BRIDGE_ADDRESS).toBuffer()
          )
        ).to.equal(0);
        expect(messageData.emitterChain).to.equal(1);
        expect(messageData.nonce).to.equal(nonce);
        expect(messageData.sequence).to.equal(3n);
        expect(messageData.vaaTime).to.equal(0);
        expect(messageData.vaaSignatureAccount.equals(web3.PublicKey.default))
          .is.true;
        expect(messageData.vaaVersion).to.equal(0);

        const tokenTransfer = parseTokenTransferPayload(messageData.payload);
        expect(tokenTransfer.payloadType).to.equal(1);
        const mintInfo = await getMint(connection, mint);
        expect(mintInfo.decimals).to.equal(8);
        expect(tokenTransfer.amount).to.equal(amount);
        expect(tokenTransfer.fee).is.not.null;
        expect(tokenTransfer.fee).to.equal(fee);
        expect(tokenTransfer.fromAddress).is.null;
        expect(Buffer.compare(tokenTransfer.to, targetAddress)).to.equal(0);
        expect(tokenTransfer.toChain).to.equal(targetChain);
        expect(
          Buffer.compare(tokenTransfer.tokenAddress, tokenAddress)
        ).to.equal(0);
        expect(tokenTransfer.tokenChain).to.equal(tokenChain);

        const sequenceTracker = await getProgramSequenceTracker(
          connection,
          TOKEN_BRIDGE_ADDRESS,
          CORE_BRIDGE_ADDRESS
        );
        expect(sequenceTracker.value()).to.equal(messageData.sequence + 1n);
      });

      it("Send Token With Payload", async () => {
        const tokenAddress = ethAddressToBuffer(WETH_ADDRESS);
        const tokenChain = ethereumTokenBridge.chain;
        const mint = deriveWrappedMintKey(
          TOKEN_BRIDGE_ADDRESS,
          tokenChain,
          tokenAddress
        );
        const mintAta = getAssociatedTokenAddressSync(mint, wallet.key());

        const walletBalanceBefore = await getAccount(connection, mintAta).then(
          (account) => account.amount
        );
        const supplyBefore = await getMint(connection, mint).then(
          (info) => info.supply
        );

        const nonce = 69;
        const amount = 4206942069n;
        const targetAddress = Buffer.alloc(32, "deadbeef", "hex");
        const targetChain = 2;

        const approveIx = createApproveAuthoritySignerInstruction(
          TOKEN_BRIDGE_ADDRESS,
          mintAta,
          wallet.key(),
          amount
        );

        const message = web3.Keypair.generate();
        const transferPayload = Buffer.from("All your base are belong to us");
        const transferNativeIx = createTransferWrappedWithPayloadInstruction(
          TOKEN_BRIDGE_ADDRESS,
          CORE_BRIDGE_ADDRESS,
          wallet.key(),
          message.publicKey,
          mintAta,
          wallet.key(),
          tokenChain,
          tokenAddress,
          nonce,
          amount,
          targetAddress,
          targetChain,
          transferPayload
        );

        const approveAndTransferTx = await web3.sendAndConfirmTransaction(
          connection,
          new web3.Transaction().add(approveIx, transferNativeIx),
          [wallet.signer(), message]
        );
        // console.log(`approveAndTransferTx: ${approveAndTransferTx}`);

        const walletBalanceAfter = await getAccount(connection, mintAta).then(
          (account) => account.amount
        );
        const supplyAfter = await getMint(connection, mint).then(
          (info) => info.supply
        );

        // check balance changes
        expect(walletBalanceBefore - walletBalanceAfter).to.equal(amount);
        expect(supplyBefore - supplyAfter).to.equal(amount);

        // verify data
        const messageData = await getPostedMessage(
          connection,
          message.publicKey
        ).then((posted) => posted.message);

        expect(messageData.consistencyLevel).to.equal(32);
        expect(
          Buffer.compare(
            messageData.emitterAddress,
            deriveWormholeEmitterKey(TOKEN_BRIDGE_ADDRESS).toBuffer()
          )
        ).to.equal(0);
        expect(messageData.emitterChain).to.equal(1);
        expect(messageData.nonce).to.equal(nonce);
        expect(messageData.sequence).to.equal(4n);
        expect(messageData.vaaTime).to.equal(0);
        expect(messageData.vaaSignatureAccount.equals(web3.PublicKey.default))
          .is.true;
        expect(messageData.vaaVersion).to.equal(0);

        const tokenTransfer = parseTokenTransferPayload(messageData.payload);
        expect(tokenTransfer.payloadType).to.equal(3);
        const mintInfo = await getMint(connection, mint);
        expect(mintInfo.decimals).to.equal(8);
        expect(tokenTransfer.amount).to.equal(amount);
        expect(Buffer.compare(tokenTransfer.to, targetAddress)).to.equal(0);
        expect(tokenTransfer.toChain).to.equal(targetChain);
        expect(
          Buffer.compare(tokenTransfer.tokenAddress, tokenAddress)
        ).to.equal(0);
        expect(tokenTransfer.tokenChain).to.equal(tokenChain);
        expect(
          Buffer.compare(tokenTransfer.tokenTransferPayload, transferPayload)
        ).to.equal(0);

        const sequenceTracker = await getProgramSequenceTracker(
          connection,
          TOKEN_BRIDGE_ADDRESS,
          CORE_BRIDGE_ADDRESS
        );
        expect(sequenceTracker.value()).to.equal(messageData.sequence + 1n);
      });
    });
  });

  describe("SDK Methods", () => {
    // nft bridge on Ethereum
    const ethereumTokenBridge = new MockEthereumTokenBridge(
      ETHEREUM_TOKEN_BRIDGE_ADDRESS,
      10 // startSequence
    );

    describe("getOriginalAssetSolana", () => {
      it("Non-existent Token", async () => {
        const mint = "wot m8?";
        const asset = await getOriginalAssetSolana(
          connection,
          TOKEN_BRIDGE_ADDRESS,
          mint
        );

        // verify results
        expect(asset.isWrapped).to.be.false;
        expect(asset.chainId).to.equal(1);
        expect(
          Buffer.compare(Buffer.from(asset.assetAddress), Buffer.alloc(32))
        ).to.equal(0);
      });

      it("Native Token", async () => {
        const mint = localVariables.mint;

        const asset = await getOriginalAssetSolana(
          connection,
          TOKEN_BRIDGE_ADDRESS,
          mint
        );

        // verify results
        expect(asset.isWrapped).to.be.false;
        expect(asset.chainId).to.equal(1);
        expect(
          Buffer.compare(Buffer.from(asset.assetAddress), mint.toBuffer())
        ).to.equal(0);
      });

      it("Wrapped Token", async () => {
        const tokenAddress = ethAddressToBuffer(WETH_ADDRESS);
        const tokenChain = ethereumTokenBridge.chain;
        const mint = deriveWrappedMintKey(
          TOKEN_BRIDGE_ADDRESS,
          tokenChain,
          tokenAddress
        );

        const asset = await getOriginalAssetSolana(
          connection,
          TOKEN_BRIDGE_ADDRESS,
          mint
        );

        // verify results
        expect(asset.isWrapped).is.true;
        expect(asset.chainId).to.equal(tokenChain);
        expect(
          Buffer.compare(Buffer.from(asset.assetAddress), tokenAddress)
        ).to.equal(0);
      });
    });

    describe("getForeignAssetSolana", () => {
      it("Wrapped Token", async () => {
        const tokenAddress = ethAddressToBuffer(WETH_ADDRESS);
        const tokenChain = ethereumTokenBridge.chain;

        const asset = await getForeignAssetSolana(
          connection,
          TOKEN_BRIDGE_ADDRESS,
          tokenChain as ChainId,
          tokenAddress
        );

        // verify results
        expect(asset).to.equal("3tUXFuBNWzZZ8p2xNx5UoWCH664M2KHdDAWrdZAD1VQ3");
      });
    });

    describe("getIsWrappedAsset", () => {
      it("Non-existent Token", async () => {
        const mint = null;

        const isWrapped = await getIsWrappedAssetSolana(
          connection,
          TOKEN_BRIDGE_ADDRESS,
          // @ts-ignore: mint is null
          mint
        );

        // verify results
        expect(isWrapped).to.be.false;
      });

      it("Native Token", async () => {
        const mint = localVariables.mint;

        const isWrapped = await getIsWrappedAssetSolana(
          connection,
          TOKEN_BRIDGE_ADDRESS,
          mint
        );

        // verify results
        expect(isWrapped).to.be.false;
      });

      it("Wrapped Token", async () => {
        const tokenAddress = ethAddressToBuffer(WETH_ADDRESS);
        const tokenChain = ethereumTokenBridge.chain;
        const mint = deriveWrappedMintKey(
          TOKEN_BRIDGE_ADDRESS,
          tokenChain,
          tokenAddress
        );

        const isWrapped = await getIsWrappedAssetSolana(
          connection,
          TOKEN_BRIDGE_ADDRESS,
          mint
        );

        // verify results
        expect(isWrapped).is.true;
      });
    });

    describe("transferNativeSol", () => {
      it("Send SOL To Ethereum", async () => {
        const balanceBefore = await connection
          .getBalance(wallet.key())
          .then((num) => BigInt(num));

        const amount = 6969696969n;
        const targetAddress = Buffer.alloc(32, "deadbeef", "hex");

        const transferNativeSolTx = await transferNativeSol(
          connection,
          CORE_BRIDGE_ADDRESS,
          TOKEN_BRIDGE_ADDRESS,
          wallet.key(),
          amount,
          targetAddress,
          "ethereum"
        )
          .then((transaction) =>
            signSendAndConfirmTransaction(
              connection,
              wallet.key(),
              wallet.signTransaction,
              transaction
            )
          )
          .then((response) => response.signature);
        //console.log(`transferNativeSolTx: ${transferNativeSolTx}`);

        const balanceAfter = await connection
          .getBalance(wallet.key())
          .then((num) => BigInt(num));

        const transactionCost = 4601551n;
        expect(balanceBefore - balanceAfter - transactionCost).to.equal(amount);
      });
    });

    describe("redeemAndUnwrapOnSolana", () => {
      it("Receive SOL From Ethereum", async () => {
        const balanceBefore = await connection
          .getBalance(wallet.key())
          .then((num) => BigInt(num));

        const tokenChain = 1;
        const mintAta = await getOrCreateAssociatedTokenAccount(
          connection,
          wallet.signer(),
          NATIVE_MINT,
          wallet.key()
        ).then((account) => account.address);

        const amount = 42042042n;
        const recipientChain = 1;
        const fee = 0n;
        const nonce = 420;
        const message = ethereumTokenBridge.publishTransferTokens(
          NATIVE_MINT.toBuffer().toString("hex"),
          tokenChain,
          amount,
          recipientChain,
          mintAta.toBuffer().toString("hex"),
          fee,
          nonce
        );

        const signedVaa = guardians.addSignatures(
          message,
          [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18]
        );

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

        const transferNativeSolTx = await redeemAndUnwrapOnSolana(
          connection,
          CORE_BRIDGE_ADDRESS,
          TOKEN_BRIDGE_ADDRESS,
          wallet.key(),
          signedVaa
        )
          .then((transaction) =>
            signSendAndConfirmTransaction(
              connection,
              wallet.key(),
              wallet.signTransaction,
              transaction
            )
          )
          .then((response) => response.signature);
        //console.log(`transferNativeSolTx: ${transferNativeSolTx}`);

        const balanceAfter = await connection
          .getBalance(wallet.key())
          .then((num) => BigInt(num));

        const transactionCost = 6821400n;
        expect(balanceAfter - (balanceBefore - transactionCost)).to.equal(
          amount * 10n
        );
      });
    });
  });
});
