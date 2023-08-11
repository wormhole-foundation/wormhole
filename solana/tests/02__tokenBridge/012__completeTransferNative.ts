import * as anchor from "@coral-xyz/anchor";
import {
  createAssociatedTokenAccount,
  getAssociatedTokenAddressSync,
  getOrCreateAssociatedTokenAccount,
  mintTo,
} from "@solana/spl-token";
import { PublicKey } from "@solana/web3.js";
import {
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  MINT_INFO_8,
  MINT_INFO_9,
  MintInfo,
  expectIxOkDetails,
  getTokenBalances,
  parallelPostVaa,
} from "../helpers";
import { CHAIN_ID_SOLANA, tryNativeToHexString, parseVaa } from "@certusone/wormhole-sdk";
import { GUARDIAN_KEYS, invokeVerifySignaturesAndPostVaa } from "../helpers";
import * as tokenBridge from "../helpers/tokenBridge";
import * as coreBridge from "../helpers/coreBridge";
import { MockTokenBridge, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";

const GUARDIAN_SET_INDEX = 2;
const dummyTokenBridge = new MockTokenBridge(
  tryNativeToHexString(ETHEREUM_TOKEN_BRIDGE_ADDRESS, 2),
  2,
  0
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

describe("Token Bridge -- Legacy Instruction: Complete Transfer (Native)", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = tokenBridge.getAnchorProgram(connection, tokenBridge.localnet());
  const wormholeProgram = coreBridge.getAnchorProgram(connection, coreBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;

  const forkedProgram = tokenBridge.getAnchorProgram(connection, tokenBridge.mainnet());

  const mints: MintInfo[] = [MINT_INFO_8, MINT_INFO_9];

  describe("Ok", () => {
    for (const { mint, decimals } of mints) {
      it(`Invoke \`complete_transfer_native\` (${decimals} Decimals)`, async () => {
        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate();
        const recipientToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          recipient.publicKey
        );
        const payerToken = getAssociatedTokenAddressSync(mint, payer.publicKey);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(
          mint,
          new anchor.BN(69999),
          new anchor.BN(0),
          recipientToken.address
        );

        // Fetch balances before.
        const recipientBalancesBefore = await getTokenBalances(
          program,
          forkedProgram,
          recipientToken.address
        );
        const relayerBalancesBefore = await getTokenBalances(program, forkedProgram, payerToken);

        console.log(recipientToken);

        // Post the VAA.
        await parallelPostVaa(connection, payer, signedVaa);

        // Create instruction.
        const ix = tokenBridge.legacyCompleteTransferNativeIx(
          program,
          {
            payer: payer.publicKey,
            recipientToken: recipientToken.address,
            mint,
            payerToken,
          },
          parseVaa(signedVaa)
        );
        const forkedIx = tokenBridge.legacyCompleteTransferNativeIx(
          forkedProgram,
          {
            payer: payer.publicKey,
            recipientToken: recipientToken.address,
            mint,
            payerToken,
          },
          parseVaa(signedVaa)
        );
        console.log(forkedIx.keys);

        await expectIxOkDetails(connection, [ix, forkedIx], [payer]);

        await tokenBridge.expectCorrectTokenBalanceChanges(
          connection,
          recipientToken.address,
          recipientBalancesBefore,
          tokenBridge.TransferDirection.In
        );

        await tokenBridge.expectCorrectTokenBalanceChanges(
          connection,
          payerToken,
          relayerBalancesBefore,
          tokenBridge.TransferDirection.In
        );
      });
    }
  });
});

function getSignedTransferVaa(
  mint: anchor.web3.PublicKey,
  amount: anchor.BN,
  fee: anchor.BN,
  recipient: anchor.web3.PublicKey
): Buffer {
  const vaaBytes = dummyTokenBridge.publishTransferTokens(
    tryNativeToHexString(mint.toString(), "solana"),
    CHAIN_ID_SOLANA,
    BigInt(amount.toString()),
    CHAIN_ID_SOLANA,
    recipient.toBuffer().toString("hex"),
    BigInt(fee.toString())
  );
  return guardians.addSignatures(vaaBytes, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
}

// async function parallelTxDetails(
//   program: tokenBridge.TokenBridgeProgram,
//   forkedProgram: tokenBridge.TokenBridgeProgram,
//   accounts: { payer: PublicKey; mint: PublicKey; srcToken: PublicKey },
//   args: tokenBridge.LegacyTransferTokensArgs,
//   payer: anchor.web3.Keypair
// ) {
//   const connection = program.provider.connection;
//   const { payer: owner, srcToken: token } = accounts;
//   const { amount } = args;
//   const coreMessage = anchor.web3.Keypair.generate();
//   const approveIx = tokenBridge.approveTransferAuthorityIx(program, token, owner, amount);
//   const ix = tokenBridge.legacyTransferTokensNativeIx(
//     program,
//     {
//       coreMessage: coreMessage.publicKey,
//       ...accounts,
//     },
//     args
//   );

//   const forkCoreMessage = anchor.web3.Keypair.generate();
//   const forkedApproveIx = tokenBridge.approveTransferAuthorityIx(
//     forkedProgram,
//     token,
//     owner,
//     amount
//   );
//   const forkedIx = tokenBridge.legacyTransferTokensNativeIx(
//     forkedProgram,
//     {
//       coreMessage: forkCoreMessage.publicKey,
//       ...accounts,
//     },
//     args
//   );

//   const [txDetails, forkTxDetails] = await Promise.all([
//     expectIxOkDetails(connection, [approveIx, ix], [payer, coreMessage]),
//     expectIxOkDetails(connection, [forkedApproveIx, forkedIx], [payer, forkCoreMessage]),
//   ]);

//   return [coreMessage, txDetails, forkCoreMessage, forkTxDetails];
// }
