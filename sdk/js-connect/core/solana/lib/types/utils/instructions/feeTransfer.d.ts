import { Commitment, Connection, PublicKeyInitData, TransactionInstruction } from '@solana/web3.js';
export declare function createBridgeFeeTransferInstruction(connection: Connection, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, commitment?: Commitment): Promise<TransactionInstruction>;
