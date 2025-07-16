// vaa-verification-service.mjs
import express from 'express';
import { createPXEClient, waitForPXE, Contract, loadContractArtifact } from '@aztec/aztec.js';
import { getInitialTestAccountsWallets } from '@aztec/accounts/testing';
import { readFileSync } from 'fs';
import WormholeJson from "./contracts/target/wormhole_contracts-Wormhole.json" assert { type: "json" };

const app = express();
app.use(express.json());

const PORT = process.env.PORT || 8080;
const PXE_URL = process.env.AZTEC_PXE_URL || 'http://localhost:8090';

let pxe, wallet, wormholeContract, isReady = false;

// Initialize Aztec
async function init() {
  console.log('🔄 Initializing Aztec connection...');
  
  pxe = createPXEClient(PXE_URL);
  await waitForPXE(pxe);
  
  const wallets = await getInitialTestAccountsWallets(pxe);
  wallet = wallets[0];
  
  // Load contract address from your existing addresses.json
  const addresses = JSON.parse(readFileSync('./packages/deploy/src/addresses.json', 'utf8'));
  const contractArtifact = loadContractArtifact(WormholeJson);
  wormholeContract = await Contract.at(addresses.wormhole, contractArtifact, wallet);
  
  isReady = true;
  console.log(`✅ Connected to Wormhole contract: ${addresses.wormhole}`);
  console.log(`✅ Using wallet: ${wallet.getAddress()}`);
}

// Health check
app.get('/health', (req, res) => {
  res.json({ 
    status: isReady ? 'healthy' : 'initializing',
    timestamp: new Date().toISOString(),
    pxeUrl: PXE_URL,
    walletAddress: wallet ? wallet.getAddress().toString() : 'not connected'
  });
});

// Verify VAA
app.post('/verify', async (req, res) => {
  if (!isReady) {
    return res.status(503).json({ 
      success: false, 
      error: 'Service not ready - Aztec connection still initializing' 
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
    
    console.log(`🔍 Verifying VAA (${vaaBuffer.length} bytes actual, ${paddedVAA.length} bytes padded)`);
    
    // Call verify_vaa function with padded bytes and actual length
    const tx = await wormholeContract.methods
      .verify_vaa(vaaArray, actualLength)
      .send()
      .wait();
    
    console.log(`✅ VAA verified successfully: ${tx.txHash}`);
    
    res.json({
      success: true,
      txHash: tx.txHash,
      message: 'VAA verified successfully',
      processedAt: new Date().toISOString()
    });
    
  } catch (error) {
    console.error('❌ VAA verification failed:', error.message);
    res.status(500).json({
      success: false,
      error: error.message,
      processedAt: new Date().toISOString()
    });
  }
});

// Test endpoint with sample data
app.post('/test', async (req, res) => {
  const sampleVAA = "01000000000000015f00081f2c84eb31fb19ea3f0161648c447d86d77bd709056" +
    "2b58880120000000000000000000000000000000000000000000000000000000000" +
    "0000000000000000000000000000000000144b90000000000000009c80000000000" +
    "0000000000000000000000000000000000000000000000000000000000000000000" +
    "0000000000000000000000000000000000000000000000000000000000000000000" +
    "0000000020000000000000000000000000000000000000000000000000000000000" +
    "0000000000000000000000000000000000000000000000000000000000000000000" +
    "0000000000000000000000000000000000000000000000000e48656c6c6f20576f" +
    "726d686f6c6521";
  
  req.body = { vaaBytes: sampleVAA };
  
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
    console.log('Available endpoints:');
    console.log('  GET  /health - Health check');
    console.log('  POST /verify - Verify VAA');
    console.log('  POST /test   - Test with sample data');
  });
}).catch(error => {
  console.error('❌ Failed to start service:', error);
  process.exit(1);
});