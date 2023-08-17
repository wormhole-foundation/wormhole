import * as anchor from "@coral-xyz/anchor";
import { Account, getOrCreateAssociatedTokenAccount } from "@solana/spl-token";
import {
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  MINT_INFO_WRAPPED_7,
  MINT_INFO_WRAPPED_8,
  MINT_INFO_WRAPPED_MAX_ONE,
  WrappedMintInfo,
  expectIxOkDetails,
  getTokenBalances,
  parallelPostVaa,
  expectIxErr,
  invokeVerifySignaturesAndPostVaa,
} from "../helpers";
import {
  CHAIN_ID_SOLANA,
  tryNativeToHexString,
  parseVaa,
  CHAIN_ID_ETH,
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
  690 // Starting sequence
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

describe("Token Bridge -- Legacy Instruction: Complete Transfer (Wrapped)", () => {
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
      it(`Invoke \`complete_transfer_wrapped\` (${decimals} Decimals, No Fee)`, async () => {
        const [mint, forkMint] = [program, forkedProgram].map((program) =>
          tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
        );
        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate();
        const [recipientToken, forkRecipientToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, recipient.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, recipient.publicKey),
        ]);

        const [payerToken, forkPayerToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, payer.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, payer.publicKey),
        ]);

        // Amounts.
        const amount = BigInt(699999);
        let fee = BigInt(0);

        // Create the signed transfer VAA.
        const [signedVaa, forkSignedVaa] = [recipientToken, forkRecipientToken].map((acct) =>
          getSignedTransferVaa(address, amount, fee, acct.address)
        );

        // Fetch balances before.
        const [recipientBalancesBefore, relayerBalancesBefore] = await Promise.all([
          getTokenBalances(
            program,
            forkedProgram,
            recipientToken.address,
            forkRecipientToken.address
          ),
          getTokenBalances(program, forkedProgram, payerToken.address, forkPayerToken.address),
        ]);

        // Complete the transfer.
        await parallelTxDetails(
          program,
          forkedProgram,
          {
            recipientToken,
            forkRecipientToken,
          },
          {
            signedVaa,
            forkSignedVaa,
          },
          payer
        );

        // Denormalize the fee.
        if (decimals > 8) {
          fee = fee * BigInt(10 ** decimals - 8);
        }

        // Check recipient and relayer token balance changes.
        await Promise.all([
          tokenBridge.expectCorrectWrappedTokenBalanceChanges(
            connection,
            recipientToken.address,
            forkRecipientToken.address,
            recipientBalancesBefore,
            tokenBridge.TransferDirection.In,
            amount
          ),
          tokenBridge.expectCorrectWrappedTokenBalanceChanges(
            connection,
            payerToken.address,
            forkPayerToken.address,
            relayerBalancesBefore,
            tokenBridge.TransferDirection.In,
            fee
          ),
        ]);
      });

      it(`Invoke \`complete_transfer_wrapped\` (${decimals} Decimals, With Fee)`, async () => {
        const [mint, forkMint] = [program, forkedProgram].map((program) =>
          tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
        );
        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate();
        const [recipientToken, forkRecipientToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, recipient.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, recipient.publicKey),
        ]);

        const [payerToken, forkPayerToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, payer.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, payer.publicKey),
        ]);

        // Amounts.
        const amount = BigInt(4206999999);
        let fee = BigInt(500000);

        // Create the signed transfer VAA.
        const [signedVaa, forkSignedVaa] = [recipientToken, forkRecipientToken].map((acct) =>
          getSignedTransferVaa(address, amount, fee, acct.address)
        );

        // Fetch balances before.
        const [recipientBalancesBefore, relayerBalancesBefore] = await Promise.all([
          getTokenBalances(
            program,
            forkedProgram,
            recipientToken.address,
            forkRecipientToken.address
          ),
          getTokenBalances(program, forkedProgram, payerToken.address, forkPayerToken.address),
        ]);

        // Complete the transfer.
        await parallelTxDetails(
          program,
          forkedProgram,
          {
            recipientToken,
            forkRecipientToken,
          },
          {
            signedVaa,
            forkSignedVaa,
          },
          payer
        );

        // Denormalize the fee.
        if (decimals > 8) {
          fee = fee * BigInt(10 ** decimals - 8);
        }

        // Check recipient and relayer token balance changes.
        await Promise.all([
          tokenBridge.expectCorrectWrappedTokenBalanceChanges(
            connection,
            recipientToken.address,
            forkRecipientToken.address,
            recipientBalancesBefore,
            tokenBridge.TransferDirection.In,
            amount - fee
          ),
          tokenBridge.expectCorrectWrappedTokenBalanceChanges(
            connection,
            payerToken.address,
            forkPayerToken.address,
            relayerBalancesBefore,
            tokenBridge.TransferDirection.In,
            fee
          ),
        ]);
      });

      it(`Invoke \`complete_transfer_wrapped\` (${decimals} Decimals, Self Redemption no Fee)`, async () => {
        const [mint, forkMint] = [program, forkedProgram].map((program) =>
          tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
        );
        const [payerToken, forkPayerToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, payer.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, payer.publicKey),
        ]);

        // Amounts.
        const amount = BigInt(4206999999);
        let fee = BigInt(0);

        // Create the signed transfer VAA.
        const [signedVaa, forkSignedVaa] = [payerToken, forkPayerToken].map((acct) =>
          getSignedTransferVaa(address, amount, fee, acct.address)
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
          },
          {
            signedVaa,
            forkSignedVaa,
          },
          payer
        );

        // Check recipient and relayer token balance changes.
        await Promise.all([
          tokenBridge.expectCorrectWrappedTokenBalanceChanges(
            connection,
            payerToken.address,
            forkPayerToken.address,
            recipientBalancesBefore,
            tokenBridge.TransferDirection.In,
            amount // no fee for this test
          ),
        ]);
      });

      it(`Invoke \`complete_transfer_wrapped\` (${decimals} Decimals, Self Redemption with Fee)`, async () => {
        const [mint, forkMint] = [program, forkedProgram].map((program) =>
          tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
        );
        const [payerToken, forkPayerToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, payer.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, payer.publicKey),
        ]);

        // Amounts, set fee to nonzero value.
        const amount = BigInt(4206999999);
        let fee = BigInt(500000);

        // Create the signed transfer VAA.
        const [signedVaa, forkSignedVaa] = [payerToken, forkPayerToken].map((acct) =>
          getSignedTransferVaa(address, amount, fee, acct.address)
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
          },
          {
            signedVaa,
            forkSignedVaa,
          },
          payer
        );

        // Check recipient and relayer token balance changes.
        await Promise.all([
          tokenBridge.expectCorrectWrappedTokenBalanceChanges(
            connection,
            payerToken.address,
            forkPayerToken.address,
            recipientBalancesBefore,
            tokenBridge.TransferDirection.In,
            amount // Fee shouldn't be paid out, so we don't need to account for it.
          ),
        ]);
      });

      it(`Invoke \`complete_transfer_wrapped\` (${decimals} Decimals, Recipient == Wallet Address)`, async () => {
        const mint = await tokenBridge.wrappedMintPda(
          program.programId,
          chain,
          Array.from(address)
        );

        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate();
        const recipientToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          recipient.publicKey
        );
        const payerToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          payer.publicKey
        );

        // Amounts.
        const amount = BigInt(699999420);
        let fee = BigInt(50000);

        // Create the signed transfer VAA.
        const signedVaa = await getSignedTransferVaa(
          address,
          amount,
          fee,
          recipient.publicKey // Recipient is the wallet address, not the ATA.
        );

        // Fetch balances before.
        const [recipientBalancesBefore, relayerBalancesBefore] = await Promise.all([
          getTokenBalances(program, forkedProgram, recipientToken.address),
          getTokenBalances(program, forkedProgram, payerToken.address),
        ]);

        // Complete the transfer.
        await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

        // Create instruction.
        const ix = tokenBridge.legacyCompleteTransferWrappedIx(
          program,
          { payer: payer.publicKey, recipientToken: recipientToken.address },
          parseVaa(signedVaa)
        );

        await expectIxOkDetails(connection, [ix], [payer]);

        // Denormalize the fee.
        if (decimals > 8) {
          fee = fee * BigInt(10 ** decimals - 8);
        }

        // Check recipient and relayer token balance changes.
        const [recipientBalancesAfter, relayerBalancesAfter] = await Promise.all([
          getTokenBalances(program, forkedProgram, recipientToken.address),
          getTokenBalances(program, forkedProgram, payerToken.address),
        ]);

        expect(recipientBalancesAfter.token - recipientBalancesBefore.token).equals(amount - fee);
        expect(relayerBalancesAfter.token - relayerBalancesBefore.token).equals(fee);
      });

      it(`Invoke \`complete_transfer_wrapped\` (${decimals} Decimals, Minimum Transfer Amount)`, async () => {
        const [mint, forkMint] = [program, forkedProgram].map((program) =>
          tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
        );
        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate();
        const [recipientToken, forkRecipientToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, recipient.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, recipient.publicKey),
        ]);

        const [payerToken, forkPayerToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, payer.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, payer.publicKey),
        ]);

        // Minimum amount.
        const amount = BigInt(1);
        let fee = BigInt(0);

        // Create the signed transfer VAA.
        const [signedVaa, forkSignedVaa] = [recipientToken, forkRecipientToken].map((acct) =>
          getSignedTransferVaa(address, amount, fee, acct.address)
        );

        // Fetch balances before.
        const [recipientBalancesBefore, relayerBalancesBefore] = await Promise.all([
          getTokenBalances(
            program,
            forkedProgram,
            recipientToken.address,
            forkRecipientToken.address
          ),
          getTokenBalances(program, forkedProgram, payerToken.address, forkPayerToken.address),
        ]);

        // Complete the transfer.
        await parallelTxDetails(
          program,
          forkedProgram,
          {
            recipientToken,
            forkRecipientToken,
          },
          {
            signedVaa,
            forkSignedVaa,
          },
          payer
        );

        // Denormalize the fee.
        if (decimals > 8) {
          fee = fee * BigInt(10 ** decimals - 8);
        }

        // Check recipient and relayer token balance changes.
        await Promise.all([
          tokenBridge.expectCorrectWrappedTokenBalanceChanges(
            connection,
            recipientToken.address,
            forkRecipientToken.address,
            recipientBalancesBefore,
            tokenBridge.TransferDirection.In,
            amount
          ),
          tokenBridge.expectCorrectWrappedTokenBalanceChanges(
            connection,
            payerToken.address,
            forkPayerToken.address,
            relayerBalancesBefore,
            tokenBridge.TransferDirection.In,
            fee
          ),
        ]);
      });

      it(`Cannot Invoke \`complete_transfer_wrapped\` (${decimals} Decimals, Invalid Target Chain)`, async () => {
        const mint = await tokenBridge.wrappedMintPda(
          program.programId,
          chain,
          Array.from(address)
        );

        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate();
        const recipientToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          recipient.publicKey
        );

        // Amounts.
        const amount = BigInt(699999420);
        let fee = BigInt(50000);

        // Create the signed transfer VAA.
        const signedVaa = await getSignedTransferVaa(
          address,
          amount,
          fee,
          recipientToken.address,
          CHAIN_ID_ETH // Invalid target chain.
        );

        // Complete the transfer.
        await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

        // Create instruction.
        const ix = tokenBridge.legacyCompleteTransferWrappedIx(
          program,
          { payer: payer.publicKey, recipientToken: recipientToken.address },
          parseVaa(signedVaa)
        );

        await expectIxErr(connection, [ix], [payer], "RecipientChainNotSolana");
      });

      it(`Cannot Invoke \`complete_transfer_wrapped\` (${decimals} Decimals, Invalid Recipient ATA)`, async () => {
        const mint = await tokenBridge.wrappedMintPda(
          program.programId,
          chain,
          Array.from(address)
        );

        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate();
        const recipientToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          recipient.publicKey
        );
        const payerToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          payer.publicKey
        );

        // Amounts.
        const amount = BigInt(699999420);
        let fee = BigInt(50000);

        // Create the signed transfer VAA.
        const signedVaa = await getSignedTransferVaa(address, amount, fee, recipientToken.address);

        // Complete the transfer.
        await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

        // Create instruction.
        const ix = tokenBridge.legacyCompleteTransferWrappedIx(
          program,
          { payer: payer.publicKey, recipientToken: payerToken.address }, // Pass invalid recipient ATA
          parseVaa(signedVaa)
        );

        await expectIxErr(connection, [ix], [payer], "ConstraintTokenOwner");
      });

      it(`Invoke \`complete_transfer_wrapped\` (${decimals} Decimals, Minimum Transfer Amount)`, async () => {
        const [mint, forkMint] = [program, forkedProgram].map((program) =>
          tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
        );
        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate();
        const [recipientToken, forkRecipientToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, recipient.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, recipient.publicKey),
        ]);

        const [payerToken, forkPayerToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, payer.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, payer.publicKey),
        ]);

        // Amounts.
        const amount = BigInt(1);
        let fee = BigInt(0);

        // Create the signed transfer VAA.
        const [signedVaa, forkSignedVaa] = [recipientToken, forkRecipientToken].map((acct) =>
          getSignedTransferVaa(address, amount, fee, acct.address)
        );

        // Fetch balances before.
        const [recipientBalancesBefore, relayerBalancesBefore] = await Promise.all([
          getTokenBalances(
            program,
            forkedProgram,
            recipientToken.address,
            forkRecipientToken.address
          ),
          getTokenBalances(program, forkedProgram, payerToken.address, forkPayerToken.address),
        ]);

        // Complete the transfer.
        await parallelTxDetails(
          program,
          forkedProgram,
          {
            recipientToken,
            forkRecipientToken,
          },
          {
            signedVaa,
            forkSignedVaa,
          },
          payer
        );

        // Denormalize the fee.
        if (decimals > 8) {
          fee = fee * BigInt(10 ** decimals - 8);
        }

        // Check recipient and relayer token balance changes.
        await Promise.all([
          tokenBridge.expectCorrectWrappedTokenBalanceChanges(
            connection,
            recipientToken.address,
            forkRecipientToken.address,
            recipientBalancesBefore,
            tokenBridge.TransferDirection.In,
            amount
          ),
          tokenBridge.expectCorrectWrappedTokenBalanceChanges(
            connection,
            payerToken.address,
            forkPayerToken.address,
            relayerBalancesBefore,
            tokenBridge.TransferDirection.In,
            fee
          ),
        ]);
      });
    }

    it(`Invoke \`complete_transfer_wrapped\` (8 Decimals, Maximum Transfer Amount)`, async () => {
      // Fetch special mint for this test.
      const { chain, decimals, address } = wrappedMaxMint;

      const [mint, forkMint] = [program, forkedProgram].map((program) =>
        tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
      );
      // Create recipient token account.
      const [payerToken, forkPayerToken] = await Promise.all([
        getOrCreateAssociatedTokenAccount(connection, payer, mint, payer.publicKey),
        getOrCreateAssociatedTokenAccount(connection, payer, forkMint, payer.publicKey),
      ]);

      // Amounts.
      const amount = Buffer.alloc(8, "ffffffff", "hex").readBigUInt64BE() - BigInt(1);

      // Create the signed transfer VAA.
      const [signedVaa, forkSignedVaa] = [payerToken, forkPayerToken].map((acct) =>
        getSignedTransferVaa(address, amount, BigInt(0), acct.address)
      );

      // Fetch balances before.
      const payerBalancesBefore = await getTokenBalances(
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
        },
        {
          signedVaa,
          forkSignedVaa,
        },
        payer
      );

      // Check recipient and relayer token balance changes.
      await Promise.all([
        tokenBridge.expectCorrectWrappedTokenBalanceChanges(
          connection,
          payerToken.address,
          forkPayerToken.address,
          payerBalancesBefore,
          tokenBridge.TransferDirection.In,
          amount
        ),
      ]);
    });
  });
});

