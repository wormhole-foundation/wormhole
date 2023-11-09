import {
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  ParsedVaa,
  parseVaa,
  tryNativeToUint8Array,
} from "@certusone/wormhole-sdk";
import { BN } from "@coral-xyz/anchor";
import { MockGuardians, MockEmitter } from "@certusone/wormhole-sdk/lib/cjs/mock";
import {
  createAssociatedTokenAccountInstruction,
  getAccount,
  getAssociatedTokenAddressSync,
} from "@solana/spl-token";
import {
  ComputeBudgetProgram,
  ConfirmOptions,
  Connection,
  Keypair,
  LAMPORTS_PER_SOL,
  PublicKey,
  Signer,
  SystemProgram,
  Transaction,
  TransactionInstruction,
  TransactionMessage,
  VersionedTransaction,
  sendAndConfirmTransaction,
} from "@solana/web3.js";
import { expect } from "chai";
import { execSync } from "child_process";
import { Err, Ok } from "ts-results";
import { ethers } from "ethers";
import * as coreBridge from "./coreBridge";
import * as tokenBridge from "./tokenBridge";
import { createSecp256k1Instruction, GOVERNANCE_EMITTER_ADDRESS } from "./";
import {
  createReadOnlyWormholeProgramInterface,
  getVerifySignatureAccounts,
} from "@certusone/wormhole-sdk/lib/cjs/solana/wormhole";

export type InvalidAccountConfig = {
  label: string;
  contextName: string;
  errorMsg: string;
  dataLength?: number;
  owner?: PublicKey;
};

export type NullableAccountConfig = {
  label: string;
  contextName: string;
};

export type InvalidArgConfig = {
  label: string;
  argName: string;
  value: any;
  errorMsg: string;
};

export type MintInfo = {
  mint: PublicKey;
  decimals: number;
};

export type WrappedMintInfo = {
  chain: number;
  address: Uint8Array;
  decimals: number;
};

export function expectDeepEqual<T>(a: T, b: T) {
  expect(JSON.stringify(a)).to.equal(JSON.stringify(b));
}

async function confirmLatest(connection: Connection, signature: string) {
  return connection.getLatestBlockhash().then(({ blockhash, lastValidBlockHeight }) =>
    connection.confirmTransaction(
      {
        blockhash,
        lastValidBlockHeight,
        signature,
      },
      "confirmed"
    )
  );
}

export async function expectIxOk(
  connection: Connection,
  instructions: TransactionInstruction[],
  signers: Signer[],
  confirmOptions?: ConfirmOptions
) {
  return debugSendAndConfirmTransaction(connection, instructions, signers, {
    logError: true,
    confirmOptions,
  }).then((result) => result.unwrap());
}

export async function expectIxErr(
  connection: Connection,
  instructions: TransactionInstruction[],
  signers: Signer[],
  expectedError: string,
  confirmOptions?: ConfirmOptions
) {
  const errorMsg = await debugSendAndConfirmTransaction(connection, instructions, signers, {
    logError: false,
    confirmOptions,
  }).then((result) => {
    if (result.err) {
      return result.toString();
    } else {
      throw new Error("Expected transaction to fail");
    }
  });
  try {
    expect(errorMsg).includes(expectedError);
  } catch (err) {
    console.log(errorMsg);
    throw err;
  }
}

export async function expectIxOkDetails(
  connection: Connection,
  ixs: TransactionInstruction[],
  signers: Signer[],
  confirmOptions?: ConfirmOptions
) {
  const txSig = await expectIxOk(connection, ixs, signers, confirmOptions);
  await confirmLatest(connection, txSig);
  return connection.getTransaction(txSig, {
    commitment: "confirmed",
    maxSupportedTransactionVersion: 0,
  });
}

