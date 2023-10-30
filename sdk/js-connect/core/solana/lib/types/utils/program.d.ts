import { Connection, PublicKeyInitData } from '@solana/web3.js';
import { Program, Provider } from '@project-serum/anchor';
import { WormholeCoder } from './coder';
import { Wormhole } from '../types';
export declare function createWormholeProgramInterface(programId: PublicKeyInitData, provider?: Provider): Program<Wormhole>;
export declare function createReadOnlyWormholeProgramInterface(programId: PublicKeyInitData, connection?: Connection): Program<Wormhole>;
export declare function coder(): WormholeCoder;
