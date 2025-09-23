import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import request from 'supertest';
import { PeerServer } from '../src/server.js';
import { loadConfig } from '../src/index.js';
import { Display } from '../src/display.js';
import { WormholeGuardianData, ServerConfig, PeerRegistration, Peer } from '../src/types.js';
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
const testGuardianWallets: ethers.Wallet[] = [];
const testGuardianAddresses: string[] = [];

for (let i = 0; i < 19; i++) {
  const wallet = ethers.Wallet.createRandom();
  testGuardianWallets.push(wallet);
  testGuardianAddresses.push(wallet.address);
}

// Utility function to create a peer registration with a valid signature
async function createPeerRegistration(
  wallet: ethers.Wallet,
  peer: Peer,
  guardianIndex: number = 0
): Promise<PeerRegistration> {
  const messageHash = ethers.keccak256(
    ethers.solidityPacked(
      ['string', 'string', 'uint256'],
      [peer.Hostname, peer.TlsX509, peer.Port]
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
    keys: testGuardianAddresses
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
    it('should return empty object when no peers exist', async () => {
      const response = await request(app)
        .get('/peers')
        .expect(200);

      expect(response.body).toEqual({});
    });

    it('should return all peers mapped by guardian keys', async () => {
      // Add a test peer first with valid signature using one of our generated guardians
      const peer: Peer = {
        Hostname: 'test.example.com',
        TlsX509: 'test-cert',
        Port: 8080
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

      // Should be an object with guardian key mapping to peer data
      expect(typeof response.body).toBe('object');
      expect(response.body[testGuardianWallet.address]).toBeDefined();
      expect(response.body[testGuardianWallet.address].Hostname).toBe(peer.Hostname);
      expect(response.body[testGuardianWallet.address].TlsX509).toBe(peer.TlsX509);
      expect(response.body[testGuardianWallet.address].Port).toBe(peer.Port);
    });
  });

  describe('POST /peers', () => {
    it('should add a new peer with valid signatures', async () => {
      const peer: Peer = {
        Hostname: 'newpeer.example.com',
        TlsX509: 'new-cert-data',
        Port: 9090
      };

      // Use the first generated guardian wallet to create and sign the peer registration
      const testGuardianWallet = testGuardianWallets[0];
      const peerRegistration = await createPeerRegistration(testGuardianWallet, peer);

      const response = await request(app)
        .post('/peers')
        .send(peerRegistration)
        .expect(201);

      expect(response.body.peer.Hostname).toBe(peer.Hostname);
      expect(response.body.peer.TlsX509).toBe(peer.TlsX509);
      expect(response.body.peer.Port).toBe(peer.Port);
      expect(response.body.guardianAddress.toLowerCase()).toBe(testGuardianWallet.address.toLowerCase());
    });

    it('should reject peer registration with missing fields', async () => {
      const incompleteRegistration = {
        peer: {
          Hostname: 'incomplete.example.com'
          // Missing TlsX509 and Port
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
        Hostname: 'invalid.example.com',
        TlsX509: 'invalid-cert',
        Port: 8080
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
        Hostname: 'nosigs.example.com',
        TlsX509: 'nosigs-cert',
        Port: 8080
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
