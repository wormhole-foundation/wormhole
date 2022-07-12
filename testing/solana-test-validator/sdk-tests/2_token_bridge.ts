import { expect } from "chai";
import * as web3 from "@solana/web3.js";
import {
  createMint,
  getAccount,
  getAssociatedTokenAddress,
  getMint,
  getOrCreateAssociatedTokenAccount,
  mintTo,
  NATIVE_MINT,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import {
  MockGuardians,
  TokenBridgeGovernanceEmitter,
  MockEthereumTokenBridge,
} from "../../../sdk/js/src/mock";
import {
  createApproveAuthoritySignerInstruction,
  createAttestTokenInstruction,
  createCompleteTransferNativeInstruction,
  createCompleteTransferWrappedInstruction,
  createCreateWrappedInstruction,
  createInitializeInstruction,
  createRegisterChainInstruction,
  createTransferNativeInstruction,
  createTransferNativeWithPayloadInstruction,
  createTransferWrappedInstruction,
  createTransferWrappedWithPayloadInstruction,
  deriveCustodyKey,
  deriveEndpointKey,
  deriveMintAuthorityKey,
  deriveWrappedMintKey,
  getAttestTokenAccounts,
  getCompleteTransferNativeAccounts,
  getCompleteTransferWrappedAccounts,
  getCreateWrappedAccounts,
  getEndpointRegistration,
  getInitializeAccounts,
  getRegisterChainAccounts,
  getTokenBridgeConfig,
  getTransferNativeAccounts,
  getTransferNativeWithPayloadAccounts,
  getTransferWrappedAccounts,
  getTransferWrappedWithPayloadAccounts,
} from "../../../sdk/js/src/solana/tokenBridge";
import { postVaa } from "../../../sdk/js/src/solana/sendAndConfirmPostVaa";
import {
  deriveWormholeEmitterKey,
  getPostedMessage,
  getPostedVaa,
  NodeWallet,
  SplTokenMetadataProgram,
} from "../../../sdk/js/src/solana";
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
} from "./helpers/consts";
import { ethAddressToBuffer, now } from "./helpers/utils";

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
    const governance = new TokenBridgeGovernanceEmitter(
      GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex")
    );

    // token bridge on Ethereum
    const ethereumTokenBridge = new MockEthereumTokenBridge(
      ETHEREUM_TOKEN_BRIDGE_ADDRESS
    );

    const payer = new web3.PublicKey(
      "6sbzC1eH4FTujJXWj51eQe25cYvr4xfXbJ1vAj7j2k5J"
    );

    it("Instruction 0: Initialize", () => {
      const accounts = getInitializeAccounts(TOKEN_BRIDGE_ADDRESS, payer);

      // verify accounts
      expect(accounts.config.toString()).to.equal(
        "3GwVs8GSLdo4RUsoXTkGQhojauQ1sXcDNjm7LSDicw19"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).to.be.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
    });

    it("Instruction 1: Attest Token", () => {
      const mint = NATIVE_MINT;
      const message = web3.Keypair.generate();
      const accounts = getAttestTokenAccounts(
        TOKEN_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        wallet.key(),
        mint,
        message.publicKey
      );

      // verify accounts
      expect(accounts.config.toString()).to.equal(
        "3GwVs8GSLdo4RUsoXTkGQhojauQ1sXcDNjm7LSDicw19"
      );
      expect(accounts.wrappedMeta.toString()).to.equal(
        "Euey6bDcoZ7it4fYpeLFwF7riApfVAkp7qsZ8Wp88diu"
      );
      expect(accounts.splMetadata.toString()).to.equal(
        "6dM4TqWyWJsbx7obrdLcviBkTafD5E8av61zfU6jq57X"
      );
      expect(accounts.wormholeConfig.toString()).to.equal(
        "FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP"
      );
      expect(accounts.wormholeEmitter.toString()).to.equal(
        "ENG1wQ7CQKH8ibAJ1hSLmJgL9Ucg6DRDbj752ZAfidLA"
      );
      expect(accounts.wormholeSequence.toString()).to.equal(
        "7F4RNrCkBJxs1uidvF96iPieZ8upkEnc8NdpHoJ8YjxH"
      );
      expect(accounts.wormholeFeeCollector.toString()).to.equal(
        "GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).to.be.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).to.be.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).to.be.true;
    });

    it("Instruction 2: Complete Native", async () => {
      const mint = NATIVE_MINT;
      const mintAta = await getAssociatedTokenAddress(mint, payer);

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
      expect(accounts.config.toString()).to.equal(
        "3GwVs8GSLdo4RUsoXTkGQhojauQ1sXcDNjm7LSDicw19"
      );
      expect(accounts.vaa.toString()).to.equal(
        "fk2fuDPoyhEw9Ln7KT5ZgXgttsM5un9pk6HjQH3xGw5"
      );
      expect(accounts.claim.toString()).to.equal(
        "9bHANw61DHWs8ZX66dwQWnkZZkADQKwmMy5ccKqgnzjv"
      );
      expect(accounts.endpoint.toString()).to.equal(
        "7UqWgfVW1TrjrqauMfDoNMcw8kEStSsQXWNoT2BbhDS5"
      );
      expect(accounts.to.equals(mintAta)).to.be.true;
      expect(accounts.toFees.equals(mintAta)).to.be.true;
      expect(accounts.custody.toString()).to.equal(
        "GFoobF1UjsycpBwaZ11BwbHB37qiwXj1iiNYpfJt3GXP"
      );
      expect(accounts.mint.toString()).to.equal(
        "So11111111111111111111111111111111111111112"
      );
      expect(accounts.custodySigner.toString()).to.equal(
        "JCQ1JdJ3vgnvurNAqMvpwaiSwJXaoMFJN53F6sRKejxQ"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).to.be.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).to.be.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).to.be.true;
    });

    it("Instruction 3: Complete Wrapped", async () => {
      const tokenAddress = ethAddressToBuffer(WETH_ADDRESS);
      const tokenChain = ethereumTokenBridge.chain;
      const mint = deriveWrappedMintKey(
        TOKEN_BRIDGE_ADDRESS,
        tokenChain,
        tokenAddress
      );
      const mintAta = await getAssociatedTokenAddress(mint, payer);

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
      expect(accounts.config.toString()).to.equal(
        "3GwVs8GSLdo4RUsoXTkGQhojauQ1sXcDNjm7LSDicw19"
      );
      expect(accounts.vaa.toString()).to.equal(
        "FzwYv4qYHfDct2WPdbgW9qgU5vZKyuT2XZhSkqiw1Vkq"
      );
      expect(accounts.claim.toString()).to.equal(
        "HAS1yNoBFvoyZLowV4DdyA8Ap9SzHNBHUzQszJ4arCpd"
      );
      expect(accounts.endpoint.toString()).to.equal(
        "7UqWgfVW1TrjrqauMfDoNMcw8kEStSsQXWNoT2BbhDS5"
      );
      expect(accounts.to.equals(mintAta)).to.be.true;
      expect(accounts.toFees.equals(mintAta)).to.be.true;
      expect(accounts.mint.equals(mint)).to.be.true;
      expect(accounts.wrappedMeta.toString()).to.equal(
        "GQfedrnBFCqhUDjVWzpvnv5hXzpixcxyVPoeWeTRVhoT"
      );
      expect(accounts.mintAuthority.toString()).to.equal(
        "8P2wAnHr2t4pAVEyJftzz7k6wuCE7aP1VugNwehzCJJY"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).to.be.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).to.be.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).to.be.true;
    });

    it("Instruction 4: Transfer Wrapped", async () => {
      const tokenAddress = ethAddressToBuffer(WETH_ADDRESS);
      const tokenChain = ethereumTokenBridge.chain;
      const mint = deriveWrappedMintKey(
        TOKEN_BRIDGE_ADDRESS,
        tokenChain,
        tokenAddress
      );
      const mintAta = await getAssociatedTokenAddress(mint, payer);

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
      expect(accounts.config.toString()).to.equal(
        "3GwVs8GSLdo4RUsoXTkGQhojauQ1sXcDNjm7LSDicw19"
      );
      expect(accounts.from.equals(mintAta)).to.be.true;
      expect(accounts.fromOwner.equals(payer)).to.be.true;
      expect(accounts.mint.equals(mint)).to.be.true;
      expect(accounts.wrappedMeta.toString()).to.equal(
        "GQfedrnBFCqhUDjVWzpvnv5hXzpixcxyVPoeWeTRVhoT"
      );
      expect(accounts.mint.equals(mint)).to.be.true;
      expect(accounts.wrappedMeta.toString()).to.equal(
        "GQfedrnBFCqhUDjVWzpvnv5hXzpixcxyVPoeWeTRVhoT"
      );
      expect(accounts.authoritySigner.toString()).to.equal(
        "C1AVBd8PpfHGe1zW42XXVbHsAQf6q5khiRKuGPLbwHkh"
      );
      expect(accounts.wormholeConfig.toString()).to.equal(
        "FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP"
      );
      expect(accounts.wormholeEmitter.toString()).to.equal(
        "ENG1wQ7CQKH8ibAJ1hSLmJgL9Ucg6DRDbj752ZAfidLA"
      );
      expect(accounts.wormholeSequence.toString()).to.equal(
        "7F4RNrCkBJxs1uidvF96iPieZ8upkEnc8NdpHoJ8YjxH"
      );
      expect(accounts.wormholeFeeCollector.toString()).to.equal(
        "GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).to.be.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).to.be.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).to.be.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).to.be.true;
    });

    it("Instruction 5: Transfer Native", async () => {
      const mint = NATIVE_MINT;
      const mintAta = await getAssociatedTokenAddress(mint, payer);
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
      expect(accounts.config.toString()).to.equal(
        "3GwVs8GSLdo4RUsoXTkGQhojauQ1sXcDNjm7LSDicw19"
      );
      expect(accounts.custody.toString()).to.equal(
        "GFoobF1UjsycpBwaZ11BwbHB37qiwXj1iiNYpfJt3GXP"
      );
      expect(accounts.authoritySigner.toString()).to.equal(
        "C1AVBd8PpfHGe1zW42XXVbHsAQf6q5khiRKuGPLbwHkh"
      );
      expect(accounts.custodySigner.toString()).to.equal(
        "JCQ1JdJ3vgnvurNAqMvpwaiSwJXaoMFJN53F6sRKejxQ"
      );
      expect(accounts.wormholeConfig.toString()).to.equal(
        "FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP"
      );
      expect(accounts.wormholeEmitter.toString()).to.equal(
        "ENG1wQ7CQKH8ibAJ1hSLmJgL9Ucg6DRDbj752ZAfidLA"
      );
      expect(accounts.wormholeSequence.toString()).to.equal(
        "7F4RNrCkBJxs1uidvF96iPieZ8upkEnc8NdpHoJ8YjxH"
      );
      expect(accounts.wormholeFeeCollector.toString()).to.equal(
        "GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).to.be.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).to.be.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).to.be.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).to.be.true;
    });

    it("Instruction 6: Register Chain", () => {
      const timestamp = 45678901;
      const message = governance.publishRegisterChain(
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
      expect(accounts.config.toString()).to.equal(
        "3GwVs8GSLdo4RUsoXTkGQhojauQ1sXcDNjm7LSDicw19"
      );
      expect(accounts.endpoint.toString()).to.equal(
        "7UqWgfVW1TrjrqauMfDoNMcw8kEStSsQXWNoT2BbhDS5"
      );
      expect(accounts.vaa.toString()).to.equal(
        "LH1KyVAFFXUwbPgcF62MA3hBpCGcs6NZKuX6xXwemea"
      );
      expect(accounts.claim.toString()).to.equal(
        "5Sqq4RSDcfy5trhMzmNekyyYxhRrNoy4jeef4PrKsgzF"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).to.be.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).to.be.true;
    });

    it("Instruction 7: Create Wrapped", () => {
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
      expect(accounts.config.toString()).to.equal(
        "3GwVs8GSLdo4RUsoXTkGQhojauQ1sXcDNjm7LSDicw19"
      );
      expect(accounts.endpoint.toString()).to.equal(
        "7UqWgfVW1TrjrqauMfDoNMcw8kEStSsQXWNoT2BbhDS5"
      );
      expect(accounts.vaa.toString()).to.equal(
        "E8f1LsqStguXL8Rw4YAoVN97vo8yzWf2Cg3JienT6zVy"
      );
      expect(accounts.claim.toString()).to.equal(
        "9fUXMQnTsmgdtMuSpzzorKgYrHWBnVPAiiKEewr1NMWP"
      );
      expect(accounts.mint.toString()).to.equal(
        "3idbCQz6ZpWJ4hzraeEVUSbwMSSTZDihFMVzdLu15659"
      );
      expect(accounts.wrappedMeta.toString()).to.equal(
        "GQfedrnBFCqhUDjVWzpvnv5hXzpixcxyVPoeWeTRVhoT"
      );
      expect(accounts.splMetadata.toString()).to.equal(
        "BFsFGG9d3Xj6AoKf64pmNyMdtvyXeSMZsXGGf2BTFnXs"
      );
      expect(accounts.mintAuthority.toString()).to.equal(
        "8P2wAnHr2t4pAVEyJftzz7k6wuCE7aP1VugNwehzCJJY"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).to.be.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).to.be.true;
      expect(
        accounts.splMetadataProgram.equals(SplTokenMetadataProgram.programId)
      ).to.be.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).to.be.true;
    });

    it("Instruction 8: Upgrade Contract", () => {
      const timestamp = 56789012;
      // TODO
    });

    it("Instruction 11: Transfer Wrapped With Payload", async () => {
      const tokenAddress = ethAddressToBuffer(WETH_ADDRESS);
      const tokenChain = ethereumTokenBridge.chain;
      const mint = deriveWrappedMintKey(
        TOKEN_BRIDGE_ADDRESS,
        tokenChain,
        tokenAddress
      );
      const mintAta = await getAssociatedTokenAddress(mint, payer);

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
      expect(accounts.config.toString()).to.equal(
        "3GwVs8GSLdo4RUsoXTkGQhojauQ1sXcDNjm7LSDicw19"
      );
      expect(accounts.from.equals(mintAta)).to.be.true;
      expect(accounts.fromOwner.equals(payer)).to.be.true;
      expect(accounts.mint.equals(mint)).to.be.true;
      expect(accounts.wrappedMeta.toString()).to.equal(
        "GQfedrnBFCqhUDjVWzpvnv5hXzpixcxyVPoeWeTRVhoT"
      );
      expect(accounts.mint.equals(mint)).to.be.true;
      expect(accounts.wrappedMeta.toString()).to.equal(
        "GQfedrnBFCqhUDjVWzpvnv5hXzpixcxyVPoeWeTRVhoT"
      );
      expect(accounts.authoritySigner.toString()).to.equal(
        "C1AVBd8PpfHGe1zW42XXVbHsAQf6q5khiRKuGPLbwHkh"
      );
      expect(accounts.wormholeConfig.toString()).to.equal(
        "FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP"
      );
      expect(accounts.wormholeEmitter.toString()).to.equal(
        "ENG1wQ7CQKH8ibAJ1hSLmJgL9Ucg6DRDbj752ZAfidLA"
      );
      expect(accounts.wormholeSequence.toString()).to.equal(
        "7F4RNrCkBJxs1uidvF96iPieZ8upkEnc8NdpHoJ8YjxH"
      );
      expect(accounts.wormholeFeeCollector.toString()).to.equal(
        "GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).to.be.true;
      expect(accounts.sender.equals(payer)).to.be.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).to.be.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).to.be.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).to.be.true;
    });

    it("Instruction 12: Transfer Native With Payload", async () => {
      const mint = NATIVE_MINT;
      const mintAta = await getAssociatedTokenAddress(mint, payer);
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
      expect(accounts.config.toString()).to.equal(
        "3GwVs8GSLdo4RUsoXTkGQhojauQ1sXcDNjm7LSDicw19"
      );
      expect(accounts.custody.toString()).to.equal(
        "GFoobF1UjsycpBwaZ11BwbHB37qiwXj1iiNYpfJt3GXP"
      );
      expect(accounts.authoritySigner.toString()).to.equal(
        "C1AVBd8PpfHGe1zW42XXVbHsAQf6q5khiRKuGPLbwHkh"
      );
      expect(accounts.custodySigner.toString()).to.equal(
        "JCQ1JdJ3vgnvurNAqMvpwaiSwJXaoMFJN53F6sRKejxQ"
      );
      expect(accounts.wormholeConfig.toString()).to.equal(
        "FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP"
      );
      expect(accounts.wormholeEmitter.toString()).to.equal(
        "ENG1wQ7CQKH8ibAJ1hSLmJgL9Ucg6DRDbj752ZAfidLA"
      );
      expect(accounts.wormholeSequence.toString()).to.equal(
        "7F4RNrCkBJxs1uidvF96iPieZ8upkEnc8NdpHoJ8YjxH"
      );
      expect(accounts.wormholeFeeCollector.toString()).to.equal(
        "GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).to.be.true;
      expect(accounts.sender.equals(payer)).to.be.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).to.be.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).to.be.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).to.be.true;
    });
  });

  describe("Token Bridge Program Interaction", () => {
    // for generating governance wormhole messages
    const governance = new TokenBridgeGovernanceEmitter(
      GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex")
    );

    // token bridge on Ethereum
    const ethereumTokenBridge = new MockEthereumTokenBridge(
      ETHEREUM_TOKEN_BRIDGE_ADDRESS
    );

    describe("Setup Token Bridge", () => {
      it("Initialize", async () => {
        const initializeTx = await web3.sendAndConfirmTransaction(
          connection,
          new web3.Transaction().add(
            createInitializeInstruction(
              TOKEN_BRIDGE_ADDRESS,
              wallet.key(),
              CORE_BRIDGE_ADDRESS
            )
          ),
          [wallet.signer()]
        );
        // console.log(`initializeTx: ${initializeTx}`);

        // verify data
        const config = await getTokenBridgeConfig(
          connection,
          TOKEN_BRIDGE_ADDRESS
        );
        expect(config.wormhole.equals(CORE_BRIDGE_ADDRESS)).to.be.true;
      });

      it("Register Ethereum Token Bridge", async () => {
        const timestamp = now();
        const message = governance.publishRegisterChain(
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
          .to.be.true;
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
      });

      it("Attest Mint With Metadata", async () => {
        // TODO
      });

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
          .to.be.true;
        expect(messageData.vaaVersion).to.equal(0);

        const tokenTransfer = parseTokenTransferPayload(messageData.payload);
        expect(tokenTransfer.payloadType).to.equal(1);
        const mintInfo = await getMint(connection, mint);
        expect(mintInfo.decimals).greaterThan(8);
        // decimals will be 8 on Ethereum token bridge
        const amountEncoded =
          amount / BigInt(Math.pow(10, mintInfo.decimals - 8));
        expect(tokenTransfer.amount).to.equal(amountEncoded);
        expect(tokenTransfer.fee).to.equal(fee);
        expect(Buffer.compare(tokenTransfer.to, targetAddress)).to.equal(0);
        expect(tokenTransfer.toChain).to.equal(targetChain);
        expect(
          Buffer.compare(tokenTransfer.tokenAddress, mint.toBuffer())
        ).to.equal(0);
        expect(tokenTransfer.tokenChain).to.equal(1);
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
        expect(messageData.sequence).to.equal(1n);
        expect(messageData.vaaTime).to.equal(0);
        expect(messageData.vaaVersion).to.equal(1);
        expect(
          Buffer.compare(parseVaa(signedVaa).payload, messageData.payload)
        ).to.equal(0);

        const tokenTransfer = parseTokenTransferPayload(messageData.payload);
        expect(tokenTransfer.payloadType).to.equal(1);
        expect(tokenTransfer.amount).to.equal(amountEncoded);
        expect(tokenTransfer.fee).to.equal(fee);
        expect(Buffer.compare(tokenTransfer.to, mintAta.toBuffer())).to.equal(
          0
        );
        expect(tokenTransfer.toChain).to.equal(recipientChain);
        expect(
          Buffer.compare(tokenTransfer.tokenAddress, mint.toBuffer())
        ).to.equal(0);
        expect(tokenTransfer.tokenChain).to.equal(tokenChain);
      });

      it("Send SOL", async () => {
        // TODO
      });

      it("Receive SOL", async () => {
        // TODO
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
          .to.be.true;
        expect(messageData.vaaVersion).to.equal(0);

        const tokenTransfer = parseTokenTransferPayload(messageData.payload);
        expect(tokenTransfer.payloadType).to.equal(3);
        const mintInfo = await getMint(connection, mint);
        expect(mintInfo.decimals).greaterThan(8);
        // decimals will be 8 on Ethereum token bridge
        const amountEncoded =
          amount / BigInt(Math.pow(10, mintInfo.decimals - 8));
        expect(tokenTransfer.amount).to.equal(amountEncoded);
        expect(Buffer.compare(tokenTransfer.to, targetAddress)).to.equal(0);
        expect(tokenTransfer.toChain).to.equal(targetChain);
        expect(
          Buffer.compare(tokenTransfer.tokenAddress, mint.toBuffer())
        ).to.equal(0);
        expect(tokenTransfer.tokenChain).to.equal(1);
        expect(
          Buffer.compare(tokenTransfer.tokenTransferPayload, transferPayload)
        ).to.equal(0);
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
        ).to.be.true;
        expect(mintInfo.supply).to.equal(0n);
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
        expect(messageData.sequence).to.equal(3n);
        expect(messageData.vaaTime).to.equal(0);
        expect(messageData.vaaVersion).to.equal(1);
        expect(
          Buffer.compare(parseVaa(signedVaa).payload, messageData.payload)
        ).to.equal(0);

        const tokenTransfer = parseTokenTransferPayload(messageData.payload);
        expect(tokenTransfer.payloadType).to.equal(1);
        expect(tokenTransfer.amount).to.equal(amount);
        expect(tokenTransfer.fee).to.equal(fee);
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
        const mintAta = await getAssociatedTokenAddress(mint, wallet.key());

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
          .to.be.true;
        expect(messageData.vaaVersion).to.equal(0);

        const tokenTransfer = parseTokenTransferPayload(messageData.payload);
        expect(tokenTransfer.payloadType).to.equal(1);
        const mintInfo = await getMint(connection, mint);
        expect(mintInfo.decimals).to.equal(8);
        expect(tokenTransfer.amount).to.equal(amount);
        expect(tokenTransfer.fee).to.equal(fee);
        expect(Buffer.compare(tokenTransfer.to, targetAddress)).to.equal(0);
        expect(tokenTransfer.toChain).to.equal(targetChain);
        expect(
          Buffer.compare(tokenTransfer.tokenAddress, tokenAddress)
        ).to.equal(0);
        expect(tokenTransfer.tokenChain).to.equal(tokenChain);
      });

      it("Send Token With Payload", async () => {
        const tokenAddress = ethAddressToBuffer(WETH_ADDRESS);
        const tokenChain = ethereumTokenBridge.chain;
        const mint = deriveWrappedMintKey(
          TOKEN_BRIDGE_ADDRESS,
          tokenChain,
          tokenAddress
        );
        const mintAta = await getAssociatedTokenAddress(mint, wallet.key());

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
          .to.be.true;
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
      });
    });
  });
});
