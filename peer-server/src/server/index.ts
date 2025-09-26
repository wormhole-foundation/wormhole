import { PeerServer } from './server.js';
import { getWormholeGuardianData } from './wormhole.js';
import { Display } from './display.js';
import { ServerConfig, ServerConfigSchema, validateOrFail } from '../shared/types.js';
import fs from 'fs';
import path from 'path';

// Load configuration from file
export function loadConfig(configPath?: string): ServerConfig {
  const configFile = configPath || path.join(process.cwd(), 'config.json');

  if (!fs.existsSync(configFile)) {
    throw new Error(`Config file not found: ${configFile}`);
  }

  try {
    const configData = fs.readFileSync(configFile, 'utf8');
    const config = JSON.parse(configData);

    // Validate configuration using Zod schema
    return validateOrFail(ServerConfigSchema, config, `Invalid configuration in ${configFile}`);
  } catch (error: any) {
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

Usage: npm start [options]

Options:
  --config <path>        Config file path (default: ./config.json)
  --help                 Show this help message

Examples:
  npm start
  npm start -- --config ./my-config.json
        `);
        process.exit(0);
        break;
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

  const server = new PeerServer(config, wormholeData, display);
  server.start();

  // Handle graceful shutdown
  process.on('SIGINT', () => {
    display.log('\n👋 Shutting down server...');
    server.close();
    process.exit(0);
  });

  process.on('SIGTERM', () => {
    display.log('\nShutting down server...');
    server.close();
    process.exit(0);
  });
}

main().catch((error) => {
  const display = new Display();
  display.error('Failed to start server:', error);
  process.exit(1);
});
