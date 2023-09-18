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
import { TokenBridgeProgram, coreBridgeProgramId } from "../../..";
import { Claim, PostedVaaV1 } from "../../../../coreBridge";
import { Config, RegisteredEmitter, custodyAuthorityPda, custodyTokenPda } from "../../state";

export type LegacyCompleteTransferWithPayloadNativeContext = {
  payer: PublicKey;
  config?: PublicKey; // TODO: demonstrate this isn't needed in tests
  vaa?: PublicKey;
  claim?: PublicKey;
  registeredEmitter?: PublicKey;
  dstToken: PublicKey;
  redeemerAuthority?: PublicKey;
  custodyToken?: PublicKey;
  mint: PublicKey;
  custodyAuthority?: PublicKey;
  rent?: PublicKey;
  coreBridgeProgram?: PublicKey;
};

export function legacyCompleteTransferWithPayloadNativeAccounts(
  program: TokenBridgeProgram,
  accounts: LegacyCompleteTransferWithPayloadNativeContext,
  parsedVaa: ParsedVaa,
  legacyRegisteredEmitterDerive: boolean
): LegacyCompleteTransferWithPayloadNativeContext {
  const programId = program.programId;
  const { emitterChain, emitterAddress, sequence, hash } = parsedVaa;

  let {
    payer,
    config,
    vaa,
    claim,
    registeredEmitter,
    dstToken,
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

  if (custodyToken === undefined) {
    custodyToken = custodyTokenPda(programId, mint);
  }

  if (custodyAuthority === undefined) {
    custodyAuthority = custodyAuthorityPda(programId);
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
    custodyToken,
    mint,
    custodyAuthority,
    rent,
    coreBridgeProgram,
  };
}

export function legacyCompleteTransferWithPayloadNativeIx(
  program: TokenBridgeProgram,
  accounts: LegacyCompleteTransferWithPayloadNativeContext,
  parsedVaa: ParsedVaa,
  legacyRegisteredEmitterDerive: boolean = true
) {
  const {
    payer,
    config,
    vaa,
    claim,
    registeredEmitter,
    dstToken,
    redeemerAuthority,
    custodyToken,
    mint,
    custodyAuthority,
    rent,
    coreBridgeProgram,
  } = legacyCompleteTransferWithPayloadNativeAccounts(
    program,
    accounts,
    parsedVaa,
    legacyRegisteredEmitterDerive
  );

  const keys: AccountMeta[] = [
    {
      pubkey: payer,
      isWritable: true,
      isSigner: true,
    },
    {
      pubkey: config!,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: vaa!,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: claim!,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: registeredEmitter!,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: dstToken,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: redeemerAuthority!,
      isWritable: false,
      isSigner: true,
    },
    {
      pubkey: dstToken, // NOTE: This exists because of a bug in the legacy program.
      isWritable: false, // TODO: check this
      isSigner: false,
    },
    {
      pubkey: custodyToken!,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: mint,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: custodyAuthority!,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: rent!,
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

  const data = Buffer.alloc(1, 9);

  return new TransactionInstruction({
    keys,
    programId: program.programId,
    data,
  });
}