export function loadProgramBpf(artifactPath: string, bufferAuthority: PublicKey): PublicKey {
  // Write keypair to temporary file.
  const keypath = `${__dirname}/../keys/pFCBP4bhqdSsrWUVTgqhPsLrfEdChBK17vgFM7TxjxQ.json`;

  // Invoke BPF Loader Upgradeable `write-buffer` instruction.
  const buffer = (() => {
    const output = execSync(`solana -u l -k ${keypath} program write-buffer ${artifactPath}`);
    return new PublicKey(output.toString().match(/^.{8}([A-Za-z0-9]+)/)[1]);
  })();

  // Invoke BPF Loader Upgradeable `set-buffer-authority` instruction.
  execSync(
    `solana -k ${keypath} program set-buffer-authority ${buffer.toString()} --new-buffer-authority ${bufferAuthority.toString()} -u localhost`
  );

  // Return the pubkey for the buffer (our new program implementation).
  return buffer;
}

async function debugSendAndConfirmTransaction(
  connection: Connection,
  instructions: TransactionInstruction[],
  signers: Signer[],
  options?: {
    logError?: boolean;
    confirmOptions?: ConfirmOptions;
  }
) {
  const logError = options === undefined ? true : options.logError;
  const confirmOptions = options === undefined ? undefined : options.confirmOptions;

  const latestBlockhash = await connection.getLatestBlockhash();

  const messageV0 = new TransactionMessage({
    payerKey: signers[0].publicKey,
    recentBlockhash: latestBlockhash.blockhash,
    instructions,
  }).compileToV0Message();

  const tx = new VersionedTransaction(messageV0);

  // sign your transaction with the required `Signers`
  tx.sign(signers);

  return connection
    .sendTransaction(tx, confirmOptions)
    .then(async (signature) => {
      await connection.confirmTransaction({ signature, ...latestBlockhash });
      return new Ok(signature);
    })
    .catch((err) => {
      if (logError) {
        console.log(err);
      }
      if (err.logs !== undefined) {
        const logs: string[] = err.logs;
        return new Err(logs.join("\n"));
      } else {
        return new Err(err.message);
      }
    });
}

export const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

export async function invokeVerifySignaturesAndPostVaa(
  program: coreBridge.CoreBridgeProgram,
  payer: Keypair,
  signedVaa: Buffer
) {
  const signatureSet = Keypair.generate();
  await invokeVerifySignatures(program, payer, signatureSet, signedVaa);

  const parsed = parseVaa(signedVaa);
  const args = {
    version: parsed.version,
    guardianSetIndex: parsed.guardianSetIndex,
    timestamp: parsed.timestamp,
    nonce: parsed.nonce,
    emitterChain: parsed.emitterChain,
    emitterAddress: Array.from(parsed.emitterAddress),
    sequence: new BN(parsed.sequence.toString()),
    consistencyLevel: parsed.consistencyLevel,
    payload: parsed.payload,
  };

  return expectIxOk(
    program.provider.connection,
    [
      coreBridge.legacyPostVaaIx(
        program,
        { payer: payer.publicKey, signatureSet: signatureSet.publicKey },
        args
      ),
    ],
    [payer]
  );
}

export async function parallelPostVaa(connection: Connection, payer: Keypair, signedVaa: Buffer) {
  await Promise.all([
    invokeVerifySignaturesAndPostVaa(
      coreBridge.getAnchorProgram(connection, coreBridge.localnet()),
      payer,
      signedVaa
    ),
    invokeVerifySignaturesAndPostVaa(
      coreBridge.getAnchorProgram(connection, coreBridge.mainnet()),
      payer,
      signedVaa
    ),
  ]);

  return parseVaa(signedVaa);
}

export type TokenBalances = {
  token: bigint;
  forkToken: bigint;
  custodyToken: bigint;
  forkCustodyToken: bigint;
};

