// NOTE: modify these to reflect current versions of packages
export const VERSION_WORMHOLE = 1;
export const VERSION_TOKEN_BRIDGE = 1;

// keystore
export const KEYSTORE = [
  "AB522qKKEsXMTFRD2SG3Het/02S/ZBOugmcH3R1CDG6l",
  "AOmPq9B16F3W3ijO/4s9hI6v8LdiYCawKAW31PKpg4Qp",
  "AGA20wtGcwbcNAG4nwapbQ5wIuXwkYQEWFUoSVAxctHb",
];

// wallets
export const WALLET_PRIVATE_KEY = Buffer.from(KEYSTORE[0], "base64").subarray(
  1
);
export const RELAYER_PRIVATE_KEY = Buffer.from(KEYSTORE[1], "base64").subarray(
  1
);
export const CREATOR_PRIVATE_KEY = Buffer.from(KEYSTORE[2], "base64").subarray(
  1
);

// guardian signer
export const GUARDIAN_PRIVATE_KEY =
  "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0";

// wormhole
export const WORMHOLE_STATE_ID =
  "0xc561a02a143575e53b87ba6c1476f053a307eac5179cb1c8121a3d3b220b81c1";

// token bridge
export const TOKEN_BRIDGE_STATE_ID =
  "0x1c8de839f6331f2d745eb53b1b595bc466b4001c11617b0b66214b2e25ee72fc";

// governance
export const GOVERNANCE_EMITTER =
  "0000000000000000000000000000000000000000000000000000000000000004";

// file encoding
export const UTF8: BufferEncoding = "utf-8";
