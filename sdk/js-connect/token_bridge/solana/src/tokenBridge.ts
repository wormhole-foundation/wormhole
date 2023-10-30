import {
  ChainId,
  Network,
  toChainId,
  toChainName,
  TokenBridge,
  ChainAddress,
  TokenId,
  UniversalAddress,
  toNative,
  ErrNotWrapped,
  RpcConnection,
  NativeAddress,
  ChainsConfig,
  Contracts,
} from '@wormhole-foundation/connect-sdk';
import {
  SolanaUnsignedTransaction,
  AnySolanaAddress,
  SolanaChainName,
  SolanaPlatform,
  SolanaAddress,
} from '@wormhole-foundation/connect-sdk-solana';
import { utils } from '@wormhole-foundation/wormhole-connect-sdk-core-solana';

import {
  Connection,
  Keypair,
  PublicKey,
  SystemProgram,
  Transaction,
  TransactionInstruction,
} from '@solana/web3.js';
import {
  ACCOUNT_SIZE,
  NATIVE_MINT,
  TOKEN_PROGRAM_ID,
  createAssociatedTokenAccountInstruction,
  createCloseAccountInstruction,
  createInitializeAccountInstruction,
  getAssociatedTokenAddress,
  getMinimumBalanceForRentExemptAccount,
} from '@solana/spl-token';
import { Program } from '@project-serum/anchor';

import { TokenBridge as TokenBridgeContract } from './types';
import {
  createApproveAuthoritySignerInstruction,
  createAttestTokenInstruction,
  createCompleteTransferNativeInstruction,
  createCompleteTransferWrappedInstruction,
  createCreateWrappedInstruction,
  createReadOnlyTokenBridgeProgramInterface,
  createTransferNativeInstruction,
  createTransferNativeWithPayloadInstruction,
  createTransferWrappedInstruction,
  createTransferWrappedWithPayloadInstruction,
  deriveWrappedMintKey,
  getWrappedMeta,
} from './utils';

export class SolanaTokenBridge implements TokenBridge<'Solana'> {
  readonly chainId: ChainId;

  readonly tokenBridgeAddress: string;
  readonly coreAddress: string;

  readonly tokenBridge: Program<TokenBridgeContract>;

  private constructor(
    readonly network: Network,
    readonly chain: SolanaChainName,
    readonly connection: Connection,
    readonly contracts: Contracts,
  ) {
    this.chainId = toChainId(chain);

    const tokenBridgeAddress = contracts.tokenBridge;
    if (!tokenBridgeAddress)
      throw new Error(
        `TokenBridge contract Address for chain ${chain} not found`,
      );
    this.tokenBridgeAddress = tokenBridgeAddress;

    const coreBridgeAddress = contracts.coreBridge;
    if (!coreBridgeAddress)
      throw new Error(
        `CoreBridge contract Address for chain ${chain} not found`,
      );

    this.coreAddress = coreBridgeAddress;

    this.tokenBridge = createReadOnlyTokenBridgeProgramInterface(
      tokenBridgeAddress,
      connection,
    );
  }

  static async fromRpc(
    connection: RpcConnection<'Solana'>,
    config: ChainsConfig,
  ): Promise<SolanaTokenBridge> {
    const [network, chain] = await SolanaPlatform.chainFromRpc(connection);
    return new SolanaTokenBridge(
      network,
      chain,
      connection,
      config[chain]!.contracts,
    );
  }

  async isWrappedAsset(token: AnySolanaAddress): Promise<boolean> {
    return getWrappedMeta(
      this.connection,
      this.tokenBridge.programId,
      new SolanaAddress(token).toUint8Array(),
    )
      .catch((_) => null)
      .then((meta) => meta != null);
  }

  async getOriginalAsset(token: AnySolanaAddress): Promise<TokenId> {
    if (!(await this.isWrappedAsset(token)))
      throw ErrNotWrapped(token.toString());

    const tokenAddr = new SolanaAddress(token).toUint8Array();
    const mint = new PublicKey(tokenAddr);

    try {
      const meta = await getWrappedMeta(
        this.connection,
        this.tokenBridge.programId,
        tokenAddr,
      );

      if (meta === null)
        return {
          chain: this.chain,
          address: toNative(this.chain, mint.toBytes()),
        };

      return {
        chain: toChainName(meta.chain as ChainId),
        address: new UniversalAddress(meta.tokenAddress),
      };
    } catch (_) {
      // TODO: https://github.com/wormhole-foundation/wormhole/blob/main/sdk/js/src/token_bridge/getOriginalAsset.ts#L200
      // the current one returns 0s for address
      throw ErrNotWrapped(token.toString());
    }
  }

