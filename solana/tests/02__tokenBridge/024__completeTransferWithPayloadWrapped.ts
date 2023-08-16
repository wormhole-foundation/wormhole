import * as anchor from "@coral-xyz/anchor";
import { Account, getOrCreateAssociatedTokenAccount } from "@solana/spl-token";
import {
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  MINT_INFO_WRAPPED_7,
  MINT_INFO_WRAPPED_8,
  WrappedMintInfo,
  expectIxOkDetails,
  getTokenBalances,
  parallelPostVaa,
  expectIxErr,
  invokeVerifySignaturesAndPostVaa,
  MINT_INFO_WRAPPED_MAX_7,
  MINT_INFO_WRAPPED_MAX_8,
} from "../helpers";
import {
  CHAIN_ID_SOLANA,
  tryNativeToHexString,
  parseVaa,
  CHAIN_ID_ETH,
  tryNativeToUint8Array,
} from "@certusone/wormhole-sdk";
import { GUARDIAN_KEYS } from "../helpers";
import * as tokenBridge from "../helpers/tokenBridge";
import * as coreBridge from "../helpers/coreBridge";
import { MockTokenBridge, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import { expect } from "chai";

const GUARDIAN_SET_INDEX = 2;
const dummyTokenBridge = new MockTokenBridge(
  tryNativeToHexString(ETHEREUM_TOKEN_BRIDGE_ADDRESS, 2),
  2,
  0, // Consistency level
  6900 // Starting sequence
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

describe("Token Bridge -- Legacy Instruction: Complete Transfer With Payload (Wrapped)", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = tokenBridge.getAnchorProgram(connection, tokenBridge.localnet());
  const wormholeProgram = coreBridge.getAnchorProgram(connection, coreBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;

  const forkedProgram = tokenBridge.getAnchorProgram(connection, tokenBridge.mainnet());

  const wrappedMints: WrappedMintInfo[] = [MINT_INFO_WRAPPED_8, MINT_INFO_WRAPPED_7];
  const wrappedMaxMints: WrappedMintInfo[] = [MINT_INFO_WRAPPED_MAX_7, MINT_INFO_WRAPPED_MAX_8];

  describe("Ok", () => {
    for (const { chain, decimals, address } of wrappedMints) {
      it(`Invoke \`complete_transfer_with_payload_wrapped\` (${decimals} Decimals, Redeemer == Redeemer Authority)`, async () => {
        const [mint, forkMint] = [program, forkedProgram].map((program) =>
          tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
        );
        // Create recipient token account.
        const [payerToken, forkPayerToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, payer.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, payer.publicKey),
        ]);

        // Amounts.
        const amount = BigInt(699999);

        // Create the signed transfer VAA.
        const signedVaa = await getSignedTransferVaa(
          address,
          amount,
          payer.publicKey,
          "0xdeadbeef"
        );

        // Fetch balances before.
        const recipientBalancesBefore = await getTokenBalances(
          program,
          forkedProgram,
          payerToken.address,
          forkPayerToken.address
        );

        // Complete the transfer.
        await parallelTxDetails(
          program,
          forkedProgram,
          {
            recipientToken: payerToken,
            forkRecipientToken: forkPayerToken,
            redeemerAuthority: payer,
          },
          signedVaa,
          payer
        );

        // Check recipient and relayer token balance changes.
        await tokenBridge.expectCorrectWrappedTokenBalanceChanges(
          connection,
          payerToken.address,
          forkPayerToken.address,
          recipientBalancesBefore,
          tokenBridge.TransferDirection.In,
          amount
        );
      });

      it(`Invoke \`complete_transfer_with_payload_wrapped\` (${decimals} Decimals, Redeemer != Redeemer Authority)`, async () => {
        const [mint, forkMint] = [program, forkedProgram].map((program) =>
          tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
        );
        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate();
        const [recipientToken, forkRecipientToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, recipient.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, recipient.publicKey),
        ]);

        // Amounts.
        const amount = BigInt(699999);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(address, amount, recipient.publicKey, "0xdeadbeef");

        // Fetch balances before.
        const recipientBalancesBefore = await getTokenBalances(
          program,
          forkedProgram,
          recipientToken.address,
          forkRecipientToken.address
        );

        // Complete the transfer.
        await parallelTxDetails(
          program,
          forkedProgram,
          {
            recipientToken,
            forkRecipientToken,
            redeemerAuthority: recipient,
          },
          signedVaa,
          payer
        );

        // Check recipient and relayer token balance changes.
        await tokenBridge.expectCorrectWrappedTokenBalanceChanges(
          connection,
          recipientToken.address,
          forkRecipientToken.address,
          recipientBalancesBefore,
          tokenBridge.TransferDirection.In,
          amount
        );
      });

      it(`Invoke \`complete_transfer_with_payload_wrapped\` (${decimals} Decimals, Minimum Transfer Amount)`, async () => {
        const [mint, forkMint] = [program, forkedProgram].map((program) =>
          tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
        );
        // Create recipient token account.
        const [payerToken, forkPayerToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, payer.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, payer.publicKey),
        ]);

        // Minimum amount.
        const amount = BigInt(1);

        // Create the signed transfer VAA.
        const signedVaa = await getSignedTransferVaa(
          address,
          amount,
          payer.publicKey,
          "0xdeadbeef"
        );

        // Fetch balances before.
        const recipientBalancesBefore = await getTokenBalances(
          program,
          forkedProgram,
          payerToken.address,
          forkPayerToken.address
        );

        // Complete the transfer.
        await parallelTxDetails(
          program,
          forkedProgram,
          {
            recipientToken: payerToken,
            forkRecipientToken: forkPayerToken,
            redeemerAuthority: payer,
          },
          signedVaa,
          payer
        );

        // Check recipient and relayer token balance changes.
        await tokenBridge.expectCorrectWrappedTokenBalanceChanges(
          connection,
          payerToken.address,
          forkPayerToken.address,
          recipientBalancesBefore,
          tokenBridge.TransferDirection.In,
          amount
        );
      });
    }

    for (const { chain, decimals, address } of wrappedMints) {
      it(`Invoke \`complete_transfer_with_payload_wrapped\` (${decimals} Decimals, Maximum Transfer Amount)`, async () => {
        const [mint, forkMint] = [program, forkedProgram].map((program) =>
          tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
        );
        // Create recipient token account.
        const [payerToken, forkPayerToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, payer.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, payer.publicKey),
        ]);

        // Minimum amount.
        const amount = Buffer.alloc(8, "ffffffff", "hex").readBigUInt64BE() - BigInt(5);

        // Create the signed transfer VAA.
        const signedVaa = await getSignedTransferVaa(
          address,
          amount,
          payer.publicKey,
          "0xdeadbeef"
        );

        console.log(signedVaa.toString("hex"));

        // Fetch balances before.
        const recipientBalancesBefore = await getTokenBalances(
          program,
          forkedProgram,
          payerToken.address,
          forkPayerToken.address
        );

        // Complete the transfer.
        await parallelTxDetails(
          program,
          forkedProgram,
          {
            recipientToken: payerToken,
            forkRecipientToken: forkPayerToken,
            redeemerAuthority: payer,
          },
          signedVaa,
          payer
        );

        // Check recipient and relayer token balance changes.
        await tokenBridge.expectCorrectWrappedTokenBalanceChanges(
          connection,
          payerToken.address,
          forkPayerToken.address,
          recipientBalancesBefore,
          tokenBridge.TransferDirection.In,
          amount
        );
      });
    }
  });
});

