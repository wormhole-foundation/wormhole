import { registerProtocol } from '@wormhole-foundation/connect-sdk';
import { SolanaWormholeCore } from './core';

registerProtocol('Solana', 'WormholeCore', SolanaWormholeCore);

export * from './core';
export * from './types';
export * as utils from './utils';
