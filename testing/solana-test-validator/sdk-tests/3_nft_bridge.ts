import { expect } from "chai";
import * as web3 from "@solana/web3.js";
import {
  Metadata,
  PROGRAM_ID as TOKEN_METADATA_PROGRAM_ID,
  createCreateMetadataAccountV3Instruction,
} from "@metaplex-foundation/mpl-token-metadata";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
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
  MockEthereumNftBridge,
} from "../../../sdk/js/src/mock";
import { postVaa } from "../../../sdk/js/src/solana/sendAndConfirmPostVaa";
import {
  BpfLoaderUpgradeable,
  NodeWallet,
  deriveTokenMetadataKey,
} from "../../../sdk/js/src/solana";
import {
  deriveWormholeEmitterKey,
  getPostedMessage,
  getPostedVaa,
} from "../../../sdk/js/src/solana/wormhole";
import {
  parseNftBridgeRegisterChainVaa,
  parseNftTransferPayload,
  parseVaa,
} from "../../../sdk/js/src/vaa";

import {
  CORE_BRIDGE_ADDRESS,
  NFT_BRIDGE_ADDRESS,
  ETHEREUM_NFT_BRIDGE_ADDRESS,
  GOVERNANCE_EMITTER_ADDRESS,
  GUARDIAN_KEYS,
  GUARDIAN_SET_INDEX,
  LOCALHOST,
  WETH_ADDRESS,
} from "./helpers/consts";
import { ethAddressToBuffer, makeErc721Token, now } from "./helpers/utils";
import {
  createApproveAuthoritySignerInstruction,
  createCompleteTransferNativeInstruction,
  createCompleteTransferWrappedInstruction,
  createCompleteWrappedMetaInstruction,
  createRegisterChainInstruction,
  createTransferNativeInstruction,
  createTransferWrappedInstruction,
  deriveCustodyKey,
  deriveEndpointKey,
  deriveWrappedMintKey,
  getCompleteTransferNativeAccounts,
  getCompleteTransferWrappedAccounts,
  getCompleteWrappedMetaAccounts,
  getEndpointRegistration,
  getInitializeAccounts,
  getRegisterChainAccounts,
  getTransferNativeAccounts,
  getTransferWrappedAccounts,
  getUpgradeContractAccounts,
  getWrappedMeta,
  mintToTokenId,
  NFT_TRANSFER_NATIVE_TOKEN_ADDRESS,
} from "../../../sdk/js/src/solana/nftBridge";
import {
  getForeignAssetSolana,
  getIsWrappedAssetSolana,
  getOriginalAssetSolana,
} from "../../../sdk/js/src/nft_bridge";
import { ChainId } from "../../../sdk/js/src";

