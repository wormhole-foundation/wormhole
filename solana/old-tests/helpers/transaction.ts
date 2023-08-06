import {
  ConfirmOptions,
  Connection,
  LAMPORTS_PER_SOL,
  PublicKey,
  Signer,
  Transaction,
  TransactionInstruction,
  sendAndConfirmTransaction,
} from "@solana/web3.js";
import { expect } from "chai";
import { Err, Ok } from "ts-results";

async function confirmLatest(connection: Connection, signature: string) {
  return connection
    .getLatestBlockhash()
    .then(({ blockhash, lastValidBlockHeight }) =>
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
  ixs: TransactionInstruction[],
  signers: Signer[],
  confirmOptions?: ConfirmOptions
) {
  return debugSendAndConfirmTransaction(
    connection,
    new Transaction().add(...ixs),
    signers,
    {
      logError: true,
      confirmOptions,
    }
  ).then((result) => result.unwrap());
}

export async function expectIxErr(
  connection: Connection,
  ixs: TransactionInstruction[],
  signers: Signer[],
  expectedError: string,
  confirmOptions?: ConfirmOptions
) {
  const errorMsg = await debugSendAndConfirmTransaction(
    connection,
    new Transaction().add(...ixs),
    signers,
    {
      logError: false,
      confirmOptions,
    }
  ).then((result) => {
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

export async function expectIxTransactionDetails(
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

export async function airdrop(connection: Connection, account: PublicKey) {
  const lamports = 69 * LAMPORTS_PER_SOL;
  await connection
    .requestAirdrop(account, lamports)
    .then((sig) => confirmLatest(connection, sig));

  return lamports;
}

async function debugSendAndConfirmTransaction(
  connection: Connection,
  tx: Transaction,
  signers: Signer[],
  options?: {
    logError?: boolean;
    confirmOptions?: ConfirmOptions;
  }
) {
  const logError = options === undefined ? true : options.logError;
  const confirmOptions =
    options === undefined ? undefined : options.confirmOptions;

  return sendAndConfirmTransaction(connection, tx, signers, confirmOptions)
    .then((sig) => new Ok(sig))
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
