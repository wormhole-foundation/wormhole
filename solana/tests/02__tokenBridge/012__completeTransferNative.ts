import * as anchor from "@coral-xyz/anchor";
import {
  createAssociatedTokenAccount,
  getAssociatedTokenAddressSync,
  mintTo,
} from "@solana/spl-token";
import { PublicKey } from "@solana/web3.js";
import {
  MINT_INFO_8,
  MINT_INFO_9,
  MintInfo,
  expectIxOkDetails,
  getTokenBalances,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import * as tokenBridge from "../helpers/tokenBridge";
import { MockTokenBridge, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";

describe("Token Bridge -- Legacy Instruction: Complete Transfer (Native)", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = tokenBridge.getAnchorProgram(connection, tokenBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;

  const forkedProgram = tokenBridge.getAnchorProgram(connection, tokenBridge.mainnet());

  const mints: MintInfo[] = [MINT_INFO_8, MINT_INFO_9];

  describe("Ok", () => {
    for (const { mint, decimals } of mints) {
      const srcToken = getAssociatedTokenAddressSync(mint, payer.publicKey);

      it.skip(`Invoke \`complete_transfer_native\` (${decimals} Decimals)`, async () => {
        const amount = new anchor.BN("88888888");
        const relayerFee = new anchor.BN("11111111");

        const balancesBefore = await getTokenBalances(program, forkedProgram, srcToken);

        const [coreMessage, txDetails, forkCoreMessage, forkTxDetails] = await parallelTxDetails(
          program,
          forkedProgram,
          { payer: payer.publicKey, mint: mint, srcToken },
          defaultArgs(amount, relayerFee),
          payer
        );

        // TODO: Check message accounts.
      });
    }
  });
});

function defaultVaa(amount: anchor.BN, recipient: anchor.web3.PublicKey): Buffer {
  return Buffer.alloc(0);
  // const timestamp = 12345678;
  // const chain = 1;
  // const published = governance.publishWormholeTransferFees(
  //   timestamp,
  //   chain,
  //   BigInt(amount.toString()),
  //   recipient.toBuffer()
  // );
  // return guardians.addSignatures(
  //   published,
  //   [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]
  // );
}

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
  accounts: { payer: PublicKey; mint: PublicKey; srcToken: PublicKey },
  args: tokenBridge.LegacyTransferTokensArgs,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;
  const { payer: owner, srcToken: token } = accounts;
  const { amount } = args;
  const coreMessage = anchor.web3.Keypair.generate();
  const approveIx = tokenBridge.approveTransferAuthorityIx(program, token, owner, amount);
  const ix = tokenBridge.legacyTransferTokensNativeIx(
    program,
    {
      coreMessage: coreMessage.publicKey,
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
  const forkedIx = tokenBridge.legacyTransferTokensNativeIx(
    forkedProgram,
    {
      coreMessage: forkCoreMessage.publicKey,
      ...accounts,
    },
    args
  );

  const [txDetails, forkTxDetails] = await Promise.all([
    expectIxOkDetails(connection, [approveIx, ix], [payer, coreMessage]),
    expectIxOkDetails(connection, [forkedApproveIx, forkedIx], [payer, forkCoreMessage]),
  ]);

  return [coreMessage, txDetails, forkCoreMessage, forkTxDetails];
}
