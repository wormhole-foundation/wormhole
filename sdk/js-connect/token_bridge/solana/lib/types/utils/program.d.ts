import { Connection, PublicKeyInitData } from '@solana/web3.js';
import { Program, Provider } from '@project-serum/anchor';
import { TokenBridgeCoder } from './coder';
import { TokenBridge } from '../types';
export declare function createTokenBridgeProgramInterface(programId: PublicKeyInitData, provider?: Provider): Program<TokenBridge>;
export declare function createReadOnlyTokenBridgeProgramInterface(programId: PublicKeyInitData, connection?: Connection): Program<TokenBridge>;
export declare function coder(): TokenBridgeCoder;
