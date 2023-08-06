import { ParsedVaa } from "@certusone/wormhole-sdk";
import { BN } from "@coral-xyz/anchor";
import {
  AccountMeta,
  PublicKey,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { CoreBridgeProgram } from "../../..";
import {
  BridgeProgramData,
  Claim,
  FeeCollector,
  PostedVaaV1,
} from "../../state";

export type LegacyTransferFeesContext = {
  payer: PublicKey;
  bridge?: PublicKey;
  postedVaa?: PublicKey;
  claim?: PublicKey;
  feeCollector?: PublicKey;
  recipient: PublicKey;
  rent?: PublicKey;
};

export function legacyTransferFeesIx(
  program: CoreBridgeProgram,
  accounts: LegacyTransferFeesContext,
  parsed: ParsedVaa
) {
  const programId = program.programId;
  const { emitterChain, emitterAddress, sequence, hash } = parsed;

  let { payer, bridge, postedVaa, claim, feeCollector, recipient, rent } =
    accounts;

  if (bridge === undefined) {
    bridge = BridgeProgramData.address(programId);
  }

  if (postedVaa === undefined) {
    postedVaa = PostedVaaV1.address(programId, Array.from(hash));
  }

  if (claim === undefined) {
    claim = Claim.address(
      programId,
      emitterChain,
      Array.from(emitterAddress),
      new BN(sequence.toString())
    );
  }

  if (feeCollector === undefined) {
    feeCollector = FeeCollector.address(programId);
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
      pubkey: bridge,
      isWritable: true,
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
      pubkey: feeCollector,
      isWritable: true,
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
  ];
  const data = Buffer.alloc(1, 4);

  return new TransactionInstruction({
    keys,
    programId,
    data,
  });
}
