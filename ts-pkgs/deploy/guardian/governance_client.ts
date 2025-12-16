import fs from "fs";
import { createWalletClient, defineChain, http, isHex, encodePacked, Hex } from "viem";
import { privateKeyToAccount } from "viem/accounts";
import { waitForTransactionReceipt } from "viem/actions";
import yargs from "yargs";
import { hideBin } from 'yargs/helpers';
import { parseGuardianKey, errorMsg, errorStack } from '@xlabs-xyz/peer-lib';

// Default contract address for WormholeVerifier
const DEFAULT_CONTRACT_ADDRESS = "0x0000000000000000000000000000000000000000"; // TODO: Update with actual deployed address

const UPDATE_SET_SHARD_ID = 0;
const UPDATE_APPEND_SCHNORR_KEY = 1;
const UPDATE_PULL_MULTISIG_KEY_DATA = 2;

const UPDATE_ABI = [
  {
    name: 'update',
    type: 'function',
    stateMutability: 'nonpayable',
    inputs: [{ name: 'data', type: 'bytes' }],
    outputs: [],
  },
] as const;

type Args = {
  contractAddress: string;
  rpcUrl: string;
  chainId: number;
  signer: string;
  dryRun: boolean;
  limit: number;
} & ({
  command: "set_shard_id";
  guardianMessage: string;
} | {
  command: "append_schnorr";
  vaaFile: string;
} | {
  command: "pull_multisigs";
})

// TODO: Use binary-layout for these
function encodeSetShardId(guardianMessage: Buffer): Hex {
  // set_shard_id: opcode (1 byte) + guardian message data
  return encodePacked(
    ['uint8', 'bytes'],
    [UPDATE_SET_SHARD_ID, `0x${guardianMessage.toString('hex')}`]
  );
}

function encodeAppendSchnorrKey(vaa: Buffer): Hex {
  // append_schnorr_KEY: opcode (1 byte) + vaa length (2 bytes) + vaa data
  return encodePacked(
    ['uint8', 'uint16', 'bytes'],
    [UPDATE_APPEND_SCHNORR_KEY, vaa.length, `0x${vaa.toString('hex')}`]
  );
}

function encodePullMultisigKeyData(limit: number): `0x${string}` {
  // PULL_MULTISIG_KEY_DATA: opcode (1 byte) + limit (4 bytes)
  return encodePacked(
    ['uint8', 'uint32'],
    [UPDATE_PULL_MULTISIG_KEY_DATA, limit]
  );
}

function encodeUpdate(args: Args, dataBytes: Buffer): `0x${string}` {
  if (args.command === "append_schnorr") {
    const pullData = encodePullMultisigKeyData(args.limit);
    const appendData = encodeAppendSchnorrKey(dataBytes);
    console.log(`Prepared pull_multisigs with limit ${args.limit}`);
    console.log(`Prepared append_schnorr with ${dataBytes.length} bytes of data`);
    return encodePacked(['bytes', 'bytes'], [pullData, appendData]);
  } else if (args.command === "set_shard_id") {
    console.log(`Prepared set_shard_id with ${dataBytes.length} bytes of data`);
    return encodeSetShardId(dataBytes);
  } else {
    console.log(`Prepared pull_multisigs with limit ${args.limit}`);
    return encodePullMultisigKeyData(args.limit);
  }
}

async function main() {
  const parser = yargs(hideBin(process.argv))
    .command('append_schnorr <vaa-file>', 'Append a Schnorr key to the guardian',
      (yargs) => yargs.positional('vaa-file', {
        description: 'Path to file containing base64-encoded governance VAA',
        type: 'string',
      }
    ))
    .command('set_shard_id <guardian-message>', 'Set the shard ID of the guardian',
      (yargs) => yargs.positional('guardian-message', {
        description: 'Path to file containing base64-encoded signed guardian message',
        type: 'string',
      }
    ))
    .command('pull_multisigs', 'Pull multisig sets from the core contract')
    .demandCommand(1, 'A command is required')
    .strictCommands()
    //TODO: add support for ledger signer
    .option('signer', {
      description: 'Path to gpg armor guardian private key file',
      demandOption: true,
      type: 'string',
      alias: 's',
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
    .option('dry-run', {
      description: 'Simulate the transaction without actually sending it',
      type: 'boolean',
      default: false,
      alias: 'd',
    })
    .option('limit', {
      description: 'Maximum number of multisig sets to pull.',
      defaultDescription: '0 (Pull all necessary multisig sets)',
      type: 'number',
      alias: 'l',
      default: 0,
    })
    .help()
    .alias('help', 'h');

  const parsedArgs = await parser.parse();
  const args = { ...parsedArgs, command: parsedArgs._[0] } as Args;

  let dataBytes = Buffer.alloc(0);
  
  if (args.command === "set_shard_id" || args.command === "append_schnorr") {
    const path = args.command === "set_shard_id" ? args.guardianMessage : args.vaaFile;
    // Load guardian message or VAA from base64 encoded file
    try {
      const messageBase64 = fs.readFileSync(path, 'utf-8').trim();
      dataBytes = Buffer.from(messageBase64, 'base64');
    } catch (error) {
      console.error(`Failed to load data from ${path}: ${errorMsg(error)}`);
      process.exit(1);
    }
    console.log(`Loaded ${dataBytes.length} bytes of data from ${path}`);
  }

  let signerKey: Hex;
  try {
    const signerFile = fs.readFileSync(args.signer, 'utf-8');
    const keyBytes = parseGuardianKey(signerFile);
    signerKey = `0x${Buffer.from(keyBytes).toString('hex')}`;
  } catch (error) {
    console.error(`Failed to parse signer file: ${errorMsg(error)}`);
    process.exit(1);
  }

  const account = privateKeyToAccount(signerKey);
  console.log(`Using signer address: ${account.address}`);

  // Validate contract address
  if (!isHex(args.contractAddress)) {
    console.error("Contract address must be a valid hex address");
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

  const updateData = encodeUpdate(args, dataBytes);

  console.log(`Target contract: ${args.contractAddress}`);
  console.log(`Chain ID: ${args.chainId}`);
  console.log(`RPC URL: ${args.rpcUrl}`);

  if (args.dryRun) {
    console.log('\nDRY RUN MODE - Transaction will not be sent');
    console.log('Update data (hex):', updateData);
    return;
  }

  try {
    console.log('\nSending transaction...');
    const txHash = await walletClient.writeContract({
      address: args.contractAddress,
      abi: UPDATE_ABI,
      functionName: 'update',
      args: [updateData],
    });

    console.log(`Transaction sent: ${txHash}`);
    console.log('Waiting for confirmation...');

    const receipt = await waitForTransactionReceipt(walletClient, {
      hash: txHash,
    });

    if (receipt.status === 'success') {
      console.log(`Transaction confirmed in block ${receipt.blockNumber}`);
      console.log(`Gas used: ${receipt.gasUsed}`);
    } else {
      console.error('Transaction failed');
      process.exit(1);
    }
  } catch (error) {
    console.error(`Transaction failed: ${errorStack(error)}`);
    process.exit(1);
  }
}

main().catch((error: unknown) => {
  console.error(`[ERROR] Unhandled error: ${errorStack(error)}`);
  process.exit(1);
});
