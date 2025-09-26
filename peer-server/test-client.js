import { PeerServer } from './dist/server.js';
import { Display } from './dist/display.js';
import { WormholeGuardianData, ServerConfig } from './dist/types.js';
import { PeerClient } from './dist/client.js';
import { ethers } from 'ethers';

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

async function testClient() {
  console.log('Testing PeerClient integration...');

  // Generate test guardians
  const testGuardianWallets = [];
  const testGuardianAddresses = [];

  // Using 2 guardians for faster testing
  for (let i = 0; i < 2; i++) {
    const wallet = ethers.Wallet.createRandom();
    testGuardianWallets.push(wallet);
    testGuardianAddresses.push(wallet.address);
  }

  const testConfig = {
    port: 0, // Use 0 for automatic port assignment
    ethereum: {
      rpcUrl: 'https://eth.llamarpc.com',
      chainId: 1
    },
    wormholeContractAddress: '0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B'
  };

  // Mock guardian data for testing using generated wallets
  const mockWormholeData = {
    keys: testGuardianAddresses
  };

  // Create a new server instance
  const mockDisplay = new MockDisplay();
  const server = new PeerServer(testConfig, mockWormholeData, mockDisplay);

  // Start the server
  server.start();

  // Give server a moment to start
  await new Promise(resolve => setTimeout(resolve, 100));

  // Get the actual port the server is running on
  const address = server.server.address();
  const serverUrl = `http://localhost:${address.port}`;

  console.log(`Server started on ${serverUrl}`);

  // Create test peer data for each guardian
  const testPeers = [
    {
      Hostname: 'guardian-0.example.com',
      TlsX509: 'test-cert-0',
      Port: 8080
    },
    {
      Hostname: 'guardian-1.example.com',
      TlsX509: 'test-cert-1',
      Port: 8081
    }
  ];

  try {
    // Create and run test clients
    const clientPromises = [];

    for (let i = 0; i < 2; i++) {
      const clientConfig = {
        guardianIndex: i,
        guardianKey: testGuardianWallets[i].privateKey,
        serverUrl: serverUrl,
        peer: testPeers[i]
      };

      const client = new PeerClient(clientConfig);
      clientPromises.push(client.run());
    }

    // Wait for all clients to complete
    const results = await Promise.all(clientPromises);

    // Verify each client received complete results
    results.forEach((peers, index) => {
      console.log(`Client ${index} received ${Object.keys(peers).length} peers:`, Object.keys(peers));

      // Should have 2 peers (both guardians submitted)
      if (Object.keys(peers).length !== 2) {
        throw new Error(`Expected 2 peers, got ${Object.keys(peers).length}`);
      }

      // Should have our own peer data
      const ourGuardianAddr = testGuardianWallets[index].address;
      if (!peers[ourGuardianAddr]) {
        throw new Error(`Missing peer data for guardian ${ourGuardianAddr}`);
      }
      if (JSON.stringify(peers[ourGuardianAddr]) !== JSON.stringify(testPeers[index])) {
        throw new Error(`Peer data mismatch for guardian ${ourGuardianAddr}`);
      }

      // Should have the other guardian's peer data
      const otherGuardianAddr = testGuardianWallets[1 - index].address;
      if (!peers[otherGuardianAddr]) {
        throw new Error(`Missing peer data for guardian ${otherGuardianAddr}`);
      }
      if (JSON.stringify(peers[otherGuardianAddr]) !== JSON.stringify(testPeers[1 - index])) {
        throw new Error(`Peer data mismatch for guardian ${otherGuardianAddr}`);
      }
    });

    console.log('✅ Integration test passed: All clients received complete peer data');

    // Clean up
    server.close();
    process.exit(0);
  } catch (error) {
    console.error('❌ Test failed:', error);
    server.close();
    process.exit(1);
  }
}

testClient().catch(error => {
  console.error('Unhandled error:', error);
  process.exit(1);
});
