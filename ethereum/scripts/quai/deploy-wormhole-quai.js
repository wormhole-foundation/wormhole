const hre = require("hardhat");
const { deployMetadata } = require("hardhat");
require("dotenv").config();
const fs = require("fs");

// Quai SDK (ethers v6 fork)
const {
  Wallet,
  ContractFactory,
  JsonRpcProvider,
  Interface,
  getAddress,
  hexlify,
  isAddress,
} = require("quais");

// ABIs & Bytecode (Hardhat artifacts)
const ImplementationArtifact = require("../../artifacts/contracts/Implementation.sol/Implementation.json"); // Implementation deployed at: 0x0061bE6522a55E2AAefb3ad72526b61f8B63D226
const SetupArtifact = require("../../artifacts/contracts/Setup.sol/Setup.json"); // Setup deployed at: 0x002f4E65Bee725C69925Fa41e4Ecc03792C8D7EC
const WormholeArtifact = require("../../artifacts/contracts/Wormhole.sol/Wormhole.json"); // Wormhole proxy deployed at: 0x004Accf29dD34f88E885e2BdFB1B0105059b3D08

/*
 * Expected environment variables (see .env.example)
 *   PASSWORD                  – password to decrypt the wallet.json file
 *   RPC_URL                  – Quai JSON-RPC endpoint
 *   INIT_SIGNERS        – comma-separated list of guardian addresses
 *   INIT_CHAIN_ID                 – Wormhole chain ID for this network (uint16)
 *   INIT_GOV_CHAIN_ID      – Governance chain ID (uint16)
 *   INIT_GOV_CONTRACT      – 32-byte hex string of governance contract address (bytes32)
 *   INIT_EVM_CHAIN_ID             – Underlying EVM chainId (uint256)
 */
function getEnv(name) {
  const value = process.env[name];
  if (value === undefined || value === "") {
    throw new Error(`Missing required env var: ${name}`);
  }
  return value;
}

/*
   loadWalletFromFile loads an encrypted wallet from a file and decrypts it with a password
   walletPath: path to the wallet.json file
   password: password to decrypt the wallet.json file
   returns: a Wallet object
*/
async function loadWalletFromFile(walletPath, password) {
  try {
    const walletData = fs.readFileSync(walletPath, "utf8");
    const wallet = await Wallet.fromEncryptedJson(walletData, password);
    return wallet;
  } catch (error) {
    if (error.code === "ENOENT") {
      throw new Error(`Wallet file not found: ${walletPath}`);
    } else if (error.message.includes("invalid password")) {
      throw new Error("Invalid password for wallet decryption");
    } else {
      throw new Error(`Failed to load wallet: ${error.message}`);
    }
  }
}

