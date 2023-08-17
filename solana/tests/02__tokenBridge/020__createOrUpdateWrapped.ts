import {
  ParsedVaa,
  coalesceChainId,
  parseAttestMetaPayload,
  parseVaa,
  tryHexToNativeString,
  tryNativeToHexString,
} from "@certusone/wormhole-sdk";
import { MockGuardians, MockTokenBridge } from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as anchor from "@coral-xyz/anchor";
import { Metadata } from "@metaplex-foundation/mpl-token-metadata";
import { getMint } from "@solana/spl-token";
import { expect } from "chai";
import {
  ETHEREUM_DEADBEEF_TOKEN_ADDRESS,
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  ETHEREUM_STEAK_TOKEN_ADDRESS,
  GUARDIAN_KEYS,
  expectDeepEqual,
  expectIxOk,
  parallelPostVaa,
  ETHEREUM_TOKEN_ADDRESS_MAX_ONE,
  ETHEREUM_TOKEN_ADDRESS_MAX_TWO,
} from "../helpers";
import * as tokenBridge from "../helpers/tokenBridge";

const GUARDIAN_SET_INDEX = 2;
const ETHEREUM_TOKEN_BRIDGE_SEQ = 2_020_000;

const ethereumTokenBridge = new MockTokenBridge(
  tryNativeToHexString(ETHEREUM_TOKEN_BRIDGE_ADDRESS, "ethereum"),
  coalesceChainId("ethereum"),
  1,
  ETHEREUM_TOKEN_BRIDGE_SEQ - 1
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

describe("Token Bridge -- Legacy Instruction: Create or Update Wrapped", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = tokenBridge.getAnchorProgram(connection, tokenBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;

  const forkedProgram = tokenBridge.getAnchorProgram(connection, tokenBridge.mainnet());

  describe("Ok", () => {
    it("Invoke `create_or_update_wrapped` for New Asset (18 decimals)", async () => {
      const signedVaa = defaultVaa();

      const parsed = await parallelTxOk(
        program,
        forkedProgram,
        { payer: payer.publicKey },
        signedVaa,
        payer
      );

      // Check metadata.
      const {
        wrappedAsset,
        dataV1,
        decomposedMetadata: metadata,
      } = await expectCorrectData(program, parsed);
      const {
        wrappedAsset: forkWrappedAsset,
        dataV1: forkDataV1,
        decomposedMetadata: forkMetadata,
      } = await expectCorrectData(forkedProgram, parsed);

      expectDeepEqual(wrappedAsset, forkWrappedAsset);
      expectDeepEqual(metadata, forkMetadata);

      expect(dataV1.symbol).equals(forkDataV1.symbol);
      expect(dataV1.sellerFeeBasisPoints).equals(forkDataV1.sellerFeeBasisPoints);

      // Note the differences between the new implementation and the fork.
      expect(dataV1.name).equals("Dead beef. Moo.".padEnd(32, "\x00"));
      expect(forkDataV1.name).equals("Dead beef. Moo. (Wormhole)".padEnd(32, "\x00"));

      // Instead of adding the suffix " (Wormhole)", we use the URI to describe the original asset.
      const uri: { wormholeChainId: number; canonicalAddress: string; nativeDecimals: number } =
        JSON.parse(dataV1.uri.replace(/\0/g, ""));
      expectDeepEqual(uri, {
        wormholeChainId: 2,
        canonicalAddress: "0x000000000000000000000000deadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
        nativeDecimals: 18,
      });
    });

    it("Invoke `create_or_update_wrapped` to Update Asset", async () => {
      const newSymbol = "BEEEEEEEEEEEEF";
      expect(newSymbol).length.greaterThan(10);

      const newName = "Beef. Moooooooooooooooooooooooooooooooooooooooooooooooo.";
      expect(newName).length.greaterThan(32);

      {
        const mint = tokenBridge.wrappedMintPda(
          program.programId,
          2,
          Array.from(ETHEREUM_DEADBEEF_TOKEN_ADDRESS)
        );
        const {
          data: { name, symbol },
        } = await Metadata.fromAccountAddress(connection, tokenBridge.tokenMetadataPda(mint));
        expect(symbol).not.equals(newSymbol.padEnd(10, "\x00"));
        expect(name).not.equals(newName.padEnd(10, "\x00"));
      }

      const signedVaa = defaultVaa({ symbol: newSymbol, name: newName });

      const parsed = await parallelTxOk(
        program,
        forkedProgram,
        { payer: payer.publicKey },
        signedVaa,
        payer
      );

      // Check metadata.
      const { dataV1 } = await expectCorrectData(program, parsed);
      const { dataV1: forkDataV1 } = await expectCorrectData(forkedProgram, parsed);

      expect(dataV1.symbol).equals(forkDataV1.symbol);

      // Let's only check the rewrite since we know the names are different between this and the
      // fork.
      expect(dataV1.name).equals(newName.substring(0, 32));
    });

    it("Invoke `create_or_update_wrapped` for New Asset (7 decimals)", async () => {
      const signedVaa = defaultVaa({
        symbol: "STEAK",
        name: "medium rare",
        decimals: 7,
        address: ETHEREUM_STEAK_TOKEN_ADDRESS,
      });

      const parsed = await parallelTxOk(
        program,
        forkedProgram,
        { payer: payer.publicKey },
        signedVaa,
        payer
      );

      // Check metadata.
      const {
        wrappedAsset,
        dataV1,
        decomposedMetadata: metadata,
      } = await expectCorrectData(program, parsed);
      const {
        wrappedAsset: forkWrappedAsset,
        dataV1: forkDataV1,
        decomposedMetadata: forkMetadata,
      } = await expectCorrectData(forkedProgram, parsed);

      expectDeepEqual(wrappedAsset, forkWrappedAsset);
      expectDeepEqual(metadata, forkMetadata);

      expect(dataV1.symbol).equals(forkDataV1.symbol);
      expect(dataV1.sellerFeeBasisPoints).equals(forkDataV1.sellerFeeBasisPoints);

      // Note the differences between the new implementation and the fork.
      expect(dataV1.name).equals("medium rare".padEnd(32, "\x00"));
      expect(forkDataV1.name).equals("medium rare (Wormhole)".padEnd(32, "\x00"));

      // Instead of adding the suffix " (Wormhole)", we use the URI to describe the original asset.
      const uri: { wormholeChainId: number; canonicalAddress: string; nativeDecimals: number } =
        JSON.parse(dataV1.uri.replace(/\0/g, ""));
      expectDeepEqual(uri, {
        wormholeChainId: 2,
        canonicalAddress: "0x000000000000000000000000beefdeadbeefdeadbeefdeadbeefdeadbeefdead",
        nativeDecimals: 7,
      });
    });

    it("Invoke `create_or_update_wrapped` for Boundary Test Assets", async () => {
      for (let i = 0; i < 2; i++) {
        const signedVaa = defaultVaa({
          symbol: `MAX`,
          name: `Max Amount`,
          decimals: 8,
          address: i == 0 ? ETHEREUM_TOKEN_ADDRESS_MAX_ONE : ETHEREUM_TOKEN_ADDRESS_MAX_TWO,
        });

        await parallelTxOk(program, forkedProgram, { payer: payer.publicKey }, signedVaa, payer);
      }
    });
  });
});

function defaultVaa(args?: {
  symbol?: string;
  name?: string;
  decimals?: number;
  address?: Uint8Array;
}): Buffer {
  if (args === undefined) {
    args = {};
  }

  let { symbol, name, decimals, address } = args;
  if (symbol === undefined) {
    symbol = "DEADBEEF";
  }
  if (name === undefined) {
    name = "Dead beef. Moo.";
  }
  if (decimals === undefined) {
    decimals = 18;
  }
  if (address === undefined) {
    address = ETHEREUM_DEADBEEF_TOKEN_ADDRESS;
  }
  const nonce = 420;
  const timestamp = 12345678;
  const published = ethereumTokenBridge.publishAttestMeta(
    Buffer.from(address).toString("hex"),
    decimals,
    symbol,
    name,
    nonce,
    timestamp
  );
  return guardians.addSignatures(published, [0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 14]);
}

async function parallelTxOk(
  program: tokenBridge.TokenBridgeProgram,
  forkedProgram: tokenBridge.TokenBridgeProgram,
  accounts: tokenBridge.LegacyCreateOrUpdateWrappedContext,
  signedVaa: Buffer,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;

  // Post the VAAs.
  await parallelPostVaa(connection, payer, signedVaa);

  const parsed = parseVaa(signedVaa);
  const ix = tokenBridge.legacyCreateOrUpdateWrappedIx(program, accounts, parsed);
  const forkedIx = tokenBridge.legacyCreateOrUpdateWrappedIx(forkedProgram, accounts, parsed);

  await expectIxOk(connection, [ix, forkedIx], [payer]);

  return parsed;
}

async function expectCorrectData(program: tokenBridge.TokenBridgeProgram, parsed: ParsedVaa) {
  const programId = program.programId;
  const connection = program.provider.connection;

  const {
    tokenChain,
    tokenAddress,
    decimals: nativeDecimals,
    symbol,
  } = parseAttestMetaPayload(parsed.payload);

  const mint = tokenBridge.wrappedMintPda(programId, tokenChain, Array.from(tokenAddress));
  const mintInfo = await getMint(connection, mint);
  expect(mintInfo.isInitialized).is.true;
  expect(mintInfo.supply).equals(BigInt(0));
  expect(mintInfo.decimals).equals(nativeDecimals > 8 ? 8 : nativeDecimals);

  const mintAuthority = tokenBridge.mintAuthorityPda(programId);
  expectDeepEqual(mintInfo.mintAuthority, mintAuthority);

  // Check wrapped asset.
  const wrappedAsset = await tokenBridge.WrappedAsset.fromPda(connection, programId, mint);
  expectDeepEqual(wrappedAsset, {
    tokenChain,
    tokenAddress: Array.from(tokenAddress),
    nativeDecimals,
  });

  // Check Token Metadata.
  const metadata = await Metadata.fromAccountAddress(
    connection,
    tokenBridge.tokenMetadataPda(mint)
  );
  const {
    updateAuthority,
    mint: expectedMint,
    editionNonce,
    data: dataV1,
    ...decomposedMetadata
  } = metadata;
  expectDeepEqual(updateAuthority, mintAuthority);
  expectDeepEqual(mint, expectedMint);

  const { symbol: expectedSymbol } = dataV1;

  if (symbol.length >= 10) {
    expect(symbol.substring(0, 10)).to.equal(expectedSymbol);
  } else {
    expect(symbol.padEnd(10, "\x00")).to.equal(expectedSymbol);
  }

  return { wrappedAsset, decomposedMetadata, dataV1 };
}
