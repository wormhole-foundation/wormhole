
import { registerProtocol } from '@wormhole-foundation/connect-sdk';
import { EvmTokenBridge } from './tokenBridge';
import { EvmAutomaticTokenBridge } from './automaticTokenBridge';

registerProtocol('Evm', 'TokenBridge', EvmTokenBridge);
registerProtocol('Evm', 'AutomaticTokenBridge', EvmAutomaticTokenBridge);

export * as ethers_contracts from './ethers-contracts';
export * from './tokenBridge';
export * from './automaticTokenBridge';
