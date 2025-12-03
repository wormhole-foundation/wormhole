import fs from "fs";
import yargs from "yargs";
import { hideBin } from 'yargs/helpers';
import { createWalletClient, defineChain, http, isHex, encodePacked } from "viem";
import { privateKeyToAccount } from "viem/accounts";
import { waitForTransactionReceipt } from "viem/actions";

// Default contract address for WormholeVerifier
const DEFAULT_CONTRACT_ADDRESS = "0x0000000000000000000000000000000000000000"; // TODO: Update with actual deployed address

interface Args {
  vaaFile?: string;
  guardianMessage?: string;
  limit?: number;
  contractAddress: string;
  rpcUrl: string;
  chainId: number;
  signer: string;
  opcode: number;
  dryRun: boolean;
}

async function main() {
  const parser = yargs(hideBin(process.argv))
    .option('opcode', {
      description: 'Update opcode: 0=SET_SHARD_ID, 1=APPEND_SCHNORR_KEY, 2=PULL_MULTISIG_KEY_DATA',
      demandOption: true,
      type: 'number',
      alias: 'o',
      choices: [0, 1, 2],
    })
    .option('vaa-file', {
      description: '[Opcode 1 only] Path to file containing base64-encoded governance VAA for APPEND_SCHNORR_KEY',
      type: 'string',
      alias: 'v',
    })
    .option('guardian-message', {
      description: '[Opcode 0 only] Path to file containing base64-encoded signed guardian message for SET_SHARD_ID',
      type: 'string',
      alias: 'g',
    })
    .option('limit', {
      description: '[Opcode 2 only] Number of multisig keys to pull (uint32)',
      type: 'number',
      alias: 'l',
    })
    .option('contract-address', {
      description: 'Address of the WormholeVerifier contract',
      type: 'string',
      default: DEFAULT_CONTRACT_ADDRESS,
      alias: 'c',
    })
    .option('rpc-url', {
      description: 'RPC endpoint URL for the target chain',
      type: 'string',
      default: 'https://eth.llamarpc.com',
      alias: 'r',
    })
    .option('chain-id', {
      description: 'EIP-155 Chain ID',
      type: 'number',
      default: 1,
      alias: 'i',
    })
    .option('signer', {
      description: 'Path to JSON file containing signer private key (hex string with 0x prefix)',
      demandOption: true,
      type: 'string',
      alias: 's',
    })
    .option('dry-run', {
      description: 'Simulate the transaction without actually sending it',
      type: 'boolean',
      default: false,
      alias: 'd',
    })
    .help()
    .alias('help', 'h')
    .check((argv) => {
      // Validate required parameters based on opcode
      if (argv.opcode === 0 && !argv.guardianMessage) {
        throw new Error('--guardian-message is required for opcode 0 (SET_SHARD_ID)');
      }
      if (argv.opcode === 1 && !argv.vaaFile) {
        throw new Error('--vaa-file is required for opcode 1 (APPEND_SCHNORR_KEY)');
      }
      if (argv.opcode === 2 && argv.limit === undefined) {
        throw new Error('--limit is required for opcode 2 (PULL_MULTISIG_KEY_DATA)');
      }
      return true;
    });

  const args = await parser.parse() as Args;

  let dataBytes: Buffer | undefined;
  
  // Load data based on opcode
  if (args.opcode === 0) {
    // SET_SHARD_ID: Load guardian message
    if (!fs.existsSync(args.guardianMessage!)) {
      console.error(`❌ Guardian message file not found at ${args.guardianMessage}`);
      process.exit(1);
    }

    const messageBase64 = fs.readFileSync(args.guardianMessage!, 'utf-8').trim();
    try {
      dataBytes = Buffer.from(messageBase64, 'base64');
    } catch (error: any) {
      console.error(`❌ Failed to decode base64 guardian message: ${error?.message || error}`);
      process.exit(1);
    }

    console.log(`📄 Loaded guardian message (${dataBytes.length} bytes) from ${args.guardianMessage}`);
  } else if (args.opcode === 1) {
    // APPEND_SCHNORR_KEY: Load VAA
    if (!fs.existsSync(args.vaaFile!)) {
      console.error(`❌ VAA file not found at ${args.vaaFile}`);
      process.exit(1);
    }

    const vaaBase64 = fs.readFileSync(args.vaaFile!, 'utf-8').trim();
    try {
      dataBytes = Buffer.from(vaaBase64, 'base64');
    } catch (error: any) {
      console.error(`❌ Failed to decode base64 VAA: ${error?.message || error}`);
      process.exit(1);
    }

    console.log(`📄 Loaded VAA (${dataBytes.length} bytes) from ${args.vaaFile}`);
  } else {
    // PULL_MULTISIG_KEY_DATA: No file needed, just the limit
    console.log(`📄 Using limit: ${args.limit}`);
  }

  // Read and validate signer
  if (!fs.existsSync(args.signer)) {
    console.error(`❌ Signer file not found at ${args.signer}`);
    process.exit(1);
  }

  const signerFile = fs.readFileSync(args.signer, 'utf-8');
  let signerKey: string;
  try {
    signerKey = JSON.parse(signerFile);
  } catch (error: any) {
    console.error(`❌ Failed to parse signer file: ${error?.message || error}`);
    process.exit(1);
  }

  // TODO: Do we want parseCrypto.ts to be used here?
  if (typeof signerKey !== "string" || !isHex(signerKey)) {
    console.error("❌ Signer file must contain a hex string with 0x prefix");
    process.exit(1);
  }

  const account = privateKeyToAccount(signerKey);
  console.log(`🔑 Using signer address: ${account.address}`);

  // Validate contract address
  if (!isHex(args.contractAddress)) {
    console.error("❌ Contract address must be a valid hex address");
    process.exit(1);
  }

  // Setup chain and wallet client
  const viemChain = defineChain({
    id: args.chainId,
    name: `Chain ${args.chainId}`,
    nativeCurrency: {
      decimals: 18,
      name: 'ETH',
      symbol: 'ETH',
    },
    rpcUrls: {
      default: {
        http: [args.rpcUrl],
      },
    },
  });

  const walletClient = createWalletClient({
    chain: viemChain,
    transport: http(args.rpcUrl),
    account,
  });

  // Prepare the update call data based on opcode
  let updateData: `0x${string}`;
  
  if (args.opcode === 1) {
    // APPEND_SCHNORR_KEY: opcode (1 byte) + vaa length (2 bytes) + vaa data
    const vaaLength = dataBytes!.length;
    updateData = encodePacked(
      ['uint8', 'uint16', 'bytes'],
      [args.opcode, vaaLength, `0x${dataBytes!.toString('hex')}`]
    );
    console.log(`📦 Prepared APPEND_SCHNORR_KEY data (${updateData.length} bytes)`);
  } else if (args.opcode === 0) {
    // SET_SHARD_ID: opcode (1 byte) + guardian message data
    updateData = encodePacked(
      ['uint8', 'bytes'],
      [args.opcode, `0x${dataBytes!.toString('hex')}`]
    );
    console.log(`📦 Prepared SET_SHARD_ID data (${updateData.length} bytes)`);
  } else if (args.opcode === 2) {
    // PULL_MULTISIG_KEY_DATA: opcode (1 byte) + limit (4 bytes)
    updateData = encodePacked(
      ['uint8', 'uint32'],
      [args.opcode, args.limit!]
    );
    console.log(`📦 Prepared PULL_MULTISIG_KEY_DATA with limit ${args.limit}`);
  } else {
    console.error("❌ Invalid opcode");
    process.exit(1);
  }

  console.log(`📍 Target contract: ${args.contractAddress}`);
  console.log(`🌐 Chain ID: ${args.chainId}`);
  console.log(`🔗 RPC URL: ${args.rpcUrl}`);

  // Simple ABI for the update function
  const abi = [
    {
      name: 'update',
      type: 'function',
      stateMutability: 'nonpayable',
      inputs: [{ name: 'data', type: 'bytes' }],
      outputs: [],
    },
  ] as const;

  if (args.dryRun) {
    console.log('\n🧪 DRY RUN MODE - Transaction will not be sent');
    console.log('Update data (hex):', updateData);
    return;
  }

  try {
    console.log('\n📤 Sending transaction...');
    const txHash = await walletClient.writeContract({
      address: args.contractAddress as `0x${string}`,
      abi,
      functionName: 'update',
      args: [updateData],
    });

    console.log(`✅ Transaction sent: ${txHash}`);
    console.log('⏳ Waiting for confirmation...');

    const receipt = await waitForTransactionReceipt(walletClient, {
      hash: txHash,
    });

    if (receipt.status === 'success') {
      console.log(`✅ Transaction confirmed in block ${receipt.blockNumber}`);
      console.log(`⛽ Gas used: ${receipt.gasUsed}`);
    } else {
      console.error('❌ Transaction failed');
      process.exit(1);
    }
  } catch (error: any) {
    console.error(`❌ Transaction failed: ${error?.message || error}`);
    if (error?.cause) {
      console.error('Cause:', error.cause);
    }
    process.exit(1);
  }
}

main().catch((error) => {
  console.error(error?.stack || error);
  process.exit(1);
});
