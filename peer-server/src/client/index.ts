import fs from "fs";
import path from "path";
import process from "process";
import {
  SelfConfig,
  validateOrFail,
  SelfConfigSchema,
  Peer,
} from "../shared/types.js";
import { PeerClient } from "./client.js";

type ClientAction = "upload" | "poll";

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
      return validateOrFail(SelfConfigSchema, parsedConfig, "Invalid self_config.json");
    } catch (error: any) {
      console.error(`[ERROR] Invalid JSON in self_config.json: ${error?.message || error}`);
      process.exit(1);
    }
  }

  private savePeerConfig(peers: Peer[], threshold: number): void {
    const outputPath = path.resolve("peer_config.json");

    try {
      // Convert to array and sort by guardian address for consistency
      const outputData = {
        Peers: peers
          .map((peer) => ({
            Hostname: peer.hostname,
            TlsX509: peer.tlsX509,
            Port: peer.port,
          })),
        Self: {
          Hostname: this.config.peer.hostname,
          TlsX509: this.config.peer.tlsX509,
          Port: this.config.peer.port,
        },
        NumParticipants: peers.length,
        WantedThreshold: threshold,
      };

      fs.writeFileSync(outputPath, JSON.stringify(outputData, null, 2), 'utf-8');
      console.log(`[SAVED] Peer configuration saved to: ${outputPath}`);
      console.log(`[INFO] Threshold: ${threshold}`);
    } catch (error: any) {
      console.error(`[ERROR] Error saving peer config: ${error?.message || error}`);
      process.exit(1);
    }
  }

  public async run(action: ClientAction): Promise<void> {
    if (action === "upload") {
      await this.client.submitPeerData();
    } else {
      const response = await this.client.waitForAllPeers();
      // Save the final configuration
      this.savePeerConfig(response.peers, response.threshold);
    }
  }
}

// Main execution
async function main() {
  const action = process.argv[2];
  if (action !== "upload" && action !== "poll") {
    console.log("Usage: npm run start:client [upload | poll]");
    console.log("    upload: Uploads the peer data to the server");
    console.log("    poll: Polls until the server has all the peer data");
    process.exit(1);
  }
  const client = new ConfigClient();
  await client.run(action as ClientAction);
}

main().catch((error) => {
  console.error(`[ERROR] Unhandled error: ${error}`);
  process.exit(1);
});

export { ConfigClient };
