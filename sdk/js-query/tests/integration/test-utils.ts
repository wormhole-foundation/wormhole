import {
  createWalletClient,
  http,
  parseEther,
  formatEther,
  type Address,
  publicActions,
  defineChain,
  encodeFunctionData,
  WalletClient,
  Transport,
  Chain,
  PublicActions,
} from "viem";
import { LocalAccount, privateKeyToAccount } from "viem/accounts";
import axios from "axios";
import { Buffer } from "buffer";
import {
  EthCallData,
  EthCallQueryRequest,
  Network,
  PerChainQueryRequest,
  QueryRequest,
  sign,
} from "../../src";

// ============================================================================
// Constants
// ============================================================================

export const CI = process.env.CI;
export const ETH_NODE_URL = CI
  ? "http://eth-devnet:8545"
  : "http://localhost:8545";
export const SOLANA_NODE_URL = CI
  ? "http://solana-devnet:8899"
  : "http://localhost:8899";

// Define the local devnet chain (Ganache with chain ID 1337)
const devnet = defineChain({
  id: 1337,
  name: "Devnet",
  nativeCurrency: { name: "Ether", symbol: "ETH", decimals: 18 },
  rpcUrls: {
    default: { http: [ETH_NODE_URL] },
  },
});
export const SERVER_URL = CI ? "http://query-server:" : "http://localhost:";
export const CCQ_SERVER_URL = SERVER_URL + "6069/v1";
export const QUERY_URL = CCQ_SERVER_URL + "/query";

// Minter account (ganacheDefaults[0])
export const MINTER_PRIVATE_KEY =
  "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d" as `0x${string}`;
export const MINTER_ADDRESS =
  "0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1" as Address;

// Contract addresses from devnet
export const WETH_ADDRESS = "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E";
export const W_TOKEN_ADDRESS =
  "0x2D8BE6BF0baA74e0A907016679CaE9190e80dD0A" as Address;
export const STAKING_FACTORY_ADDRESS =
  (process.env.STAKING_FACTORY_ADDRESS as Address) ||
  ("0x8fed3F9126e7051DeA6c530920cb0BAE5ffa17a8" as Address);

// Query types -- get from staking-pools-deploy logs
// e.g. ENCODED_QUERY_TYPE: 0x0000000000000000000000000000000000000000000000000000000000000732
export const EVM_QUERY_TYPE =
  "0x0000000000000000000000000000000000000000000000000000000000000705";
export const SOLANA_QUERY_TYPE =
  "0x0000000000000000000000000000000000000000000000000000000000001805";

// ============================================================================
// Type Definitions
// ============================================================================

export interface StakeInfo {
  amount: string;
  conversionTableIndex: string;
  lockupEnd: string;
  accessEnd: string;
  lastClaimed: string;
  capacity: string;
}

export interface TransactionReceipt {
  status: boolean | bigint;
  transactionHash: string;
  blockNumber: number | bigint;
  [key: string]: any;
}

// ============================================================================
// Contract ABIs
// ============================================================================

export const ERC20_MINT_ABI = [
  {
    inputs: [
      { name: "to", type: "address" },
      { name: "amount", type: "uint256" },
    ],
    name: "mint",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function",
  },
] as const;

export const ERC20_ABI = [
  {
    inputs: [
      { name: "spender", type: "address" },
      { name: "amount", type: "uint256" },
    ],
    name: "approve",
    outputs: [{ name: "", type: "bool" }],
    stateMutability: "nonpayable",
    type: "function",
  },
  {
    inputs: [{ name: "account", type: "address" }],
    name: "balanceOf",
    outputs: [{ name: "", type: "uint256" }],
    stateMutability: "view",
    type: "function",
  },
  {
    inputs: [
      { name: "to", type: "address" },
      { name: "amount", type: "uint256" },
    ],
    name: "transfer",
    outputs: [{ name: "", type: "bool" }],
    stateMutability: "nonpayable",
    type: "function",
  },
] as const;