describe("NFT Bridge", () => {
  const connection = new web3.Connection(LOCALHOST, "processed");

  const wallet = new NodeWallet(web3.Keypair.generate());

  // for signing wormhole messages
  const guardians = new MockGuardians(GUARDIAN_SET_INDEX + 1, GUARDIAN_KEYS);

  const erc721Token = makeErc721Token(
    WETH_ADDRESS,
    6969n,
    "Wetherean",
    "WETH",
    "https://ethereum.org/en/developers/tutorials/how-to-write-and-deploy-an-nft/"
  );

  const localVariables: any = {};

  before("Airdrop SOL", async () => {
    await connection
      .requestAirdrop(wallet.key(), 1000 * web3.LAMPORTS_PER_SOL)
      .then(async (signature) => connection.confirmTransaction(signature));
  });

  before("Create NFT", async () => {
    localVariables.mint = await createMint(
      connection,
      wallet.signer(),
      wallet.key(),
      null,
      0
    );

    localVariables.nftMeta = {
      name: "Space Cadet",
      symbol: "CADET",
      uri: "https://spl.solana.com/token#example-create-a-non-fungible-token",
    };

    const mint: web3.PublicKey = localVariables.mint;
    const name: string = localVariables.nftMeta.name;
    const symbol: string = localVariables.nftMeta.symbol;
    const uri: string = localVariables.nftMeta.uri;

    const accounts = {
      metadata: deriveTokenMetadataKey(mint),
      mint,
      mintAuthority: wallet.key(),
      payer: wallet.key(),
      updateAuthority: wallet.key(),
    };
    const args = {
      createMetadataAccountArgsV3: {
        data: {
          name,
          symbol,
          uri,
          sellerFeeBasisPoints: 0,
          creators: null,
          collection: null,
          uses: null,
        },
        isMutable: false,
        collectionDetails: null,
      },
    };
    const createMetadataIx = createCreateMetadataAccountV3Instruction(
      accounts,
      args,
      TOKEN_METADATA_PROGRAM_ID
    );

    const createMetadataTx = await web3.sendAndConfirmTransaction(
      connection,
      new web3.Transaction().add(createMetadataIx),
      [wallet.signer()]
    );
    // console.log("createMatadataTx", createMetadataTx);

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
      1
    );
  });

  describe("Accounts", () => {
    // for generating governance wormhole messages
    const governance = new GovernanceEmitter(
      GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex")
    );

    // nft bridge on Ethereum
    const ethereumNftBridge = new MockEthereumNftBridge(
      ETHEREUM_NFT_BRIDGE_ADDRESS
    );

    const payer = new web3.PublicKey(
      "6sbzC1eH4FTujJXWj51eQe25cYvr4xfXbJ1vAj7j2k5J"
    );

    it("Instruction 0: Initialize", () => {
      const accounts = getInitializeAccounts(NFT_BRIDGE_ADDRESS, payer);

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.config.toString()).to.equal(
        "J1oLBQPejgP75y9mKAAfftaQtmLhLkuQzbCufmYKMSQz"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
    });

    it("Instruction 1: Complete Native", () => {
      const timestamp = 12345678;
      const mint = NATIVE_MINT;
      const mintAta = getAssociatedTokenAddressSync(mint, payer);

      const nftMeta = localVariables.nftMeta;
      const tokenId = mintToTokenId(mint);
      const nonce = 420;
      const message = ethereumNftBridge.publishTransferNft(
        NFT_TRANSFER_NATIVE_TOKEN_ADDRESS.toString("hex"),
        1,
        nftMeta.name,
        nftMeta.symbol,
        tokenId,
        nftMeta.uri,
        1,
        mintAta.toBuffer().toString("hex"),
        nonce,
        timestamp
      );

      const signedVaa = guardians.addSignatures(
        message,
        [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18]
      );

      const accounts = getCompleteTransferNativeAccounts(
        NFT_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        signedVaa
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.config.toString()).to.equal(
        "J1oLBQPejgP75y9mKAAfftaQtmLhLkuQzbCufmYKMSQz"
      );
      expect(accounts.vaa.toString()).to.equal(
        "8b6y2t5NngJxhyicDkMQGWbrFagKEiiuGz2bqsVtf6Ks"
      );
      expect(accounts.claim.toString()).to.equal(
        "G3CvkGY9Pf7zMKmxxfN4UdaqDYcfQoKCoEZeCqh6ZQLR"
      );
      expect(accounts.endpoint.toString()).to.equal(
        "GGobvHkLNgwD7qnMRLFniLjoAtr12H4bqPD6AEHWzCou"
      );
      expect(accounts.to.equals(mintAta)).is.true;
      expect(accounts.toAuthority.equals(payer)).is.true;
      expect(accounts.custody.toString()).to.equal(
        "3ju9P66Ng9PEPhjY9HUDiC6taExZgWGULTERcP8RzT2j"
      );
      expect(accounts.mint.equals(mint)).is.true;
      expect(accounts.custodySigner.toString()).to.equal(
        "HHJPgZGoLrh8VmpmR4kPWmVoo8SZyAWQAVe15UtFJKQ1"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).is.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });

    it("Instruction 2: Complete Wrapped", () => {
      const timestamp = 23456789;
      const tokenAddress = ethAddressToBuffer(erc721Token.address);
      const tokenChain = ethereumNftBridge.chain;
      const tokenId = erc721Token.tokenId;
      const mint = deriveWrappedMintKey(
        NFT_BRIDGE_ADDRESS,
        tokenChain,
        tokenAddress,
        tokenId
      );
      const mintAta = getAssociatedTokenAddressSync(mint, payer);

      const name = erc721Token.name;
      const symbol = erc721Token.symbol;
      const uri = erc721Token.uri;

      const recipientChain = 1;
      const nonce = 420;
      const message = ethereumNftBridge.publishTransferNft(
        tokenAddress.toString("hex"),
        tokenChain,
        name,
        symbol,
        tokenId,
        uri,
        recipientChain,
        mintAta.toBuffer().toString("hex"),
        nonce,
        timestamp
      );

      const signedVaa = guardians.addSignatures(
        message,
        [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18]
      );

      const accounts = getCompleteTransferWrappedAccounts(
        NFT_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        signedVaa
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.config.toString()).to.equal(
        "J1oLBQPejgP75y9mKAAfftaQtmLhLkuQzbCufmYKMSQz"
      );
      expect(accounts.vaa.toString()).to.equal(
        "2bNynrs1qu3tRah8ztXG1eY6XaiP1xPXSDxYim8DGVsQ"
      );
      expect(accounts.claim.toString()).to.equal(
        "5svagh2zqoHkBewBq2KRKQDJHYHwUzLJNt2sfgpRWLuj"
      );
      expect(accounts.endpoint.toString()).to.equal(
        "GGobvHkLNgwD7qnMRLFniLjoAtr12H4bqPD6AEHWzCou"
      );
      expect(accounts.to.equals(mintAta)).is.true;
      expect(accounts.toAuthority.equals(payer)).is.true;
      expect(accounts.mint.equals(mint)).is.true;
      expect(accounts.wrappedMeta.toString()).to.equal(
        "GjxBVsD3fHPa3R97B4uA6TaMbuKXrQ9So8ktdNA6agXs"
      );
      expect(accounts.mintAuthority.toString()).to.equal(
        "ESeW7kNyP8mvfeqKkZdWkuFsKYeZnQUbrxYnNL8N11hi"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).is.true;
      expect(accounts.splMetadataProgram.equals(TOKEN_METADATA_PROGRAM_ID)).is
        .true;
      expect(
        accounts.associatedTokenProgram.equals(ASSOCIATED_TOKEN_PROGRAM_ID)
      ).is.true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });

    it("Instruction 3: Complete Wrapped Meta", () => {
      const timestamp = 34567890;
      const tokenAddress = ethAddressToBuffer(erc721Token.address);
      const tokenChain = ethereumNftBridge.chain;
      const tokenId = erc721Token.tokenId;
      const mint = deriveWrappedMintKey(
        NFT_BRIDGE_ADDRESS,
        tokenChain,
        tokenAddress,
        tokenId
      );
      const mintAta = getAssociatedTokenAddressSync(mint, payer);

      const name = erc721Token.name;
      const symbol = erc721Token.symbol;
      const uri = erc721Token.uri;

      const recipientChain = 1;
      const nonce = 420;
      const message = ethereumNftBridge.publishTransferNft(
        tokenAddress.toString("hex"),
        tokenChain,
        name,
        symbol,
        tokenId,
        uri,
        recipientChain,
        mintAta.toBuffer().toString("hex"),
        nonce,
        timestamp
      );

      const signedVaa = guardians.addSignatures(
        message,
        [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18]
      );

      const accounts = getCompleteWrappedMetaAccounts(
        NFT_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        signedVaa
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.config.toString()).to.equal(
        "J1oLBQPejgP75y9mKAAfftaQtmLhLkuQzbCufmYKMSQz"
      );
      expect(accounts.vaa.toString()).to.equal(
        "4ZxueqFJAomm96wiDcxgedvyGTGVSp8HMxCz731xUGFd"
      );
      expect(accounts.endpoint.toString()).to.equal(
        "GGobvHkLNgwD7qnMRLFniLjoAtr12H4bqPD6AEHWzCou"
      );
      expect(accounts.mint.equals(mint)).is.true;
      expect(accounts.wrappedMeta.toString()).to.equal(
        "GjxBVsD3fHPa3R97B4uA6TaMbuKXrQ9So8ktdNA6agXs"
      );
      expect(accounts.splMetadata.toString()).to.equal(
        "HgFdrXZJt1LzGuowL7KFP5JH7dgp2r22wRHijpgH4x9s"
      );
      expect(accounts.mintAuthority.toString()).to.equal(
        "ESeW7kNyP8mvfeqKkZdWkuFsKYeZnQUbrxYnNL8N11hi"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).is.true;
      expect(accounts.splMetadataProgram.equals(TOKEN_METADATA_PROGRAM_ID)).is
        .true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });

    it("Instruction 4: Transfer Wrapped", () => {
      const tokenAddress = ethAddressToBuffer(erc721Token.address);
      const tokenChain = ethereumNftBridge.chain;
      const tokenId = erc721Token.tokenId;
      const mint = deriveWrappedMintKey(
        NFT_BRIDGE_ADDRESS,
        tokenChain,
        tokenAddress,
        tokenId
      );
      const mintAta = getAssociatedTokenAddressSync(mint, payer);

      const message = web3.Keypair.generate();
      const accounts = getTransferWrappedAccounts(
        NFT_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        message.publicKey,
        mintAta,
        payer,
        tokenChain,
        tokenAddress,
        tokenId
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.config.toString()).to.equal(
        "J1oLBQPejgP75y9mKAAfftaQtmLhLkuQzbCufmYKMSQz"
      );
      expect(accounts.from.equals(mintAta)).is.true;
      expect(accounts.fromOwner.equals(payer)).is.true;
      expect(accounts.mint.equals(mint)).is.true;
      expect(accounts.wrappedMeta.toString()).to.equal(
        "GjxBVsD3fHPa3R97B4uA6TaMbuKXrQ9So8ktdNA6agXs"
      );
      expect(accounts.splMetadata.toString()).to.equal(
        "HgFdrXZJt1LzGuowL7KFP5JH7dgp2r22wRHijpgH4x9s"
      );
      expect(accounts.authoritySigner.toString()).to.equal(
        "9xMX62GupB5AhucZyD3oC6aBd1NHsBCA6e1x9fez1zHe"
      );
      expect(accounts.wormholeBridge.toString()).to.equal(
        "DNN2VhmrGTGj6QVnPz4NVfsiSk64cRHzKBLP5kUaQrf8"
      );
      expect(accounts.wormholeMessage.equals(message.publicKey)).is.true;
      expect(accounts.wormholeEmitter.toString()).to.equal(
        "6rbfWsCH5vFbwKAkaAvBSP1Mom6ZkCaCk2pGAbxWt1CH"
      );
      expect(accounts.wormholeSequence.toString()).to.equal(
        "J1hwmV7YfVygE96jVFgWRE8F1niWRQ9pkQDr61551XpG"
      );
      expect(accounts.wormholeFeeCollector.toString()).to.equal(
        "Cxt3Uka7X8vyHYjU6szcuYVPPFyg1fAtoeVy7eyzPjGV"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).is.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).is.true;
      expect(accounts.splMetadataProgram.equals(TOKEN_METADATA_PROGRAM_ID)).is
        .true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });

    it("Instruction 5: Transfer Native", () => {
      const mint = NATIVE_MINT;
      const mintAta = getAssociatedTokenAddressSync(mint, payer);

      const message = web3.Keypair.generate();
      const accounts = getTransferNativeAccounts(
        NFT_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        message.publicKey,
        mintAta,
        mint
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.config.toString()).to.equal(
        "J1oLBQPejgP75y9mKAAfftaQtmLhLkuQzbCufmYKMSQz"
      );
      expect(accounts.from.equals(mintAta)).is.true;
      expect(accounts.mint.equals(mint)).is.true;
      expect(accounts.splMetadata.toString()).to.equal(
        "6dM4TqWyWJsbx7obrdLcviBkTafD5E8av61zfU6jq57X"
      );
      expect(accounts.custody.toString()).to.equal(
        "3ju9P66Ng9PEPhjY9HUDiC6taExZgWGULTERcP8RzT2j"
      );
      expect(accounts.authoritySigner.toString()).to.equal(
        "9xMX62GupB5AhucZyD3oC6aBd1NHsBCA6e1x9fez1zHe"
      );
      expect(accounts.custodySigner.toString()).to.equal(
        "HHJPgZGoLrh8VmpmR4kPWmVoo8SZyAWQAVe15UtFJKQ1"
      );
      expect(accounts.wormholeBridge.toString()).to.equal(
        "DNN2VhmrGTGj6QVnPz4NVfsiSk64cRHzKBLP5kUaQrf8"
      );
      expect(accounts.wormholeMessage.equals(message.publicKey)).is.true;
      expect(accounts.wormholeEmitter.toString()).to.equal(
        "6rbfWsCH5vFbwKAkaAvBSP1Mom6ZkCaCk2pGAbxWt1CH"
      );
      expect(accounts.wormholeSequence.toString()).to.equal(
        "J1hwmV7YfVygE96jVFgWRE8F1niWRQ9pkQDr61551XpG"
      );
      expect(accounts.wormholeFeeCollector.toString()).to.equal(
        "Cxt3Uka7X8vyHYjU6szcuYVPPFyg1fAtoeVy7eyzPjGV"
      );
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).is.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.tokenProgram.equals(TOKEN_PROGRAM_ID)).is.true;
      expect(accounts.splMetadataProgram.equals(TOKEN_METADATA_PROGRAM_ID)).is
        .true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });

    it("Instruction 6: Register Chain", () => {
      const timestamp = 45678901;
      const message = governance.publishNftBridgeRegisterChain(
        timestamp,
        2,
        ETHEREUM_NFT_BRIDGE_ADDRESS
      );
      const signedVaa = guardians.addSignatures(
        message,
        [0, 1, 2, 3, 5, 7, 8, 9, 10, 12, 15, 16, 18]
      );

      const accounts = getRegisterChainAccounts(
        NFT_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        signedVaa
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.config.toString()).to.equal(
        "J1oLBQPejgP75y9mKAAfftaQtmLhLkuQzbCufmYKMSQz"
      );
      expect(accounts.endpoint.toString()).to.equal(
        "GGobvHkLNgwD7qnMRLFniLjoAtr12H4bqPD6AEHWzCou"
      );
      expect(accounts.vaa.toString()).to.equal(
        "DU2VB93gzJ7Qb8xskdHXF4u3nFoAHD4L5DE1kLtXBHCJ"
      );
      expect(accounts.claim.toString()).to.equal(
        "6QuJAFuXYpz8WvzbMoS41mFki5mdLQswhkxoccg2tmTS"
      );
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
      expect(accounts.wormholeProgram.equals(CORE_BRIDGE_ADDRESS)).is.true;
    });

    it("Instruction 7: Upgrade Contract", () => {
      const timestamp = 56789012;
      const chain = 1;
      const implementation = new web3.PublicKey(
        "2B5wMnErS8oKWV1wPTNQQhM1WLyxee2obtBMDtsYeHgA"
      );
      const message = governance.publishNftBridgeUpgradeContract(
        timestamp,
        chain,
        implementation.toString()
      );
      const signedVaa = guardians.addSignatures(message, [0]);

      const accounts = getUpgradeContractAccounts(
        NFT_BRIDGE_ADDRESS,
        CORE_BRIDGE_ADDRESS,
        payer,
        signedVaa
      );

      // verify accounts
      expect(accounts.payer.equals(payer)).is.true;
      expect(accounts.vaa.toString()).to.equal(
        "Evar3arhnjy84wPDUpKPifxFRT9oRJFzwYZAVUKpsTnd"
      );
      expect(accounts.claim.toString()).to.equal(
        "3gHw5uPbhk1dDoCUxtK5VosaqTgV9H7wbNJsttswB4At"
      );
      expect(accounts.upgradeAuthority.toString()).to.equal(
        "8GUsAHTGAjJv5XgdQrHprwVnzqARbWaxZ3GHvquhZhpp"
      );
      expect(accounts.spill.equals(payer)).is.true;
      expect(accounts.implementation.equals(implementation)).is.true;
      expect(accounts.programData.toString()).to.equal(
        "2oC7qvaYxBLg1msKS8rPgEab5VSQ7TUFfhJBv75BzdSS"
      );
      expect(accounts.nftBridgeProgram.equals(NFT_BRIDGE_ADDRESS)).is.true;
      expect(accounts.rent.equals(web3.SYSVAR_RENT_PUBKEY)).is.true;
      expect(accounts.clock.equals(web3.SYSVAR_CLOCK_PUBKEY)).is.true;
      expect(
        accounts.bpfLoaderUpgradeable.equals(BpfLoaderUpgradeable.programId)
      ).is.true;
      expect(accounts.systemProgram.equals(web3.SystemProgram.programId)).to.be
        .true;
    });
  });

  describe("NFT Bridge Program Interaction", () => {
    // for generating governance wormhole messages
    const governance = new GovernanceEmitter(
      GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
      30
    );

    // nft bridge on Ethereum
    const ethereumNftBridge = new MockEthereumNftBridge(
      ETHEREUM_NFT_BRIDGE_ADDRESS
    );

    describe("Setup NFT Bridge", () => {
      it("Register Ethereum NFT Bridge", async () => {
        const timestamp = now();
        const message = governance.publishNftBridgeRegisterChain(
          timestamp,
          2,
          ETHEREUM_NFT_BRIDGE_ADDRESS
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
              NFT_BRIDGE_ADDRESS,
              CORE_BRIDGE_ADDRESS,
              wallet.key(),
              signedVaa
            )
          ),
          [wallet.signer()]
        );
        // console.log(`registerChainTx: ${registerChainTx}`);

        // verify data
        const parsed = parseNftBridgeRegisterChainVaa(signedVaa);
        const endpoint = deriveEndpointKey(
          NFT_BRIDGE_ADDRESS,
          parsed.foreignChain,
          parsed.foreignAddress
        );
        const endpointRegistration = await getEndpointRegistration(
          connection,
          endpoint
        );
        expect(endpointRegistration.chain).to.equal(2);
        const expectedEmitter = ethAddressToBuffer(ETHEREUM_NFT_BRIDGE_ADDRESS);
        expect(
          Buffer.compare(endpointRegistration.contract, expectedEmitter)
        ).to.equal(0);
      });
    });

    describe("Native Token Handling", () => {
      it("Send NFT", async () => {
        const mint: web3.PublicKey = localVariables.mint;
        const mintAta: web3.PublicKey = localVariables.mintAta;
        const custodyAccount = deriveCustodyKey(NFT_BRIDGE_ADDRESS, mint);

        const walletBalanceBefore = await getAccount(connection, mintAta).then(
          (account) => account.amount
        );
        const custodyBalanceBefore = 0n;

        const nonce = 69;
        const targetAddress = Buffer.alloc(32, "deadbeef", "hex");
        const targetChain = 2;

        const approveIx = createApproveAuthoritySignerInstruction(
          NFT_BRIDGE_ADDRESS,
          mintAta,
          wallet.key()
        );

        const message = web3.Keypair.generate();
        const transferNativeIx = createTransferNativeInstruction(
          NFT_BRIDGE_ADDRESS,
          CORE_BRIDGE_ADDRESS,
          wallet.key(),
          message.publicKey,
          mintAta,
          mint,
          nonce,
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
        expect(walletBalanceBefore - walletBalanceAfter).to.equal(1n);
        expect(custodyBalanceAfter - custodyBalanceBefore).to.equal(1n);

        // verify data
        const messageData = await getPostedMessage(
          connection,
          message.publicKey
        ).then((posted) => posted.message);
        expect(messageData.consistencyLevel).to.equal(32);
        expect(
          Buffer.compare(
            messageData.emitterAddress,
            deriveWormholeEmitterKey(NFT_BRIDGE_ADDRESS).toBuffer()
          )
        ).to.equal(0);
        expect(messageData.emitterChain).to.equal(1);
        expect(messageData.nonce).to.equal(nonce);
        expect(messageData.sequence).to.equal(0n);
        expect(messageData.vaaTime).to.equal(0);
        expect(messageData.vaaSignatureAccount.equals(web3.PublicKey.default))
          .is.true;
        expect(messageData.vaaVersion).to.equal(0);

        const nftTransfer = parseNftTransferPayload(messageData.payload);
        const nftMeta = localVariables.nftMeta;
        expect(nftTransfer.payloadType).to.equal(1);
        expect(
          Buffer.compare(
            nftTransfer.tokenAddress,
            NFT_TRANSFER_NATIVE_TOKEN_ADDRESS
          )
        ).to.equal(0);
        expect(nftTransfer.tokenChain).to.equal(1);
        expect(nftTransfer.name).to.equal(nftMeta.name);
        expect(nftTransfer.symbol).to.equal(nftMeta.symbol);
        expect(nftTransfer.tokenId).to.equal(mintToTokenId(mint));
        const expectedUri = Buffer.alloc(200);
        expectedUri.write(nftMeta.uri, 0);
        expect(nftTransfer.uri).to.equal(expectedUri.toString());
        expect(Buffer.compare(nftTransfer.to, targetAddress)).to.equal(0);
        expect(nftTransfer.toChain).to.equal(targetChain);
      });

      it("Receive NFT", async () => {
        const mint: web3.PublicKey = localVariables.mint;
        const mintAta: web3.PublicKey = localVariables.mintAta;
        const custodyAccount = deriveCustodyKey(NFT_BRIDGE_ADDRESS, mint);
        const walletBalanceBefore = await getAccount(connection, mintAta).then(
          (account) => account.amount
        );
        const custodyBalanceBefore = await getAccount(
          connection,
          custodyAccount
        ).then((account) => account.amount);

        const metadata = await Metadata.fromAccountAddress(
          connection,
          deriveTokenMetadataKey(mint)
        );

        const tokenChain = 1;
        const tokenId = mintToTokenId(mint);
        const recipientChain = 1;
        const nonce = 420;
        const message = ethereumNftBridge.publishTransferNft(
          NFT_TRANSFER_NATIVE_TOKEN_ADDRESS.toString("hex"),
          tokenChain,
          metadata.data.name,
          metadata.data.symbol,
          tokenId,
          metadata.data.uri,
          recipientChain,
          mintAta.toBuffer().toString("hex"),
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
            NFT_BRIDGE_ADDRESS,
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
        expect(walletBalanceAfter - walletBalanceBefore).to.equal(1n);
        expect(custodyBalanceBefore - custodyBalanceAfter).to.equal(1n);

        // verify data
        const messageData = await getPostedVaa(
          connection,
          CORE_BRIDGE_ADDRESS,
          parseVaa(signedVaa).hash
        ).then((posted) => posted.message);
        expect(messageData.consistencyLevel).to.equal(
          ethereumNftBridge.consistencyLevel
        );
        expect(
          Buffer.compare(
            messageData.emitterAddress,
            ethAddressToBuffer(ETHEREUM_NFT_BRIDGE_ADDRESS)
          )
        ).to.equal(0);
        expect(messageData.emitterChain).to.equal(ethereumNftBridge.chain);
        expect(messageData.nonce).to.equal(nonce);
        expect(messageData.sequence).to.equal(1n);
        expect(messageData.vaaTime).to.equal(0);
        expect(messageData.vaaVersion).to.equal(1);
        expect(
          Buffer.compare(parseVaa(signedVaa).payload, messageData.payload)
        ).to.equal(0);

        const nftTransfer = parseNftTransferPayload(messageData.payload);
        const nftMeta = localVariables.nftMeta;
        expect(nftTransfer.payloadType).to.equal(1);
        expect(
          Buffer.compare(
            nftTransfer.tokenAddress,
            NFT_TRANSFER_NATIVE_TOKEN_ADDRESS
          )
        ).to.equal(0);
        expect(nftTransfer.tokenChain).to.equal(tokenChain);
        expect(nftTransfer.name).to.equal(nftMeta.name);
        expect(nftTransfer.symbol).to.equal(nftMeta.symbol);
        expect(nftTransfer.tokenId).to.equal(mintToTokenId(mint));
        const expectedUri = Buffer.alloc(200);
        expectedUri.write(nftMeta.uri, 0);
        expect(nftTransfer.uri).to.equal(expectedUri.toString());
        expect(Buffer.compare(nftTransfer.to, mintAta.toBuffer())).to.equal(0);
        expect(nftTransfer.toChain).to.equal(recipientChain);
      });
    });

    describe("NFT Bridge Wrapped Token Handling", () => {
      it("Receive NFT and Create Metadata", async () => {
        const tokenAddress = ethAddressToBuffer(erc721Token.address);
        const tokenChain = ethereumNftBridge.chain;
        const tokenId = erc721Token.tokenId;
        const mint = deriveWrappedMintKey(
          NFT_BRIDGE_ADDRESS,
          tokenChain,
          tokenAddress,
          tokenId
        );
        const mintAta = getAssociatedTokenAddressSync(mint, wallet.key());

        const name = erc721Token.name;
        const symbol = erc721Token.symbol;
        const uri = erc721Token.uri;

        // token account and mint don't exist yet, so there is no balance
        const walletBalanceBefore = 0n;
        const supplyBefore = 0n;

        const recipientChain = 1;
        const nonce = 420;
        const message = ethereumNftBridge.publishTransferNft(
          tokenAddress.toString("hex"),
          tokenChain,
          name,
          symbol,
          tokenId,
          uri,
          recipientChain,
          mintAta.toBuffer().toString("hex"),
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

        const completeWrappedTransferIx =
          createCompleteTransferWrappedInstruction(
            NFT_BRIDGE_ADDRESS,
            CORE_BRIDGE_ADDRESS,
            wallet.key(),
            signedVaa
          );

        const completeWrappedTransferTx = await web3.sendAndConfirmTransaction(
          connection,
          new web3.Transaction().add(completeWrappedTransferIx),
          [wallet.signer()]
        );
        // console.log(`completeWrappedTransferTx: ${completeWrappedTransferTx}`);

        const walletBalanceAfter = await getAccount(connection, mintAta).then(
          (account) => account.amount
        );
        const supplyAfter = await getMint(connection, mint).then(
          (info) => info.supply
        );

        // check balance changes
        expect(walletBalanceAfter - walletBalanceBefore).to.equal(1n);
        expect(supplyAfter - supplyBefore).to.equal(1n);

        // we need a separate transaction to execute complete_wrapped_meta instruction
        // following complete_wrapped because... ???
        const completeWrappedMetaIx = createCompleteWrappedMetaInstruction(
          NFT_BRIDGE_ADDRESS,
          CORE_BRIDGE_ADDRESS,
          wallet.key(),
          signedVaa
        );

        const completeWrappedMetaTx = await web3.sendAndConfirmTransaction(
          connection,
          new web3.Transaction().add(completeWrappedMetaIx),
          [wallet.signer()]
        );
        // console.log(`completeWrappedMetaTx:     ${completeWrappedMetaTx}`);

        // verify data
        const messageData = await getPostedVaa(
          connection,
          CORE_BRIDGE_ADDRESS,
          parseVaa(signedVaa).hash
        ).then((posted) => posted.message);
        expect(messageData.consistencyLevel).to.equal(
          ethereumNftBridge.consistencyLevel
        );
        expect(
          Buffer.compare(
            messageData.emitterAddress,
            ethAddressToBuffer(ETHEREUM_NFT_BRIDGE_ADDRESS)
          )
        ).to.equal(0);
        expect(messageData.emitterChain).to.equal(ethereumNftBridge.chain);
        expect(messageData.nonce).to.equal(nonce);
        expect(messageData.sequence).to.equal(2n);
        expect(messageData.vaaTime).to.equal(0);
        expect(messageData.vaaVersion).to.equal(1);
        expect(
          Buffer.compare(parseVaa(signedVaa).payload, messageData.payload)
        ).to.equal(0);

        const nftTransfer = parseNftTransferPayload(messageData.payload);
        expect(nftTransfer.payloadType).to.equal(1);
        expect(Buffer.compare(nftTransfer.tokenAddress, tokenAddress)).to.equal(
          0
        );
        expect(nftTransfer.tokenChain).to.equal(tokenChain);
        expect(nftTransfer.name).to.equal(name);
        expect(nftTransfer.symbol).to.equal(symbol);
        expect(nftTransfer.tokenId).to.equal(tokenId);
        expect(nftTransfer.uri).to.equal(uri);
        expect(Buffer.compare(nftTransfer.to, mintAta.toBuffer())).to.equal(0);
        expect(nftTransfer.toChain).to.equal(recipientChain);

        // check wrapped meta
        const wrappedMeta = await getWrappedMeta(
          connection,
          NFT_BRIDGE_ADDRESS,
          mint
        );
        expect(wrappedMeta.chain).to.equal(tokenChain);
        expect(Buffer.compare(wrappedMeta.tokenAddress, tokenAddress)).to.equal(
          0
        );
        expect(wrappedMeta.tokenId).to.equal(tokenId);
      });

      it("Send NFT", async () => {
        const tokenAddress = ethAddressToBuffer(erc721Token.address);
        const tokenChain = ethereumNftBridge.chain;
        const tokenId = erc721Token.tokenId;
        const mint = deriveWrappedMintKey(
          NFT_BRIDGE_ADDRESS,
          tokenChain,
          tokenAddress,
          tokenId
        );
        const mintAta = getAssociatedTokenAddressSync(mint, wallet.key());

        const walletBalanceBefore = await getAccount(connection, mintAta).then(
          (account) => account.amount
        );
        const supplyBefore = await getMint(connection, mint).then(
          (info) => info.supply
        );

        const nonce = 69;
        const targetAddress = Buffer.alloc(32, "deadbeef", "hex");
        const targetChain = 2;

        const approveIx = createApproveAuthoritySignerInstruction(
          NFT_BRIDGE_ADDRESS,
          mintAta,
          wallet.key()
        );

        const message = web3.Keypair.generate();
        const transferWrappedIx = createTransferWrappedInstruction(
          NFT_BRIDGE_ADDRESS,
          CORE_BRIDGE_ADDRESS,
          wallet.key(),
          message.publicKey,
          mintAta,
          wallet.key(),
          tokenChain,
          tokenAddress,
          tokenId,
          nonce,
          targetAddress,
          targetChain
        );

        const approveAndTransferTx = await web3.sendAndConfirmTransaction(
          connection,
          new web3.Transaction().add(approveIx, transferWrappedIx),
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
        expect(walletBalanceBefore - walletBalanceAfter).to.equal(1n);
        expect(supplyBefore - supplyAfter).to.equal(1n);

        // verify data
        const messageData = await getPostedMessage(
          connection,
          message.publicKey
        ).then((posted) => posted.message);
        expect(messageData.consistencyLevel).to.equal(32);
        expect(
          Buffer.compare(
            messageData.emitterAddress,
            deriveWormholeEmitterKey(NFT_BRIDGE_ADDRESS).toBuffer()
          )
        ).to.equal(0);
        expect(messageData.emitterChain).to.equal(1);
        expect(messageData.nonce).to.equal(nonce);
        expect(messageData.sequence).to.equal(1n);
        expect(messageData.vaaTime).to.equal(0);
        expect(messageData.vaaSignatureAccount.equals(web3.PublicKey.default))
          .is.true;
        expect(messageData.vaaVersion).to.equal(0);

        const nftTransfer = parseNftTransferPayload(messageData.payload);
        expect(nftTransfer.payloadType).to.equal(1);
        expect(Buffer.compare(nftTransfer.tokenAddress, tokenAddress)).to.equal(
          0
        );
        expect(nftTransfer.tokenChain).to.equal(tokenChain);
        expect(nftTransfer.name).to.equal(erc721Token.name);
        expect(nftTransfer.symbol).to.equal(erc721Token.symbol);
        expect(nftTransfer.tokenId).to.equal(tokenId);
        // bridge does this cool thing of adding padding to the uri when being
        // transferred out when it's wrapped
        const expectedUri = Buffer.alloc(200);
        expectedUri.write(erc721Token.uri, 0);
        expect(nftTransfer.uri).to.equal(expectedUri.toString());
        expect(Buffer.compare(nftTransfer.to, targetAddress)).to.equal(0);
        expect(nftTransfer.toChain).to.equal(targetChain);
      });
    });
  });

  describe("Asset Queries", () => {
    // nft bridge on Ethereum
    const ethereumNftBridge = new MockEthereumNftBridge(
      ETHEREUM_NFT_BRIDGE_ADDRESS
    );

    describe("getOriginalAssetSolana", () => {
      it("Non-existent NFT", async () => {
        const mint = "wot m8?";
        const asset = await getOriginalAssetSolana(
          connection,
          NFT_BRIDGE_ADDRESS,
          mint
        );

        // verify results
        expect(asset.isWrapped).to.be.false;
        expect(asset.chainId).to.equal(1);
        expect(
          Buffer.compare(Buffer.from(asset.assetAddress), Buffer.alloc(32))
        ).to.equal(0);
        expect(asset.tokenId).is.undefined;
      });

      it("Native NFT", async () => {
        const mint = localVariables.mint;

        const asset = await getOriginalAssetSolana(
          connection,
          NFT_BRIDGE_ADDRESS,
          mint
        );

        // verify results
        expect(asset.isWrapped).to.be.false;
        expect(asset.chainId).to.equal(1);
        expect(
          Buffer.compare(Buffer.from(asset.assetAddress), mint.toBuffer())
        ).to.equal(0);
        expect(asset.tokenId).is.undefined;
      });

      it("Wrapped NFT", async () => {
        const tokenAddress = ethAddressToBuffer(erc721Token.address);
        const tokenChain = ethereumNftBridge.chain;
        const tokenId = erc721Token.tokenId;
        const mint = deriveWrappedMintKey(
          NFT_BRIDGE_ADDRESS,
          tokenChain,
          tokenAddress,
          tokenId
        );

        const asset = await getOriginalAssetSolana(
          connection,
          NFT_BRIDGE_ADDRESS,
          mint
        );

        // verify results
        expect(asset.isWrapped).is.true;
        expect(asset.chainId).to.equal(tokenChain);
        expect(
          Buffer.compare(Buffer.from(asset.assetAddress), tokenAddress)
        ).to.equal(0);
        expect(asset.tokenId).to.equal(tokenId.toString());
      });
    });

    describe("getForeignAssetSolana", () => {
      it("Wrapped NFT", async () => {
        const tokenAddress = ethAddressToBuffer(erc721Token.address);
        const tokenChain = ethereumNftBridge.chain;
        const tokenId = erc721Token.tokenId;

        const asset = await getForeignAssetSolana(
          NFT_BRIDGE_ADDRESS,
          tokenChain as ChainId,
          tokenAddress,
          tokenId
        );

        // verify results
        expect(asset).to.equal("GrWwR7tTfJvCLuNbcHAyhqBhuuCr8kG7xg47x47GshF4");
      });
    });

    describe("getIsWrappedAsset", () => {
      it("Non-existent NFT", async () => {
        const mint = null;

        const isWrapped = await getIsWrappedAssetSolana(
          connection,
          NFT_BRIDGE_ADDRESS,
          // @ts-ignore
          mint
        );

        // verify results
        expect(isWrapped).to.be.false;
      });

      it("Native NFT", async () => {
        const mint = localVariables.mint;

        const isWrapped = await getIsWrappedAssetSolana(
          connection,
          NFT_BRIDGE_ADDRESS,
          mint
        );

        // verify results
        expect(isWrapped).to.be.false;
      });

      it("Wrapped NFT", async () => {
        const tokenAddress = ethAddressToBuffer(erc721Token.address);
        const tokenChain = ethereumNftBridge.chain;
        const tokenId = erc721Token.tokenId;
        const mint = deriveWrappedMintKey(
          NFT_BRIDGE_ADDRESS,
          tokenChain,
          tokenAddress,
          tokenId
        );

        const isWrapped = await getIsWrappedAssetSolana(
          connection,
          NFT_BRIDGE_ADDRESS,
          mint
        );

        // verify results
        expect(isWrapped).is.true;
      });
    });
  });
});
