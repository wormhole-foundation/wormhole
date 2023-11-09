import * as anchor from "@coral-xyz/anchor";
import { Account, TOKEN_PROGRAM_ID, getOrCreateAssociatedTokenAccount } from "@solana/spl-token";
import {
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  WRAPPED_MINT_INFO_MAX_TWO,
  WRAPPED_MINT_INFO_7,
  WRAPPED_MINT_INFO_8,
  WrappedMintInfo,
  expectIxOkDetails,
  getTokenBalances,
  parallelPostVaa,
  expectIxErr,
  invokeVerifySignaturesAndPostVaa,
  ETHEREUM_TOKEN_ADDRESS_MAX_ONE,
  ETHEREUM_DEADBEEF_TOKEN_ADDRESS,
  expectDeepEqual,
  expectIxOk,
  processVaa,
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

const GUARDIAN_SET_INDEX = 4;
const dummyTokenBridge = new MockTokenBridge(
  tryNativeToHexString(ETHEREUM_TOKEN_BRIDGE_ADDRESS, 2),
  2,
  0, // Consistency level
  6900 // Starting sequence
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

const localVariables = new Map<string, any>();

describe("Token Bridge -- Legacy Instruction: Complete Transfer With Payload (Wrapped)", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = tokenBridge.getAnchorProgram(connection, tokenBridge.localnet());
  const wormholeProgram = coreBridge.getAnchorProgram(connection, coreBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;

  const forkedProgram = tokenBridge.getAnchorProgram(connection, tokenBridge.mainnet());

  const wrappedMints: WrappedMintInfo[] = [WRAPPED_MINT_INFO_8, WRAPPED_MINT_INFO_7];
  const wrappedMaxMint: WrappedMintInfo = WRAPPED_MINT_INFO_MAX_TWO;

  describe("Ok", () => {
    const unorderedPrograms = [
      {
        name: "System",
        pubkey: anchor.web3.SystemProgram.programId,
        forkPubkey: anchor.web3.SystemProgram.programId,
        idx: 12,
      },
      { name: "Token", pubkey: TOKEN_PROGRAM_ID, forkPubkey: TOKEN_PROGRAM_ID, idx: 13 },
      {
        name: "Core Bridge",
        pubkey: tokenBridge.coreBridgeProgramId(program),
        forkPubkey: tokenBridge.coreBridgeProgramId(forkedProgram),
        idx: 14,
      },
    ];

    const possibleIndices = [11, 12, 13, 14];

    for (const { name, pubkey, forkPubkey, idx } of unorderedPrograms) {
      for (const possibleIdx of possibleIndices) {
        if (possibleIdx == idx) {
          continue;
        }

        it(`Invoke \`complete_transfer_with_payload_wrapped\` with ${name} Program at Index == ${possibleIdx}`, async () => {
          const { chain, address } = WRAPPED_MINT_INFO_8;

          const mint = tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address));
          const dstToken = await getOrCreateAssociatedTokenAccount(
            connection,
            payer,
            mint,
            payer.publicKey
          );

          const forkMint = tokenBridge.wrappedMintPda(
            forkedProgram.programId,
            chain,
            Array.from(address)
          );
          const forkDstToken = await getOrCreateAssociatedTokenAccount(
            connection,
            payer,
            forkMint,
            payer.publicKey
          );

          const amount = new anchor.BN(10);
          const signedVaa = getSignedTransferVaa(
            address,
            BigInt(amount.toString()),
            payer.publicKey
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
          const ix = tokenBridge.legacyCompleteTransferWithPayloadWrappedIx(
            program,
            {
              payer: payer.publicKey,
              vaa: encodedVaa,
              dstToken: dstToken.address,
              wrappedMint: mint,
            },
            parsed
          );
          expectDeepEqual(ix.keys[idx].pubkey, pubkey);
          ix.keys[idx].pubkey = ix.keys[possibleIdx].pubkey;
          ix.keys[possibleIdx].pubkey = pubkey;

          const forkedIx = tokenBridge.legacyCompleteTransferWithPayloadWrappedIx(
            forkedProgram,
            {
              payer: payer.publicKey,
              dstToken: forkDstToken.address,
              wrappedMint: forkMint,
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
        const signedVaa = getSignedTransferVaa(address, amount, payer.publicKey);

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
            forkDstToken: forkPayerToken,
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

        // Save for later
        localVariables.set(`signedVaa${decimals}`, signedVaa);
        localVariables.set(`dstToken${decimals}`, payerToken.address);
      });

      it(`Invoke \`complete_transfer_with_payload_wrapped\` (${decimals} Decimals, Redeemer != Redeemer Authority)`, async () => {
        const [mint, forkMint] = [program, forkedProgram].map((program) =>
          tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
        );
        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate();
        const [dstToken, forkDstToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, recipient.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, recipient.publicKey),
        ]);

        // Amounts.
        const amount = BigInt(699999);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(address, amount, recipient.publicKey);

        // Fetch balances before.
        const recipientBalancesBefore = await getTokenBalances(
          program,
          forkedProgram,
          dstToken.address,
          forkDstToken.address
        );

        // Complete the transfer.
        await parallelTxDetails(
          program,
          forkedProgram,
          {
            dstToken,
            forkDstToken,
            redeemerAuthority: recipient,
          },
          signedVaa,
          payer
        );

        // Check recipient and relayer token balance changes.
        await tokenBridge.expectCorrectWrappedTokenBalanceChanges(
          connection,
          dstToken.address,
          forkDstToken.address,
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
        const signedVaa = getSignedTransferVaa(address, amount, payer.publicKey);

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
            forkDstToken: forkPayerToken,
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
      const signedVaa = getSignedTransferVaa(address, amount, payer.publicKey);

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
          forkDstToken: forkPayerToken,
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
    it("Cannot Invoke `complete_transfer_wrapped` on Same VAA", async () => {
      const signedVaa = localVariables.get("signedVaa8") as Buffer;
      const dstToken = localVariables.get("dstToken8") as anchor.web3.PublicKey;

      const ix = tokenBridge.legacyCompleteTransferWithPayloadWrappedIx(
        program,
        { payer: payer.publicKey, dstToken },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "already in use");
    });
    it("Cannot Invoke `complete_transfer_wrapped` on Same VAA Buffer using Encoded Vaa", async () => {
      const signedVaa = localVariables.get("signedVaa8") as Buffer;
      const dstToken = localVariables.get("dstToken8") as anchor.web3.PublicKey;

      const encodedVaa = await processVaa(
        tokenBridge.getCoreBridgeProgram(program),
        payer,
        signedVaa,
        GUARDIAN_SET_INDEX
      );

      const ix = tokenBridge.legacyCompleteTransferWithPayloadWrappedIx(
        program,
        { payer: payer.publicKey, vaa: encodedVaa, dstToken },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "already in use");
    });

    for (const { chain, decimals, address } of wrappedMints) {
      it(`Cannot Invoke \`complete_transfer_with_payload_wrapped\` (${decimals} Decimals, Invalid Mint)`, async () => {
        const mint = tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address));

        // Create payer token account.
        const payerToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          payer.publicKey
        );

        // Amount.
        const amount = BigInt(42069);

        // Create the signed transfer VAA, pass an invalid token address.
        const signedVaa = getSignedTransferVaa(
          ETHEREUM_TOKEN_ADDRESS_MAX_ONE, // Pass invalid address.
          amount,
          payer.publicKey
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

      it(`Cannot Invoke \`complete_transfer_with_payload_wrapped\` (${decimals} Decimals, Invalid Redeemer Chain)`, async () => {
        const mint = tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address));

        // Create payer token account.
        const payerToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          payer.publicKey
        );

        // Amount.
        const amount = BigInt(42069);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(
          address,
          amount,
          payer.publicKey,
          undefined,
          CHAIN_ID_ETH // Pass invalid target chain.
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
          parseVaa(signedVaa)
        );

        await expectIxErr(connection, [ix], [payer], "RedeemerChainNotSolana");
      });
    }

    it(`Cannot Invoke \`complete_transfer_with_payload_wrapped\` (Native Asset)`, async () => {
      const wrappedAssetInfo = wrappedMints[0];
      const { chain, address } = wrappedAssetInfo;

      // Mint.
      const mint = tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address));

      // Create payer token account.
      const payerToken = await getOrCreateAssociatedTokenAccount(
        connection,
        payer,
        mint,
        payer.publicKey
      );

      // Amount.
      const amount = BigInt(42069);

      // Create the signed transfer VAA, pass an invalid token address.
      const signedVaa = getSignedTransferVaa(
        ETHEREUM_TOKEN_ADDRESS_MAX_ONE, // Pass invalid address.
        amount,
        payer.publicKey,
        undefined,
        undefined,
        CHAIN_ID_SOLANA // Pass a token chain that is not ETH.
      );

      // Complete the transfer.
      await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

      // Create instruction.
      const ix = tokenBridge.legacyCompleteTransferWithPayloadWrappedIx(
        program,
        {
          payer: payer.publicKey,
          dstToken: payerToken.address,
          wrappedMint: mint,
        },
        parseVaa(signedVaa),
        undefined,
        undefined,
        CHAIN_ID_ETH // Pass ETH chain ID so the wrapped asset account is derived correctly.
      );

      await expectIxErr(connection, [ix], [payer], "NativeAsset");
    });

    it(`Cannot Invoke \`complete_transfer_with_payload_wrapped\` (U64Overflow)`, async () => {
      const wrappedAssetInfo = wrappedMints[0];
      const { chain, address } = wrappedAssetInfo;

      // Mint.
      const mint = tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address));

      // Create payer token account.
      const payerToken = await getOrCreateAssociatedTokenAccount(
        connection,
        payer,
        mint,
        payer.publicKey
      );

      // MAX U64.
      const amount = Buffer.alloc(8, "ffffffff", "hex").readBigUInt64BE() + BigInt(10000);

      // Create the signed transfer VAA. Specify an amount that is > u64::MAX.
      const signedVaa = getSignedTransferVaa(address, amount, payer.publicKey);

      // Complete the transfer.
      await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

      // Create instruction.
      const ix = tokenBridge.legacyCompleteTransferWithPayloadWrappedIx(
        program,
        {
          payer: payer.publicKey,
          dstToken: payerToken.address,
        },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "U64Overflow");
    });

    it(`Cannot Invoke \`complete_transfer_with_payload_wrapped\` (Invalid Program Redeemer)`, async () => {
      const wrappedAssetInfo = wrappedMints[0];
      const { chain, address } = wrappedAssetInfo;

      // Mint.
      const mint = tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address));

      // Create payer token account.
      const payerToken = await getOrCreateAssociatedTokenAccount(
        connection,
        payer,
        mint,
        payer.publicKey
      );

      // Amount.
      const amount = BigInt(42069);

      // Create the signed transfer VAA.
      const signedVaa = getSignedTransferVaa(
        address,
        amount,
        anchor.web3.Keypair.generate().publicKey // Pass invalid redeemer authority.
      );

      // Complete the transfer.
      await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

      // Create instruction.
      const ix = tokenBridge.legacyCompleteTransferWithPayloadWrappedIx(
        program,
        {
          payer: payer.publicKey,
          dstToken: payerToken.address,
          redeemerAuthority: payer.publicKey, // Pass different redeemer authority.
        },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "InvalidProgramRedeemer");
    });

    it(`Cannot Invoke \`complete_transfer_with_payload_wrapped\` (Constraint Token Owner)`, async () => {
      const wrappedAssetInfo = wrappedMints[0];
      const { chain, address } = wrappedAssetInfo;

      // Mint.
      const mint = tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address));

      // Create random token account.
      const invalidTokenAccount = await getOrCreateAssociatedTokenAccount(
        connection,
        payer,
        mint,
        anchor.web3.Keypair.generate().publicKey
      );

      // Amount.
      const amount = BigInt(42069);

      // Create the signed transfer VAA.
      const signedVaa = getSignedTransferVaa(address, amount, payer.publicKey);

      // Complete the transfer.
      await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

      // Create instruction.
      const ix = tokenBridge.legacyCompleteTransferWithPayloadWrappedIx(
        program,
        {
          payer: payer.publicKey,
          dstToken: invalidTokenAccount.address,
        },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "ConstraintTokenOwner");
    });

    it(`Cannot Invoke \`complete_transfer_with_payload_wrapped\` (Invalid Token Bridge VAA)`, async () => {
      const wrappedAssetInfo = wrappedMints[0];
      const { chain, address } = wrappedAssetInfo;

      // Mint.
      const mint = tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address));

      // Create random token account.
      const payerToken = await getOrCreateAssociatedTokenAccount(
        connection,
        payer,
        mint,
        payer.publicKey
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

      // Complete the transfer.
      await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

      // Create instruction.
      const ix = tokenBridge.legacyCompleteTransferWithPayloadWrappedIx(
        program,
        {
          payer: payer.publicKey,
          dstToken: payerToken.address,
          wrappedMint: mint,
        },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "InvalidTokenBridgeVaa");
    });
  });
});