export async function getTokenBalances(
  tokenBridgeProgram: tokenBridge.TokenBridgeProgram,
  forkTokenBridgeProgram: tokenBridge.TokenBridgeProgram,
  token: PublicKey,
  forkToken?: PublicKey
): Promise<TokenBalances> {
  const connection = tokenBridgeProgram.provider.connection;
  const tokenAccount = await getAccount(connection, token);
  const forkTokenAccount =
    forkToken !== undefined ? await getAccount(connection, forkToken) : tokenAccount;

  const custodyToken = await getAccount(
    connection,
    tokenBridge.custodyTokenPda(tokenBridgeProgram.programId, tokenAccount.mint)
  )
    .then((token) => token.amount)
    .catch((_) => BigInt(0));
  const forkCustodyToken = await getAccount(
    connection,
    tokenBridge.custodyTokenPda(forkTokenBridgeProgram.programId, forkTokenAccount.mint)
  )
    .then((token) => token.amount)
    .catch((_) => BigInt(0));
  return {
    token: tokenAccount.amount,
    forkToken: forkTokenAccount.amount,
    custodyToken,
    forkCustodyToken,
  };
}

export function range(start: number, end: number): number[] {
  return Array.from({ length: end - start }, (_, i) => start + i);
}

export async function invokeVerifySignatures(
  program: coreBridge.CoreBridgeProgram,
  payer: Keypair,
  signatureSet: Keypair,
  signedVaa: Buffer
) {
  const connection = program.provider.connection;
  const ixs = await createVerifySignaturesInstructions(
    connection,
    program.programId,
    payer.publicKey,
    signedVaa,
    signatureSet.publicKey
  );
  if (ixs.length % 2 !== 0) {
    throw new Error("impossible");
  }

  const ixGroups: TransactionInstruction[][] = [];
  for (let i = 0; i < ixs.length; i += 2) {
    ixGroups.push(ixs.slice(i, i + 2));
  }

  return Promise.all(
    ixGroups.map((instructions) =>
      debugSendAndConfirmTransaction(connection, instructions, [payer, signatureSet])
    )
  );
}

export class SignatureSets {
  signatureSet: Keypair;
  forkSignatureSet: Keypair;

  constructor() {
    this.signatureSet = Keypair.generate();
    this.forkSignatureSet = Keypair.generate();
  }
}

export async function parallelVerifySignatures(
  connection: Connection,
  payer: Keypair,
  signedVaa: Buffer
) {
  const { signatureSet, forkSignatureSet } = new SignatureSets();
  await Promise.all([
    invokeVerifySignatures(
      coreBridge.getAnchorProgram(connection, coreBridge.localnet()),
      payer,
      signatureSet,
      signedVaa
    ),
    invokeVerifySignatures(
      coreBridge.getAnchorProgram(connection, coreBridge.mainnet()),
      payer,
      forkSignatureSet,
      signedVaa
    ),
  ]);

  return { signatureSet, forkSignatureSet };
}

export async function airdrop(connection: Connection, account: PublicKey) {
  const lamports = 69 * LAMPORTS_PER_SOL;
  await connection.requestAirdrop(account, lamports).then((sig) => confirmLatest(connection, sig));

  return lamports;
}

export async function createAccountIx(
  connection: Connection,
  programId: PublicKey,
  payer: Keypair,
  accountKeypair: Keypair,
  dataLength: number
) {
  return connection.getMinimumBalanceForRentExemption(dataLength).then((lamports) =>
    SystemProgram.createAccount({
      fromPubkey: payer.publicKey,
      newAccountPubkey: accountKeypair.publicKey,
      space: dataLength,
      lamports,
      programId,
    })
  );
}

export async function createIfNeeded<T>(
  connection: Connection,
  cfg: InvalidAccountConfig,
  payer: Keypair,
  accounts: T
) {
  const created = Keypair.generate();

  if (cfg.dataLength !== undefined) {
    const ix = await createAccountIx(connection, cfg.owner, payer, created, cfg.dataLength);
    await expectIxOk(connection, [ix], [payer, created]);
  }
  accounts[cfg.contextName] = created.publicKey;

  return accounts;
}

function generateSignature(guardians: MockGuardians, message: Buffer, guardianIndex: number) {
  return guardians.addSignatures(message, [guardianIndex]).subarray(7, 7 + 65);
}

