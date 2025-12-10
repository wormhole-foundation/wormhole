import { ethers } from 'ethers';
import { WormholeGuardianData, WormholeConfig, PeerRegistration, Guardian, ValidationError, BasePeer, Peer } from './types.js';
import { errorMsg } from './error.js';

// Core Bridge ABI based on ICoreBridge interface
const CORE_BRIDGE_ABI = [
  'function getGuardianSet(uint32 index) view returns (tuple(address[] keys, uint32 expirationTime))',
  'function getCurrentGuardianSetIndex() view returns (uint32)'
];

/**
 * Simple function to connect to Ethereum and get the current Wormhole guardian data
 */
export async function getWormholeGuardianData(
  config: WormholeConfig
): Promise<WormholeGuardianData> {
  console.log('Connecting to Wormhole contract...');

  const provider = new ethers.JsonRpcProvider(config.ethereum.rpcUrl, undefined, {staticNetwork: true});
  const contract = new ethers.Contract(config.wormholeContractAddress, CORE_BRIDGE_ABI, provider);

  try {
    // Get current guardian set index
    const currentGuardianSetIndex = await contract.getCurrentGuardianSetIndex() as number;
    console.log(`Current guardian set index: ${currentGuardianSetIndex}`);

    // Load current guardian set - returns a struct/tuple
    const currentSetResult = await contract.getGuardianSet(currentGuardianSetIndex) as [string[]];

    console.log(`Loaded current guardian set with ${currentSetResult[0].length} guardians`);

    return {
      guardians: currentSetResult[0]
    };
  } catch (error) {
    console.error('Failed to fetch Wormhole guardian data:', errorMsg(error));
    throw error;
  }
}

export function hashPeerData(basePeer: BasePeer): string {
  return ethers.keccak256(
    ethers.solidityPacked(
      ['string', 'string'],
      [`${basePeer.hostname}:${basePeer.port}`, basePeer.tlsX509]
    )
  );
}

export function validateGuardianSignature(
  peerRegistration: PeerRegistration,
  wormholeData: WormholeGuardianData
): ValidationError<Guardian> {
  // The message hash that should have been signed by the guardian
  const messageHash = hashPeerData(peerRegistration.peer);
  try {
    // Recover the address that signed the message
    const guardianAddress = ethers.verifyMessage(
      ethers.getBytes(messageHash),
      peerRegistration.signature
    );
    const guardianIndex = wormholeData.guardians.findIndex(
      guardian => guardian.toLowerCase() === guardianAddress.toLowerCase()
    );
    if (guardianIndex === -1) {
      return { success: false, error: `Invalid signature: guardian ${guardianAddress} not found in guardian set` };
    }
    return { success: true, value: { guardianAddress, guardianIndex } };
  } catch (error) {
    return { success: false, error: `Failed to verify signature: ${errorMsg(error)}` };
  }
}

export function validatePeers(
  initialPeers: Peer[],
  wormholeData: WormholeGuardianData,
): (Peer | undefined)[] {
  const sparsePeers = Array<Peer | undefined>(wormholeData.guardians.length);
  for (const peer of initialPeers) {
    if (peer.guardianIndex < 0 || peer.guardianIndex >= wormholeData.guardians.length) {
      throw new Error(`Invalid initial peer index: ${peer.guardianIndex}`);
    }
    if (sparsePeers[peer.guardianIndex] !== undefined) {
      throw new Error(`Duplicate initial peer: ${JSON.stringify(peer)}`);
    }
    const guardianAddress = wormholeData.guardians[peer.guardianIndex];
    if (guardianAddress.toLowerCase() !== peer.guardianAddress.toLowerCase()) {
      throw new Error(`Peer address at index ${peer.guardianIndex} is not ${peer.guardianAddress}`);
    }
    const signature = peer.signature;
    const guardian = validateGuardianSignature({ peer, signature }, wormholeData);
    if (!guardian.success) {
      throw new Error(`Invalid guardian signature: ${guardian.error}`);
    }
    sparsePeers[peer.guardianIndex] = peer;
  }
  return sparsePeers;
}