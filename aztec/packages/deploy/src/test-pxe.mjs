// pxe-test.mjs
// A minimal script to test PXE connection

// Enable verbose debugging
process.env.DEBUG = '@aztec:*,node-fetch,*,http';

import { createPXEClient } from '@aztec/aztec.js';
import http from 'http';
import https from 'https';

const PXE_URL = 'http://localhost:8080';

// Direct fetch using node-fetch to test basic connectivity
async function testDirectFetch() {
  try {
    console.log("Testing direct HTTP request to PXE endpoint...");
    const response = await fetch(`${PXE_URL}/status`);
    const text = await response.text();
    console.log(`Direct HTTP request successful. Response: ${text}`);
    return true;
  } catch (error) {
    console.error("Direct HTTP request failed:", error);
    return false;
  }
}

// Manual implementation of PXE connection check
async function manualPXECheck() {
  try {
    console.log("Testing manual PXE status check...");
    const response = await fetch(`${PXE_URL}/status`);
    const text = await response.text();
    
    if (text === "OK") {
      console.log("PXE status check successful");
      return true;
    } else {
      console.log(`PXE status check received unexpected response: ${text}`);
      return false;
    }
  } catch (error) {
    console.error("Manual PXE status check failed:", error);
    return false;
  }
}

// Test actual PXE client connection
async function testPXEClient() {
  try {
    console.log("Creating PXE client...");
    
    // Configure HTTP agents
    const httpAgent = new http.Agent({
      keepAlive: true,
      timeout: 10000,
    });

    const httpsAgent = new https.Agent({
      keepAlive: true,
      timeout: 10000,
      rejectUnauthorized: false,
    });
    
    // Create PXE client
    const pxe = createPXEClient(PXE_URL, {
      httpAgent,
      httpsAgent,
    });
    
    console.log("PXE client created, testing getStatus method...");
    
    // Set a timeout to avoid hanging indefinitely
    const timeoutPromise = new Promise((_, reject) => {
      setTimeout(() => reject(new Error("Timeout after 10 seconds")), 10000);
    });
    
    // Test getStatus
    const statusPromise = pxe.getStatus().then(status => {
      console.log("PXE status:", status);
      return status;
    });
    
    // Race the promises
    await Promise.race([statusPromise, timeoutPromise]);
    console.log("PXE client test completed successfully");
    return true;
  } catch (error) {
    console.error("PXE client test failed:", error);
    return false;
  }
}

// Test basic block query
async function testBlockQuery() {
  try {
    console.log("Testing block number query...");
    
    const pxe = createPXEClient(PXE_URL);
    
    // Set a timeout to avoid hanging indefinitely
    const timeoutPromise = new Promise((_, reject) => {
      setTimeout(() => reject(new Error("Timeout after 10 seconds")), 10000);
    });
    
    // Try to get block number
    const blockPromise = pxe.getBlockNumber().then(blockNumber => {
      console.log("Current block number:", blockNumber);
      return blockNumber;
    });
    
    // Race the promises
    await Promise.race([blockPromise, timeoutPromise]);
    console.log("Block query test completed successfully");
    return true;
  } catch (error) {
    console.error("Block query test failed:", error);
    return false;
  }
}

// Get PXE version info
async function testVersionInfo() {
  try {
    console.log("Testing version info query...");
    
    const pxe = createPXEClient(PXE_URL);
    
    // Set a timeout to avoid hanging indefinitely
    const timeoutPromise = new Promise((_, reject) => {
      setTimeout(() => reject(new Error("Timeout after 10 seconds")), 10000);
    });
    
    // Try to get version info
    const versionPromise = fetch(`${PXE_URL}/version`).then(async (response) => {
      const text = await response.text();
      console.log("Version info:", text);
      return text;
    });
    
    // Race the promises
    await Promise.race([versionPromise, timeoutPromise]);
    console.log("Version info test completed successfully");
    return true;
  } catch (error) {
    console.error("Version info test failed:", error);
    return false;
  }
}

// Run all tests
async function runTests() {
  console.log("=== Beginning PXE Connection Tests ===");
  
  // Basic connectivity test
  const directFetchSuccess = await testDirectFetch();
  console.log("Direct fetch test:", directFetchSuccess ? "PASSED" : "FAILED");
  
  // Manual PXE status check
  const manualCheckSuccess = await manualPXECheck();
  console.log("Manual PXE check:", manualCheckSuccess ? "PASSED" : "FAILED");
  
  // PXE client test
  const pxeClientSuccess = await testPXEClient();
  console.log("PXE client test:", pxeClientSuccess ? "PASSED" : "FAILED");
  
  // Block query test
  const blockQuerySuccess = await testBlockQuery();
  console.log("Block query test:", blockQuerySuccess ? "PASSED" : "FAILED");
  
  // Version info test
  const versionInfoSuccess = await testVersionInfo();
  console.log("Version info test:", versionInfoSuccess ? "PASSED" : "FAILED");
  
  console.log("=== PXE Connection Tests Complete ===");
  
  // Print overall result
  if (directFetchSuccess && manualCheckSuccess && pxeClientSuccess && blockQuerySuccess && versionInfoSuccess) {
    console.log("All tests PASSED");
  } else {
    console.log("Some tests FAILED");
  }
}

// Run tests and exit
runTests()
  .catch(error => {
    console.error("Test error:", error);
  })
  .finally(() => {
    console.log("Tests completed");
  });