import { registerProtocol } from '@wormhole-foundation/connect-sdk';
import { SolanaTokenBridge } from './tokenBridge';

registerProtocol('Solana', 'TokenBridge', SolanaTokenBridge);


export * from './types';
export * from './utils';
export * from './tokenBridge';