export const POOL_STAKE_ABI = [
  {
    inputs: [{ name: "_amount", type: "uint256" }],
    name: "stake",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function",
  },
  {
    inputs: [],
    name: "minimumStake",
    outputs: [{ name: "", type: "uint256" }],
    stateMutability: "view",
    type: "function",
  },
] as const;

export const POOL_ABI = [
  {
    inputs: [{ name: "staker", type: "address" }],
    name: "getStakeInfo",
    outputs: [
      { name: "amount", type: "uint256" },
      { name: "conversionTableIndex", type: "uint256" },
      { name: "lockupEnd", type: "uint48" },
      { name: "accessEnd", type: "uint48" },
      { name: "lastClaimed", type: "uint48" },
      { name: "capacity", type: "uint256" },
    ],
    stateMutability: "view",
    type: "function",
  },
  {
    inputs: [],
    name: "unstake",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function",
  },
] as const;

export const POOL_DELEGATION_ABI = [
  {
    inputs: [{ name: "_newSigner", type: "address" }],
    name: "setSigner",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function",
  },
  {
    inputs: [{ name: "staker", type: "address" }],
    name: "stakerSigners",
    outputs: [{ name: "", type: "address" }],
    stateMutability: "view",
    type: "function",
  },
] as const;

export const FACTORY_ABI = [
  {
    inputs: [{ name: "queryType", type: "bytes32" }],
    name: "queryTypePools",
    outputs: [{ name: "poolAddress", type: "address" }],
    stateMutability: "view",
    type: "function",
  },
] as const;

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Sleep for a specified number of milliseconds
 */
export function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Get current timestamp in seconds
 */
export function getCurrentTimestampSeconds(): number {
  return Math.floor(Date.now() / 1000);
}

/**
 * Creates a viem wallet client that can both read and write.
 * Uses publicActions extension to add read capabilities to wallet client.
 */
type ExtendedClient = WalletClient<Transport, Chain, LocalAccount> &
  PublicActions;

export function createClient(
  privateKey: `0x${string}` = MINTER_PRIVATE_KEY
): ExtendedClient {
  // Disable caching to avoid stale nonce issues in parallel tests
  // Each call creates a fresh client with up-to-date nonce
  const account = privateKeyToAccount(privateKey);

  const client = createWalletClient({
    account,
    chain: devnet,
    transport: http(ETH_NODE_URL),
  }).extend(publicActions);

  return client;
}

/**
 * Mint and transfer tokens to a wallet
 * Note: Uses transfer from account 0 (deployer with 1M tokens) since token contract doesn't support public minting
 */
export async function mintAndTransferTokens(
  to: Address,
  amount: string
): Promise<void> {
  const amountWei = parseEther(amount);

  // Retry logic for nonce conflicts - create fresh client on each attempt
  let lastError: any;
  for (let attempt = 0; attempt < 3; attempt++) {
    try {
      const client = createClient(); // Fresh client to get current nonce
      const hash = await client.writeContract({
        address: W_TOKEN_ADDRESS,
        abi: ERC20_ABI,
        functionName: "transfer",
        args: [to, amountWei],
      } as any);

      await client.waitForTransactionReceipt({ hash });
      return; // Success
    } catch (error: any) {
      lastError = error;
      if (error?.message?.includes("replacement transaction underpriced") ||
          error?.message?.includes("nonce too low")) {
        // Wait briefly before retry to let new nonce be picked up
        await new Promise(resolve => setTimeout(resolve, 100 * (attempt + 1)));
        continue;
      }
      throw error; // Re-throw if not a nonce error
    }
  }
  throw lastError; // All retries failed
}

/**
 * Unstake all tokens for an address
 */
