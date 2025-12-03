
import path from 'path';
import fs from 'fs';
import { Peer, PeerArraySchema, validateOrFail } from '../shared/types.js';
import { Display } from './display.js';

const GUARDIAN_PEERS_FILE = 'guardian_peers.json';

export function saveGuardianPeers(peers: Peer[], display: Display): void {
  display.log(`Saving guardian peers to ${path.resolve(GUARDIAN_PEERS_FILE)}`);
  fs.writeFileSync(path.resolve(GUARDIAN_PEERS_FILE), JSON.stringify(peers, null, 2));
}

export function loadGuardianPeers(display: Display): Peer[] {
  if (!fs.existsSync(path.resolve(GUARDIAN_PEERS_FILE))) {
    display.log(`WARNING: No guardian peers file found at ${path.resolve(GUARDIAN_PEERS_FILE)}`);
    return [];
  }
  const content = fs.readFileSync(path.resolve(GUARDIAN_PEERS_FILE), 'utf-8');
  return validateOrFail(PeerArraySchema, JSON.parse(content), 'Invalid guardian peers');
}