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
  UploadResponse
} from '@xlabs-xyz/peer-lib';

export class PeerClient {
  private config: SelfConfig;
  private serverUrl: string;

  constructor(config: SelfConfig) {
    this.config = config;
    this.serverUrl = this.config.serverUrl;
  }

  private async signPeerData(): Promise<PeerRegistration> {
    const { peer } = this.config;
    // Create wallet from private key
    const wallet = new ethers.Wallet(this.config.guardianPrivateKey);
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

  private async uploadPeerData(): Promise<UploadResponse> {
    try {
      const peerRegistration = await this.signPeerData();

      console.log(`[UPLOAD] Uploading peer data for guardian...`);

      const response = await fetch(`${this.serverUrl}/peers`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(peerRegistration),
      });

      if (response.ok) {
        const jsonResponse = await response.json();

        const result = validateOrFail(UploadResponseSchema, jsonResponse, "Invalid server response");
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
    } catch (error: any) {
      console.error(`[ERROR] Error uploading peer data: ${error?.stack || error}`);
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
        console.error(`[ERROR] Error polling for completion: ${error?.stack || error}`);
      }

      // Wait 5 seconds before next poll
      await this.sleep(5000);
    }
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  private async run<T>(action: () => Promise<T>, message: string): Promise<T> {
    try {
      console.log(`[STARTING] Peer Client starting...`);
      console.log(`   Server: ${this.serverUrl}`);
      console.log(`   Peer: ${this.config.peer.hostname}`);
      console.log(`   ${message}`);
      const result = await action();
      console.log(`[COMPLETED] Completed successfully!`);
      return result;
    } catch (error: any) {
      console.error(`[ERROR] Client failed: ${error?.stack || error}`);
      throw error;
    }
  }

  public async submitPeerData(): Promise<UploadResponse> {
    return this.run(() => this.uploadPeerData(), "Uploading peer data...");
  }

  public async waitForAllPeers(): Promise<PeersResponse> {
    return this.run(() => this.pollForCompletion(), "Polling all peers...");
  }

  public async submitAndWaitForAllPeers(): Promise<PeersResponse> {
    return this.run(async () => {
      await this.uploadPeerData();
      return this.pollForCompletion();
    }, "Uploading peer data and waiting for all peers...");
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
