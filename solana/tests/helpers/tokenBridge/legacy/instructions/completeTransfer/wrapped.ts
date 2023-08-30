import { ParsedVaa } from "@certusone/wormhole-sdk";
import { BN } from "@coral-xyz/anchor";
import { TOKEN_PROGRAM_ID, getAssociatedTokenAddressSync } from "@solana/spl-token";
import {
  AccountMeta,
  PublicKey,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { TokenBridgeProgram, coreBridgeProgramId } from "../../..";
import { Claim, PostedVaaV1 } from "../../../../coreBridge";
import {
  Config,
  RegisteredEmitter,
  WrappedAsset,
  mintAuthorityPda,
  wrappedMintPda,
} from "../../state";

export type LegacyCompleteTransferWrappedContext = {
  payer: PublicKey;
  config?: PublicKey; // TODO: demonstrate this isn't needed in tests
  postedVaa?: PublicKey;
  claim?: PublicKey;
  registeredEmitter?: PublicKey;
  recipientToken: PublicKey;
  payerToken?: PublicKey;
  wrappedMint?: PublicKey;
  wrappedAsset?: PublicKey;
  mintAuthority?: PublicKey;
  rent?: PublicKey;
  coreBridgeProgram?: PublicKey;
};

export function legacyCompleteTransferWrappedIx(
  program: TokenBridgeProgram,
  accounts: LegacyCompleteTransferWrappedContext,
  parsedVaa: ParsedVaa,
  legacyRegisteredEmitterDerive: boolean = true,
  tokenAddressOverride?: number[]
) {
  const programId = program.programId;
  const { emitterChain, emitterAddress, sequence, hash, payload } = parsedVaa;

  let tokenAddress = Array.from(payload.subarray(33, 65));
  const tokenChain = payload.readUInt16BE(65);

  let {
    payer,
    config,
    postedVaa,
    claim,
    registeredEmitter,
    recipientToken,
    payerToken,
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

  if (postedVaa === undefined) {
    postedVaa = PostedVaaV1.address(coreBridgeProgram, Array.from(hash));
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

  if (wrappedMint === undefined) {
    wrappedMint = wrappedMintPda(programId, tokenChain, tokenAddressOverride ?? tokenAddress);
  }

  if (payerToken === undefined) {
    payerToken = getAssociatedTokenAddressSync(wrappedMint, payer);
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
      pubkey: postedVaa,
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
      pubkey: recipientToken,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: payerToken,
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
      pubkey: coreBridgeProgram,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: TOKEN_PROGRAM_ID,
      isWritable: false,
      isSigner: false,
    },
  ];

  const data = Buffer.alloc(1, 3);

  return new TransactionInstruction({
    keys,
    programId,
    data,
  });
}
