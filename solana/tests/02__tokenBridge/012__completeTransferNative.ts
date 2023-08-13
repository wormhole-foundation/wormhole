import * as anchor from "@coral-xyz/anchor";
import {
  getAssociatedTokenAddressSync,
  getOrCreateAssociatedTokenAccount,
} from "@solana/spl-token";
import {
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  MINT_INFO_8,
  MINT_INFO_9,
  MintInfo,
  expectIxOkDetails,
  getTokenBalances,
  parallelPostVaa,
} from "../helpers";
import { CHAIN_ID_SOLANA, tryNativeToHexString, parseVaa, relayer } from "@certusone/wormhole-sdk";
import { GUARDIAN_KEYS } from "../helpers";
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
      it(`Invoke \`complete_transfer_native\` (${decimals} Decimals (No Fee))`, async () => {
        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate();
        const recipientToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          recipient.publicKey
        );
        const payerToken = getAssociatedTokenAddressSync(mint, payer.publicKey);

        // Amounts.
        const amount = BigInt(699999);
        const fee = BigInt(0);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(mint, amount, fee, recipientToken.address);

        // Fetch balances before.
        const recipientBalancesBefore = await getTokenBalances(
          program,
          forkedProgram,
          recipientToken.address
        );
        const relayerBalancesBefore = await getTokenBalances(program, forkedProgram, payerToken);

        // Complete the transfer.
        await parallelTxDetails(
          program,
          forkedProgram,
          {
            payer: payer.publicKey,
            recipientToken: recipientToken.address,
            mint,
            payerToken,
          },
          signedVaa,
          payer
        );

        // Check recipient and relayer token balance changes.
        await tokenBridge.expectCorrectTokenBalanceChanges(
          connection,
          recipientToken.address,
          recipientBalancesBefore,
          tokenBridge.TransferDirection.In
        );
        await tokenBridge.expectCorrectRelayerBalanceChanges(
          connection,
          payerToken,
          relayerBalancesBefore,
          BigInt(fee.toString())
        );
      });

      it.skip(`Invoke \`complete_transfer_native\` (${decimals} Decimals (With Fee))`, async () => {
        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate();
        const recipientToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          recipient.publicKey
        );
        const payerToken = getAssociatedTokenAddressSync(mint, payer.publicKey);

        // Amounts.
        const amount = BigInt(699999);
        const fee = BigInt(199999);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(mint, amount, fee, recipientToken.address);

        // Fetch balances before.
        const recipientBalancesBefore = await getTokenBalances(
          program,
          forkedProgram,
          recipientToken.address
        );
        const relayerBalancesBefore = await getTokenBalances(program, forkedProgram, payerToken);

        // Complete the transfer.
        await parallelTxDetails(
          program,
          forkedProgram,
          {
            payer: payer.publicKey,
            recipientToken: recipientToken.address,
            mint,
            payerToken,
          },
          signedVaa,
          payer
        );

        console.log(recipientBalancesBefore);
        console.log(await getTokenBalances(program, forkedProgram, recipientToken.address));

        console.log("Relayer:");
        console.log(relayerBalancesBefore);
        console.log(await getTokenBalances(program, forkedProgram, payerToken));

        // Check recipient and relayer token balance changes.
        await tokenBridge.expectCorrectTokenBalanceChanges(
          connection,
          recipientToken.address,
          recipientBalancesBefore,
          tokenBridge.TransferDirection.In,
          fee
        );
        await tokenBridge.expectCorrectRelayerBalanceChanges(
          connection,
          payerToken,
          relayerBalancesBefore,
          fee
        );
      });
    }
  });
});

function getSignedTransferVaa(
  mint: anchor.web3.PublicKey,
  amount: bigint,
  fee: bigint,
  recipient: anchor.web3.PublicKey
): Buffer {
  const vaaBytes = dummyTokenBridge.publishTransferTokens(
    tryNativeToHexString(mint.toString(), "solana"),
    CHAIN_ID_SOLANA,
    amount,
    CHAIN_ID_SOLANA,
    recipient.toBuffer().toString("hex"),
    fee
  );
  return guardians.addSignatures(vaaBytes, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
}

async function parallelTxDetails(
  program: tokenBridge.TokenBridgeProgram,
  forkedProgram: tokenBridge.TokenBridgeProgram,
  accounts: tokenBridge.LegacyCompleteTransferNativeContext,
  signedVaa: Buffer,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;

  // Post the VAA.
  await parallelPostVaa(connection, payer, signedVaa);

  // Create instruction.
  const ix = tokenBridge.legacyCompleteTransferNativeIx(program, accounts, parseVaa(signedVaa));
  const forkedIx = tokenBridge.legacyCompleteTransferNativeIx(
    forkedProgram,
    accounts,
    parseVaa(signedVaa)
  );

  return await Promise.all([
    expectIxOkDetails(connection, [ix], [payer]),
    expectIxOkDetails(connection, [forkedIx], [payer]),
  ]);
}
