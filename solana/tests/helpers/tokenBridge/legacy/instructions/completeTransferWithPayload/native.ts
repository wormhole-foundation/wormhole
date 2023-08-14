import { BN } from "@coral-xyz/anchor";
import {
  AccountMeta,
  PublicKey,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { TokenBridgeProgram, coreBridgeProgramId } from "../../..";
import { Config, custodyAuthorityPda, custodyTokenPda, RegisteredEmitter } from "../../state";
import { PostedVaaV1, Claim } from "../../../../coreBridge";
import { ParsedVaa } from "@certusone/wormhole-sdk";
import { getAssociatedTokenAddressSync, TOKEN_PROGRAM_ID } from "@solana/spl-token";

export type LegacyCompleteTransferWithPayloadNativeContext = {
  payer: PublicKey;
  config?: PublicKey; // TODO: demonstrate this isn't needed in tests
  postedVaa?: PublicKey;
  claim?: PublicKey;
  registeredEmitter?: PublicKey;
  recipientToken: PublicKey;
  redeemerAuthority?: PublicKey;
  custodyToken?: PublicKey;
  mint: PublicKey;
  custodyAuthority?: PublicKey;
  rent?: PublicKey;
  coreBridgeProgram?: PublicKey;
};

export function legacyCompleteTransferWithPayloadNativeIx(
  program: TokenBridgeProgram,
  accounts: LegacyCompleteTransferWithPayloadNativeContext,
  parsedVaa: ParsedVaa
) {
  const programId = program.programId;
  const { emitterChain, emitterAddress, sequence, hash } = parsedVaa;

  let {
    payer,
    config,
    postedVaa,
    claim,
    registeredEmitter,
    recipientToken,
    redeemerAuthority,
    custodyToken,
    mint,
    custodyAuthority,
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
      Array.from(emitterAddress)
    );
  }

  if (redeemerAuthority === undefined) {
    redeemerAuthority = payer;
  }

  if (custodyToken === undefined) {
    custodyToken = custodyTokenPda(programId, mint);
  }

  if (custodyAuthority === undefined) {
    custodyAuthority = custodyAuthorityPda(programId);
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
      pubkey: redeemerAuthority,
      isWritable: false,
      isSigner: true,
    },
    {
      pubkey: recipientToken, // NOTE: This exists because of a bug in the legacy program.
      isWritable: false, // TODO: check this
      isSigner: false,
    },
    {
      pubkey: custodyToken,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: mint,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: custodyAuthority,
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

  const data = Buffer.alloc(1, 9);

  return new TransactionInstruction({
    keys,
    programId,
    data,
  });
}
