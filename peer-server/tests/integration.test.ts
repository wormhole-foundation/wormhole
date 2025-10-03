import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { PeerServer } from '../src/server/server.js';
import { Display } from '../src/server/display.js';
import { WormholeGuardianData, ServerConfig, SelfConfig, PeersResponse } from '../src/shared/types.js';
import { PeerClient } from '../src/client/client.js';
import { ethers } from 'ethers';
import fs from 'fs';
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


describe('Peer Server Integration Tests', () => {
  let server: PeerServer;
  let app: any;
  let serverUrl: string;

  // Generate test guardians
  const testGuardianWallets: ethers.HDNodeWallet[] = [];
  const testGuardianAddresses: string[] = [];

  // Using 2 guardians for faster testing
  for (let i = 0; i < 2; i++) {
    const wallet = ethers.Wallet.createRandom();
    testGuardianWallets.push(wallet as ethers.HDNodeWallet);
    testGuardianAddresses.push(wallet.address);
  }

  const testConfig: ServerConfig = {
    port: 0, // Use 0 for automatic port assignment
    ethereum: {
      rpcUrl: 'https://eth.llamarpc.com',
      chainId: 1
    },
    wormholeContractAddress: '0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B',
    threshold: 1
  };

  // Mock guardian data for testing using generated wallets
  const mockWormholeData: WormholeGuardianData = {
    guardians: testGuardianAddresses
  };

  beforeEach(async () => {
    const mockDisplay = new MockDisplay();
    server = new PeerServer(testConfig, mockWormholeData, mockDisplay);
    app = server.getApp();

    // Start the server and get the actual port
    const serverPromise = new Promise<void>((resolve) => {
      server.start();
      // Give server a moment to start
      setTimeout(resolve, 100);
    });
    await serverPromise;

    // Get the actual port the server is running on
    const address = (server as any).server.address();
    serverUrl = `http://localhost:${address.port}`;
  });

  afterEach(() => {
    server.close();
  });

  it('should handle multiple clients submitting and all receiving complete results', async () => {
    // Create test files for guardian keys and TLS certificates
    const testDir = path.join(__dirname, '..', 'test-temp');
    if (!fs.existsSync(testDir)) {
      fs.mkdirSync(testDir, { recursive: true });
    }

    const testFiles: string[] = [];
    
    for (let i = 0; i < 2; i++) {
      // Create guardian key file with proper Wormhole format
      const keyPath = path.join(testDir, `guardian-${i}-key.txt`);
      const privateKeyBytes = Buffer.from(testGuardianWallets[i].privateKey.slice(2), 'hex');
      // Format: [tag (0x0A), length (32), ...32 bytes of key]
      const wormholeKeyData = Buffer.concat([
        Buffer.from([0x0A]), // tag
        Buffer.from([privateKeyBytes.length]), // length
        privateKeyBytes // the actual private key
      ]);
      const keyContent = `-----BEGIN WORMHOLE GUARDIAN PRIVATE KEY-----\n${wormholeKeyData.toString('base64')}\n-----END WORMHOLE GUARDIAN PRIVATE KEY-----`;
      fs.writeFileSync(keyPath, keyContent);
      testFiles.push(keyPath);

      // Create TLS cert file
      const certPath = path.join(testDir, `guardian-${i}-cert.pem`);
      const certContent = `-----BEGIN CERTIFICATE-----\nMIIDXTCCAkWgAwIBAgIJAKL5Z5Z5Z5Z5MA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV\nBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX\naWRnaXRzIFB0eSBMdGQwHhcNMjEwMTAxMDAwMDAwWhcNMjIwMTAxMDAwMDAwWjBF\nMQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50\nZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB\nCgKCAQEA0Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z\n3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z\n3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z\n3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z\n3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z\n3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z\n3Z3Z3Z3Z3Z3Z3Z3Z3QIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQAxMjM0NTY3ODkw\nMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDEyMzQ1Njc4\nOTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2\nNzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0\nNTY3ODkwMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE=\n-----END CERTIFICATE-----`;
      fs.writeFileSync(certPath, certContent);
      testFiles.push(certPath);
    }

    // Create test peer data for each guardian (guardianAddress will be set by server)
    const testPeers = [
      {
        hostname: 'guardian-0.example.com',
        tlsX509: path.join(testDir, 'guardian-0-cert.pem')
      },
      {
        hostname: 'guardian-1.example.com',
        tlsX509: path.join(testDir, 'guardian-1-cert.pem')
      }
    ];

    // Create and run test clients
    const clientPromises: Promise<PeersResponse>[] = [];

    for (let i = 0; i < 2; i++) {
      const clientConfig: SelfConfig = {
        guardianIndex: i,
        // @ts-ignore
        guardianPrivateKeyPath: path.join(testDir, `guardian-${i}-key.txt`),
        serverUrl: serverUrl,
        peer: testPeers[i]
      };

      const client = new PeerClient(clientConfig);
      clientPromises.push(client.run());
    }

    try {
      // Wait for all clients to complete
      const results = await Promise.all(clientPromises);

      // Verify each client received complete results
      results.forEach((result, index) => {
        console.log(`Client ${index} received ${result.peers.length} peers`);

        // Should have 2 peers (both guardians submitted)
        expect(Array.isArray(result.peers)).toBe(true);
        expect(result.peers).toHaveLength(2);

        // Should have our own peer data
        const ourGuardianAddr = testGuardianWallets[index].address;
        const ourPeer = result.peers.find((p: any) => p.guardianAddress === ourGuardianAddr);
        expect(ourPeer).toBeDefined();
        expect(ourPeer?.hostname).toBe(testPeers[index].hostname);

        // Should have the other guardian's peer data
        const otherGuardianAddr = testGuardianWallets[1 - index].address;
        const otherPeer = result.peers.find((p: any) => p.guardianAddress === otherGuardianAddr);
        expect(otherPeer).toBeDefined();
        expect(otherPeer?.hostname).toBe(testPeers[1 - index].hostname);
      });

      console.log('✅ Integration test passed: All clients received complete peer data');
    } finally {
      // Cleanup test files
      testFiles.forEach(file => {
        if (fs.existsSync(file)) {
          fs.unlinkSync(file);
        }
      });
      if (fs.existsSync(testDir)) {
        fs.rmdirSync(testDir);
      }
    }
  }, 30000); // 30 second timeout for integration test

  it('should handle staggered client submissions correctly', async () => {
    // Create test files for guardian keys and TLS certificates
    const testDir = path.join(__dirname, '..', 'test-temp-staggered');
    if (!fs.existsSync(testDir)) {
      fs.mkdirSync(testDir, { recursive: true });
    }

    const testFiles: string[] = [];
    
    for (let i = 0; i < 2; i++) {
      // Create guardian key file with proper Wormhole format
      const keyPath = path.join(testDir, `guardian-${i}-key.txt`);
      const privateKeyBytes = Buffer.from(testGuardianWallets[i].privateKey.slice(2), 'hex');
      // Format: [tag (0x0A), length (32), ...32 bytes of key]
      const wormholeKeyData = Buffer.concat([
        Buffer.from([0x0A]), // tag
        Buffer.from([privateKeyBytes.length]), // length
        privateKeyBytes // the actual private key
      ]);
      const keyContent = `-----BEGIN WORMHOLE GUARDIAN PRIVATE KEY-----\n\n${wormholeKeyData.toString('base64')}\n-----END WORMHOLE GUARDIAN PRIVATE KEY-----`;
      fs.writeFileSync(keyPath, keyContent);
      testFiles.push(keyPath);

      // Create TLS cert file
      const certPath = path.join(testDir, `guardian-${i}-cert.pem`);
      const certContent = `-----BEGIN CERTIFICATE-----\nMIIDXTCCAkWgAwIBAgIJAKL5Z5Z5Z5Z5MA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV\nBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX\naWRnaXRzIFB0eSBMdGQwHhcNMjEwMTAxMDAwMDAwWhcNMjIwMTAxMDAwMDAwWjBF\nMQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50\nZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB\nCgKCAQEA0Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z\n3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z\n3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z\n3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z\n3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z\n3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z3Z\n3Z3Z3Z3Z3Z3Z3Z3Z3QIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQAxMjM0NTY3ODkw\nMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDEyMzQ1Njc4\nOTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2\nNzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0\nNTY3ODkwMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE=\n-----END CERTIFICATE-----`;
      fs.writeFileSync(certPath, certContent);
      testFiles.push(certPath);
    }

    const testPeers = [
      {
        hostname: 'guardian-staggered-0.example.com',
        tlsX509: path.join(testDir, 'guardian-0-cert.pem')
      },
      {
        hostname: 'guardian-staggered-1.example.com',
        tlsX509: path.join(testDir, 'guardian-1-cert.pem')
      }
    ];

    try {
      // Create clients but submit with delays
      const clientConfigs: SelfConfig[] = [];
      for (let i = 0; i < 2; i++) {
        const clientConfig: SelfConfig = {
          guardianIndex: i,
          // @ts-ignore
          guardianPrivateKeyPath: path.join(testDir, `guardian-${i}-key.txt`),
          serverUrl: serverUrl,
          peer: testPeers[i]
        };
        clientConfigs.push(clientConfig);
      }

      // Submit first client immediately
      const firstClient = new PeerClient(clientConfigs[0]);
      const firstClientPromise = firstClient.run();

      // Wait a bit then submit second client
      setTimeout(async () => {
        const secondClient = new PeerClient(clientConfigs[1]);
        await secondClient.run();
      }, 1000);

      // Wait for both to complete
      const results = await Promise.all([
        firstClientPromise,
        new PeerClient(clientConfigs[1]).run()
      ]);

      // Verify results are consistent
      results.forEach((result) => {
        expect(Array.isArray(result.peers)).toBe(true);
        expect(result.peers).toHaveLength(2);
        
        const peer0 = result.peers.find((p: any) => p.guardianAddress === testGuardianWallets[0].address);
        const peer1 = result.peers.find((p: any) => p.guardianAddress === testGuardianWallets[1].address);
        
        expect(peer0).toBeDefined();
        expect(peer1).toBeDefined();
        expect(peer0?.hostname).toBe(testPeers[0].hostname);
        expect(peer1?.hostname).toBe(testPeers[1].hostname);
      });

      console.log('✅ Staggered submission test passed');
    } finally {
      // Cleanup test files
      testFiles.forEach(file => {
        if (fs.existsSync(file)) {
          fs.unlinkSync(file);
        }
      });
      if (fs.existsSync(testDir)) {
        fs.rmdirSync(testDir);
      }
    }
  }, 30000);
});
