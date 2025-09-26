import express from 'express';
import cors from 'cors';
import { ethers } from 'ethers';
import { Display } from './display.js';
import { Peer, PeerRegistration, ServerConfig, WormholeGuardianData } from '../shared/types.js';

export class PeerServer {
  private app: express.Application;
  private guardianPeers: Peer[] = []; // array of peer data with guardian keys
  private wormholeData: WormholeGuardianData;
  private config: ServerConfig;
  private server?: any;
  private display: Display;

  constructor(config: ServerConfig, wormholeData: WormholeGuardianData, display: Display) {
    this.config = config;
    this.wormholeData = wormholeData;
    this.display = display;
    this.app = express();
    this.setupMiddleware();
    this.setupRoutes();
    
    // Show initial progress
    this.display.setProgress(this.submittedCount, this.guardianSetLength, 'Guardian Collection Progress');
  }

  private setupMiddleware(): void {
    this.app.use(cors());
    this.app.use(express.json());
  }

  private setupRoutes(): void {
    // Get all peers (returns array of peer data)
    this.app.get('/peers', async (req, res) => {
      try {
        // Sort peers by guardian index
        const peers = this.allPeers.sort((a, b) => {
          const aIndex = this.wormholeData.guardians.indexOf(a.guardianAddress);
          const bIndex = this.wormholeData.guardians.indexOf(b.guardianAddress);
          return aIndex - bIndex;
        })

        res.json({
          peers,
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
        const peerRegistration: PeerRegistration = req.body;

        // Basic validation
        if (!peerRegistration.peer || !peerRegistration.signature) {
          return res.status(400).json({
            error: 'Missing required fields: peer, signature'
          });
        }

        const { hostname, tlsX509 } = peerRegistration.peer;
        if (!hostname || !tlsX509 || hostname.trim() === '' || tlsX509.trim() === '') {
          return res.status(400).json({
            error: 'Missing required peer fields: Hostname, TlsX509, Port'
          });
        }

        // Validate guardian signature and get guardian address
        const guardianAddress = this.validateGuardianSignature(peerRegistration);
        if (!guardianAddress) {
          return res.status(401).json({ error: 'Invalid guardian signature' });
        }

        // Check if this guardian has already submitted
        if (this.guardianPeers.find(peer => peer.guardianAddress === guardianAddress)) {
          this.display.log(`Guardian ${guardianAddress} attempted resubmission - ignoring`);
          return res.status(409).json({ 
            error: 'Guardian has already submitted peer data',
            guardianAddress
          });
        }

        this.display.log(`Adding peer ${hostname} from guardian ${guardianAddress}`);

        // Store peer data for this guardian
        const peer: Peer = { 
          guardianAddress: guardianAddress, 
          hostname, 
          tlsX509,
        };
        this.guardianPeers.push(peer);

        // Update progress display (will automatically show peers when complete)
        this.display.setProgress(
          this.guardianPeers.length, 
          this.wormholeData.guardians.length, 
          'Guardian Collection Progress',
          this.guardianPeers
        );

        res.status(201).json({
          peer: { guardianAddress, hostname, tlsX509 },
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

  get guardianSetLength(): number {
    return this.wormholeData?.guardians.length || 0;
  }

  get submittedCount(): number {
    return this.guardianPeers.length;
  }

  get allPeers(): Peer[] {
    return this.guardianPeers;
  }

  getApp(): express.Application {
    return this.app;
  }

  private validateGuardianSignature(peerRegistration: PeerRegistration): string | null {
    // Validate guardian index is within bounds
    const guardianIndex = peerRegistration.signature.guardianIndex;
    if (guardianIndex < 0 || guardianIndex >= this.wormholeData.guardians.length) {
      this.display.log(`Invalid guardian index: ${guardianIndex}. Must be between 0 and ${this.wormholeData.guardians.length - 1}`);
      return null;
    }

    // Get the expected guardian address at this index
    const expectedGuardianAddress = this.wormholeData.guardians[guardianIndex];

    // Create the message hash that should have been signed
    // Message format: keccak256(abi.encodePacked(hostname, tlsX509))
    const messageHash = ethers.keccak256(
      ethers.solidityPacked(
        ['string', 'string'],
        [peerRegistration.peer.hostname, peerRegistration.peer.tlsX509]
      )
    );

    try {
      // Recover the address that signed the message
      const recoveredAddress = ethers.verifyMessage(
        ethers.getBytes(messageHash),
        peerRegistration.signature.signature
      );

      // Check if the recovered address matches the expected guardian at the given index
      if (recoveredAddress.toLowerCase() === expectedGuardianAddress.toLowerCase()) {
        this.display.log(`Valid signature from guardian ${guardianIndex}: ${recoveredAddress}`);
        return recoveredAddress;
      } else {
        this.display.log(`Invalid signature: expected ${expectedGuardianAddress}, got ${recoveredAddress} for guardian index ${guardianIndex}`);
        return null;
      }
    } catch (error) {
      this.display.log('Failed to verify signature:' + (error instanceof Error ? error.message : String(error)));
      return null;
    }
  }

  close(): void {
    if (this.server) {
      this.server.close();
    }
  }
}