export async function unstakeAll(
  poolAddress: Address,
  stakerPrivateKey: `0x${string}`
): Promise<void> {
  try {
    const client = createClient(stakerPrivateKey);
    const stakerAddress = client.account.address;

    const stakeInfo = (await client.readContract({
      address: poolAddress,
      abi: POOL_ABI,
      functionName: "getStakeInfo",
      args: [stakerAddress],
    } as any)) as any;

    const stakeAmount = BigInt(stakeInfo[0] || "0");
    if (stakeAmount === BigInt(0)) {
      console.log(`  Cleanup: ${stakerAddress} has no stake to unstake`);
      return;
    }

    const hash = await client.writeContract({
      address: poolAddress,
      abi: POOL_ABI,
      functionName: "unstake",
    } as any);

    await client.waitForTransactionReceipt({ hash });

    console.log(`  Cleanup: Successfully unstaked for ${stakerAddress}`);
  } catch (error: any) {
    const address = privateKeyToAccount(stakerPrivateKey).address;
    console.warn(`  Cleanup: Failed to unstake for ${address}:`, error.message);
  }
}

/**
 * Ensure a staker has the required stake amount
 */
export async function ensureStakerHasStake(
  poolAddress: Address,
  stakerPrivateKey: `0x${string}`,
  stakeAmountTokens: string
): Promise<void> {
  const client = createClient(stakerPrivateKey);
  const stakerAddress = client.account.address;
  const stakeAmountWei = parseEther(stakeAmountTokens);

  const stakeInfo = (await client.readContract({
    address: poolAddress,
    abi: POOL_ABI,
    functionName: "getStakeInfo",
    args: [stakerAddress],
  } as any)) as any;

  const currentStake = BigInt(stakeInfo[0]);

  // Check if staker already has enough stake
  if (currentStake >= stakeAmountWei) {
    return;
  }

  const neededAmount = stakeAmountWei - currentStake;

  // Minimum stake threshold to avoid contract rejections (typically 1000 tokens)
  // If we need less than this, skip staking to avoid errors
  const minStakeThreshold = parseEther("1000");

  if (neededAmount < minStakeThreshold) {
    // Close enough - staker has sufficient stake for the test
    // (within 1000 tokens of target, which is acceptable for rate limiting)
    console.log(`  Skipping small top-up: staker has ${formatEther(currentStake)} / ${stakeAmountTokens} tokens staked`);
    return;
  }

  if (neededAmount > BigInt(0)) {
    // Stake the exact needed amount
    const amountToStake = neededAmount;

    const balance = (await client.readContract({
      address: W_TOKEN_ADDRESS,
      abi: ERC20_ABI,
      functionName: "balanceOf",
      args: [stakerAddress],
    } as any)) as bigint;

    if (balance < amountToStake) {
      throw new Error(
        `Insufficient token balance. Need ${formatEther(amountToStake)} tokens, have ${formatEther(
          balance
        )} tokens`
      );
    }

    // Check if staker has ETH for gas, send if needed
    const ethBalance = await client.getBalance({
      address: stakerAddress,
    });

    if (ethBalance === BigInt(0)) {
      const minterClient = createClient();
      // Send ETH with retry for nonce conflicts
      let ethHash: `0x${string}` | undefined;
      for (let attempt = 0; attempt < 3; attempt++) {
        try {
          ethHash = await minterClient.sendTransaction({
            to: stakerAddress,
            value: parseEther("1"),
          } as any);
          break; // Success
        } catch (error: any) {
          if (attempt === 2 ||
              (!error?.message?.includes("replacement transaction underpriced") &&
               !error?.message?.includes("nonce too low"))) {
            throw error;
          }
          await new Promise(resolve => setTimeout(resolve, 100 * (attempt + 1)));
        }
      }
      if (ethHash) {
        await minterClient.waitForTransactionReceipt({ hash: ethHash });
      }
    }

    // Approve tokens
    const approveHash = await client.writeContract({
      address: W_TOKEN_ADDRESS,
      abi: ERC20_ABI,
      functionName: "approve",
      args: [poolAddress, amountToStake],
    } as any);
    await client.waitForTransactionReceipt({ hash: approveHash });

    // Stake tokens
    const stakeHash = await client.writeContract({
      address: poolAddress,
      abi: POOL_STAKE_ABI,
      functionName: "stake",
      args: [amountToStake],
    } as any);
    await client.waitForTransactionReceipt({ hash: stakeHash });
  }
}