export async function createSigVerifyIx(
  program: coreBridge.CoreBridgeProgram,
  guardians: MockGuardians,
  guardianSetIndex: number,
  message: Buffer,
  guardianIndices: number[]
) {
  const guardianSet = coreBridge.GuardianSet.address(program.programId, guardianSetIndex);
  const ethAddresses = await coreBridge.GuardianSet.fromAccountAddress(
    program.provider.connection,
    guardianSet
  ).then((acct) => guardianIndices.map((i) => Buffer.from(acct.keys[i])));
  const signatures = guardianIndices.map((i) => generateSignature(guardians, message, i));

  return createSecp256k1Instruction(
    signatures,
    ethAddresses,
    Buffer.from(ethers.utils.arrayify(ethers.utils.keccak256(message)))
  );
}

export async function processVaa(
  program: coreBridge.CoreBridgeProgram,
  payer: Keypair,
  signedVaa: Buffer,
  guardianSetIndex: number,
  verify: boolean = true
) {
  const connection = program.provider.connection;

  const vaaLen = signedVaa.length;

  const encodedVaa = Keypair.generate();
  const createIx = await createAccountIx(
    program.provider.connection,
    program.programId,
    payer,
    encodedVaa,
    46 + vaaLen
  );

  const initIx = await coreBridge.initEncodedVaaIx(program, {
    writeAuthority: payer.publicKey,
    encodedVaa: encodedVaa.publicKey,
  });

  const endAfterInit = 840;
  const firstProcessIx = await coreBridge.writeEncodedVaaIx(
    program,
    {
      writeAuthority: payer.publicKey,
      draftVaa: encodedVaa.publicKey,
    },
    { index: 0, data: signedVaa.subarray(0, endAfterInit) }
  );

  if (vaaLen > endAfterInit) {
    await expectIxOk(
      program.provider.connection,
      [createIx, initIx, firstProcessIx],
      [payer, encodedVaa]
    );

    const chunkSize = 900;
    for (let start = endAfterInit; start < vaaLen; start += chunkSize) {
      const end = Math.min(start + chunkSize, vaaLen);

      const writeIx = await coreBridge.writeEncodedVaaIx(
        program,
        {
          writeAuthority: payer.publicKey,
          draftVaa: encodedVaa.publicKey,
        },
        { index: start, data: signedVaa.subarray(start, end) }
      );

      if (verify && end === vaaLen) {
        const computeIx = ComputeBudgetProgram.setComputeUnitLimit({ units: 360_000 });
        const verifyIx = await coreBridge.verifyEncodedVaaV1Ix(program, {
          writeAuthority: payer.publicKey,
          draftVaa: encodedVaa.publicKey,
          guardianSet: coreBridge.GuardianSet.address(program.programId, guardianSetIndex),
        });
        await expectIxOk(connection, [computeIx, writeIx, verifyIx], [payer]);
      } else {
        await expectIxOk(connection, [writeIx], [payer]);
      }
    }
  } else if (verify) {
    const computeIx = ComputeBudgetProgram.setComputeUnitLimit({ units: 420_000 });
    const verifyIx = await coreBridge.verifyEncodedVaaV1Ix(program, {
      writeAuthority: payer.publicKey,
      draftVaa: encodedVaa.publicKey,
      guardianSet: coreBridge.GuardianSet.address(program.programId, guardianSetIndex),
    });

    await expectIxOk(
      program.provider.connection,
      [computeIx, createIx, initIx, firstProcessIx, verifyIx],
      [payer, encodedVaa]
    );
  }

  return encodedVaa.publicKey;
}

export async function createAssociatedTokenAccountOffCurve(
  connection: Connection,
  payer: Signer,
  mint: PublicKey,
  owner: PublicKey,
  confirmOptions?: ConfirmOptions
): Promise<PublicKey> {
  const associatedToken = getAssociatedTokenAddressSync(mint, owner, true);

  const transaction = new Transaction().add(
    createAssociatedTokenAccountInstruction(payer.publicKey, associatedToken, owner, mint)
  );

  await sendAndConfirmTransaction(connection, transaction, [payer], confirmOptions);

  return associatedToken;
}

