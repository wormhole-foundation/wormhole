import * as anchor from "@coral-xyz/anchor";
import { getOrCreateAssociatedTokenAccount } from "@solana/spl-token";
import {
  MINT_INFO_WRAPPED_7,
  MINT_INFO_WRAPPED_8,
  MINT_INFO_WRAPPED_MAX_ONE,
  WrappedMintInfo,
  expectIxOkDetails,
  getTokenBalances,
} from "../helpers";
import * as tokenBridge from "../helpers/tokenBridge";
import * as coreBridge from "../helpers/coreBridge";
import { PublicKey } from "@solana/web3.js";

describe("Token Bridge -- Legacy Instruction: Transfer Tokens (Wrapped)", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = tokenBridge.getAnchorProgram(connection, tokenBridge.localnet());
  const wormholeProgram = coreBridge.getAnchorProgram(connection, coreBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;

  const forkedProgram = tokenBridge.getAnchorProgram(connection, tokenBridge.mainnet());

  const wrappedMints: WrappedMintInfo[] = [MINT_INFO_WRAPPED_8, MINT_INFO_WRAPPED_7];
  const wrappedMaxMint: WrappedMintInfo = MINT_INFO_WRAPPED_MAX_ONE;

  describe("Ok", () => {
    for (const { chain, decimals, address } of wrappedMints) {
      it(`Invoke \`transfer_tokens_wrapped\` (${decimals} Decimals)`, async () => {
        const [mint, forkMint] = [program, forkedProgram].map((program) =>
          tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
        );

        // Fetch recipient token account, these accounts should've been created in other tests.
        const [srcToken, forkSrcToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, payer.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, payer.publicKey),
        ]);

        // Fetch balance before the outbound transfer.
        const balancesBefore = await getTokenBalances(
          program,
          forkedProgram,
          srcToken.address,
          forkSrcToken.address
        );

        // Transfer params.
        const amount = new anchor.BN("88888888");
        const relayerFee = new anchor.BN("11111111");

        const [coreMessage, txDetails, forkCoreMessage, forkTxDetails] = await parallelTxDetails(
          program,
          forkedProgram,
          {
            payer: payer.publicKey,
            wrappedMint: mint,
            forkWrappedMint: forkMint,
            srcToken: srcToken.address,
            forkSrcToken: forkSrcToken.address,
            srcOwner: payer.publicKey,
          },
          defaultArgs(amount, relayerFee),
          payer
        );

        await tokenBridge.expectCorrectWrappedTokenBalanceChanges(
          connection,
          srcToken.address,
          forkSrcToken.address,
          balancesBefore,
          tokenBridge.TransferDirection.Out,
          BigInt(amount.toString())
        );

        // TODO: Check that the core messages are correct.
      });

      it(`Invoke \`transfer_tokens_wrapped\` (${decimals} Decimals, Minimum Transfer Amount)`, async () => {
        const [mint, forkMint] = [program, forkedProgram].map((program) =>
          tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
        );

        // Fetch recipient token account, these accounts should've been created in other tests.
        const [srcToken, forkSrcToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, payer.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, payer.publicKey),
        ]);

        // Fetch balance before the outbound transfer.
        const balancesBefore = await getTokenBalances(
          program,
          forkedProgram,
          srcToken.address,
          forkSrcToken.address
        );

        // Transfer params.
        const amount = new anchor.BN("1");
        const relayerFee = new anchor.BN("0");

        const [coreMessage, txDetails, forkCoreMessage, forkTxDetails] = await parallelTxDetails(
          program,
          forkedProgram,
          {
            payer: payer.publicKey,
            wrappedMint: mint,
            forkWrappedMint: forkMint,
            srcToken: srcToken.address,
            forkSrcToken: forkSrcToken.address,
            srcOwner: payer.publicKey,
          },
          defaultArgs(amount, relayerFee),
          payer
        );

        await tokenBridge.expectCorrectWrappedTokenBalanceChanges(
          connection,
          srcToken.address,
          forkSrcToken.address,
          balancesBefore,
          tokenBridge.TransferDirection.Out,
          BigInt(amount.toString())
        );

        // TODO: Check that the core messages are correct.
      });
    }

    it(`Invoke \`transfer_tokens_wrapped\` (8 Decimals, Max Transfer Amount)`, async () => {
      // Fetch special mint for this test.
      const { chain, address } = wrappedMaxMint;

      const [mint, forkMint] = [program, forkedProgram].map((program) =>
        tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
      );

      // Fetch recipient token account, these accounts should've been created in other tests.
      const [srcToken, forkSrcToken] = await Promise.all([
        getOrCreateAssociatedTokenAccount(connection, payer, mint, payer.publicKey),
        getOrCreateAssociatedTokenAccount(connection, payer, forkMint, payer.publicKey),
      ]);

      // Fetch balance before the outbound transfer.
      const balancesBefore = await getTokenBalances(
        program,
        forkedProgram,
        srcToken.address,
        forkSrcToken.address
      );

      // Transfer params.
      const amount = new anchor.BN(
        Buffer.alloc(8, "ffffffff", "hex").readBigUInt64BE().toString()
      ).subn(1);
      const relayerFee = new anchor.BN("11111111");

      const [coreMessage, txDetails, forkCoreMessage, forkTxDetails] = await parallelTxDetails(
        program,
        forkedProgram,
        {
          payer: payer.publicKey,
          wrappedMint: mint,
          forkWrappedMint: forkMint,
          srcToken: srcToken.address,
          forkSrcToken: forkSrcToken.address,
          srcOwner: payer.publicKey,
        },
        defaultArgs(amount, relayerFee),
        payer
      );

      await tokenBridge.expectCorrectWrappedTokenBalanceChanges(
        connection,
        srcToken.address,
        forkSrcToken.address,
        balancesBefore,
        tokenBridge.TransferDirection.Out,
        BigInt(amount.toString())
      );

      // TODO: Check that the core messages are correct.
    });
  });
});

