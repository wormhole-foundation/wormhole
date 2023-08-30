import * as anchor from "@coral-xyz/anchor";
import { Account, getOrCreateAssociatedTokenAccount } from "@solana/spl-token";
import {
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  MINT_INFO_WRAPPED_MAX_TWO,
  MINT_INFO_WRAPPED_7,
  MINT_INFO_WRAPPED_8,
  WrappedMintInfo,
  expectIxOkDetails,
  getTokenBalances,
  parallelPostVaa,
  expectIxErr,
  invokeVerifySignaturesAndPostVaa,
  ETHEREUM_TOKEN_ADDRESS_MAX_ONE,
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
  const wrappedMaxMint: WrappedMintInfo = MINT_INFO_WRAPPED_MAX_TWO;

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
            dstToken: payerToken,
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
        const [dstToken, forkRecipientToken] = await Promise.all([
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
          dstToken.address,
          forkRecipientToken.address
        );

        // Complete the transfer.
        await parallelTxDetails(
          program,
          forkedProgram,
          {
            dstToken,
            forkRecipientToken,
            redeemerAuthority: recipient,
          },
          signedVaa,
          payer
        );

        // Check recipient and relayer token balance changes.
        await tokenBridge.expectCorrectWrappedTokenBalanceChanges(
          connection,
          dstToken.address,
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
            dstToken: payerToken,
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

    it(`Invoke \`complete_transfer_with_payload_wrapped\` (8 Decimals, Maximum Transfer Amount)`, async () => {
      // Fetch special mint for this test.
      const { chain, address } = wrappedMaxMint;

      const [mint, forkMint] = [program, forkedProgram].map((program) =>
        tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
      );
      // Create recipient token account.
      const [payerToken, forkPayerToken] = await Promise.all([
        getOrCreateAssociatedTokenAccount(connection, payer, mint, payer.publicKey),
        getOrCreateAssociatedTokenAccount(connection, payer, forkMint, payer.publicKey),
      ]);

      // Maximum amount.
      const amount = Buffer.alloc(8, "ffffffff", "hex").readBigUInt64BE() - BigInt(1);

      // Create the signed transfer VAA.
      const signedVaa = await getSignedTransferVaa(address, amount, payer.publicKey, "0xdeadbeef");

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
          dstToken: payerToken,
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
  });

  describe("New Implementation", () => {
    for (const { chain, decimals, address } of wrappedMints) {
      it(`Cannot Invoke \`complete_transfer_with_payload_wrapped)\` (${decimals} Decimals, Invalid Mint)`, async () => {
        const mint = tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address));

        // Create payer token account.
        const payerToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          payer.publicKey
        );

        // Maximum amount.
        const amount = BigInt(42069);

        // Create the signed transfer VAA, pass an invalid token address.
        const signedVaa = await getSignedTransferVaa(
          ETHEREUM_TOKEN_ADDRESS_MAX_ONE, // Pass invalid address.
          amount,
          payer.publicKey,
          "0xdeadbeef"
        );

        // Complete the transfer.
        await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

        // Create instruction.
        const ix = tokenBridge.legacyCompleteTransferWithPayloadWrappedIx(
          program,
          {
            payer: payer.publicKey,
            dstToken: payerToken.address,
          },
          parseVaa(signedVaa),
          undefined,
          Array.from(address) // Pass correct token address to derive mint.
        );

        await expectIxErr(connection, [ix], [payer], "InvalidMint");
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
    dstToken: Account;
    forkRecipientToken: Account;
    redeemerAuthority: anchor.web3.Keypair;
  },
  signedVaa,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;
  const { dstToken, forkRecipientToken, redeemerAuthority } = tokenAccounts;

  // Post the VAA.
  const parsed = await parallelPostVaa(connection, payer, signedVaa);

  // Create instruction.
  const ix = tokenBridge.legacyCompleteTransferWithPayloadWrappedIx(
    program,
    {
      payer: payer.publicKey,
      dstToken: dstToken.address,
      redeemerAuthority: redeemerAuthority.publicKey,
    },
    parsed
  );
  const forkedIx = tokenBridge.legacyCompleteTransferWithPayloadWrappedIx(
    forkedProgram,
    {
      payer: payer.publicKey,
      dstToken: forkRecipientToken.address,
      redeemerAuthority: redeemerAuthority.publicKey,
    },
    parsed
  );
  return expectIxOkDetails(connection, [ix, forkedIx], [payer, redeemerAuthority]);
}
