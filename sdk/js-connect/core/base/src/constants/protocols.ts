export const protocols = [
  "WormholeCore",
  "TokenBridge",
  "AutomaticTokenBridge",
  "CircleBridge",
  "AutomaticCircleBridge",
  "Relayer",
  "IbcBridge",
  // not implemented
  "NftBridge",
] as const;

export type ProtocolName = (typeof protocols)[number];
export const isProtocolName = (protocol: string): protocol is ProtocolName =>
  protocols.includes(protocol as ProtocolName);
