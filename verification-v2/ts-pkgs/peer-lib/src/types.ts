import { z } from 'zod';
import { readFileSync } from 'fs';
import { checkTlsCertificate, parseGuardianKey } from './parseCrypto.js';
import { errorMsg } from './error.js';

const portSchema = z.int().min(1, "Port must be between 1 and 65535").max(65535, "Port must be between 1 and 65535");

/// Encodes the information necessary to construct the peer description message
export const BasePeerSchema = z.object({
  hostname: z.string().min(1, "Hostname cannot be empty"),
  port: portSchema,
  tlsX509: z.string().min(1, "TlsX509 certificate cannot be empty"),
});

const guardianAddressSchema = z.string().startsWith("0x", "Guardian address must be an EVM address hex encoded with 0x prefix").length(42, "Guardian address must be an EVM address hex encoded with 0x prefix");

export const GuardianSchema = z.object({
  guardianAddress: guardianAddressSchema,
  guardianIndex: z.int().min(0, "Guardian index must be non-negative"),
});

/// Contains the signature of the peer description message
export const PeerSignatureSchema = z.object({
  signature: z.string().min(1, "Signature cannot be empty"),
});

export const PeerSchema = z.intersection(z.intersection(BasePeerSchema, GuardianSchema), PeerSignatureSchema);

export const PeerRegistrationSchema = z.intersection(
  z.object({ peer: BasePeerSchema }),
  PeerSignatureSchema,
);

export const WormholeConfigSchema = z.object({
  ethereum: z.object({
    rpcUrl: z.url("Ethereum RPC URL must be a valid URL")
  }),
  wormholeContractAddress: z.string().min(1, "Wormhole contract address cannot be empty"),
});

// Config schema that reads from file paths and transforms to runtime values
export const PeerClientConfigSchema = z.object({
  // TODO: move this to specific CLI option/command type
  guardianPrivateKeyPath: z.string().optional().transform((value) => value ?? undefined),
  guardianPrivateKeyArn: z.string().optional().transform((value) => value ?? undefined),
  serverUrl: z.url("Server URL must be a valid HTTP(S) URL"),
  peer: BasePeerSchema,
  wormhole: WormholeConfigSchema.optional(),
}).transform((data) => {
  // Validate that at most one of guardianPrivateKeyPath or guardianPrivateKeyArn is set
  if (data.guardianPrivateKeyPath !== undefined && data.guardianPrivateKeyArn !== undefined) {
    throw new Error("Only one of guardianPrivateKeyPath or guardianPrivateKeyArn must be set, not both");
  }

  // Load and validate guardian private key from file if path is provided
  let guardianPrivateKey: string | undefined = undefined;
  if (data.guardianPrivateKeyPath !== undefined) {
    try {
      const keyContents = readFileSync(data.guardianPrivateKeyPath, 'utf-8');
      const keyBytes = parseGuardianKey(keyContents); // Extract and validate the key
      // Convert to hex format for ethers.Wallet
      guardianPrivateKey = '0x' + Buffer.from(keyBytes).toString('hex');
    } catch (error) {
      throw new Error(`Failed to read or validate guardian private key from ${data.guardianPrivateKeyPath}: ${errorMsg(error)}`);
    }
  }

  // Load and validate TLS certificate
  let tlsX509: string;
  try {
    const certContents = readFileSync(data.peer.tlsX509, 'utf-8');
    if (!checkTlsCertificate(certContents)) {
      throw new Error("Invalid TLS X509 certificate format");
    }
    tlsX509 = certContents;
  } catch (error) {
    throw new Error(`Failed to read or validate TLS X509 certificate from ${data.peer.tlsX509}: ${errorMsg(error)}`);
  }

  return {
    // guardianPrivateKeyOrArn can be undefined if neither is provided (for commands that don't need it)
    guardianPrivateKeyOrArn: guardianPrivateKey ?? data.guardianPrivateKeyArn,
    serverUrl: data.serverUrl,
    peer: {
      hostname: data.peer.hostname,
      port: data.peer.port,
      tlsX509,
    },
    wormhole: data.wormhole,
  };
});

const thresholdSchema = z.int().min(1, "Threshold must be a positive integer");

export const BaseServerConfigSchema = z.object({
  port: portSchema,
  threshold: thresholdSchema,
  peerListStore: z.string().min(1, "Peer list store path cannot be empty"),
});

export const ServerConfigSchema = z.intersection(BaseServerConfigSchema, WormholeConfigSchema);

export const WormholeGuardianDataSchema = z.object({
  guardians: z.array(guardianAddressSchema),
});

export const UploadResponseSchema = z.object({
  peer: PeerSchema,
  threshold: thresholdSchema,
});

export const PeerArraySchema = z.array(PeerSchema);

export const PeersResponseSchema = z.object({
  peers: PeerArraySchema,
  threshold: thresholdSchema,
  totalExpectedGuardians: z.int().min(1, "Total expected guardians must be a positive integer")
});

// Type definitions inferred from Zod schemas
export type Peer = z.infer<typeof PeerSchema>;
export type BasePeer = z.infer<typeof BasePeerSchema>;
export type Guardian = z.infer<typeof GuardianSchema>;
export type PeerSignature = z.infer<typeof PeerSignatureSchema>;
export type PeerRegistration = z.infer<typeof PeerRegistrationSchema>;
export type PeerClientConfig = z.infer<typeof PeerClientConfigSchema>;
export type BaseServerConfig = z.infer<typeof BaseServerConfigSchema>;
export type WormholeConfig = z.infer<typeof WormholeConfigSchema>;
export type ServerConfig = z.infer<typeof ServerConfigSchema>;
export type WormholeGuardianData = z.infer<typeof WormholeGuardianDataSchema>;
export type UploadResponse = z.infer<typeof UploadResponseSchema>;
export type PeersResponse = z.infer<typeof PeersResponseSchema>;

// We use this type in validation functions to avoid relying on unchecked input.
export type UncheckedPeer = BasePeer & PeerSignature;

export type ValidationError<T> = {
  success: true;
  value: T;
} | {
  success: false;
  error: string;
};

// Validation helper function
export function validate<IN, OUT>(
  schema: z.ZodType<OUT, IN>,
  data: IN,
  errorMessage: string,
): ValidationError<OUT> {
  const validationResult = schema.safeParse(data);
  if (!validationResult.success) {
    let fullMessage = errorMessage + '\n';
    validationResult.error.issues.forEach((error) => {
      fullMessage += `  - ${error.path.flat().join('.')}: ${error.message}\n`;
    });
    return { success: false, error: fullMessage };
  }
  return { success: true, value: validationResult.data };
}

export function validateOrFail<IN, OUT>(schema: z.ZodType<OUT, IN>, data: IN, errorMessage: string): OUT {
  const validationResult = validate(schema, data, errorMessage);
  if (!validationResult.success) {
    console.error(`[ERROR] ${validationResult.error}`);
    throw new Error(validationResult.error);
  }
  return validationResult.value;
}