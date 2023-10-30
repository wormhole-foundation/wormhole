import { registerProtocol } from '@wormhole-foundation/connect-sdk';
import { EvmWormholeCore } from './wormholeCore';

registerProtocol('Evm', 'WormholeCore', EvmWormholeCore);

export * as ethers_contracts from './ethers-contracts';
export * from './wormholeCore';
