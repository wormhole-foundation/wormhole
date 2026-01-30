import express from 'express';
import cors from 'cors';
import {
  BaseServerConfig,
  Peer,
  PeerRegistrationSchema,
  validate,
  validateGuardianSignature,
  WormholeGuardianData,
  validatePeers,
  errorStack
} from '@xlabs-xyz/peer-lib';
import { createServer, Server } from 'node:http';

import { Display } from './display.js';
import { saveGuardianPeers } from './peers.js';
import { inspect } from 'node:util';

export class PeerServer {
  private sparseGuardianPeers: (Peer | undefined)[];
  private port?: number;

  protected constructor(
    private app: express.Application,
    private server: Server,
    private config: BaseServerConfig,
    private wormholeData: WormholeGuardianData,
    private display: Display,
    initialPeers: Peer[] = []
  ) {
    this.sparseGuardianPeers = validatePeers(initialPeers, wormholeData);

    this.setupRoutes();
    // Show initial progress
    this.display.setProgress(initialPeers, this.wormholeData.guardians.length);
  }

  private partialGuardianPeers(): Peer[] {
    return this.sparseGuardianPeers.filter(peer => peer !== undefined);
  }

  private setupRoutes(): void {
    // Get all peers (returns array of peer data)
    this.app.get('/peers', (req, res) => {
      try {
        res.json({
          peers: this.partialGuardianPeers(),
          threshold: this.config.threshold,
          totalExpectedGuardians: this.wormholeData.guardians.length
        });
      } catch (error) {
        this.display.error(`Error fetching peers: ${errorStack(error)}`);
        res.status(500).json({ error: 'Failed to fetch peers' });
      }
    });

    // Add a new peer with signature validation
    this.app.post('/peers', (req, res) => {
      try {
        const validationResult = validate(
          PeerRegistrationSchema, req.body, "Invalid peer registration"
        );
        if (!validationResult.success) {
          res.status(400).json({ error: validationResult.error });
          return;
        }
        const peerRegistration = validationResult.value;

        // Validate guardian signature and get guardian address
        const guardian = validateGuardianSignature(peerRegistration, this.wormholeData);
        if (!guardian.success) {
          this.display.error(`Error validating guardian signature: ${guardian.error}`);
          res.status(401).json({ error: 'Invalid guardian signature' });
          return;
        }

        const { guardianAddress, guardianIndex } = guardian.value;
        const { hostname, port, tlsX509 } = peerRegistration.peer;
        const signature = peerRegistration.signature;
        this.display.log(`Adding peer ${hostname} from guardian ${guardianAddress}`);

        // Store peer data for this guardian
        const peer: Peer = { 
          guardianAddress,
          guardianIndex,
          signature,
          hostname, 
          port,
          tlsX509,
        };
        // We allow re-submission of peer data for the same guardian
        if (this.sparseGuardianPeers[guardianIndex] !== undefined) {
          const oldPeer = this.sparseGuardianPeers[guardianIndex];
          this.display.log('-------------------------------------');
          this.display.log(`WARNING: Guardian ${guardianAddress} resubmitted peer data`);
          this.display.log(`Old peer: ${JSON.stringify(oldPeer, null, 2)}`);
          this.display.log(`New peer: ${JSON.stringify(peer, null, 2)}`);
          this.display.log('-------------------------------------');
        }
        this.sparseGuardianPeers[guardianIndex] = peer;
        // Save the updated guardian peers
        const currentGuardianPeers = this.partialGuardianPeers();
        saveGuardianPeers(currentGuardianPeers, this.display, this.config.peerListStore);
        // Update progress display (will automatically show peers when complete)
        this.display.setProgress(currentGuardianPeers, this.wormholeData.guardians.length);
        res.status(201).json({
          peer: { guardianAddress, guardianIndex, signature, hostname, port, tlsX509 },
          threshold: this.config.threshold
        });
      } catch (error) {
        this.display.error(`Error adding peer: ${errorStack(error)}`);
        res.status(500).json({ error: 'Failed to add peer' });
      }
    });
  }

  protected static createApp()  {
    const app = express();
    app.use(cors());
    app.use(express.json());
    return app;
  }

  static start(
    config: BaseServerConfig,
    wormholeData: WormholeGuardianData,
    display: Display,
    initialPeers: Peer[] = [],
  ): Promise<PeerServer> {
    const app = this.createApp();
    const server = createServer(app);

    const peerServer = new PeerServer(app, server, config, wormholeData, display, initialPeers);
    return new Promise((resolve) => {
      server.listen(config.port, () => {
        const address = server.address();
        peerServer.port = typeof address === 'object' ? address?.port : undefined;
        display.log(`Peer server running on port ${inspect(peerServer.port ?? address)}`);
        display.log('\nWaiting for guardians to submit their peer data...');
        resolve(peerServer);
      });
    });
  }

  getPort(): number | undefined {
    return this.port;
  }

  getApp(): express.Application {
    return this.app;
  }

  close(): void {
    this.server.close();
  }
}
