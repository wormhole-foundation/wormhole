import express from 'express';
import cors from 'cors';
import { Display } from './display.js';
import {
  BaseServerConfig,
  Peer,
  PeerRegistrationSchema,
  validate,
  validateGuardianSignature,
  WormholeGuardianData,
  validatePeers
} from '@xlabs-xyz/peer-lib';
import { saveGuardianPeers } from './peers.js';

export class PeerServer {
  private app: express.Application;
  private sparseGuardianPeers: (Peer | undefined)[];
  private guardianSetLength: number;
  private wormholeData: WormholeGuardianData;
  private config: BaseServerConfig;
  private server?: any;
  private display: Display;

  constructor(
    config: BaseServerConfig,
    wormholeData: WormholeGuardianData,
    display: Display,
    initialPeers: Peer[] = []
  ) {
    this.config = config;
    this.wormholeData = wormholeData;
    this.display = display;
    this.sparseGuardianPeers = validatePeers(initialPeers, wormholeData);
    this.guardianSetLength = wormholeData.guardians.length;
    this.app = express();
    this.setupMiddleware();
    this.setupRoutes();
    // Show initial progress
    this.display.setProgress(this.sparseGuardianPeers.length, this.guardianSetLength, 'Guardian Collection Progress');
  }

  private partialGuardianPeers(): Peer[] {
    return this.sparseGuardianPeers.filter(peer => peer !== undefined);
  }

  private setupMiddleware(): void {
    this.app.use(cors());
    this.app.use(express.json());
  }

  private setupRoutes(): void {
    // Get all peers (returns array of peer data)
    this.app.get('/peers', async (req, res) => {
      try {
        res.json({
          peers: this.partialGuardianPeers(),
          threshold: this.config.threshold,
          totalExpectedGuardians: this.guardianSetLength
        });
      } catch (error) {
        this.display.error('Error fetching peers:', error);
        res.status(500).json({ error: 'Failed to fetch peers' });
      }
    });

    // Add a new peer with signature validation
    this.app.post('/peers', async (req, res) => {
      try {
        const validationResult = validate(PeerRegistrationSchema, req.body, "Invalid peer registration");
        if (!validationResult.success) {
          return res.status(400).json({ error: validationResult.error });
        }
        const peerRegistration = validationResult.value;

        // Validate guardian signature and get guardian address
        const guardian = validateGuardianSignature(peerRegistration, this.wormholeData);
        if (!guardian.success) {
          this.display.error('Error validating guardian signature:', guardian.error);
          return res.status(401).json({ error: 'Invalid guardian signature' });
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
          this.display.log(`WARNING: Guardian ${guardianIndex} resubmitted peer data`);
          this.display.log(`   Old peer: ${this.sparseGuardianPeers[guardianIndex]}`);
          this.display.log(`   New peer: ${peer}`);
        }
        this.sparseGuardianPeers[guardianIndex] = peer;
        // Save the updated guardian peers
        saveGuardianPeers(this.partialGuardianPeers(), this.display);
        // Update progress display (will automatically show peers when complete)
        this.display.setProgress(
          this.sparseGuardianPeers.length, 
          this.wormholeData.guardians.length, 
          'Guardian Collection Progress',
          this.partialGuardianPeers()
        );
        res.status(201).json({
          peer: { guardianAddress, guardianIndex, signature, hostname, port, tlsX509 },
          threshold: this.config.threshold
        });
      } catch (error) {
        this.display.error('Error adding peer:', error);
        res.status(500).json({ error: 'Failed to add peer' });
      }
    });
  }

  start(): void {
    this.server = this.app.listen(this.config.port, () => {
      this.display.log(`Peer server running on port ${this.config.port}`);
      this.display.log('\nWaiting for guardians to submit their peer data...');
    });
  }

  getApp(): express.Application {
    return this.app;
  }

  close(): void {
    if (this.server) {
      this.server.close();
    }
  }
}