  async hasWrappedAsset(token: TokenId): Promise<boolean> {
    try {
      await this.getWrappedAsset(token);
      return true;
    } catch (_) { }
    return false;
  }

  async getWrappedAsset(token: TokenId): Promise<NativeAddress<'Solana'>> {
    const mint = deriveWrappedMintKey(
      this.tokenBridge.programId,
      toChainId(token.chain),
      token.address.toUniversalAddress().toUint8Array(),
    );

    // If we don't throw an error getting wrapped meta, we're good to return
    // the derived mint address back to the caller.
    try {
      await getWrappedMeta(this.connection, this.tokenBridge.programId, mint);
      return toNative(this.chain, mint.toBase58());
    } catch (_) { }

    throw ErrNotWrapped(token.address.toUniversalAddress().toString());
  }

  async isTransferCompleted(
    vaa: TokenBridge.VAA<'Transfer' | 'TransferWithPayload'>,
  ): Promise<boolean> {
    return utils
      .getClaim(
        this.connection,
        this.tokenBridge.programId,
        vaa.emitterAddress.toUint8Array(),
        toChainId(vaa.emitterChain),
        vaa.sequence,
        this.connection.commitment,
      )
      .catch((e) => false);
  }

  async getWrappedNative(): Promise<NativeAddress<'Solana'>> {
    return toNative(this.chain, NATIVE_MINT.toBase58());
  }

  async *createAttestation(
    token: AnySolanaAddress,
    payer?: AnySolanaAddress,
  ): AsyncGenerator<SolanaUnsignedTransaction> {
    if (!payer) throw new Error('Payer required to create attestation');
    const senderAddress = new SolanaAddress(payer).unwrap();
    // TODO:
    const nonce = 0; // createNonce().readUInt32LE(0);

    const transferIx = await utils.createBridgeFeeTransferInstruction(
      this.connection,
      this.coreAddress,
      senderAddress,
    );
    const messageKey = Keypair.generate();
    const attestIx = createAttestTokenInstruction(
      this.connection,
      this.tokenBridge.programId,
      this.coreAddress,
      senderAddress,
      new SolanaAddress(token).toUint8Array(),
      messageKey.publicKey,
      nonce,
    );

    const transaction = new Transaction().add(transferIx, attestIx);
    const { blockhash } = await this.connection.getLatestBlockhash();
    transaction.recentBlockhash = blockhash;
    transaction.feePayer = senderAddress;
    transaction.partialSign(messageKey);

    yield this.createUnsignedTx(transaction, 'Solana.AttestToken');
  }

  async *submitAttestation(
    vaa: TokenBridge.VAA<'AttestMeta'>,
    payer?: AnySolanaAddress,
  ): AsyncGenerator<SolanaUnsignedTransaction> {
    if (!payer) throw new Error('Payer required to create attestation');
    const senderAddress = new SolanaAddress(payer).unwrap();

    const { blockhash } = await this.connection.getLatestBlockhash();

    // Yield transactions to verify sigs and post the VAA
    yield* this.postVaa(senderAddress, vaa, blockhash);

    // Now yield the transaction to actually create the token
    const transaction = new Transaction().add(
      createCreateWrappedInstruction(
        this.connection,
        this.tokenBridge.programId,
        this.coreAddress,
        senderAddress,
        vaa,
      ),
    );
    transaction.recentBlockhash = blockhash;
    transaction.feePayer = senderAddress;

    yield this.createUnsignedTx(transaction, 'Solana.CreateWrapped');
  }

