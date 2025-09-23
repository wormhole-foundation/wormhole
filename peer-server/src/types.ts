export interface Peer {
  Hostname: string;
  TlsX509: string;
  Port: number;
}

export interface PeerSignature {
  signature: string;  // Ethereum signature (r,s,v format) from a guardian
  guardianIndex: number;
}

export interface PeerRegistration {
  peer: Peer;
  signature: PeerSignature;
}

export interface ServerConfig {
  port: number;
  ethereum: {
    rpcUrl: string;
    chainId?: number;
  };
  wormholeContractAddress: string;
}

export interface WormholeGuardianData {
  keys: string[];
}
