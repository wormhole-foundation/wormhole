import { ethers } from 'ethers';
import {
  Peer,
  PeerRegistration,
  SelfConfig,
  PeersResponse,
  validateOrFail,
  SelfConfigSchema,
  PeerRegistrationSchema,
  PeersResponseSchema,
  ServerResponseSchema
} from '../shared/types.js';

export class PeerClient {
  private config: SelfConfig;
  private serverUrl: string;

  constructor(config: SelfConfig) {
    // Validate with Zod
    this.config = validateOrFail(SelfConfigSchema, config, "Invalid client configuration");
    this.serverUrl = this.config.serverUrl;
  }

  private async signPeerData(): Promise<PeerRegistration> {
    const { peer, guardianIndex } = this.config;

    // Create wallet from private key
    const wallet = new ethers.Wallet(this.config.guardianPrivateKey);

    // Create message hash as per server implementation
    const messageHash = ethers.keccak256(
      ethers.solidityPacked(
        ['string', 'string'],
        [peer.hostname, peer.tlsX509]
      )
    );

    // Sign the message
    const signature = await wallet.signMessage(ethers.getBytes(messageHash));

    const peerRegistration = {
      peer,
      signature: {
        signature,
        guardianIndex
      }
    };

    // Validate the generated PeerRegistration
    return validateOrFail(PeerRegistrationSchema, peerRegistration, "Generated PeerRegistration is invalid");
  }

  private async uploadPeerData(): Promise<void> {
    try {
      const peerRegistration = await this.signPeerData();

      console.log(`[UPLOAD] Uploading peer data for guardian ${this.config.guardianIndex}...`);

      const response = await fetch(`${this.serverUrl}/peers`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(peerRegistration),
      });

      if (response.ok) {
        const jsonResponse = await response.json();

        // Validate response with Zod
        const result = validateOrFail(ServerResponseSchema, jsonResponse, "Invalid server response");
        console.log(`[SUCCESS] Successfully uploaded peer data!`);
        console.log(`   Guardian Address: ${result.peer.guardianAddress}`);
        console.log(`   Hostname: ${result.peer.hostname}`);
      } else {
        const error = await response.text();
        console.error(`[ERROR] Failed to upload peer data: ${response.status} ${response.statusText}`);
        console.error(`   Error: ${error}`);
        throw new Error(`Upload failed: ${response.status} ${response.statusText}`);
      }
    } catch (error: any) {
      console.error(`[ERROR] Error uploading peer data: ${error?.message || error}`);
      throw error;
    }
  }

  private async pollForCompletion(): Promise<PeersResponse> {
    console.log(`[POLLING] Starting to poll for completion...`);

    let lastPeerCount = 0;

    while (true) {
      try {
        const response = await fetch(`${this.serverUrl}/peers`);

        if (response.ok) {
          const jsonResponse = await response.json();

          // Validate response with Zod
          const responseData = validateOrFail(PeersResponseSchema, jsonResponse, "Invalid peers response");
          const peers = responseData.peers;
          const threshold = responseData.threshold;
          const totalExpectedGuardians = responseData.totalExpectedGuardians;
          const currentCount = Object.keys(peers).length;

          // Check if all expected guardians have submitted
          if (currentCount >= totalExpectedGuardians) {
            console.log(`[SUCCESS] All ${totalExpectedGuardians} expected guardians have submitted their peer data!`);
            return { peers, threshold, totalExpectedGuardians };
          }

          // Show progress if we have new submissions
          if (currentCount > lastPeerCount) {
            console.log(`[PROGRESS] ${currentCount}/${totalExpectedGuardians} guardians have submitted`);
            lastPeerCount = currentCount;
          } else if (currentCount > 0) {
            console.log(`[PROGRESS] ${currentCount}/${totalExpectedGuardians} guardians have submitted (waiting for more...)`);
          } else {
            console.log(`[PROGRESS] 0/${totalExpectedGuardians} guardians have submitted (waiting for peers...)`);
          }
        } else {
          console.error(`[ERROR] Failed to fetch peers: ${response.status} ${response.statusText}`);
        }
      } catch (error: any) {
        console.error(`[ERROR] Error polling for completion: ${error?.message || error}`);
      }

      // Wait 5 seconds before next poll
      await this.sleep(5000);
    }
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  public async run(): Promise<PeersResponse> {
    try {
      console.log(`[STARTING] Peer Client starting...`);
      console.log(`   Server: ${this.serverUrl}`);
      console.log(`   Guardian Index: ${this.config.guardianIndex}`);
      console.log(`   Peer: ${this.config.peer.hostname}`);

      // Upload our peer data
      await this.uploadPeerData();

      // Poll for completion
      const response = await this.pollForCompletion();

      console.log(`[COMPLETED] Client completed successfully!`);
      return response;
    } catch (error: any) {
      console.error(`[ERROR] Client failed: ${error?.message || error}`);
      throw error;
    }
  }

  // Test helper method to get peer data without polling for completion
  public async submitPeerData(): Promise<void> {
    await this.uploadPeerData();
  }

  // Test helper method to get current peer data from server
  public async getCurrentPeers(): Promise<PeersResponse> {
    const response = await fetch(`${this.serverUrl}/peers`);
    if (!response.ok) {
      throw new Error(`Failed to fetch peers: ${response.status} ${response.statusText}`);
    }
    const jsonResponse = await response.json();
    return validateOrFail(PeersResponseSchema, jsonResponse, "Invalid peers response");
  }
}
