/* TODO:
 *   governance actions have a module parameter:
 *     - "Core" - https://github.com/wormhole-foundation/wormhole/blob/9e61d151c61bedb18ab1d4ca6ffb1c6c91b108f0/ethereum/contracts/Governance.sol#L21
 *     - "TokenBridge" - https://github.com/wormhole-foundation/wormhole/blob/9e61d151c61bedb18ab1d4ca6ffb1c6c91b108f0/ethereum/contracts/bridge/BridgeGovernance.sol#L24
 *     - "NFTBridge" - https://github.com/wormhole-foundation/wormhole/blob/9e61d151c61bedb18ab1d4ca6ffb1c6c91b108f0/ethereum/contracts/nft/NFTBridgeGovernance.sol#L23
 *     - "WormholeRelayer" - https://github.com/wormhole-foundation/wormhole/blob/9e61d151c61bedb18ab1d4ca6ffb1c6c91b108f0/ethereum/contracts/relayer/wormholeRelayer/WormholeRelayerGovernance.sol#L43
 *   while the core contract is actually called "Wormhole" in most platforms
 *     - though in solana it's called bridge - https://github.com/wormhole-foundation/wormhole/tree/main/solana/bridge
 *     - and in algorand it seems to be called wormhole_core - https://github.com/wormhole-foundation/wormhole/blob/main/algorand/wormhole_core.py
 *     - and in EVM it resides directly in the contracts directory
 *   the naming of the token bridge and the nft bridge seem to be a lot more consistent, though:
 *     - for EVM the TokenBridge and NFTbridge reside in the "bridge" and "nft" directory respectively: https://github.com/wormhole-foundation/wormhole/tree/main/ethereum/contracts)
 *   the WormholeRelayer resides in relayer/wormholeRelayer (only built for EVM so far)
 *
 *   Within the solana directory, only the token bridge and the nft bridge are considered modules
 *     (i.e. are in the modules directory: https://github.com/wormhole-foundation/wormhole/tree/main/solana/modules).
 *   While in the JS SDK, the core bridge functionality is in the "bridge" directory
 *     (notice the clash with EVM where bridge refers to the token bridge...).
 *
 * With all of this in mind: What should we name modules here?
 * My preferred choice would be ["Wormholecore", "TokenBridge", "NftBridge", "Relayer"]
 *   but ["Core", "TokenBridge", "NFTBridge", "WormholeRelayer"] seems to be more consistent given
 *   current naming "conventions"
 */

export const protocols = [
  "WormholeCore",
  "TokenBridge",
  "AutomaticTokenBridge",
  "CircleBridge",
  "AutomaticCircleBridge",
  "Relayer",
  "IbcBridge",
  // not implemented
  "NftBridge",
] as const;

export type ProtocolName = (typeof protocols)[number];
export const isProtocolName = (protocol: string): protocol is ProtocolName =>
  protocols.includes(protocol as ProtocolName);