function getSignedTransferVaa(
  tokenAddress: Uint8Array,
  amount: bigint,
  recipient: anchor.web3.PublicKey,
  payload?: Buffer,
  targetChain?: number,
  tokenChain?: number
): Buffer {
  const vaaBytes = dummyTokenBridge.publishTransferTokensWithPayload(
    Buffer.from(tokenAddress).toString("hex"),
    tokenChain ?? CHAIN_ID_ETH,
    amount,
    targetChain ?? CHAIN_ID_SOLANA,
    recipient.toBuffer().toString("hex"), // TARGET CONTRACT (redeemer)
    Buffer.from(tryNativeToUint8Array(ETHEREUM_TOKEN_BRIDGE_ADDRESS, 2)),
    payload ?? Buffer.from("deadbeef", "hex"),
    0 // Batch ID
  );
  return guardians.addSignatures(vaaBytes, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
}

async function parallelTxDetails(
  program: tokenBridge.TokenBridgeProgram,
  forkedProgram: tokenBridge.TokenBridgeProgram,
  tokenAccounts: {
    dstToken: Account;
    forkDstToken: Account;
    redeemerAuthority: anchor.web3.Keypair;
  },
  signedVaa,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;
  const { dstToken, forkDstToken, redeemerAuthority } = tokenAccounts;

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
      dstToken: forkDstToken.address,
      redeemerAuthority: redeemerAuthority.publicKey,
    },
    parsed
  );
  return expectIxOkDetails(connection, [ix, forkedIx], [payer, redeemerAuthority]);
}
