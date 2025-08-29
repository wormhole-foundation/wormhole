import anchor from '@coral-xyz/anchor';
import { Connection, Keypair, PublicKey, Signer, Transaction } from '@solana/web3.js';
import {
  Chain,
  deserializeLayout,
  encoding,
  Network,
  signAndSendWait,
} from '@wormhole-foundation/sdk-connect';
import { AnySolanaAddress, SolanaAddress, SolanaSendSigner } from '@wormhole-foundation/sdk-solana';
import { SolanaWormholeCore, utils as coreUtils } from '@wormhole-foundation/sdk-solana-core';
import {
  serialize as serializeVaa,
  deserialize as deserializeVaa,
  UniversalAddress,
  createVAA,
  Contracts,
  VAA,
} from '@wormhole-foundation/sdk-definitions';
import { mocks } from '@wormhole-foundation/sdk-definitions/testing';

import { coreV1AccountDataLayout } from './layouts.js';
import { sendAndConfirm } from './testing_helpers.js';

export type VaaMessage = VAA<'Uint8Array'>;

const guardianKey = 'cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0';
export const guardianAddress = 'beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe';

/** A Wormhole Core wrapper allowing to write tests using this program in a local environment. */
export class TestingWormholeCore<N extends Network> {
  public readonly signer: Signer;
  public readonly client: SolanaWormholeCore<N, 'Solana'>;
  private sequence = 0n;
  private readonly toProgram: PublicKey;
  private _guardians?: mocks.MockGuardians;

  /**
   *
   * @param solanaProgram The Solana Program used as a destination for the VAAs, _i.e._ the program being tested.
   * @param contracts At least the core program address `coreBridge` must be provided.
   */
  constructor(
    signer: Signer,
    connection: Connection,
    network: N,
    testedProgram: PublicKey,
    contracts: Contracts,
  ) {
    this.signer = signer;
    this.toProgram = testedProgram;
    this.client = new SolanaWormholeCore(network, 'Solana', connection, contracts);
  }

  get guardians(): mocks.MockGuardians {
    if (this._guardians === undefined) throw Error("coreV1 not initialized")

    return this._guardians;
  }

  get pda() {
    return {
      guardianSet: (): PublicKey =>
        this.findPda(Buffer.from('GuardianSet'), Buffer.from([0, 0, 0, 0])),
      bridge: (): PublicKey => this.findPda(Buffer.from('Bridge')),
      feeCollector: (): PublicKey => this.findPda(Buffer.from('fee_collector')),
    };
  }

  async initialize(
    guardians = [guardianAddress],
    guardianSetExpirationTime = 86400,
    fee = 100,
  ) {
    const initialGuardians = guardians.map((guardian) => Array.from(encoding.hex.decode(guardian)));
    const initFee = new anchor.BN(fee);

    // https://github.com/wormhole-foundation/wormhole/blob/main/solana/bridge/program/src/api/initialize.rs
    const ix = await this.client.coreBridge.methods
      .initialize(guardianSetExpirationTime, initFee, initialGuardians)
      .accounts({
        bridge: this.pda.bridge(),
        guardianSet: this.pda.guardianSet(),
        feeCollector: this.pda.feeCollector(),
        payer: this.signer.publicKey,
      })
      .instruction();

    const txid = await sendAndConfirm(this.client.connection, ix, this.signer)
    this._guardians = new mocks.MockGuardians(0, [guardianKey])
    return txid
  }

  /** Parse a VAA generated from the postVaa method, or from the Token Bridge during
   * and outbound transfer
   */
  async parseVaa(key: PublicKey): Promise<VaaMessage> {
    const info = await this.client.connection.getAccountInfo(key);
    if (info === null) {
      throw new Error(`No message account exists at that address: ${key.toString()}`);
    }

    const message = deserializeLayout(coreV1AccountDataLayout, info.data);

    const vaa = createVAA('Uint8Array', {
      guardianSet: 0,
      timestamp: message.timestamp,
      nonce: message.nonce,
      emitterChain: message.emitterChain,
      emitterAddress: message.emitterAddress,
      sequence: message.sequence,
      consistencyLevel: message.consistencyLevel,
      signatures: [],
      payload: message.payload,
    });

    return deserializeVaa('Uint8Array', serializeVaa(vaa));
  }

  /**
   * `source`: the emitter of the message.
   */
  async postVaa(
    payer: Keypair,
    source: { chain: Chain; emitterAddress: UniversalAddress },
    message: Uint8Array,
  ) {
    const seq = this.sequence++;
    const timestamp = Math.round(Date.now() / 1000);

    const emittingPeer = new mocks.MockEmitter(source.emitterAddress, source.chain, seq);

    const published = emittingPeer.publishMessage(
      0, // nonce,
      message,
      1, // consistencyLevel
      timestamp,
    );
    const vaa = this.guardians.addSignatures(published, [0]);

    let signatureSet: Keypair | undefined;

    this.client.postVaa = async function *postVaa(sender: AnySolanaAddress, vaa: VAA) {
      const postedVaaAddress = coreUtils.derivePostedVaaKey(
        this.coreBridge.programId,
        Buffer.from(vaa.hash),
      );

      // no need to do anything else, this vaa is posted
      const isPosted = await this.connection.getAccountInfo(postedVaaAddress);
      if (isPosted) return;

      const senderAddr = new SolanaAddress(sender).unwrap();
      signatureSet = Keypair.generate();

      const verifySignaturesInstructions =
        await coreUtils.createVerifySignaturesInstructions(
          this.connection,
          this.coreBridge.programId,
          senderAddr,
          vaa,
          signatureSet.publicKey,
        );

      // Create a new transaction for every 2 instructions
      for (let i = 0; i < verifySignaturesInstructions.length; i += 2) {
        const verifySigTx = new Transaction().add(
          ...verifySignaturesInstructions.slice(i, i + 2),
        );
        verifySigTx.feePayer = senderAddr;
        yield this["createUnsignedTx"](
          { transaction: verifySigTx, signers: [signatureSet] },
          'Core.VerifySignature',
          true,
        );
      }

      // Finally create the VAA posting transaction
      const postVaaTx = new Transaction().add(
        coreUtils.createPostVaaInstruction(
          this.connection,
          this.coreBridge.programId,
          senderAddr,
          vaa,
          signatureSet.publicKey,
        ),
      );
      postVaaTx.feePayer = senderAddr;

      yield this["createUnsignedTx"]({ transaction: postVaaTx }, 'Core.PostVAA');
    }

    const txs = this.client.postVaa(payer.publicKey, vaa);
    const signer = new SolanaSendSigner(this.client.connection, 'Solana', payer, false, {});
    await signAndSendWait(txs, signer);
    if (signatureSet === undefined) throw new Error("Something failed with the signature set");

    return {
      postedVaa: coreUtils.derivePostedVaaKey(this.client.coreBridge.programId, Buffer.from(vaa.hash)),
      signatureSet: signatureSet.publicKey,
    };
  }

  private findPda(...seeds: Array<Buffer | Uint8Array>) {
    return PublicKey.findProgramAddressSync(seeds, this.client.coreBridge.programId)[0];
  }
}
