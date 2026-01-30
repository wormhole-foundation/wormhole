import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import request from 'supertest';
import {
  hashPeerData,
  WormholeGuardianData,
  ServerConfig,
  PeerRegistration,
  BasePeer,
  PeersResponse,
  Peer,
  UploadResponse,
  BaseServerConfig
} from '@xlabs-xyz/peer-lib';
import { ethers } from 'ethers';

import { PeerServer } from '../src/server.js';
import { Display } from '../src/display.js';
import { Application } from 'express';

// Mock Display for tests to avoid console output during testing
class MockDisplay extends Display {
  log(): void {
    // Silent for tests
  }
  error(): void {
    // Silent for tests
  }
  setProgress(): void {
    // Silent for tests
  }
}

// Generate 19 random guardian wallets for testing
const testGuardianWallets: ethers.HDNodeWallet[] = [];
const testGuardianAddresses: string[] = [];

for (let i = 0; i < 19; i++) {
  const wallet = ethers.Wallet.createRandom();
  testGuardianWallets.push(wallet);
  testGuardianAddresses.push(wallet.address);
}

// Utility function to create a peer registration with a valid signature
async function createPeerRegistration(
  wallet: ethers.HDNodeWallet,
  peer: BasePeer,
): Promise<PeerRegistration> {
  const messageHash = hashPeerData(peer);
  const signature = await wallet.signMessage(ethers.getBytes(messageHash));

  return {
    peer,
    signature,
  };
}

class PeerServerTest extends PeerServer {
  constructor(
    config: BaseServerConfig,
    wormholeData: WormholeGuardianData,
    display: Display
  ) {
    const app = PeerServerTest.createApp();
    // eslint-disable-next-line @typescript-eslint/no-unsafe-argument, @typescript-eslint/no-explicit-any
    super(app, undefined as any, config, wormholeData, display);
  }
}

describe('PeerServer', () => {
  let server: PeerServer;
  let app: Application;

  const testConfig: ServerConfig = {
    port: 3000,
    ethereum: {
      rpcUrl: "https://eth.llamarpc.com",
      chainId: 1
    },
    wormholeContractAddress: "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B",
    threshold: 13,
    // Shouldn't really be used in these tests
    peerListStore: "/tmp/peerListStore.test.json"
  };

  // Mock guardian data for testing using generated wallets
  const mockWormholeData: WormholeGuardianData = {
    guardians: testGuardianAddresses
  };

  beforeEach(() => {
    const mockDisplay = new MockDisplay();
    server = new PeerServerTest(testConfig, mockWormholeData, mockDisplay);
    app = server.getApp();
  });

  describe('GET /peers', () => {
    it('should return empty peers object when no peers exist', async () => {
      const response = await request(app)
        .get('/peers')
        .expect(200);

      expect(response.body).toEqual({ peers: [], threshold: 13, totalExpectedGuardians: 19 });
    });

    it('should return all peers mapped by guardian keys', async () => {
      // Add a test peer first with valid signature using one of our generated guardians
      const peer: BasePeer = {
        hostname: 'test.example.com',
        port: 1,
        tlsX509: 'test-cert',
      };

      // Use the first generated guardian wallet to create and sign the peer registration
      const testGuardianWallet = testGuardianWallets[0];
      const peerRegistration = await createPeerRegistration(testGuardianWallet, peer);

      await request(app)
        .post('/peers')
        .send(peerRegistration)
        .expect(201);

      const response = await request(app)
        .get('/peers')
        .expect(200) as { body: PeersResponse };

      // Should be an array with peer data and threshold
      expect(typeof response.body).toBe('object');
      expect(Array.isArray(response.body.peers)).toBe(true);
      expect(response.body.threshold).toBe(13);
      expect(response.body.totalExpectedGuardians).toBe(19);
      expect(response.body.peers).toHaveLength(1);
      
      const submittedPeer = response.body.peers.find((p: Peer) => p.guardianAddress === testGuardianWallet.address);
      if (!submittedPeer) {
        throw new Error('Submitted peer not found');
      }
      expect(submittedPeer.hostname).toBe(peer.hostname);
      expect(submittedPeer.tlsX509).toBe(peer.tlsX509);
    });
  });

  describe('POST /peers', () => {
    it('should add a new peer with valid signatures', async () => {
      const peer: BasePeer = {
        hostname: 'newpeer.example.com',
        port: 1,
        tlsX509: 'new-cert-data',
      };

      // Use the first generated guardian wallet to create and sign the peer registration
      const testGuardianWallet = testGuardianWallets[0];
      const peerRegistration = await createPeerRegistration(testGuardianWallet, peer);

      const response = await request(app)
        .post('/peers')
        .send(peerRegistration)
        .expect(201) as { body: UploadResponse };

      expect(response.body.peer.hostname).toBe(peer.hostname);
      expect(response.body.peer.tlsX509).toBe(peer.tlsX509);
      expect(response.body.peer.guardianAddress.toLowerCase()).toBe(testGuardianWallet.address.toLowerCase());
    });

    it('should reject peer registration with missing fields', async () => {
      const incompleteRegistration = {
        peer: {
          hostname: 'incomplete.example.com'
          // Missing tlsX509 and port
        }
        // Missing signature
      };

      const response = await request(app)
        .post('/peers')
        .send(incompleteRegistration)
        .expect(400) as { body: { error: string } };

      expect(response.body.error).toContain('Invalid peer registration');
    });

    it('should reject peer registration with invalid signatures', async () => {
      const peer: BasePeer = {
        hostname: 'invalid.example.com',
        port: 1,
        tlsX509: 'invalid-cert',
      };

      const invalidRegistration: PeerRegistration = {
        peer,
        signature: '0xinvalidsignature',
      };

      const response = await request(app)
        .post('/peers')
        .send(invalidRegistration)
        .expect(401) as { body: { error: string } };

      expect(response.body.error).toBe('Invalid guardian signature');
    });

    it('should reject peer registration with missing signature', async () => {
      const peer: BasePeer = {
        hostname: 'nosigs.example.com',
        port: 1,
        tlsX509: 'nosigs-cert',
      };

      const noSigRegistration = {
        peer,
        // Missing signature field
      };

      const response = await request(app)
        .post('/peers')
        .send(noSigRegistration)
        .expect(400) as { body: { error: string } };

      expect(response.body.error).toContain('Invalid peer registration');
    });
  });

});
