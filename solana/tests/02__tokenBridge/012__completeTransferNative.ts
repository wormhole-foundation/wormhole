import {
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  parseVaa,
  tryNativeToHexString,
} from "@certusone/wormhole-sdk";
import { MockGuardians, MockTokenBridge } from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as anchor from "@coral-xyz/anchor";
import {
  TOKEN_PROGRAM_ID,
  getAssociatedTokenAddressSync,
  getOrCreateAssociatedTokenAccount,
} from "@solana/spl-token";
import { expect } from "chai";
import {
  ETHEREUM_DEADBEEF_TOKEN_ADDRESS,
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  GUARDIAN_KEYS,
  MINT_INFO_8,
  MINT_INFO_9,
  MintInfo,
  createAssociatedTokenAccountOffCurve,
  expectDeepEqual,
  expectIxErr,
  expectIxOk,
  expectIxOkDetails,
  getTokenBalances,
  invokeVerifySignaturesAndPostVaa,
  parallelPostVaa,
  processVaa,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import * as tokenBridge from "../helpers/tokenBridge";

const GUARDIAN_SET_INDEX = 4;
const dummyTokenBridge = new MockTokenBridge(
  tryNativeToHexString(ETHEREUM_TOKEN_BRIDGE_ADDRESS, 2),
  2,
  0
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

const localVariables = new Map<string, any>();

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
    const unorderedPrograms = [
      {
        name: "System",
        pubkey: anchor.web3.SystemProgram.programId,
        forkPubkey: anchor.web3.SystemProgram.programId,
        idx: 11,
      },
      { name: "Token", pubkey: TOKEN_PROGRAM_ID, forkPubkey: TOKEN_PROGRAM_ID, idx: 12 },
      {
        name: "Core Bridge",
        pubkey: tokenBridge.coreBridgeProgramId(program),
        forkPubkey: tokenBridge.coreBridgeProgramId(forkedProgram),
        idx: 13,
      },
    ];

    const possibleIndices = [10, 11, 12, 13];

    for (const { name, pubkey, forkPubkey, idx } of unorderedPrograms) {
      for (const possibleIdx of possibleIndices) {
        if (possibleIdx == idx) {
          continue;
        }

        it(`Invoke \`complete_transfer_native\` with ${name} Program at Index == ${possibleIdx}`, async () => {
          const { mint } = MINT_INFO_8;
          const recipient = anchor.web3.Keypair.generate();
          const recipientToken = await getOrCreateAssociatedTokenAccount(
            connection,
            payer,
            mint,
            recipient.publicKey
          );

          const amount = new anchor.BN(10);
          const signedVaa = getSignedTransferVaa(
            mint,
            BigInt(amount.toString()),
            BigInt(0),
            recipientToken.address
          );

          // Process the VAA for the new implementation.
          const encodedVaa = await processVaa(
            tokenBridge.getCoreBridgeProgram(program),
            payer,
            signedVaa,
            GUARDIAN_SET_INDEX
          );

          // And post the VAA.
          const parsed = await parallelPostVaa(connection, payer, signedVaa);
          const ix = tokenBridge.legacyCompleteTransferNativeIx(
            program,
            {
              payer: payer.publicKey,
              vaa: encodedVaa,
              recipientToken: recipientToken.address,
              mint,
            },
            parsed
          );
          expectDeepEqual(ix.keys[idx].pubkey, pubkey);
          ix.keys[idx].pubkey = ix.keys[possibleIdx].pubkey;
          ix.keys[possibleIdx].pubkey = pubkey;

          const forkedIx = tokenBridge.legacyCompleteTransferNativeIx(
            forkedProgram,
            {
              payer: payer.publicKey,
              recipientToken: recipientToken.address,
              mint,
            },
            parsed
          );
          expectDeepEqual(forkedIx.keys[idx].pubkey, forkPubkey);
          forkedIx.keys[idx].pubkey = forkedIx.keys[possibleIdx].pubkey;
          forkedIx.keys[possibleIdx].pubkey = forkPubkey;

          await expectIxOk(connection, [ix, forkedIx], [payer]);
        });
      }
    }

    for (const { mint, decimals } of mints) {
      it(`Invoke \`complete_transfer_native\` (${decimals} Decimals, No Fee)`, async () => {
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
        const [recipientBalancesBefore, relayerBalancesBefore] = await Promise.all([
          getTokenBalances(program, forkedProgram, recipientToken.address),
          getTokenBalances(program, forkedProgram, payerToken),
        ]);

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
        await Promise.all([
          tokenBridge.expectCorrectTokenBalanceChanges(
            connection,
            recipientToken.address,
            recipientBalancesBefore,
            tokenBridge.TransferDirection.In
          ),
          tokenBridge.expectCorrectRelayerBalanceChanges(
            connection,
            payerToken,
            relayerBalancesBefore,
            fee * BigInt(10 ** (decimals - 8))
          ),
        ]);

        // Save for later
        localVariables.set(`signedVaa${decimals}`, signedVaa);
        localVariables.set(`recipientToken${decimals}`, recipientToken.address);
        localVariables.set(`mint${decimals}`, mint);
      });

      it(`Invoke \`complete_transfer_native\` (${decimals} Decimals, With Fee)`, async () => {
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
        let fee = BigInt(199999);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(mint, amount, fee, recipientToken.address);

        // Fetch balances before.
        const [recipientBalancesBefore, relayerBalancesBefore] = await Promise.all([
          getTokenBalances(program, forkedProgram, recipientToken.address),
          getTokenBalances(program, forkedProgram, payerToken),
        ]);

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

        // Denormalize the fee.
        fee = fee * BigInt(10 ** (decimals - 8));

        // Check recipient and relayer token balance changes.
        await Promise.all([
          tokenBridge.expectCorrectTokenBalanceChanges(
            connection,
            recipientToken.address,
            recipientBalancesBefore,
            tokenBridge.TransferDirection.In,
            fee
          ),
          tokenBridge.expectCorrectRelayerBalanceChanges(
            connection,
            payerToken,
            relayerBalancesBefore,
            fee
          ),
        ]);
      });

      it(`Invoke \`complete_transfer_native\` (${decimals} Decimals, Self Redeemption with Fee)`, async () => {
        // Create recipient token account.
        const payerToken = getAssociatedTokenAddressSync(mint, payer.publicKey);

        // Amounts.
        const amount = BigInt(699999);
        let fee = BigInt(199999);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(mint, amount, fee, payerToken);

        // Fetch balances before.
        const recipientBalancesBefore = await getTokenBalances(program, forkedProgram, payerToken);

        // Complete the transfer.
        await parallelTxDetails(
          program,
          forkedProgram,
          {
            payer: payer.publicKey,
            recipientToken: payerToken,
            mint,
            payerToken,
          },
          signedVaa,
          payer
        );

        // Denormalize the fee.
        fee = fee * BigInt(10 ** (decimals - 8));

        // Check recipient and relayer token balance changes.
        await tokenBridge.expectCorrectTokenBalanceChanges(
          connection,
          payerToken,
          recipientBalancesBefore,
          tokenBridge.TransferDirection.In,
          BigInt(0) // No fee for self-redemption.
        );
      });

      it(`Invoke \`complete_transfer_native\` (${decimals} Decimals, Self Redeemption no fee)`, async () => {
        // Create recipient token account.
        const payerToken = getAssociatedTokenAddressSync(mint, payer.publicKey);

        // Amounts.
        const amount = BigInt(699999);
        let fee = BigInt(0);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(mint, amount, fee, payerToken);

        // Fetch balances before.
        const recipientBalancesBefore = await getTokenBalances(program, forkedProgram, payerToken);

        // Complete the transfer.
        await parallelTxDetails(
          program,
          forkedProgram,
          {
            payer: payer.publicKey,
            recipientToken: payerToken,
            mint,
            payerToken,
          },
          signedVaa,
          payer
        );

        // Denormalize the fee.
        fee = fee * BigInt(10 ** (decimals - 8));

        // Check recipient and relayer token balance changes.
        await tokenBridge.expectCorrectTokenBalanceChanges(
          connection,
          payerToken,
          recipientBalancesBefore,
          tokenBridge.TransferDirection.In,
          BigInt(0) // No fee for self-redemption.
        );
      });
    }
  });

  describe("New Implementation", () => {
    it("Cannot Invoke `complete_transfer_native` on Same VAA", async () => {
      const signedVaa = localVariables.get("signedVaa9") as Buffer;
      const recipientToken = localVariables.get("recipientToken9") as anchor.web3.PublicKey;
      const mint = localVariables.get("mint9") as anchor.web3.PublicKey;

      const ix = tokenBridge.legacyCompleteTransferNativeIx(
        program,
        { payer: payer.publicKey, recipientToken, mint },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "already in use");
    });

    it("Cannot Invoke `complete_transfer_native` on Same VAA Buffer using Encoded Vaa", async () => {
      const signedVaa = localVariables.get("signedVaa9") as Buffer;
      const recipientToken = localVariables.get("recipientToken9") as anchor.web3.PublicKey;
      const mint = localVariables.get("mint9") as anchor.web3.PublicKey;

      const encodedVaa = await processVaa(
        tokenBridge.getCoreBridgeProgram(program),
        payer,
        signedVaa,
        GUARDIAN_SET_INDEX
      );

      const ix = tokenBridge.legacyCompleteTransferNativeIx(
        program,
        { payer: payer.publicKey, vaa: encodedVaa, recipientToken, mint },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "already in use");
    });

    for (const { mint, decimals } of mints) {
      it(`Cannot Invoke \`complete_transfer_native\` (${decimals} Decimals, Recipient == Wallet Address with Rent Sysvar)`, async () => {
        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate().publicKey;
        const recipientToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          recipient
        );

        // Amounts.
        let amount = BigInt(42069);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(
          mint,
          amount,
          BigInt(0),
          recipient // Recipient is the wallet address, not the ATA.
        );

        // Post the VAA.
        await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

        // Create instruction.
        const ix = tokenBridge.legacyCompleteTransferNativeIx(
          program,
          {
            payer: payer.publicKey,
            recipientToken: recipientToken.address,
            mint,
          },
          parseVaa(signedVaa)
        );

        // Complete the transfer.
        await expectIxErr(connection, [ix], [payer], "InvalidRecipient");
      });

      it(`Cannot Invoke \`complete_transfer_native\` (${decimals} Decimals, Recipient == Wallet Address with an Invalid Recipient)`, async () => {
        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate().publicKey;
        const recipientToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          recipient
        );

        // Amounts.
        let amount = BigInt(42069);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(
          mint,
          amount,
          BigInt(0),
          recipient // Recipient is the wallet address, not the ATA.
        );

        // Post the VAA.
        await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

        const anotherGuy = anchor.web3.Keypair.generate().publicKey;

        // Create instruction.
        const ix = tokenBridge.legacyCompleteTransferNativeIx(
          program,
          {
            payer: payer.publicKey,
            recipientToken: recipientToken.address,
            mint,
            recipient: anotherGuy,
          },
          parseVaa(signedVaa)
        );

        // Complete the transfer.
        await expectIxErr(connection, [ix], [payer], "InvalidRecipient");
      });

      it(`Invoke \`complete_transfer_native\` (${decimals} Decimals, Recipient == Wallet Address)`, async () => {
        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate().publicKey;
        const recipientToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          recipient
        );
        const payerToken = getAssociatedTokenAddressSync(mint, payer.publicKey);

        // Amounts.
        let amount = BigInt(42069);
        let fee = BigInt(1669);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(
          mint,
          amount,
          fee,
          recipient // Recipient is the wallet address, not the ATA.
        );

        // Fetch balances before.
        const [recipientBalancesBefore, relayerBalancesBefore] = await Promise.all([
          getTokenBalances(program, forkedProgram, recipientToken.address),
          getTokenBalances(program, forkedProgram, payerToken),
        ]);

        // Post the VAA.
        await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

        // Create instruction.
        const ix = tokenBridge.legacyCompleteTransferNativeIx(
          program,
          {
            payer: payer.publicKey,
            recipientToken: recipientToken.address,
            mint,
            payerToken,
            recipient,
          },
          parseVaa(signedVaa)
        );

        // Complete the transfer.
        await expectIxOkDetails(connection, [ix], [payer]);

        // Denormalize the fee and amount.
        fee = fee * BigInt(10 ** (decimals - 8));
        amount = amount * BigInt(10 ** (decimals - 8));

        // Fetch balances after.
        const [recipientBalancesAfter, relayerBalancesAfter] = await Promise.all([
          getTokenBalances(program, forkedProgram, recipientToken.address),
          getTokenBalances(program, forkedProgram, payerToken),
        ]);

        // Check recipient and relayer token balance changes.
        expect(recipientBalancesAfter.token - recipientBalancesBefore.token).to.equal(amount - fee);
        expect(recipientBalancesBefore.custodyToken - recipientBalancesAfter.custodyToken).to.equal(
          amount
        );
        expect(relayerBalancesAfter.token - relayerBalancesBefore.token).to.equal(fee);
      });

      it(`Cannot Invoke \`complete_transfer_native\` (${decimals} Decimals, Invalid Target Chain)`, async () => {
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
        let amount = BigInt(42069);
        let fee = BigInt(1669);

        // Target chain.
        const targetChain = 2;

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(
          mint,
          amount,
          fee,
          recipientToken.address,
          targetChain
        );

        // Post the VAA.
        await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

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

        // Complete the transfer.
        await expectIxErr(connection, [ix], [payer], "RecipientChainNotSolana");
      });

      it(`Cannot Invoke \`complete_transfer_native\` (${decimals} Decimals, Invalid Recipent ATA)`, async () => {
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

        const payerToken = getAssociatedTokenAddressSync(mint, payer.publicKey);

        // Amounts.
        let amount = BigInt(42069);
        let fee = BigInt(1669);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(mint, amount, fee, recipientToken.address);

        // Post the VAA.
        await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

        // Create instruction.
        const ix = tokenBridge.legacyCompleteTransferNativeIx(
          program,
          {
            payer: payer.publicKey,
            recipientToken: trollToken, // Pass an invalid recipient ATA.
            mint,
            payerToken,
            recipient: recipientToken.address,
          },
          parseVaa(signedVaa)
        );

        // Complete the transfer.
        await expectIxErr(connection, [ix], [payer], "NestedTokenAccount");
      });

      it(`Cannot Invoke \`complete_transfer_native\` (${decimals} Decimals, Invalid Mint)`, async () => {
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
        let amount = BigInt(42069);
        let fee = BigInt(1669);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(
          anchor.web3.Keypair.generate().publicKey, // Pass bogus mint.
          amount,
          fee,
          recipientToken.address
        );

        // Post the VAA.
        await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

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

        // Complete the transfer.
        await expectIxErr(connection, [ix], [payer], "InvalidMint");
      });
    }

    it(`Cannot Invoke \`complete_transfer_native\` (Wrapped Mint)`, async () => {
      // Mint.
      const mint = mints[0].mint;

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
      let amount = BigInt(42069);
      let fee = BigInt(1669);

      // Create the signed transfer VAA. Pass a token chain that is not Solana.
      const signedVaa = getSignedTransferVaa(
        mint,
        amount,
        fee,
        recipientToken.address,
        undefined,
        CHAIN_ID_ETH // Specify a token chain that is not Solana.
      );

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

      // Create instruction.
      const ix = tokenBridge.legacyCompleteTransferNativeIx(
        program,
        {
          payer: payer.publicKey,
          recipientToken: recipientToken.address,
          mint: mint,
          payerToken,
        },
        parseVaa(signedVaa)
      );

      // Complete the transfer.
      await expectIxErr(connection, [ix], [payer], "WrappedAsset");
    });

    it("Cannot Invoke `complete_transfer_native` (Invalid Token Bridge VAA)", async () => {
      // Mint.
      const mint = mints[0].mint;

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
      const ix = tokenBridge.legacyCompleteTransferNativeIx(
        program,
        {
          payer: payer.publicKey,
          recipientToken: recipientToken.address,
          mint,
        },
        parseVaa(signedVaa)
      );

      // Complete the transfer.
      await expectIxErr(connection, [ix], [payer], "InvalidTokenBridgeVaa");
    });
  });
});

function getSignedTransferVaa(
  mint: anchor.web3.PublicKey,
  amount: bigint,
  fee: bigint,
  recipient: anchor.web3.PublicKey,
  targetChain?: number,
  tokenChain?: number
): Buffer {
  const vaaBytes = dummyTokenBridge.publishTransferTokens(
    tryNativeToHexString(mint.toString(), "solana"),
    tokenChain ?? CHAIN_ID_SOLANA,
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
  accounts: tokenBridge.LegacyCompleteTransferNativeContext,
  signedVaa: Buffer,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;

  // Post the VAA.
  const parsed = await parallelPostVaa(connection, payer, signedVaa);

  // Create instruction.
  const ix = tokenBridge.legacyCompleteTransferNativeIx(program, accounts, parsed);
  const forkedIx = tokenBridge.legacyCompleteTransferNativeIx(forkedProgram, accounts, parsed);

  return expectIxOkDetails(connection, [ix, forkedIx], [payer]);
}
