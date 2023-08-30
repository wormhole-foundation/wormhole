import * as anchor from "@coral-xyz/anchor";
import { getOrCreateAssociatedTokenAccount } from "@solana/spl-token";
import { PublicKey } from "@solana/web3.js";
import {
  WrappedMintInfo,
  WRAPPED_MINT_INFO_7,
  WRAPPED_MINT_INFO_8,
  WRAPPED_MINT_INFO_MAX_TWO,
  expectIxOkDetails,
  getTokenBalances,
} from "../helpers";
import * as tokenBridge from "../helpers/tokenBridge";

describe("Token Bridge -- Legacy Instruction: Transfer Tokens with Payload (Wrapped)", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = tokenBridge.getAnchorProgram(connection, tokenBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;

  const forkedProgram = tokenBridge.getAnchorProgram(connection, tokenBridge.mainnet());

  const wrappedMints: WrappedMintInfo[] = [WRAPPED_MINT_INFO_8, WRAPPED_MINT_INFO_7];
  const wrappedMaxMint: WrappedMintInfo = WRAPPED_MINT_INFO_MAX_TWO;

  describe("Ok", () => {
    const transferAuthority = anchor.web3.Keypair.generate();

    for (const { chain, decimals, address } of wrappedMints) {
      it(`Invoke \`transfer_tokens_with_payload_wrapped\` (${decimals} Decimals)`, async () => {
        const [mint, forkMint] = [program, forkedProgram].map((program) =>
          tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
        );

        // Fetch recipient token account, these accounts should've been created in other tests.
        const [payerToken, forkPayerToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, payer.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, payer.publicKey),
        ]);

        // Fetch balance before the outbound transfer.
        const balancesBefore = await getTokenBalances(
          program,
          forkedProgram,
          payerToken.address,
          forkPayerToken.address
        );

        // Transfer amount.
        const amount = new anchor.BN("88888888");

        // Invoke the instruction.
        const [coreMessage, txDetails, forkCoreMessage, forkTxDetails] = await parallelTxDetails(
          program,
          forkedProgram,
          {
            payer: payer.publicKey,
            wrappedMint: mint,
            forkWrappedMint: forkMint,
            srcToken: payerToken.address,
            forkSrcToken: forkPayerToken.address,
            srcOwner: payerToken.owner, // Payer owns both token accounts.
          },
          defaultArgs(amount),
          payer,
          transferAuthority
        );

        await tokenBridge.expectCorrectWrappedTokenBalanceChanges(
          connection,
          payerToken.address,
          forkPayerToken.address,
          balancesBefore,
          tokenBridge.TransferDirection.Out,
          BigInt(amount.toString())
        );

        // TODO: Check that the core messages are correct.
      });

      it(`Invoke \`transfer_tokens_with_payload_wrapped\` (${decimals} Decimals, Minimum Transfer Amount)`, async () => {
        const [mint, forkMint] = [program, forkedProgram].map((program) =>
          tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
        );

        // Fetch recipient token account, these accounts should've been created in other tests.
        const [payerToken, forkPayerToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, payer.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, payer.publicKey),
        ]);

        // Fetch balance before the outbound transfer.
        const balancesBefore = await getTokenBalances(
          program,
          forkedProgram,
          payerToken.address,
          forkPayerToken.address
        );

        // Transfer amount.
        const amount = new anchor.BN("1");

        // Invoke the instruction.
        const [coreMessage, txDetails, forkCoreMessage, forkTxDetails] = await parallelTxDetails(
          program,
          forkedProgram,
          {
            payer: payer.publicKey,
            wrappedMint: mint,
            forkWrappedMint: forkMint,
            srcToken: payerToken.address,
            forkSrcToken: forkPayerToken.address,
            srcOwner: payerToken.owner, // Payer owns both token accounts.
          },
          defaultArgs(amount),
          payer,
          transferAuthority
        );

        await tokenBridge.expectCorrectWrappedTokenBalanceChanges(
          connection,
          payerToken.address,
          forkPayerToken.address,
          balancesBefore,
          tokenBridge.TransferDirection.Out,
          BigInt(amount.toString())
        );

        // TODO: Check that the core messages are correct.
      });
    }

    it(`Invoke \`transfer_tokens_with_payload_wrapped\` (8 Decimals)`, async () => {
      // Fetch special mint for this test.
      const { chain, address } = wrappedMaxMint;

      const [mint, forkMint] = [program, forkedProgram].map((program) =>
        tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
      );

      // Fetch recipient token account, these accounts should've been created in other tests.
      const [payerToken, forkPayerToken] = await Promise.all([
        getOrCreateAssociatedTokenAccount(connection, payer, mint, payer.publicKey),
        getOrCreateAssociatedTokenAccount(connection, payer, forkMint, payer.publicKey),
      ]);

      // Fetch balance before the outbound transfer.
      const balancesBefore = await getTokenBalances(
        program,
        forkedProgram,
        payerToken.address,
        forkPayerToken.address
      );

      // Transfer amount.
      const amount = new anchor.BN(
        Buffer.alloc(8, "ffffffff", "hex").readBigUInt64BE().toString()
      ).subn(1);

      // Invoke the instruction.
      const [coreMessage, txDetails, forkCoreMessage, forkTxDetails] = await parallelTxDetails(
        program,
        forkedProgram,
        {
          payer: payer.publicKey,
          wrappedMint: mint,
          forkWrappedMint: forkMint,
          srcToken: payerToken.address,
          forkSrcToken: forkPayerToken.address,
          srcOwner: payerToken.owner, // Payer owns both token accounts.
        },
        defaultArgs(amount),
        payer,
        transferAuthority
      );

      await tokenBridge.expectCorrectWrappedTokenBalanceChanges(
        connection,
        payerToken.address,
        forkPayerToken.address,
        balancesBefore,
        tokenBridge.TransferDirection.Out,
        BigInt(amount.toString())
      );

      // TODO: Check that the core messages are correct.
    });
  });
});

function defaultArgs(amount: anchor.BN) {
  return {
    nonce: 420,
    amount,
    redeemer: Array.from(Buffer.alloc(32, "deadbeef", "hex")),
    redeemerChain: 2,
    payload: Buffer.from("All your base are belong to us."),
    cpiProgramId: null,
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
  args: tokenBridge.LegacyTransferTokensWithPayloadArgs,
  payer: anchor.web3.Keypair,
  senderAuthority: anchor.web3.Keypair
) {
  const connection = program.provider.connection;

  // Accounts and args.
  const { payer: owner, wrappedMint, forkWrappedMint, srcToken, forkSrcToken, srcOwner } = accounts;
  const { amount } = args;
  const coreMessage = anchor.web3.Keypair.generate();

  // Approve the transfer.
  const approveIx = tokenBridge.approveTransferAuthorityIx(program, srcToken, owner, amount);
  const ix = await tokenBridge.legacyTransferTokensWithPayloadWrappedIx(
    program,
    {
      coreMessage: coreMessage.publicKey,
      senderAuthority: senderAuthority.publicKey,
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
  const forkedIx = await tokenBridge.legacyTransferTokensWithPayloadWrappedIx(
    forkedProgram,
    {
      coreMessage: forkCoreMessage.publicKey,
      senderAuthority: senderAuthority.publicKey,
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
    expectIxOkDetails(connection, [approveIx, ix], [payer, coreMessage, senderAuthority]),
    expectIxOkDetails(
      connection,
      [forkedApproveIx, forkedIx],
      [payer, forkCoreMessage, senderAuthority]
    ),
  ]);
  return [coreMessage, txDetails, forkCoreMessage, forkTxDetails];
}