export async function transferLamports(
  connection: Connection,
  payer: Keypair,
  other: PublicKey,
  lamports: bigint = BigInt("1000000000")
) {
  return expectIxOk(
    connection,
    [SystemProgram.transfer({ fromPubkey: payer.publicKey, toPubkey: other, lamports })],
    [payer]
  );
}

export function createInvalidCoreGovernanceVaaFromEth(
  guardians: MockGuardians,
  signatureIndices: number[],
  sequence: number,
  args: {
    targetChain?: number;
    governanceModule?: Buffer;
    governanceAction?: number;
  }
): Buffer {
  let { targetChain, governanceModule, governanceAction } = args;

  // Create mock governance emitter.
  const mockEmitter = new MockEmitter(
    GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
    CHAIN_ID_SOLANA,
    sequence
  );

  if (targetChain === undefined) {
    targetChain = CHAIN_ID_ETH;
  }

  if (governanceModule === undefined) {
    governanceModule = Buffer.from(
      "0x000000000000000000000000000000000000000000546f6b656e427269646765",
      "hex"
    );
  }

  if (governanceAction === undefined) {
    governanceAction = 1;
  }

  // Mock payload.
  let payload = Buffer.alloc(35);
  payload.set(governanceModule, 0);
  payload.writeUint8(governanceAction, 32);
  payload.writeUint16BE(0, 33);

  // Vaa info.
  const published = mockEmitter.publishMessage(
    69, // Nonce.
    payload,
    1 // Finality.
  );
  return guardians.addSignatures(published, signatureIndices);
}

export async function createVerifySignaturesInstructions(
  connection: Connection,
  coreBridgeProgramId: PublicKey,
  payer: PublicKey,
  vaa: Buffer,
  signatureSet: PublicKey
): Promise<TransactionInstruction[]> {
  const parsed = parseVaa(vaa);

  const guardianSetData = await coreBridge.GuardianSet.fromPda(
    connection,
    coreBridgeProgramId,
    parsed.guardianSetIndex
  );

  const guardianSignatures = parsed.guardianSignatures;
  const guardianKeys = guardianSetData.keys;

  const batchSize = 7;
  const instructions: TransactionInstruction[] = [];
  for (let i = 0; i < Math.ceil(guardianSignatures.length / batchSize); ++i) {
    const start = i * batchSize;
    const end = Math.min(guardianSignatures.length, (i + 1) * batchSize);

    const signatureStatus = new Array(19).fill(-1);
    const signatures: Buffer[] = [];
    const keys: Buffer[] = [];
    for (let j = 0; j < end - start; ++j) {
      const item = guardianSignatures.at(j + start)!;
      signatures.push(item.signature);

      const key = guardianKeys.at(item.index)!;
      keys.push(Buffer.from(key));

      signatureStatus[item.index] = j;
    }

    instructions.push(createSecp256k1Instruction(signatures, keys, parsed.hash));
    instructions.push(
      createVerifySignaturesInstruction(
        coreBridgeProgramId,
        payer,
        parsed,
        signatureSet,
        signatureStatus
      )
    );
  }
  return instructions;
}

function createVerifySignaturesInstruction(
  coreBridgeProgramId: PublicKey,
  payer: PublicKey,
  vaa: ParsedVaa,
  signatureSet: PublicKey,
  signatureStatus: number[]
): TransactionInstruction {
  const methods =
    createReadOnlyWormholeProgramInterface(coreBridgeProgramId).methods.verifySignatures(
      signatureStatus
    );

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getVerifySignatureAccounts(coreBridgeProgramId, payer, signatureSet, vaa) as any,
    signers: undefined,
    remainingAccounts: undefined,
    preInstructions: undefined,
    postInstructions: undefined,
  });
}