  private async transferSol(
    sender: AnySolanaAddress,
    recipient: ChainAddress,
    amount: bigint,
    payload?: Uint8Array,
  ): Promise<SolanaUnsignedTransaction> {
    //  https://github.com/wormhole-foundation/wormhole-connect/blob/development/sdk/src/contexts/solana/context.ts#L245

    const senderAddress = new SolanaAddress(sender).unwrap();
    // TODO: the payer can actually be different from the sender. We need to allow the user to pass in an optional payer
    const payerPublicKey = senderAddress;

    const recipientAddress = recipient.address
      .toUniversalAddress()
      .toUint8Array();
    const recipientChainId = toChainId(recipient.chain);

    const nonce = 0;
    const relayerFee = 0n;

    const message = Keypair.generate();
    const ancillaryKeypair = Keypair.generate();

    const rentBalance = await getMinimumBalanceForRentExemptAccount(
      this.connection,
    );

    //This will create a temporary account where the wSOL will be created.
    const createAncillaryAccountIx = SystemProgram.createAccount({
      fromPubkey: payerPublicKey,
      newAccountPubkey: ancillaryKeypair.publicKey,
      lamports: rentBalance, //spl token accounts need rent exemption
      space: ACCOUNT_SIZE,
      programId: TOKEN_PROGRAM_ID,
    });

    //Send in the amount of SOL which we want converted to wSOL
    const initialBalanceTransferIx = SystemProgram.transfer({
      fromPubkey: payerPublicKey,
      lamports: amount,
      toPubkey: ancillaryKeypair.publicKey,
    });

    //Initialize the account as a WSOL account, with the original payerAddress as owner
    const initAccountIx = createInitializeAccountInstruction(
      ancillaryKeypair.publicKey,
      NATIVE_MINT,
      payerPublicKey,
    );

    //Normal approve & transfer instructions, except that the wSOL is sent from the ancillary account.
    const approvalIx = createApproveAuthoritySignerInstruction(
      this.tokenBridge.programId,
      ancillaryKeypair.publicKey,
      payerPublicKey,
      amount,
    );

    const tokenBridgeTransferIx = payload
      ? createTransferNativeWithPayloadInstruction(
        this.connection,
        this.tokenBridge.programId,
        this.coreAddress,
        senderAddress,
        message.publicKey,
        ancillaryKeypair.publicKey,
        NATIVE_MINT,
        nonce,
        amount,
        recipientAddress,
        recipientChainId,
        payload,
      )
      : createTransferNativeInstruction(
        this.connection,
        this.tokenBridge.programId,
        this.coreAddress,
        senderAddress,
        message.publicKey,
        ancillaryKeypair.publicKey,
        NATIVE_MINT,
        nonce,
        amount,
        relayerFee,
        recipientAddress,
        recipientChainId,
      );

    //Close the ancillary account for cleanup. Payer address receives any remaining funds
    const closeAccountIx = createCloseAccountInstruction(
      ancillaryKeypair.publicKey, //account to close
      payerPublicKey, //Remaining funds destination
      payerPublicKey, //authority
    );

    const { blockhash } = await this.connection.getLatestBlockhash();
    const transaction = new Transaction();
    transaction.recentBlockhash = blockhash;
    transaction.feePayer = payerPublicKey;
    transaction.add(
      createAncillaryAccountIx,
      initialBalanceTransferIx,
      initAccountIx,
      approvalIx,
      tokenBridgeTransferIx,
      closeAccountIx,
    );
    transaction.partialSign(message, ancillaryKeypair);

    return this.createUnsignedTx(transaction, 'Solana.TransferNative');
  }

  async *transfer(
    sender: AnySolanaAddress,
    recipient: ChainAddress,
    token: AnySolanaAddress | 'native',
    amount: bigint,
    payload?: Uint8Array,
  ): AsyncGenerator<SolanaUnsignedTransaction> {
    // TODO: payer vs sender?? can caller add diff payer later?

    if (token === 'native') {
      yield await this.transferSol(sender, recipient, amount, payload);
      return;
    }

    const tokenAddress = new SolanaAddress(token).unwrap();

    const senderAddress = new SolanaAddress(sender).unwrap();
    const senderTokenAddress = await getAssociatedTokenAddress(
      tokenAddress,
      senderAddress,
    );

    const recipientAddress = recipient.address
      .toUniversalAddress()
      .toUint8Array();
    const recipientChainId = toChainId(recipient.chain);

    const nonce = 0;
    const relayerFee = 0n;

    const isSolanaNative = !(await this.isWrappedAsset(token));

    const message = Keypair.generate();
    let tokenBridgeTransferIx: TransactionInstruction;
    if (isSolanaNative) {
      tokenBridgeTransferIx = payload
        ? createTransferNativeWithPayloadInstruction(
          this.connection,
          this.tokenBridge.programId,
          this.coreAddress,
          senderAddress,
          message.publicKey,
          senderTokenAddress,
          tokenAddress,
          nonce,
          amount,
          recipientAddress,
          recipientChainId,
          payload,
        )
        : createTransferNativeInstruction(
          this.connection,
          this.tokenBridge.programId,
          this.coreAddress,
          senderAddress,
          message.publicKey,
          senderTokenAddress,
          tokenAddress,
          nonce,
          amount,
          relayerFee,
          recipientAddress,
          recipientChainId,
        );
    } else {
      const originAsset = await this.getOriginalAsset(token);

      tokenBridgeTransferIx = payload
        ? createTransferWrappedWithPayloadInstruction(
          this.connection,
          this.tokenBridge.programId,
          this.coreAddress,
          senderAddress,
          message.publicKey,
          senderTokenAddress,
          senderAddress,
          toChainId(originAsset.chain),
          originAsset.address.toUint8Array(),
          nonce,
          amount,
          recipientAddress,
          recipientChainId,
          payload,
        )
        : createTransferWrappedInstruction(
          this.connection,
          this.tokenBridge.programId,
          this.coreAddress,
          senderAddress,
          message.publicKey,
          senderTokenAddress,
          senderAddress,
          toChainId(originAsset.chain),
          originAsset.address.toUint8Array(),
          nonce,
          amount,
          relayerFee,
          recipientAddress,
          recipientChainId,
        );
    }

    const approvalIx = createApproveAuthoritySignerInstruction(
      this.tokenBridge.programId,
      senderTokenAddress,
      senderAddress,
      amount,
    );

    const transaction = new Transaction().add(
      approvalIx,
      tokenBridgeTransferIx,
    );

    const { blockhash } = await this.connection.getLatestBlockhash();
    transaction.recentBlockhash = blockhash;
    transaction.feePayer = senderAddress;
    transaction.partialSign(message);

    yield this.createUnsignedTx(transaction, 'Solana.TransferTokens');
  }