function getSignedTransferVaa(
  tokenAddress: Uint8Array,
  amount: bigint,
  fee: bigint,
  recipient: anchor.web3.PublicKey,
  targetChain?: number
): Buffer {
  const vaaBytes = dummyTokenBridge.publishTransferTokens(
    Buffer.from(tokenAddress).toString("hex"),
    CHAIN_ID_ETH,
    amount,
    targetChain ?? CHAIN_ID_SOLANA,
    recipient.toBuffer().toString("hex"),
    fee
  );
  return guardians.addSignatures(vaaBytes, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
}

async function parallelTxDetails(
  program: tokenBridge.TokenBridgeProgram,
  forkedProgram: tokenBridge.TokenBridgeProgram,
  tokenAccounts: {
    recipientToken: Account;
    forkRecipientToken: Account;
  },
  signedVaas: { signedVaa: Buffer; forkSignedVaa: Buffer },
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;
  const { recipientToken, forkRecipientToken } = tokenAccounts;
  const { signedVaa, forkSignedVaa } = signedVaas;

  // Post the VAA.
  const parsed = await parallelPostVaa(connection, payer, signedVaa);
  const forkParsed = await parallelPostVaa(connection, payer, forkSignedVaa);

  // Create instruction.
  const ix = tokenBridge.legacyCompleteTransferWrappedIx(
    program,
    { payer: payer.publicKey, recipientToken: recipientToken.address },
    parsed
  );
  const forkedIx = tokenBridge.legacyCompleteTransferWrappedIx(
    forkedProgram,
    { payer: payer.publicKey, recipientToken: forkRecipientToken.address },
    forkParsed
  );
  return expectIxOkDetails(connection, [ix, forkedIx], [payer]);
}
