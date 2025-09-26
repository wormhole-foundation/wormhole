import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import request from 'supertest';
import { PeerServer } from '../src/server/server.js';
import { loadConfig } from '../src/server/index.js';
import { Display } from '../src/server/display.js';
import { WormholeGuardianData, ServerConfig, PeerRegistration, Peer } from '../src/shared/types.js';
import { ethers } from 'ethers';
import path from 'path';

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
  testGuardianWallets.push(wallet as ethers.HDNodeWallet);
  testGuardianAddresses.push(wallet.address);
}

// Utility function to create a peer registration with a valid signature
async function createPeerRegistration(
  wallet: ethers.HDNodeWallet,
  peer: Peer,
  guardianIndex: number = 0
): Promise<PeerRegistration> {
  const messageHash = ethers.keccak256(
    ethers.solidityPacked(
      ['string', 'string'],
      [peer.hostname, peer.tlsX509]
    )
  );
  const signature = await wallet.signMessage(ethers.getBytes(messageHash));

  return {
    peer,
    signature: {
      signature,
      guardianIndex
    }
  };
}

describe('PeerServer', () => {
  let server: PeerServer;
  let app: any;

  const testConfig: ServerConfig = loadConfig(path.join(__dirname, '..', 'config.json'));

  // Mock guardian data for testing using generated wallets
  const mockWormholeData: WormholeGuardianData = {
    guardians: testGuardianAddresses
  };

  beforeEach(async () => {
    const mockDisplay = new MockDisplay();
    server = new PeerServer(testConfig, mockWormholeData, mockDisplay);
    app = server.getApp();
  });

  afterEach(() => {
    server.close();
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
      const peer: Peer = {
        guardianAddress: '', // Will be set by server
        hostname: 'test.example.com',
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
        .expect(200);

      // Should be an array with peer data and threshold
      expect(typeof response.body).toBe('object');
      expect(Array.isArray(response.body.peers)).toBe(true);
      expect(response.body.threshold).toBe(13);
      expect(response.body.totalExpectedGuardians).toBe(19);
      expect(response.body.peers).toHaveLength(1);
      
      const submittedPeer = response.body.peers.find((p: any) => p.guardianAddress === testGuardianWallet.address);
      expect(submittedPeer).toBeDefined();
      expect(submittedPeer.hostname).toBe(peer.hostname);
      expect(submittedPeer.tlsX509).toBe(peer.tlsX509);
    });
  });

  describe('POST /peers', () => {
    it('should add a new peer with valid signatures', async () => {
      const peer: Peer = {
        guardianAddress: '', // Will be set by server
        hostname: 'newpeer.example.com',
        tlsX509: 'new-cert-data',
      };

      // Use the first generated guardian wallet to create and sign the peer registration
      const testGuardianWallet = testGuardianWallets[0];
      const peerRegistration = await createPeerRegistration(testGuardianWallet, peer);

      const response = await request(app)
        .post('/peers')
        .send(peerRegistration)
        .expect(201);

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
        .expect(400);

      expect(response.body.error).toContain('Missing required fields');
    });

    it('should reject peer registration with invalid signatures', async () => {
      const peer: Peer = {
        guardianAddress: '', // Will be set by server
        hostname: 'invalid.example.com',
        tlsX509: 'invalid-cert',
      };

      const invalidRegistration: PeerRegistration = {
        peer,
        signature: {
          signature: '0xinvalidsignature',
          guardianIndex: 0
        }
      };

      const response = await request(app)
        .post('/peers')
        .send(invalidRegistration)
        .expect(401);

      expect(response.body.error).toBe('Invalid guardian signature');
    });

    it('should reject peer registration with missing signature', async () => {
      const peer: Peer = {
        guardianAddress: '', // Will be set by server
        hostname: 'nosigs.example.com',
        tlsX509: 'nosigs-cert',
      };

      const noSigRegistration = {
        peer,
        // Missing signature field
      };

      const response = await request(app)
        .post('/peers')
        .send(noSigRegistration)
        .expect(400);

      expect(response.body.error).toContain('Missing required fields');
    });
  });

});