async function main() {
  // ---------------------------------------------------------------------------
  // Provider / Wallet ----------------------------------------------------------
  // ---------------------------------------------------------------------------
  const provider = new JsonRpcProvider(getEnv("RPC_URL"), undefined, {
    // Quai requires pathing
    usePathing: true,
  });

  // Load encrypted wallet
  // We use an encrypted wallet file for the deployer to avoid exposing the private key to the console or disk
  // But you can use a private key directly by uncommenting the following line and commenting out the loadWalletFromFile line
  // let wallet = new Wallet(getEnv("PRIVATE_KEY"), provider);
  const password = getEnv("PASSWORD");
  const walletPath = "./wallet.json";
  
  console.log("Loading encrypted wallet from:", walletPath);
  let wallet = await loadWalletFromFile(walletPath, password);
  
  // Connect wallet to provider
  wallet = wallet.connect(provider);
  console.log("Wallet address: ", wallet.address);
  console.log("Wallet balance: ", await provider.getBalance(wallet.address));
  console.log("Wallet nonce: ", await provider.getTransactionCount(wallet.address));

  // ---------------------------------------------------------------------------
  // Optional address grinding validation (wallet address should start with 0x00)
  // ---------------------------------------------------------------------------
  if (!wallet.address.toLowerCase().startsWith("0x00")) {
    console.warn(
      `WARNING: Deployer address ${wallet.address} does not start with 0x00 - ` +
        `contracts may deploy to the wrong shard. Consider grinding a key that satisfies this.`
    );
  }

  // ---------------------------------------------------------------------------
  // Gather constructor / setup parameters -------------------------------------
  // ---------------------------------------------------------------------------
  const initialGuardians = getEnv("INIT_SIGNERS")
    .split(",")
    .map((a) => getAddress(a.trim()));

  const chainId = Number(getEnv("INIT_CHAIN_ID"));
  const governanceChainId = Number(getEnv("INIT_GOV_CHAIN_ID"));
  const governanceContractRaw = getEnv("INIT_GOV_CONTRACT");

  // ensure bytes32
  const governanceContract = hexlify(governanceContractRaw);
  if (governanceContract.length !== 66) {
    throw new Error("GOVERNANCE_CONTRACT must be 32-byte hex string (0x + 64 hex chars)");
  }

  const evmChainId = BigInt(getEnv("INIT_EVM_CHAIN_ID"));

  console.log("Deploy parameters:\n", {
    deployer: wallet.address,
    initialGuardians,
    chainId,
    governanceChainId,
    governanceContract,
    evmChainId: evmChainId.toString(),
  });
  let wormholeAddress;

  // ---------------------------------------------------------------------------
  // Deploy Implementation ------------------------------------------------------
  // ---------------------------------------------------------------------------
  console.log("\nDeploying Implementation...");
  let ipfsHash = await deployMetadata.pushMetadataToIPFS("Implementation")
  const ImplementationFactory = new ContractFactory(
    ImplementationArtifact.abi,
    ImplementationArtifact.bytecode,
    wallet,
    ipfsHash
  );
  
  const implementation = await ImplementationFactory.deploy();
  console.log("  tx hash:", implementation.deploymentTransaction().hash);
  await implementation.waitForDeployment();
  const implementationAddress = await implementation.getAddress();
  console.log("  Implementation deployed at:", implementationAddress);

  // ---------------------------------------------------------------------------
  // Deploy Setup ---------------------------------------------------------------
  // ---------------------------------------------------------------------------
  console.log("\nDeploying Setup...");
  ipfsHash = await deployMetadata.pushMetadataToIPFS("Setup")
  const SetupFactory = new ContractFactory(
    SetupArtifact.abi,
    SetupArtifact.bytecode,
    wallet,
    ipfsHash
  );
  const setup = await SetupFactory.deploy();
  console.log("  tx hash:", setup.deploymentTransaction().hash);
  await setup.waitForDeployment();
  const setupAddress = await setup.getAddress();
  console.log("  Setup deployed at:", setupAddress);

  // ---------------------------------------------------------------------------
  // Encode initData for Wormhole proxy ----------------------------------------
  // ---------------------------------------------------------------------------
  const setupIface = new Interface(SetupArtifact.abi);
  const initData = setupIface.encodeFunctionData("setup", [
    implementationAddress,
    initialGuardians,
    chainId,
    governanceChainId,
    governanceContract,
    evmChainId,
  ]);

  // ---------------------------------------------------------------------------
  // Deploy Wormhole Proxy ------------------------------------------------------
  // ---------------------------------------------------------------------------
  console.log("\nDeploying Wormhole (ERC1967Proxy)...");
  ipfsHash = await deployMetadata.pushMetadataToIPFS("Wormhole")
  const WormholeFactory = new ContractFactory(
    WormholeArtifact.abi,
    WormholeArtifact.bytecode,
    wallet,
    ipfsHash
  );
  const wormhole = await WormholeFactory.deploy(setupAddress, initData);
  console.log("  tx hash:", wormhole.deploymentTransaction().hash);
  await wormhole.waitForDeployment();
  wormholeAddress = await wormhole.getAddress();
  console.log("  Wormhole proxy deployed at:", wormholeAddress);

  console.log("\n✅ Wormhole Core deployment complete!");
  console.log("\nDeployed addresses:");
  console.log("  Implementation:", implementationAddress);
  console.log("  Setup:", setupAddress);
  console.log("  Wormhole Core:", wormholeAddress);
  console.log("\nThis Wormhole Core contract can now be used with NTT deployments.");
}

main()
  .then(() => process.exit(0))
  .catch((err) => {
    console.error(err);
    process.exit(1);
  });
