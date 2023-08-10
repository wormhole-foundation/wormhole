import * as anchor from "@coral-xyz/anchor";
import { getAssociatedTokenAddressSync, mintTo } from "@solana/spl-token";
import { PublicKey } from "@solana/web3.js";
import {
  MINT_INFO_8,
  MINT_INFO_9,
  MintInfo,
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
