import fs from "fs";
import path from "path";
import {
  SelfConfig,
  validateOrFail,
  SelfConfigSchema,
} from "../shared/types.js";
import { PeerClient } from "./client.js";

/**
 * CLI wrapper for PeerClient that handles configuration loading and file I/O
 */
class ConfigClient {
  private config: SelfConfig;
  private client: PeerClient;

  constructor() {
    this.config = this.loadConfig();
    this.client = new PeerClient(this.config);
  }

  private loadConfig(): SelfConfig {
    const configPath = path.resolve("self_config.json");

    if (!fs.existsSync(configPath)) {
      console.error("[ERROR] self_config.json not found. Please create the file with your peer configuration.");
      process.exit(1);
    }

    try {
      const configData = fs.readFileSync(configPath, 'utf-8');
      const parsedConfig = JSON.parse(configData);

      // Handle guardian key file loading
      if (parsedConfig.guardianKey && !parsedConfig.guardianKey.startsWith('0x')) {
        // Treat as file path
        const keyPath = path.resolve(parsedConfig.guardianKey);
        if (fs.existsSync(keyPath)) {
          const privateKeyHex = fs.readFileSync(keyPath, 'utf-8').trim();
          // Ensure 0x prefix
          parsedConfig.guardianKey = privateKeyHex.startsWith('0x') ? privateKeyHex : '0x' + privateKeyHex;
        } else {
          console.error(`[ERROR] Guardian key file not found: ${keyPath}`);
          process.exit(1);
        }
      }

      return validateOrFail(SelfConfigSchema, parsedConfig, "Invalid self_config.json");
    } catch (error: any) {
      console.error(`[ERROR] Invalid JSON in self_config.json: ${error?.message || error}`);
      process.exit(1);
    }
  }

  private savePeerConfig(peers: Record<string, any>, threshold: number): void {
    const outputPath = path.resolve("peer_config.json");

    try {
      // Convert to array and sort by guardian address for consistency
      const peerArray = Object.entries(peers)
        .map(([guardianAddress, peer]) => ({
          guardianAddress,
          ...peer
        }))
        .sort((a, b) => a.guardianAddress.localeCompare(b.guardianAddress));

      const outputData = {
        peers: peerArray,
        totalCount: peerArray.length,
        threshold: threshold,
        generatedAt: new Date().toISOString()
      };

      fs.writeFileSync(outputPath, JSON.stringify(outputData, null, 2), 'utf-8');
      console.log(`[SAVED] Peer configuration saved to: ${outputPath}`);
      console.log(`[INFO] Threshold: ${threshold}`);
    } catch (error: any) {
      console.error(`[ERROR] Error saving peer config: ${error?.message || error}`);
      process.exit(1);
    }
  }

  public async run(): Promise<void> {
    try {
      console.log(`[STARTING] Peer Client starting...`);
      console.log(`   Server: ${this.config.serverUrl}`);
      console.log(`   Guardian Index: ${this.config.guardianIndex}`);
      console.log(`   Peer: ${this.config.peer.hostname}`);

      // Run the client and get results
      const response = await this.client.run();

      // Save the final configuration
      this.savePeerConfig(response.peers, response.threshold);

      console.log(`[COMPLETED] Client completed successfully!`);
    } catch (error: any) {
      console.error(`[ERROR] Client failed: ${error?.message || error}`);
      process.exit(1);
    }
  }
}

// Main execution
async function main() {
  const client = new ConfigClient();
  await client.run();
}

main().catch((error) => {
  console.error(`[ERROR] Unhandled error: ${error}`);
  process.exit(1);
});

export { ConfigClient };