/**
 * Get pool address from factory for a query type
 */
export async function getPoolAddress(
  factoryAddress: Address,
  queryType: string
): Promise<Address> {
  const client = createClient();
  return (await client.readContract({
    address: factoryAddress,
    abi: FACTORY_ABI,
    functionName: "queryTypePools",
    args: [queryType as `0x${string}`],
  } as any)) as Address;
}

/**
 * Setup axios error interceptor to avoid circular reference errors in Jest
 */
export function setupAxiosInterceptor() {
  const axios = require("axios");
  axios.interceptors.response.use(
    (r: any) => r,
    (err: any) => {
      const error = new Error(
        `${err.message}${err?.response?.data ? `: ${err.response.data}` : ""}`
      ) as any;
      error.response = err.response
        ? { data: err.response.data, status: err.response.status }
        : undefined;
      throw error;
    }
  );
}

/**
 * Create test EthCallData for a function call
 */
export function createTestEthCallData(
  to: string,
  name: string,
  outputType: string = "bytes"
): EthCallData {
  return {
    to,
    data: encodeFunctionData({
      abi: [
        {
          name,
          type: "function",
          inputs: [],
          outputs: [{ name, type: outputType }],
          stateMutability: "view",
        },
      ],
      functionName: name,
    }),
  };
}

/**
 * Setup wallets with stake in the pool
 */
export async function setupWalletsWithStake(
  wallets: Array<{ privateKey: `0x${string}`; address: Address }>,
  poolAddress: Address,
  stakeAmount: string,
  verbose: boolean = false
): Promise<void> {
  const stakeAmountWei = parseEther(stakeAmount);

  if (verbose) {
    console.log(
      `  Setting up ${wallets.length} wallets with ${stakeAmount} tokens each...`
    );
  }

  for (let i = 0; i < wallets.length; i++) {
    const wallet = wallets[i];

    if (verbose) {
      console.log(`    Wallet ${i + 1}/${wallets.length}: ${wallet.address}`);
    }

    // Send ETH with retry for nonce conflicts - create fresh client on each attempt
    let ethHash: `0x${string}` | undefined;
    for (let attempt = 0; attempt < 3; attempt++) {
      try {
        const minterClient = createClient(); // Fresh client to get current nonce
        ethHash = await minterClient.sendTransaction({
          to: wallet.address,
          value: parseEther("1"),
        } as any);
        break; // Success
      } catch (error: any) {
        if (attempt === 2 ||
            (!error?.message?.includes("replacement transaction underpriced") &&
             !error?.message?.includes("nonce too low"))) {
          throw error;
        }
        await new Promise(resolve => setTimeout(resolve, 100 * (attempt + 1)));
      }
    }
    if (ethHash) {
      const minterClient = createClient();
      await minterClient.waitForTransactionReceipt({ hash: ethHash });
    }

    await mintAndTransferTokens(wallet.address, stakeAmount);

    const walletClient = createClient(wallet.privateKey);

    const approveHash = await walletClient.writeContract({
      address: W_TOKEN_ADDRESS,
      abi: ERC20_ABI,
      functionName: "approve",
      args: [poolAddress, stakeAmountWei],
    } as any);
    await walletClient.waitForTransactionReceipt({ hash: approveHash });

    const stakeHash = await walletClient.writeContract({
      address: poolAddress,
      abi: POOL_STAKE_ABI,
      functionName: "stake",
      args: [stakeAmountWei],
    } as any);
    await walletClient.waitForTransactionReceipt({ hash: stakeHash });

    if (verbose) {
      console.log(`      ✓ Staked ${stakeAmount} tokens`);
    }
  }
}

