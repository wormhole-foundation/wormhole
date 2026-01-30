import {
  errorStack,
  getWormholeGuardianData,
  ServerConfig,
  ServerConfigSchema,
  validateOrFail,
} from '@xlabs-xyz/peer-lib';
import fs from 'fs';
import path from 'path';

import { Display } from './display.js';
import { PeerServer } from './server.js';
import { loadGuardianPeers } from './peers.js';

// Load configuration from file
export function loadConfig(configPath?: string): ServerConfig {
  const configFile = configPath ?? path.join(process.cwd(), 'config.json');
  try {
    const configData = fs.readFileSync(configFile, 'utf8');
    const config = JSON.parse(configData) as unknown;
    return validateOrFail(ServerConfigSchema, config, `Invalid configuration in ${configFile}`);
  } catch (error) {
    if (error instanceof SyntaxError) {
      throw new Error(`Invalid JSON in config file: ${configFile}`);
    }
    throw error;
  }
}

// Parse command line arguments for configuration
function parseConfig(): ServerConfig {
  const args = process.argv.slice(2);
  let configPath: string | undefined;

  for (let i = 0; i < args.length; i++) {
    switch (args[i]) {
      case '--config':
        configPath = args[++i];
        break;
      case '--help':
        console.log(`
Peer Server

Usage: tsx cli.ts [options]

Options:
  --config <path>        Config file path (default: ./config.json)
  --help                 Show this help message

Examples:
  tsx cli.ts
  tsx cli.ts --config ./my-config.json
        `);
        process.exit(0);
    }
  }

  // Load config from file or use defaults
  return loadConfig(configPath);
}

async function main() {
  const config = parseConfig();

  const display = new Display();
  display.log('Starting Peer Server...');
  display.log(`Port: ${config.port}`);
  display.log(`Ethereum RPC: ${config.ethereum.rpcUrl}`);
  display.log(`Wormhole contract: ${config.wormholeContractAddress}`);

  // Initialize Wormhole guardian data
  const wormholeData = await getWormholeGuardianData(config);
  display.log('Server initialized with Wormhole guardian data');

  const guardianPeers = loadGuardianPeers(display, config.peerListStore);
  display.log(`Loaded ${guardianPeers.length} guardian peers`);

  let server: PeerServer | undefined;

  const shutdown = () => {
    display.log('\nShutting down server...');
    if (server !== undefined)
      server.close();
    process.exit(0);
  };
  // Handle graceful shutdown
  process.on('SIGINT', shutdown);
  process.on('SIGTERM', shutdown);

  server = await PeerServer.start(config, wormholeData, display, guardianPeers);
}

main().catch((error: unknown) => {
  console.error(`Failed to start server: ${errorStack(error)}`);
  process.exit(1);
});