function getSignedTransferVaa(
  tokenAddress: Uint8Array,
  amount: bigint,
  recipient: anchor.web3.PublicKey,
  payload: string,
  targetChain?: number
): Buffer {
  const vaaBytes = dummyTokenBridge.publishTransferTokensWithPayload(
    Buffer.from(tokenAddress).toString("hex"),
    CHAIN_ID_ETH,
    amount,
    targetChain ?? CHAIN_ID_SOLANA,
    recipient.toBuffer().toString("hex"), // TARGET CONTRACT (redeemer)
    Buffer.from(tryNativeToUint8Array(ETHEREUM_TOKEN_BRIDGE_ADDRESS, 2)),
    Buffer.from(payload.substring(2), "hex"),
    0 // Batch ID
  );
  return guardians.addSignatures(vaaBytes, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
}

async function parallelTxDetails(
  program: tokenBridge.TokenBridgeProgram,
  forkedProgram: tokenBridge.TokenBridgeProgram,
  tokenAccounts: {
    recipientToken: Account;
    forkRecipientToken: Account;
    redeemerAuthority: anchor.web3.Keypair;
  },
  signedVaa,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;
  const { recipientToken, forkRecipientToken, redeemerAuthority } = tokenAccounts;

  // Post the VAA.
  const parsed = await parallelPostVaa(connection, payer, signedVaa);

  // Create instruction.
  const ix = tokenBridge.legacyCompleteTransferWithPayloadWrappedIx(
    program,
    {
      payer: payer.publicKey,
      recipientToken: recipientToken.address,
      redeemerAuthority: redeemerAuthority.publicKey,
    },
    parsed
  );
  const forkedIx = tokenBridge.legacyCompleteTransferWithPayloadWrappedIx(
    forkedProgram,
    {
      payer: payer.publicKey,
      recipientToken: forkRecipientToken.address,
      redeemerAuthority: redeemerAuthority.publicKey,
    },
    parsed
  );
  return expectIxOkDetails(connection, [ix, forkedIx], [payer, redeemerAuthority]);
}
