import { parseVaa } from "@certusone/wormhole-sdk";
import { createVerifySignaturesInstructions } from "@certusone/wormhole-sdk/lib/cjs/solana/wormhole";
import { BN } from "@coral-xyz/anchor";
import { getAccount } from "@solana/spl-token";
import {
  ConfirmOptions,
  Connection,
  Keypair,
  LAMPORTS_PER_SOL,
  PublicKey,
  Signer,
  TransactionInstruction,
  TransactionMessage,
  VersionedTransaction,
} from "@solana/web3.js";
import { expect } from "chai";
import { Err, Ok } from "ts-results";
import * as coreBridge from "./coreBridge";
import { CoreBridgeProgram } from "./coreBridge";
import { TokenBridgeProgram, custodyTokenPda } from "./tokenBridge";

export type InvalidAccountConfig = {
  label: string;
  contextName: string;
  address: PublicKey;
  errorMsg: string;
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

export function loadProgramBpf(
  publisher: Keypair,
  artifactPath: string,
  bufferAuthority: PublicKey
): PublicKey {
  throw new Error("not implemented yet");
  // Write keypair to temporary file.
  //   const keypath = `${tmpPath()}/payer_${new Date().toISOString()}.json`;
  //   fs.writeFileSync(keypath, JSON.stringify(Array.from(publisher.secretKey)));

  //   // Invoke BPF Loader Upgradeable `write-buffer` instruction.
  //   const buffer = (() => {
  //     const output = execSync(
  //       `solana -k ${keypath} program write-buffer ${artifactPath} -u localhost`
  //     );
  //     return new PublicKey(output.toString().match(/^.{8}([A-Za-z0-9]+)/)[1]);
  //   })();

  //   // Invoke BPF Loader Upgradeable `set-buffer-authority` instruction.
  //   execSync(
  //     `solana -k ${keypath} program set-buffer-authority ${buffer.toString()} --new-buffer-authority ${bufferAuthority.toString()} -u localhost`
  //   );

  //   // Return the pubkey for the buffer (our new program implementation).
  //   return buffer;
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

// const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

export async function invokeVerifySignaturesAndPostVaa(
  program: CoreBridgeProgram,
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
  custodyToken: bigint;
  forkCustodyToken: bigint;
};

export async function getTokenBalances(
  tokenBridgeProgram: TokenBridgeProgram,
  forkTokenBridgeProgram: TokenBridgeProgram,
  token: PublicKey
): Promise<TokenBalances> {
  const connection = tokenBridgeProgram.provider.connection;
  const tokenAccount = await getAccount(connection, token);
  const custodyToken = await getAccount(
    connection,
    custodyTokenPda(tokenBridgeProgram.programId, tokenAccount.mint)
  )
    .then((token) => token.amount)
    .catch((_) => BigInt(0));
  const forkCustodyToken = await getAccount(
    connection,
    custodyTokenPda(forkTokenBridgeProgram.programId, tokenAccount.mint)
  )
    .then((token) => token.amount)
    .catch((_) => BigInt(0));
  return {
    token: tokenAccount.amount,
    custodyToken,
    forkCustodyToken,
  };
}

export function range(start: number, end: number): number[] {
  return Array.from({ length: end - start }, (_, i) => start + i);
}

export async function invokeVerifySignatures(
  program: CoreBridgeProgram,
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

export async function parallelVerifySignatures(
  connection: Connection,
  payer: Keypair,
  signedVaa: Buffer
) {
  const signatureSets = [Keypair.generate(), Keypair.generate()];
  await Promise.all([
    invokeVerifySignatures(
      coreBridge.getAnchorProgram(connection, coreBridge.localnet()),
      payer,
      signatureSets[0],
      signedVaa
    ),
    invokeVerifySignatures(
      coreBridge.getAnchorProgram(connection, coreBridge.mainnet()),
      payer,
      signatureSets[1],
      signedVaa
    ),
  ]);

  return signatureSets;
}

export async function airdrop(connection: Connection, account: PublicKey) {
  const lamports = 69 * LAMPORTS_PER_SOL;
  await connection.requestAirdrop(account, lamports).then((sig) => confirmLatest(connection, sig));

  return lamports;
}
