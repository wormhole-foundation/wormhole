import { ethers } from 'ethers';
import { WormholeGuardianData, ServerConfig } from '../shared/types.js';

// Core Bridge ABI based on ICoreBridge interface
const CORE_BRIDGE_ABI = [
  'function getGuardianSet(uint32 index) view returns (tuple(address[] keys, uint32 expirationTime))',
  'function getCurrentGuardianSetIndex() view returns (uint32)'
];

/**
 * Simple function to connect to Ethereum and get the current Wormhole guardian data
 */
export async function getWormholeGuardianData(
  config: ServerConfig
): Promise<WormholeGuardianData> {
  console.log('Connecting to Wormhole contract...');

  const provider = new ethers.JsonRpcProvider(config.ethereum.rpcUrl);
  const contract = new ethers.Contract(config.wormholeContractAddress, CORE_BRIDGE_ABI, provider);

  try {
    // Get current guardian set index
    const currentGuardianSetIndex = await contract.getCurrentGuardianSetIndex();
    console.log(`Current guardian set index: ${currentGuardianSetIndex}`);

    // Load current guardian set - returns a struct/tuple
    const currentSetResult = await contract.getGuardianSet(currentGuardianSetIndex);

    console.log(`Loaded current guardian set with ${currentSetResult.keys.length} guardians`);

    return {
      guardians: currentSetResult.keys
    };
  } catch (error) {
    console.error('Failed to fetch Wormhole guardian data:', error);
    throw error;
  }
}
