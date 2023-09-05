import {
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  parseVaa,
  tryNativeToHexString,
} from "@certusone/wormhole-sdk";
import { MockGuardians, MockTokenBridge } from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as anchor from "@coral-xyz/anchor";
import { Account, getOrCreateAssociatedTokenAccount } from "@solana/spl-token";
import { expect } from "chai";
import {
  ETHEREUM_DEADBEEF_TOKEN_ADDRESS,
  ETHEREUM_TOKEN_ADDRESS_MAX_ONE,
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  GUARDIAN_KEYS,
  WRAPPED_MINT_INFO_7,
  WRAPPED_MINT_INFO_8,
  WRAPPED_MINT_INFO_MAX_ONE,
  WrappedMintInfo,
  createAssociatedTokenAccountOffCurve,
  expectIxErr,
  expectIxOkDetails,
  getTokenBalances,
  invokeVerifySignaturesAndPostVaa,
  parallelPostVaa,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import * as tokenBridge from "../helpers/tokenBridge";

const GUARDIAN_SET_INDEX = 4;
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

  const wrappedMints: WrappedMintInfo[] = [WRAPPED_MINT_INFO_8, WRAPPED_MINT_INFO_7];
  const wrappedMaxMint: WrappedMintInfo = WRAPPED_MINT_INFO_MAX_ONE;

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

  describe("New Implementation", () => {
    for (const { chain, decimals, address } of wrappedMints) {
      it(`Invoke \`complete_transfer_wrapped\` (${decimals} Decimals, Recipient == Wallet Address with Rent Sysvar)`, async () => {
        const mint = tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address));

        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate().publicKey;
        const recipientToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          recipient
        );

        // Amounts.
        const amount = BigInt(699999420);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(
          address,
          amount,
          BigInt(0),
          recipient // Recipient is the wallet address, not the ATA.
        );

        // Complete the transfer.
        await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

        // Create instruction.
        const ix = tokenBridge.legacyCompleteTransferWrappedIx(
          program,
          { payer: payer.publicKey, recipientToken: recipientToken.address },
          parseVaa(signedVaa)
        );

        await expectIxErr(connection, [ix], [payer], "InvalidRecipient");
      });

      it(`Invoke \`complete_transfer_wrapped\` (${decimals} Decimals, Recipient == Wallet Address with an Invalid Address)`, async () => {
        const mint = tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address));

        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate().publicKey;
        const recipientToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          recipient
        );

        // Amounts.
        const amount = BigInt(699999420);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(
          address,
          amount,
          BigInt(0),
          recipient // Recipient is the wallet address, not the ATA.
        );

        // Complete the transfer.
        await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

        const anotherGuy = anchor.web3.Keypair.generate().publicKey;

        // Create instruction.
        const ix = tokenBridge.legacyCompleteTransferWrappedIx(
          program,
          { payer: payer.publicKey, recipientToken: recipientToken.address, recipient: anotherGuy },
          parseVaa(signedVaa)
        );

        await expectIxErr(connection, [ix], [payer], "InvalidRecipient");
      });

      it(`Invoke \`complete_transfer_wrapped\` (${decimals} Decimals, Recipient == Wallet Address)`, async () => {
        const mint = tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address));

        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate().publicKey;
        const recipientToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          recipient
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
        const signedVaa = getSignedTransferVaa(
          address,
          amount,
          fee,
          recipient // Recipient is the wallet address, not the ATA.
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
          { payer: payer.publicKey, recipientToken: recipientToken.address, recipient },
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

      it(`Cannot Invoke \`complete_transfer_wrapped\` (${decimals} Decimals, Invalid Target Chain)`, async () => {
        const mint = tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address));

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
        const signedVaa = getSignedTransferVaa(
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
        const mint = tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address));

        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate();
        const recipientToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          recipient.publicKey
        );
        const trollToken = await createAssociatedTokenAccountOffCurve(
          connection,
          payer,
          mint,
          recipientToken.address
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
        const signedVaa = getSignedTransferVaa(address, amount, fee, recipientToken.address);

        // Complete the transfer.
        await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

        // Create instruction.
        const ix = tokenBridge.legacyCompleteTransferWrappedIx(
          program,
          { payer: payer.publicKey, recipientToken: trollToken, recipient: recipientToken.address }, // Pass invalid recipient ATA
          parseVaa(signedVaa)
        );

        await expectIxErr(connection, [ix], [payer], "NestedTokenAccount");
      });

      it(`Cannot Invoke \`complete_transfer_wrapped\` (${decimals} Decimals, Invalid Mint)`, async () => {
        const mint = tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address));

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

        // Create the signed transfer VAA, pass an invalid token address.
        const signedVaa = getSignedTransferVaa(
          ETHEREUM_TOKEN_ADDRESS_MAX_ONE, // Pass invalid address.
          amount,
          fee,
          recipientToken.address,
          CHAIN_ID_SOLANA
        );

        // Complete the transfer.
        await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

        // Create instruction.
        const ix = tokenBridge.legacyCompleteTransferWrappedIx(
          program,
          { payer: payer.publicKey, recipientToken: recipientToken.address },
          parseVaa(signedVaa),
          {
            tokenAddress: Array.from(address), // Pass correct token address to derive mint.
          }
        );

        await expectIxErr(connection, [ix], [payer], "InvalidMint");
      });
    }

    it(`Cannot Invoke \`complete_transfer_wrapped\` (Native Asset)`, async () => {
      const wrappedAssetInfo = wrappedMints[0];
      const { chain, address } = wrappedAssetInfo;

      // Mint.
      const mint = tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address));

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

      // Create the signed transfer VAA. Pass invalid token chain for a wrapped asset.
      const signedVaa = getSignedTransferVaa(
        address,
        amount,
        fee,
        recipientToken.address,
        undefined,
        CHAIN_ID_SOLANA // Pass a token chain that is not ETH
      );

      // Complete the transfer.
      await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

      // Create instruction.
      const ix = tokenBridge.legacyCompleteTransferWrappedIx(
        program,
        { payer: payer.publicKey, recipientToken: recipientToken.address },
        parseVaa(signedVaa),
        {
          tokenChain: CHAIN_ID_ETH, // Pass ETH chain ID so the wrapped asset account is derived correctly.
        }
      );

      await expectIxErr(connection, [ix], [payer], "NativeAsset");
    });

    it(`Cannot Invoke \`complete_transfer_wrapped\` (U64Overflow)`, async () => {
      const wrappedAssetInfo = wrappedMints[0];
      const { chain, address } = wrappedAssetInfo;

      // Mint.
      const mint = tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address));

      // Create recipient token account.
      const recipient = anchor.web3.Keypair.generate();
      const recipientToken = await getOrCreateAssociatedTokenAccount(
        connection,
        payer,
        mint,
        recipient.publicKey
      );

      // MAX U64.
      const amount = Buffer.alloc(8, "ffffffff", "hex").readBigUInt64BE() + BigInt(10000);
      let fee = BigInt(0);

      // Create the signed transfer VAA. Specify an amount that is > u64::MAX.
      const signedVaa = getSignedTransferVaa(address, amount, fee, recipientToken.address);

      // Complete the transfer.
      await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

      // Create instruction.
      const ix = tokenBridge.legacyCompleteTransferWrappedIx(
        program,
        { payer: payer.publicKey, recipientToken: recipientToken.address },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "U64Overflow");
    });

    it("Cannot Invoke `complete_transfer_wrapped` (Invalid Token Bridge VAA)", async () => {
      const wrappedAssetInfo = wrappedMints[0];
      const { chain, address } = wrappedAssetInfo;

      // Mint.
      const mint = tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address));

      // Create recipient token account.
      const recipient = anchor.web3.Keypair.generate();
      const recipientToken = await getOrCreateAssociatedTokenAccount(
        connection,
        payer,
        mint,
        recipient.publicKey
      );

      // Create a bogus attestation VAA.
      const published = dummyTokenBridge.publishAttestMeta(
        Buffer.from(ETHEREUM_DEADBEEF_TOKEN_ADDRESS).toString("hex"),
        8, // Decimals
        "EVOO", // Symbol.
        "Extra Virgin Olive Oil", // Name.
        420, // Nonce.
        1234567 // Timestamp.
      );
      const signedVaa = guardians.addSignatures(
        published,
        [0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 14]
      );

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

      // Create instruction.
      const ix = tokenBridge.legacyCompleteTransferWrappedIx(
        program,
        { payer: payer.publicKey, recipientToken: recipientToken.address, wrappedMint: mint },
        parseVaa(signedVaa)
      );

      // Complete the transfer.
      await expectIxErr(connection, [ix], [payer], "InvalidTokenBridgeVaa");
    });
  });
});

function getSignedTransferVaa(
  tokenAddress: Uint8Array,
  amount: bigint,
  fee: bigint,
  recipient: anchor.web3.PublicKey,
  targetChain?: number,
  tokenChain?: number
): Buffer {
  const vaaBytes = dummyTokenBridge.publishTransferTokens(
    Buffer.from(tokenAddress).toString("hex"),
    tokenChain ?? CHAIN_ID_ETH,
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
