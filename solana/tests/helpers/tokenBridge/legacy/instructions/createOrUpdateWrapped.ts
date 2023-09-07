import { ParsedVaa } from "@certusone/wormhole-sdk";
import { BN } from "@coral-xyz/anchor";
import { TOKEN_PROGRAM_ID } from "@solana/spl-token";
import {
  AccountMeta,
  PublicKey,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { TokenBridgeProgram, coreBridgeProgramId } from "../..";
import * as coreBridge from "../../../coreBridge";
import {
  Claim,
  Config,
  RegisteredEmitter,
  TOKEN_METADATA_PROGRAM_ID,
  WrappedAsset,
  mintAuthorityPda,
  tokenMetadataPda,
  wrappedMintPda,
} from "../state";

export type LegacyCreateOrUpdateWrappedContext = {
  payer: PublicKey;
  config?: PublicKey; // TODO: demonstrate this isn't needed in tests
  registeredEmitter?: PublicKey;
  vaa?: PublicKey;
  claim?: PublicKey;
  wrappedMint?: PublicKey;
  wrappedAsset?: PublicKey;
  tokenMetadata?: PublicKey;
  mintAuthority?: PublicKey;
  rent?: PublicKey;
  coreBridgeProgram?: PublicKey;
  mplTokenMetadataProgram?: PublicKey;
};

export function legacyCreateOrUpdateWrappedIx(
  program: TokenBridgeProgram,
  accounts: LegacyCreateOrUpdateWrappedContext,
  parsed: ParsedVaa
) {
  const programId = program.programId;
  const { emitterChain, emitterAddress, sequence, hash, payload } = parsed;

  const tokenAddress = Array.from(payload.subarray(1, 33));
  const tokenChain = payload.readUInt16BE(33);

  let {
    payer,
    config,
    registeredEmitter,
    vaa,
    claim,
    wrappedMint,
    wrappedAsset,
    tokenMetadata,
    mintAuthority,
    rent,
    coreBridgeProgram,
    mplTokenMetadataProgram,
  } = accounts;

  if (coreBridgeProgram === undefined) {
    coreBridgeProgram = coreBridgeProgramId(program);
  }

  if (config === undefined) {
    config = Config.address(programId);
  }

  if (registeredEmitter === undefined) {
    registeredEmitter = RegisteredEmitter.address(
      programId,
      emitterChain,
      Array.from(emitterAddress)
    );
  }

  if (vaa === undefined) {
    vaa = coreBridge.PostedVaaV1.address(coreBridgeProgram, Array.from(hash));
  }

  if (claim === undefined) {
    claim = Claim.address(
      programId,
      Array.from(emitterAddress),
      emitterChain,
      new BN(sequence.toString())
    );
  }

  if (wrappedMint === undefined) {
    wrappedMint = wrappedMintPda(programId, tokenChain, tokenAddress);
  }

  if (wrappedAsset === undefined) {
    wrappedAsset = WrappedAsset.address(programId, wrappedMint);
  }

  if (tokenMetadata === undefined) {
    tokenMetadata = tokenMetadataPda(wrappedMint);
  }

  if (mintAuthority === undefined) {
    mintAuthority = mintAuthorityPda(programId);
  }

  if (rent === undefined) {
    rent = SYSVAR_RENT_PUBKEY;
  }

  if (mplTokenMetadataProgram === undefined) {
    mplTokenMetadataProgram = TOKEN_METADATA_PROGRAM_ID;
  }

  const keys: AccountMeta[] = [
    {
      pubkey: payer,
      isWritable: true,
      isSigner: true,
    },
    {
      pubkey: config,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: registeredEmitter,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: vaa,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: claim,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: wrappedMint,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: wrappedAsset,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: tokenMetadata,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: mintAuthority,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: rent,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: SystemProgram.programId,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: TOKEN_PROGRAM_ID,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: mplTokenMetadataProgram,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: coreBridgeProgram,
      isWritable: false,
      isSigner: false,
    },
  ];

  const data = Buffer.alloc(1, 7);

  return new TransactionInstruction({
    keys,
    programId,
    data,
  });
}
