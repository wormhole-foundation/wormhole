import { BN } from "@coral-xyz/anchor";
import {
  AccountMeta,
  PublicKey,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { TokenBridgeProgram, coreBridgeProgramId } from "../../..";
import {
  Config,
  mintAuthorityPda,
  RegisteredEmitter,
  WrappedAsset,
  wrappedMintPda,
} from "../../state";
import { PostedVaaV1, Claim } from "../../../../coreBridge";
import { ParsedVaa } from "@certusone/wormhole-sdk";
import { TOKEN_PROGRAM_ID } from "@solana/spl-token";

export type LegacyCompleteTransferWithPayloadWrappedContext = {
  payer: PublicKey;
  config?: PublicKey; // TODO: demonstrate this isn't needed in tests
  vaa?: PublicKey;
  claim?: PublicKey;
  registeredEmitter?: PublicKey;
  dstToken: PublicKey;
  redeemerAuthority?: PublicKey;
  wrappedMint?: PublicKey;
  wrappedAsset?: PublicKey;
  mintAuthority?: PublicKey;
  rent?: PublicKey;
  coreBridgeProgram?: PublicKey;
};

export function legacyCompleteTransferWithPayloadWrappedAccounts(
  program: TokenBridgeProgram,
  accounts: LegacyCompleteTransferWithPayloadWrappedContext,
  parsedVaa: ParsedVaa,
  legacyRegisteredEmitterDerive: boolean,
  tokenAddressOverride?: number[],
  tokenChainOverride?: number
) {
  const programId = program.programId;
  const { emitterChain, emitterAddress, sequence, hash, payload } = parsedVaa;

  const tokenAddress = Array.from(payload.subarray(33, 65));
  const tokenChain = payload.readUInt16BE(65);

  let {
    payer,
    config,
    vaa,
    claim,
    registeredEmitter,
    dstToken,
    redeemerAuthority,
    wrappedMint,
    wrappedAsset,
    mintAuthority,
    rent,
    coreBridgeProgram,
  } = accounts;

  if (coreBridgeProgram === undefined) {
    coreBridgeProgram = coreBridgeProgramId(program);
  }

  if (config === undefined) {
    config = Config.address(programId);
  }

  if (vaa === undefined) {
    vaa = PostedVaaV1.address(coreBridgeProgram, Array.from(hash));
  }

  if (claim === undefined) {
    claim = Claim.address(
      programId,
      Array.from(emitterAddress),
      emitterChain,
      new BN(sequence.toString())
    );
  }

  if (registeredEmitter === undefined) {
    registeredEmitter = RegisteredEmitter.address(
      programId,
      emitterChain,
      legacyRegisteredEmitterDerive ? Array.from(emitterAddress) : undefined
    );
  }

  if (redeemerAuthority === undefined) {
    redeemerAuthority = payer;
  }

  if (wrappedMint === undefined) {
    wrappedMint = wrappedMintPda(
      programId,
      tokenChainOverride ?? tokenChain,
      tokenAddressOverride ?? tokenAddress
    );
  }

  if (wrappedAsset === undefined) {
    wrappedAsset = WrappedAsset.address(programId, wrappedMint);
  }

  if (mintAuthority === undefined) {
    mintAuthority = mintAuthorityPda(programId);
  }

  if (rent === undefined) {
    rent = SYSVAR_RENT_PUBKEY;
  }

  return {
    payer,
    config,
    vaa,
    claim,
    registeredEmitter,
    dstToken,
    redeemerAuthority,
    wrappedMint,
    wrappedAsset,
    mintAuthority,
    rent,
    coreBridgeProgram,
  };
}

export function legacyCompleteTransferWithPayloadWrappedIx(
  program: TokenBridgeProgram,
  accounts: LegacyCompleteTransferWithPayloadWrappedContext,
  parsedVaa: ParsedVaa,
  legacyRegisteredEmitterDerive?: boolean,
  tokenAddressOverride?: number[],
  tokenChainOverride?: number
) {
  if (legacyRegisteredEmitterDerive === undefined) {
    legacyRegisteredEmitterDerive = true;
  }

  const {
    payer,
    config,
    vaa,
    claim,
    registeredEmitter,
    dstToken,
    redeemerAuthority,
    wrappedMint,
    wrappedAsset,
    mintAuthority,
    rent,
    coreBridgeProgram,
  } = legacyCompleteTransferWithPayloadWrappedAccounts(
    program,
    accounts,
    parsedVaa,
    legacyRegisteredEmitterDerive,
    tokenAddressOverride,
    tokenChainOverride
  );

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
      pubkey: registeredEmitter,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: dstToken,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: redeemerAuthority,
      isWritable: false,
      isSigner: true,
    },
    {
      pubkey: dstToken, // NOTE: This exists because of a bug in the legacy program.
      isWritable: false, // TODO: check this
      isSigner: false,
    },
    {
      pubkey: wrappedMint,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: wrappedAsset,
      isWritable: false,
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
      pubkey: coreBridgeProgram!,
      isWritable: false,
      isSigner: false,
    },
  ];

  const data = Buffer.alloc(1, 10);

  return new TransactionInstruction({
    keys,
    programId: program.programId,
    data,
  });
}
