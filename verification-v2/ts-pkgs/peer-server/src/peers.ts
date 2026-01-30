
import path from 'path';
import fs from 'fs';
import { Peer, PeerArraySchema, validateOrFail } from '@xlabs-xyz/peer-lib';
import { Display } from './display.js';

export function saveGuardianPeers(peers: Peer[], display: Display, filePath: string): void {
  const resolvedPath = path.resolve(filePath);
  display.log(`Saving guardian peers to ${resolvedPath}`);
  fs.writeFileSync(resolvedPath, JSON.stringify(peers, null, 2));
}

export function loadGuardianPeers(display: Display, filePath: string): Peer[] {
  const resolvedPath = path.resolve(filePath);
  if (!fs.existsSync(resolvedPath)) {
    display.log(`WARNING: No guardian peers file found at ${resolvedPath}`);
    return [];
  }
  const content = fs.readFileSync(resolvedPath, 'utf-8');
  return validateOrFail(PeerArraySchema, JSON.parse(content), 'Invalid guardian peers');
}