/**
 * Setup delegation from a staker to a signer
 */
export async function setupDelegation(
  poolAddress: Address,
  stakerPrivateKey: `0x${string}`,
  signerAddress: string
): Promise<void> {
  const client = createClient(stakerPrivateKey);
  const stakerAddress = client.account.address;

  const hash = await client.writeContract({
    address: poolAddress,
    abi: POOL_DELEGATION_ABI,
    functionName: "setSigner",
    args: [signerAddress],
  } as any);

  const receipt = await client.waitForTransactionReceipt({ hash });

  if (receipt.status !== "success") {
    throw new Error("Failed to set signer for delegation");
  }

  const newSigner = (await client.readContract({
    address: poolAddress,
    abi: POOL_DELEGATION_ABI,
    functionName: "stakerSigners",
    args: [stakerAddress],
  } as any)) as string;

  if (newSigner.toLowerCase() !== signerAddress.toLowerCase()) {
    throw new Error("Signer was not set correctly");
  }
}

/**
 * Send a query to the CCQ server
 */
export async function sendQuery(
  privateKey: `0x${string}`,
  address: Address,
  blockNumber: number,
  nonce: number,
  network: Network = "DEVNET"
): Promise<{ status: number; elapsed: number; data?: any }> {
  const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
  const decimalsCallData = createTestEthCallData(
    WETH_ADDRESS,
    "decimals",
    "uint8"
  );

  const ethCall = new EthCallQueryRequest(blockNumber, [
    nameCallData,
    decimalsCallData,
  ]);

  const chainId = 2;
  const ethQuery = new PerChainQueryRequest(chainId, ethCall);
  const timestamp = getCurrentTimestampSeconds();
  const request = new QueryRequest(nonce, timestamp, [ethQuery], address);

  const serialized = request.serialize();
  const digest = QueryRequest.digest(network, serialized);
  const signature = sign(privateKey.slice(2), digest);

  const startTime = Date.now();
  try {
    const response = await axios.post(QUERY_URL, {
      signature,
      bytes: Buffer.from(serialized).toString("hex"),
      staker: address,
    });
    return {
      status: response.status,
      elapsed: Date.now() - startTime,
      data: response.data,
    };
  } catch (error: any) {
    return {
      status: error.response?.status || 0,
      elapsed: Date.now() - startTime,
      data: error.response?.data,
    };
  }
}

/**
 * Get the current Solana slot for a given commitment level
 */
export async function getSolanaSlot(commitment: string): Promise<bigint> {
  const response = await axios.post(SOLANA_NODE_URL, {
    jsonrpc: "2.0",
    id: 1,
    method: "getSlot",
    params: [{ commitment, transactionDetails: "none" }],
  });

  return response.data.result;
}

/**
 * Send a Solana query to the CCQ server
 */
export async function sendSolanaQuery(
  privateKey: `0x${string}`,
  stakerAddress: Address,
  nonce: number,
  queries: PerChainQueryRequest[],
  network: Network = "DEVNET"
): Promise<{ status: number; data?: any }> {
  const timestamp = getCurrentTimestampSeconds();
  const request = new QueryRequest(nonce, timestamp, queries, stakerAddress);
  const serialized = request.serialize();
  const digest = QueryRequest.digest(network, serialized);
  const signature = sign(privateKey.slice(2), digest);

  try {
    const response = await axios.post(QUERY_URL, {
      signature,
      bytes: Buffer.from(serialized).toString("hex"),
      staker: stakerAddress,
    });
    return {
      status: response.status,
      data: response.data,
    };
  } catch (error: any) {
    return {
      status: error.response?.status || 0,
      data: error.response?.data,
    };
  }
}
