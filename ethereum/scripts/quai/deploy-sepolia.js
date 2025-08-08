const { ethers } = require("ethers");
const fs = require("fs");
require("dotenv").config({ path: ".env.sepolia" });

const ImplementationArtifact = require("../../artifacts/contracts/Implementation.sol/Implementation.json"); // Implementation deployed at: 0x002d7949233231c161aC8dd2E15Ea7eD2Aa68D24
const SetupArtifact = require("../../artifacts/contracts/Setup.sol/Setup.json"); // Setup deployed at: 0x006F923410a29c8f82eC4c53D245b8472A992A8B
const WormholeArtifact = require("../../artifacts/contracts/Wormhole.sol/Wormhole.json"); // Wormhole proxy deployed at: 0x004Accf29dD34f88E885e2BdFB1B0105059b3D08

async function main() {
  // Initialize provider and wallet
  const provider = new ethers.JsonRpcProvider(process.env.RPC_URL);
  const wallet = new ethers.Wallet(process.env.CYPRUS1_PK, provider);
  
  console.log("Deploying to Sepolia...");
  console.log("Deployer address:", wallet.address);
  console.log("Chain ID:", process.env.INIT_EVM_CHAIN_ID);
  console.log("Guardian address:", process.env.INIT_SIGNERS);

  // Deploy Implementation
  console.log("\n1. Deploying Implementation...");
  const Implementation = new ethers.ContractFactory(
    ImplementationArtifact.abi,
    ImplementationArtifact.bytecode,
    wallet
  );
  const implementation = await Implementation.deploy();
  await implementation.waitForDeployment();
  console.log("Implementation deployed at:", await implementation.getAddress());

  // Deploy Setup
  console.log("\n2. Deploying Setup...");
  const Setup = new ethers.ContractFactory(
    SetupArtifact.abi,
    SetupArtifact.bytecode,
    wallet
  );
  const setup = await Setup.deploy();
  await setup.waitForDeployment();
  console.log("Setup deployed at:", await setup.getAddress());

  // Prepare guardian set
  const guardianAddresses = process.env.INIT_SIGNERS.split(",");
  console.log("\n3. Guardian addresses:", guardianAddresses);

  // Create setup data for initial guardian set
  const setupInterface = new ethers.Interface(SetupArtifact.abi);
  const setupData = setupInterface.encodeFunctionData("setup", [
    await implementation.getAddress(),
    guardianAddresses,
    process.env.INIT_CHAIN_ID,
    process.env.INIT_GOV_CHAIN_ID,
    process.env.INIT_GOV_CONTRACT,
    process.env.INIT_EVM_CHAIN_ID
  ]);

  // Deploy Wormhole proxy
  console.log("\n4. Deploying Wormhole proxy...");
  const Wormhole = new ethers.ContractFactory(
    WormholeArtifact.abi,
    WormholeArtifact.bytecode,
    wallet
  );
  const wormhole = await Wormhole.deploy(
    await setup.getAddress(),
    setupData
  );
  await wormhole.waitForDeployment();
  const wormholeAddress = await wormhole.getAddress();
  console.log("Wormhole Core deployed at:", wormholeAddress);

  // Verify deployment
  console.log("\n5. Verifying deployment...");
  const wormholeContract = new ethers.Contract(
    wormholeAddress,
    ImplementationArtifact.abi,
    wallet
  );
  
  const chainId = await wormholeContract.chainId();
  const guardianSetIndex = await wormholeContract.getCurrentGuardianSetIndex();
  const guardianSet = await wormholeContract.getGuardianSet(guardianSetIndex);
  
  console.log("Chain ID:", chainId.toString());
  console.log("Guardian Set Index:", guardianSetIndex.toString());
  console.log("Guardian Set Keys:", guardianSet.keys);

  // Save deployment info
  const deploymentInfo = {
    network: "sepolia",
    chainId: process.env.INIT_EVM_CHAIN_ID,
    wormholeChainId: process.env.INIT_CHAIN_ID,
    contracts: {
      implementation: await implementation.getAddress(),
      setup: await setup.getAddress(),
      wormhole: wormholeAddress
    },
    guardians: guardianAddresses,
    deployer: wallet.address
  };

  fs.writeFileSync(
    "deployment-sepolia.json",
    JSON.stringify(deploymentInfo, null, 2)
  );

  console.log("\n=".repeat(60));
  console.log("SEPOLIA DEPLOYMENT COMPLETE");
  console.log("=".repeat(60));
  console.log("Wormhole Core Address:", wormholeAddress);
  console.log("Guardian Address:", guardianAddresses[0]);
  console.log("Deployment saved to: deployment-sepolia.json");
  console.log("=".repeat(60));
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });