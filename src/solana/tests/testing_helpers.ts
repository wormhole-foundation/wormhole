

import * as fs from "fs"
import * as toml from "toml"

import { Connection, LAMPORTS_PER_SOL, PublicKey, Signer, Transaction, TransactionInstruction, ComputeBudgetProgram, sendAndConfirmTransaction } from "@solana/web3.js"
import { NodeWallet, postVaaSolana, signSendAndConfirmTransaction } from "@certusone/wormhole-sdk/lib/cjs/solana"
import { MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock"
import { CONTRACTS } from "@certusone/wormhole-sdk";

import { expect, use as chaiUse } from "chai"
import chaiAsPromised from "chai-as-promised"
chaiUse(chaiAsPromised)

// Copied from @wormhole-foundation/wormhole-scaffolding/solana/ts/helpers/utils.ts
// TODO: There's probably some way to import this?

export const MOCK_GUARDIANS = new MockGuardians(0, ["cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0"])

class SendIxError extends Error {
  logs: string;

  constructor(originalError: Error & { logs?: string[] }) {
    //The newlines don't actually show up correctly in chai's assertion error, but at least
    // we have all the information and can just replace '\n' with a newline manually to see
    // what's happening without having to change the code.
    const logs = originalError.logs?.join('\n') || "error had no logs";
    super(originalError.message + "\nlogs:\n" + logs);
    this.stack = originalError.stack;
    this.logs = logs;
  }
}

export const boilerPlateReduction = (connection: Connection, defaultSigner: Signer) => {
  // for signing wormhole messages
  const defaultNodeWallet = NodeWallet.fromSecretKey(defaultSigner.secretKey);

  const payerToWallet = (payer?: Signer) =>
    !payer || payer === defaultSigner
    ? defaultNodeWallet
    : NodeWallet.fromSecretKey(payer.secretKey);
  
  const requestAirdrop = async (account: PublicKey) =>
    connection.confirmTransaction(
      await connection.requestAirdrop(account, 1000 * LAMPORTS_PER_SOL)
    );
  
  const guardianSign = (message: Buffer) =>
    MOCK_GUARDIANS.addSignatures(message, [0])

  const postSignedMsgAsVaaOnSolana = async (signedMsg: Buffer, payer?: Signer) => {
    const wallet = payerToWallet(payer);
    const coreV1 = new PublicKey(CONTRACTS.DEVNET.solana.core);
    await postVaaSolana(
      connection,
      wallet.signTransaction,
      coreV1,
      wallet.key(),
      signedMsg
    );
  }

  const sendAndConfirmIx = async (
    ix: TransactionInstruction | Promise<TransactionInstruction>,
    signerOrSignersOrComputeUnits?: Signer | Signer[] | number,
    computeUnits?: number,
  ) => {
    let [signers, units] = (() => {
      if (!signerOrSignersOrComputeUnits)
        return [[defaultSigner], computeUnits];

      if (typeof signerOrSignersOrComputeUnits === "number") {
        if(computeUnits !== undefined)
          throw new Error("computeUnits can't be specified twice");
        return [[defaultSigner], signerOrSignersOrComputeUnits];
      }

      return [
        Array.isArray(signerOrSignersOrComputeUnits)
          ? signerOrSignersOrComputeUnits
          : [signerOrSignersOrComputeUnits],
          computeUnits
      ];
    })();

    const tx = new Transaction().add(await ix);
    if (units)
      tx.add(ComputeBudgetProgram.setComputeUnitLimit({units}));
    try {
      return await sendAndConfirmTransaction(connection, tx, signers);
    }
    catch (error: any) {
      throw new SendIxError(error);
    }
  }
  
  const expectIxToSucceed = async (
    ix: TransactionInstruction | Promise<TransactionInstruction>,
    signerOrSignersOrComputeUnits?: Signer | Signer[] | number,
    computeUnits?: number,
  ) =>
    expect(sendAndConfirmIx(ix, signerOrSignersOrComputeUnits, computeUnits)).to.be.fulfilled;
    
  const expectIxToFailWithError = async (
    ix: TransactionInstruction | Promise<TransactionInstruction>,
    errorMessage: string,
    signerOrSignersOrComputeUnits?: Signer | Signer[] | number,
    computeUnits?: number,
  ) => {
    try {
      await sendAndConfirmIx(ix, signerOrSignersOrComputeUnits, computeUnits);
    } catch (error) {
      expect((error as SendIxError).logs).includes(errorMessage);
      return;
    }
    expect.fail("Expected transaction to fail");
  }

  const expectTxToSucceed = async (
    tx: Transaction | Promise<Transaction>,
    payer?: Signer,
  ) => {
    const wallet = payerToWallet(payer);
    return expect(
      signSendAndConfirmTransaction(
      connection,
      wallet.key(),
      wallet.signTransaction,
      await tx,
    )).to.be.fulfilled;
  }

  return {
    requestAirdrop,
    guardianSign,
    postSignedMsgAsVaaOnSolana,
    sendAndConfirmIx,
    expectIxToSucceed,
    expectIxToFailWithError,
    expectTxToSucceed,
  };
}
