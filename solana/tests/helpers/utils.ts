import {
  ConfirmOptions,
  Connection,
  Keypair,
  PublicKey,
  Signer,
  Transaction,
  TransactionInstruction,
  sendAndConfirmTransaction,
} from "@solana/web3.js";
import { expect } from "chai";
import { Err, Ok } from "ts-results";
import { postVaaSolana } from "@certusone/wormhole-sdk";
import { NodeWallet } from "@certusone/wormhole-sdk/lib/cjs/solana";
import { CoreBridgeProgram } from "./coreBridge";
import { TokenBridgeProgram, custodyTokenPda } from "./tokenBridge";
import { getAccount, getAssociatedTokenAddress } from "@solana/spl-token";
import * as tokenBridge from "./tokenBridge";
import * as coreBridge from "./coreBridge";

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
  ixs: TransactionInstruction[],
  signers: Signer[],
  confirmOptions?: ConfirmOptions
) {
  return debugSendAndConfirmTransaction(connection, new Transaction().add(...ixs), signers, {
    logError: true,
    confirmOptions,
  }).then((result) => result.unwrap());
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
  tx: Transaction,
  signers: Signer[],
  options?: {
    logError?: boolean;
    confirmOptions?: ConfirmOptions;
  }
) {
  const logError = options === undefined ? true : options.logError;
  const confirmOptions = options === undefined ? undefined : options.confirmOptions;

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

export const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

export async function verifySignaturesAndPostVaa(
  program: CoreBridgeProgram,
  payer: Keypair,
  signedVaa: Buffer
) {
  const connection = program.provider.connection;
  const wallet = new NodeWallet(payer);
  return postVaaSolana(
    connection,
    wallet.signTransaction,
    program.programId,
    wallet.key(),
    signedVaa
  );
}

export async function parallelPostVaa(connection: Connection, payer: Keypair, signedVaa: Buffer) {
  return Promise.all([
    verifySignaturesAndPostVaa(
      coreBridge.getAnchorProgram(
        connection,
        coreBridge.getProgramId("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o")
      ),
      payer,
      signedVaa
    ),
    verifySignaturesAndPostVaa(
      coreBridge.getAnchorProgram(
        connection,
        coreBridge.getProgramId("worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth")
      ),
      payer,
      signedVaa
    ),
  ]);
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

export enum TransferDirection {
  Out,
  In,
}

export async function expectCorrectTokenBalanceChanges(
  connection: Connection,
  token: PublicKey,
  balancesBefore: TokenBalances,
  direction: TransferDirection
) {
  const program = tokenBridge.getAnchorProgram(
    connection,
    tokenBridge.getProgramId("B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE")
  );
  const forkedProgram = tokenBridge.getAnchorProgram(
    connection,
    tokenBridge.getProgramId("wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb")
  );
  const balancesAfter = await getTokenBalances(program, forkedProgram, token);

  switch (direction) {
    case TransferDirection.Out: {
      const totalTokenBalanceChange = balancesBefore.token - balancesAfter.token;
      expect(totalTokenBalanceChange % BigInt(2)).to.equal(BigInt(0));
      const balanceChange = totalTokenBalanceChange / BigInt(2);
      expect(balancesAfter.custodyToken - balancesBefore.custodyToken).to.equal(balanceChange);
      expect(balancesAfter.forkCustodyToken - balancesBefore.forkCustodyToken).to.equal(
        balanceChange
      );
      return;
    }
    case TransferDirection.In: {
      throw new Error("not implemented yet");
    }
    default: {
      throw new Error("impossible");
    }
  }
}

export function range(start: number, end: number): number[] {
  return Array.from({ length: end - start }, (_, i) => start + i);
}
