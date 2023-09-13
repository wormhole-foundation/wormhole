import * as anchor from "@coral-xyz/anchor";
import { TOKEN_PROGRAM_ID, getAssociatedTokenAddressSync, mintTo } from "@solana/spl-token";
import { PublicKey } from "@solana/web3.js";
import {
  MINT_INFO_8,
  MINT_INFO_9,
  MintInfo,
  expectDeepEqual,
  expectIxOk,
  expectIxOkDetails,
  getTokenBalances,
} from "../helpers";
import * as tokenBridge from "../helpers/tokenBridge";

describe("Token Bridge -- Legacy Instruction: Transfer Tokens with Payload (Native)", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = tokenBridge.getAnchorProgram(connection, tokenBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;

  const forkedProgram = tokenBridge.getAnchorProgram(connection, tokenBridge.mainnet());

  const mints: MintInfo[] = [MINT_INFO_8, MINT_INFO_9];

  const senderAuthority = anchor.web3.Keypair.generate();

  before("Set Up Mints and Token Accounts", async () => {
    for (const { mint } of mints) {
      const token = getAssociatedTokenAddressSync(mint, payer.publicKey);

      await mintTo(connection, payer, mint, token, payer, BigInt("1000000000000000000"));
    }
  });

  describe("Ok", () => {
    const unorderedPrograms = [
      {
        name: "System",
        pubkey: anchor.web3.SystemProgram.programId,
        forkPubkey: anchor.web3.SystemProgram.programId,
        idx: 15,
      },
      { name: "Token", pubkey: TOKEN_PROGRAM_ID, forkPubkey: TOKEN_PROGRAM_ID, idx: 16 },
      {
        name: "Core Bridge",
        pubkey: tokenBridge.coreBridgeProgramId(program),
        forkPubkey: tokenBridge.coreBridgeProgramId(forkedProgram),
        idx: 17,
      },
    ];

    const possibleIndices = [14, 15, 16, 17];

    for (const { name, pubkey, forkPubkey, idx } of unorderedPrograms) {
      for (const possibleIdx of possibleIndices) {
        if (possibleIdx == idx) {
          continue;
        }

        it(`Invoke \`transfer_tokens_with_payload_native\` with ${name} Program at Index == ${possibleIdx}`, async () => {
          const { mint } = MINT_INFO_8;
          const srcToken = getAssociatedTokenAddressSync(mint, payer.publicKey);

          const amount = new anchor.BN(10);
          const approveIx = tokenBridge.approveTransferAuthorityIx(
            program,
            srcToken,
            payer.publicKey,
            amount
          );

          const args = defaultArgs(amount);
          const coreMessage = anchor.web3.Keypair.generate();
          const ix = tokenBridge.legacyTransferTokensWithPayloadNativeIx(
            program,
            {
              payer: payer.publicKey,
              srcToken,
              mint,
              coreMessage: coreMessage.publicKey,
              senderAuthority: payer.publicKey,
            },
            args
          );
          expectDeepEqual(ix.keys[idx].pubkey, pubkey);
          ix.keys[idx].pubkey = ix.keys[possibleIdx].pubkey;
          ix.keys[possibleIdx].pubkey = pubkey;

          const forkCoreMessage = anchor.web3.Keypair.generate();
          const forkedApproveIx = tokenBridge.approveTransferAuthorityIx(
            forkedProgram,
            srcToken,
            payer.publicKey,
            amount
          );
          const forkedIx = tokenBridge.legacyTransferTokensWithPayloadNativeIx(
            forkedProgram,
            {
              payer: payer.publicKey,
              srcToken,
              mint,
              coreMessage: forkCoreMessage.publicKey,
              senderAuthority: payer.publicKey,
            },
            args
          );
          expectDeepEqual(forkedIx.keys[idx].pubkey, forkPubkey);
          forkedIx.keys[idx].pubkey = forkedIx.keys[possibleIdx].pubkey;
          forkedIx.keys[possibleIdx].pubkey = forkPubkey;

          await Promise.all([
            expectIxOk(connection, [approveIx, ix], [payer, coreMessage]),
            expectIxOk(connection, [forkedApproveIx, forkedIx], [payer, forkCoreMessage]),
          ]);
        });
      }
    }

    for (const { mint, decimals } of mints) {
      const srcToken = getAssociatedTokenAddressSync(mint, payer.publicKey);

      it(`Invoke \`transfer_tokens_with_payload_native\` (${decimals} Decimals)`, async () => {
        const amount = new anchor.BN("88888888");

        const balancesBefore = await getTokenBalances(program, forkedProgram, srcToken);

        const [coreMessage, txDetails, forkCoreMessage, forkTxDetails] = await parallelTxDetails(
          program,
          forkedProgram,
          { payer: payer.publicKey, mint: mint, srcToken },
          defaultArgs(amount),
          payer,
          senderAuthority
        );

        await tokenBridge.expectCorrectTokenBalanceChanges(
          connection,
          srcToken,
          balancesBefore,
          tokenBridge.TransferDirection.Out
        );

        // TODO: Check that the core messages are correct.
      });
    }
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
  accounts: { payer: PublicKey; mint: PublicKey; srcToken: PublicKey },
  args: tokenBridge.LegacyTransferTokensWithPayloadArgs,
  payer: anchor.web3.Keypair,
  senderAuthority: anchor.web3.Keypair
) {
  const connection = program.provider.connection;
  const { payer: owner, srcToken: token } = accounts;
  const { amount } = args;
  const coreMessage = anchor.web3.Keypair.generate();
  const approveIx = tokenBridge.approveTransferAuthorityIx(program, token, owner, amount);
  const ix = tokenBridge.legacyTransferTokensWithPayloadNativeIx(
    program,
    {
      coreMessage: coreMessage.publicKey,
      senderAuthority: senderAuthority.publicKey,
      ...accounts,
    },
    args
  );

  const forkCoreMessage = anchor.web3.Keypair.generate();
  const forkedApproveIx = tokenBridge.approveTransferAuthorityIx(
    forkedProgram,
    token,
    owner,
    amount
  );
  const forkedIx = tokenBridge.legacyTransferTokensWithPayloadNativeIx(
    forkedProgram,
    {
      coreMessage: forkCoreMessage.publicKey,
      senderAuthority: senderAuthority.publicKey,
      ...accounts,
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
