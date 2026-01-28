import { ethers } from 'ethers';
import {
  hashPeerData,
  PeerRegistration,
  SelfConfig,
  PeersResponse,
  validateOrFail,
  PeerRegistrationSchema,
  PeersResponseSchema,
  UploadResponseSchema,
  UploadResponse,
  errorStack,
  validatePeers,
  WormholeGuardianData
} from '@xlabs-xyz/peer-lib';

export class PeerClient {
  private config: SelfConfig;
  private serverUrl: string;

  constructor(config: SelfConfig, private readonly pollPeriod = 5000) {
    this.config = config;
    this.serverUrl = this.config.serverUrl;
  }

  private async signPeerData(guardianPrivateKey: string): Promise<PeerRegistration> {
    const { peer } = this.config;
    // Create wallet from private key
    const wallet = new ethers.Wallet(guardianPrivateKey);
    // Create message hash as per server implementation
    const messageHash = hashPeerData(peer);
    // Sign the message
    const signature = await wallet.signMessage(ethers.getBytes(messageHash));
    const peerRegistration = {
      peer,
      signature
    };
    // Validate the generated PeerRegistration
    return validateOrFail(PeerRegistrationSchema, peerRegistration, "Generated PeerRegistration is invalid");
  }

  private async uploadPeerData(peerRegistration: PeerRegistration): Promise<UploadResponse> {
    console.log(`[UPLOAD] Uploading peer data for guardian...`);

    const response = await fetch(`${this.serverUrl}/peers`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(peerRegistration),
    });

    if (response.ok) {
      const result = validateOrFail(UploadResponseSchema, await response.json(), "Invalid server response");
      console.log(`[SUCCESS] Successfully uploaded peer data!`);
      console.log(`   Guardian Address: ${result.peer.guardianAddress}`);
      console.log(`   Guardian Index: ${result.peer.guardianIndex}`);
      console.log(`   Hostname: ${result.peer.hostname}`);
      return result;
    } else {
      const error = await response.text();
      console.error(`[ERROR] Failed to upload peer data: ${response.status} ${response.statusText}`);
      console.error(`   Error: ${error}`);
      throw new Error(`Upload failed: ${response.status} ${response.statusText}`);
    }
  }

  private async signAndUploadPeerData(guardianPrivateKey: string): Promise<UploadResponse> {
    const peerRegistration = await this.signPeerData(guardianPrivateKey);
    return this.uploadPeerData(peerRegistration);
  }

  private async pollForCompletion(): Promise<PeersResponse> {
    console.log(`[POLLING] Starting to poll for completion...`);

    let lastPeerCount = 0;

    for (;;) {
      try {
        const response = await fetch(`${this.serverUrl}/peers`);

        if (response.ok) {
          const jsonResponse = await response.json() as PeersResponse;

          // Validate response with Zod
          const { peers, threshold, totalExpectedGuardians } = validateOrFail(
            PeersResponseSchema, jsonResponse, "Invalid peers response"
          );

          if (peers.length > totalExpectedGuardians) {
            throw new Error(`More guardians than expected have submitted their peer data`);
          }

          // Check if all expected guardians have submitted
          if (peers.length === totalExpectedGuardians) {
            console.log(`[SUCCESS] All ${totalExpectedGuardians} expected guardians have submitted their peer data!`);
            return { peers, threshold, totalExpectedGuardians };
          }

          // Show progress if we have new submissions
          const progressMessage = `${peers.length}/${totalExpectedGuardians} guardians have submitted`;
          if (peers.length > lastPeerCount) {
            console.log(`[PROGRESS] ${progressMessage}`);
            lastPeerCount = peers.length;
          } else {
            console.log(`[PROGRESS] ${progressMessage} (waiting for more...)`);
          }
        } else {
          console.error(`[ERROR] Failed to fetch peers: ${response.status} ${response.statusText}`);
        }
      } catch (error) {
        console.error(`[ERROR] Error polling for completion: ${errorStack(error)}`);
      }

      // Wait before next poll
      await this.sleep(this.pollPeriod);
    }
  }

  private validatePeers(response: PeersResponse, wormholeData: WormholeGuardianData): void {
    if (response.peers.length !== wormholeData.guardians.length) {
      throw new Error(`Expected ${wormholeData.guardians.length} guardians, got ${response.peers.length}`);
    }
    validatePeers(response.peers, wormholeData);
  }

  private async pollAllPeersAndValidate(wormholeData: WormholeGuardianData): Promise<PeersResponse> {
    const response = await this.pollForCompletion();
    this.validatePeers(response, wormholeData);
    return response;
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  private async run<T>(action: () => Promise<T>, message: string): Promise<T> {
    console.log(`[STARTING] Peer Client starting...`);
    console.log(`   Server: ${this.serverUrl}`);
    console.log(`   Peer: ${this.config.peer.hostname}`);
    console.log(`   ${message}`);
    const result = await action();
    console.log(`[COMPLETED] Completed successfully!`);
    return result;
  }

  public async submitPeerData(): Promise<UploadResponse> {
    const guardianPrivateKey = this.config.guardianPrivateKey;
    if (guardianPrivateKey === undefined) {
      throw new Error(`Guardian private key path was not set`);
    }
    return this.run(() => this.signAndUploadPeerData(guardianPrivateKey), "Uploading peer data...");
  }

  public async waitForAllPeers(wormholeData: WormholeGuardianData): Promise<PeersResponse> {
    return this.run(() => this.pollAllPeersAndValidate(wormholeData), "Polling all peers...");
  }

  public async submitAndWaitForAllPeers(wormholeData: WormholeGuardianData): Promise<PeersResponse> {
    await this.submitPeerData();
    return this.waitForAllPeers(wormholeData);
  }

  // Test helper method to get current peer data from server
  public async getCurrentPeers(): Promise<PeersResponse> {
    const response = await fetch(`${this.serverUrl}/peers`);
    if (!response.ok) {
      throw new Error(`Failed to fetch peers: ${response.status} ${response.statusText}`);
    }
    return validateOrFail(PeersResponseSchema, await response.json(), "Invalid peers response");
  }
}
