// vaa-verification-service.mjs - TESTNET VERSION
import express from 'express';
import { createPXEClient, waitForPXE, Contract, loadContractArtifact } from '@aztec/aztec.js';
import { AccountWalletWithSecretKey } from '@aztec/aztec.js';
import { readFileSync } from 'fs';
import WormholeJson from "./contracts/target/wormhole_contracts-Wormhole.json" assert { type: "json" };

const app = express();
app.use(express.json());

const PORT = process.env.PORT || 8080;

// TESTNET CONFIGURATION
const PXE_URL = 'https://aztec-alpha-testnet-fullnode.zkv.xyz'; // Testnet PXE
const PRIVATE_KEY = '0x11914f36318813102e4838022ceed7b45643523f0332561678d8810bfc0db890'; // Your testnet private key
const CONTRACT_ADDRESS = '0x0db2e7e75a9b116ff046414fd0f3d5c7c356930f8abb84cb1d10e9dc436d9d04'; // Your deployed contract address on testnet

let pxe, wallet, wormholeContract, isReady = false;

// Initialize Aztec for Testnet
async function init() {
  console.log('🔄 Initializing Aztec TESTNET connection...');
  
  if (!PRIVATE_KEY) {
    throw new Error('PRIVATE_KEY environment variable is required for testnet');
  }
  
  if (!CONTRACT_ADDRESS) {
    throw new Error('CONTRACT_ADDRESS environment variable is required for testnet');
  }
  
  pxe = createPXEClient(PXE_URL);
  await waitForPXE(pxe);
  
  // Use your specific wallet for testnet (not test accounts)
  wallet = new AccountWalletWithSecretKey(pxe, PRIVATE_KEY);
  
  // Load your deployed contract on testnet
  const contractArtifact = loadContractArtifact(WormholeJson);
  wormholeContract = await Contract.at(CONTRACT_ADDRESS, contractArtifact, wallet);
  
  isReady = true;
  console.log(`✅ Connected to Wormhole contract on TESTNET: ${CONTRACT_ADDRESS}`);
  console.log(`✅ Using wallet: ${wallet.getAddress()}`);
  console.log(`✅ PXE URL: ${PXE_URL}`);
}

// Health check
app.get('/health', (req, res) => {
  res.json({ 
    status: isReady ? 'healthy' : 'initializing',
    network: 'testnet',
    timestamp: new Date().toISOString(),
    pxeUrl: PXE_URL,
    contractAddress: CONTRACT_ADDRESS,
    walletAddress: wallet ? wallet.getAddress().toString() : 'not connected'
  });
});

// Verify VAA
app.post('/verify', async (req, res) => {
  if (!isReady) {
    return res.status(503).json({ 
      success: false, 
      error: 'Service not ready - Aztec testnet connection still initializing' 
    });
  }

  try {
    const { vaaBytes } = req.body;
    
    if (!vaaBytes) {
      return res.status(400).json({
        success: false,
        error: 'vaaBytes is required'
      });
    }
    
    // Convert hex to buffer
    const hexString = vaaBytes.startsWith('0x') ? vaaBytes.slice(2) : vaaBytes;
    const vaaBuffer = Buffer.from(hexString, 'hex');
    
    // Pad to 2000 bytes for contract but pass actual length
    const paddedVAA = Buffer.alloc(2000);
    vaaBuffer.copy(paddedVAA, 0, 0, Math.min(vaaBuffer.length, 2000));
    
    // Convert to array for Aztec contract
    const vaaArray = Array.from(paddedVAA);
    const actualLength = vaaBuffer.length;
    
    console.log(`🔍 Verifying VAA on TESTNET (${vaaBuffer.length} bytes actual, ${paddedVAA.length} bytes padded)`);
    console.log(`📍 Contract: ${CONTRACT_ADDRESS}`);
    
    // Call verify_vaa function with padded bytes and actual length
    const tx = await wormholeContract.methods
      .verify_vaa(vaaArray, actualLength)
      .send()
      .wait();
    
    console.log(`✅ VAA verified successfully on TESTNET: ${tx.txHash}`);
    
    res.json({
      success: true,
      network: 'testnet',
      txHash: tx.txHash,
      contractAddress: CONTRACT_ADDRESS,
      message: 'VAA verified successfully on Aztec testnet',
      processedAt: new Date().toISOString()
    });
    
  } catch (error) {
    console.error('❌ VAA verification failed on TESTNET:', error.message);
    res.status(500).json({
      success: false,
      network: 'testnet',
      error: error.message,
      processedAt: new Date().toISOString()
    });
  }
});

// Test endpoint with Jorge's real Arbitrum Sepolia VAA
app.post('/test', async (req, res) => {
  // Jorge's real VAA from Arbitrum Sepolia that uses Guardian 0x13947Bd48b18E53fdAeEe77F3473391aC727C638
  // This VAA contains "Hello Wormhole!" message and has been verified on Wormholescan
  // Link: https://wormholescan.io/#/tx/0xf93fd41efeb09ff28174824d4abf6dbc06ac408953a9975aa4a403d434051efc?network=Testnet&view=advanced
  const realVAA = "010000000001004682bc4d5ff2e54dc2ee5e0eb64f5c6c07aa449ac539abc63c2be5c306a48f233e9300170a82adf3c3b7f43f23176fb079174a58d67d142477f646675d86eb6301684bfad4499602d22713000000000000000000000000697f31e074bf2c819391d52729f95506e0a72ffb0000000000000000c8000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000000e48656c6c6f20576f726d686f6c6521000000000000000000000000000000000000";
  
  console.log('🧪 Testing with Jorge\'s real Arbitrum Sepolia VAA on TESTNET');
  console.log('📍 Guardian: 0x13947Bd48b18E53fdAeEe77F3473391aC727C638');
  console.log('📍 Signature: 0x4682bc4d5ff2e54dc2ee5e0eb64f5c6c07aa449ac539abc63c2be5c306a48f233e9300170a82adf3c3b7f43f23176fb079174a58d67d142477f646675d86eb6301');
  console.log('📍 Expected message hash: 0xe64320fba193c98f2d0acf3a8c7479ec9b163192bfc19d4024497d4e4159758c');
  console.log('📍 WormholeScan: https://wormholescan.io/#/tx/0xf93fd41efeb09ff28174824d4abf6dbc06ac408953a9975aa4a403d434051efc?network=Testnet&view=advanced');
  
  req.body = { vaaBytes: realVAA };
  
  // Reuse the verify endpoint logic
  const verifyHandler = app._router.stack.find(layer => 
    layer.route && layer.route.path === '/verify'
  ).route.stack[0].handle;
  
  return verifyHandler(req, res);
});

// Start server
init().then(() => {
  app.listen(PORT, () => {
    console.log(`🚀 VAA Verification Service running on port ${PORT}`);
    console.log(`🌐 Network: TESTNET`);
    console.log(`📡 PXE: ${PXE_URL}`);
    console.log(`📄 Contract: ${CONTRACT_ADDRESS}`);
    console.log('Available endpoints:');
    console.log('  GET  /health - Health check');
    console.log('  POST /verify - Verify VAA on testnet');
    console.log('  POST /test   - Test with Jorge\'s real Arbitrum Sepolia VAA');
  });
}).catch(error => {
  console.error('❌ Failed to start testnet service:', error);
  console.log('\n📝 Required environment variables:');
  console.log('  PRIVATE_KEY=your_testnet_private_key');
  console.log('  CONTRACT_ADDRESS=your_deployed_contract_address');
  console.log('  AZTEC_PXE_URL=https://api.aztec.network (optional, defaults to testnet)');
  process.exit(1);
});