function defaultArgs(amount: anchor.BN, relayerFee: anchor.BN) {
  return {
    nonce: 420,
    amount,
    relayerFee,
    recipient: Array.from(Buffer.alloc(32, "deadbeef", "hex")),
    recipientChain: 2,
  };
}

async function parallelTxDetails(
  program: tokenBridge.TokenBridgeProgram,
  forkedProgram: tokenBridge.TokenBridgeProgram,
  accounts: {
    payer: PublicKey;
    wrappedMint: PublicKey;
    forkWrappedMint: PublicKey;
    srcToken: PublicKey;
    forkSrcToken: PublicKey;
    srcOwner: PublicKey;
  },
  args: tokenBridge.LegacyTransferTokensArgs,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;

  // Accounts and args.
  const { payer: owner, wrappedMint, forkWrappedMint, srcToken, forkSrcToken, srcOwner } = accounts;
  const { amount } = args;
  const coreMessage = anchor.web3.Keypair.generate();

  // Approve the transfer.
  const approveIx = tokenBridge.approveTransferAuthorityIx(program, srcToken, owner, amount);
  const ix = await tokenBridge.legacyTransferTokensWrappedIx(
    program,
    {
      coreMessage: coreMessage.publicKey,
      ...{
        payer: owner,
        wrappedMint,
        srcToken,
        srcOwner,
      },
    },
    args
  );

  // Approve the forked transfer.
  const forkCoreMessage = anchor.web3.Keypair.generate();
  const forkedApproveIx = tokenBridge.approveTransferAuthorityIx(
    forkedProgram,
    forkSrcToken,
    owner,
    amount
  );
  const forkedIx = await tokenBridge.legacyTransferTokensWrappedIx(
    forkedProgram,
    {
      coreMessage: forkCoreMessage.publicKey,
      ...{
        payer: owner,
        wrappedMint: forkWrappedMint,
        srcToken: forkSrcToken,
        srcOwner,
      },
    },
    args
  );

  const [txDetails, forkTxDetails] = await Promise.all([
    expectIxOkDetails(connection, [approveIx, ix], [payer, coreMessage]),
    expectIxOkDetails(connection, [forkedApproveIx, forkedIx], [payer, forkCoreMessage]),
  ]);

  return [coreMessage, txDetails, forkCoreMessage, forkTxDetails];
}