  async *redeem(
    sender: AnySolanaAddress,
    vaa: TokenBridge.VAA<'Transfer' | 'TransferWithPayload'>,
    unwrapNative: boolean = true,
  ): AsyncGenerator<SolanaUnsignedTransaction> {
    // TODO unwrapNative?
    // TODO: check if vaa.payload.token.address is native Sol

    const { blockhash } = await this.connection.getLatestBlockhash();
    const senderAddress = new SolanaAddress(sender).unwrap();
    const ataAddress = new SolanaAddress(
      vaa.payload.to.address.toUint8Array(),
    ).unwrap();
    const wrappedToken = await this.getWrappedAsset(vaa.payload.token);

    // If the ata doesn't exist yet, create it
    const acctInfo = await this.connection.getAccountInfo(ataAddress);
    if (acctInfo === null) {
      const ataCreationTx = new Transaction().add(
        createAssociatedTokenAccountInstruction(
          senderAddress,
          ataAddress,
          senderAddress,
          new PublicKey(wrappedToken.toUint8Array()),
        ),
      );
      ataCreationTx.feePayer = senderAddress;
      ataCreationTx.recentBlockhash = blockhash;
      yield this.createUnsignedTx(ataCreationTx, 'Redeem.CreateATA');
    }

    // Yield transactions to verify sigs and post the VAA
    yield* this.postVaa(sender, vaa, blockhash);

    const createCompleteTransferInstruction =
      vaa.payload.token.chain == this.chain
        ? createCompleteTransferNativeInstruction
        : createCompleteTransferWrappedInstruction;

    const transaction = new Transaction().add(
      createCompleteTransferInstruction(
        this.connection,
        this.tokenBridge.programId,
        this.coreAddress,
        senderAddress,
        vaa,
      ),
    );

    transaction.recentBlockhash = blockhash;
    transaction.feePayer = senderAddress;
    yield this.createUnsignedTx(transaction, 'Solana.RedeemTransfer');
  }

  private async *postVaa(
    sender: AnySolanaAddress,
    vaa: TokenBridge.VAA<'Transfer' | 'TransferWithPayload' | 'AttestMeta'>,
    blockhash: string,
  ) {
    const senderAddr = new SolanaAddress(sender).unwrap();
    const signatureSet = Keypair.generate();

    const verifySignaturesInstructions =
      await utils.createVerifySignaturesInstructions(
        this.connection,
        this.coreAddress,
        senderAddr,
        vaa,
        signatureSet.publicKey,
      );

    // Create a new transaction for every 2 signatures we have to Verify
    for (let i = 0; i < verifySignaturesInstructions.length; i += 2) {
      const verifySigTx = new Transaction().add(
        ...verifySignaturesInstructions.slice(i, i + 2),
      );
      verifySigTx.recentBlockhash = blockhash;
      verifySigTx.feePayer = senderAddr;
      verifySigTx.partialSign(signatureSet);

      yield this.createUnsignedTx(verifySigTx, 'Redeem.VerifySignature', true);
    }

    // Finally create the VAA posting transaction
    const postVaaTx = new Transaction().add(
      utils.createPostVaaInstruction(
        this.connection,
        this.coreAddress,
        senderAddr,
        vaa,
        signatureSet.publicKey,
      ),
    );
    postVaaTx.recentBlockhash = blockhash;
    postVaaTx.feePayer = senderAddr;

    yield this.createUnsignedTx(postVaaTx, 'Redeem.PostVAA');
  }

  private createUnsignedTx(
    txReq: Transaction,
    description: string,
    parallelizable: boolean = false,
  ): SolanaUnsignedTransaction {
    return new SolanaUnsignedTransaction(
      txReq,
      this.network,
      'Solana',
      description,
      parallelizable,
    );
  }
}
