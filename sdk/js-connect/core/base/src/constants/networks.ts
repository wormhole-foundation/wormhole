//Mainnet: speaks for itself
//Testnet: blockchain of a given environment intended for developer testing
//  Warning: Therefore, what we call Testnet, e.g. Solana calls its devnet cluster!
//Devnet: local network/blockchain (tilt environment)
//TODO: not convinced that Devnet should really be a thing
export const networks = ["Mainnet", "Testnet", "Devnet"] as const;
export type Network = (typeof networks)[number];
export const isNetwork = (network: string): network is Network =>
  networks.includes(network as Network);
