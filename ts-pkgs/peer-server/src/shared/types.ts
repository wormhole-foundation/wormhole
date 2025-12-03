import { z } from 'zod';
import { readFileSync } from 'fs';
import { checkTlsCertificate, parseGuardianKey } from './parseCrypto.js';

// Zod validation schemas (these also serve as type definitions)
export const BasePeerSchema = z.object({
  hostname: z.string().min(1, "Hostname cannot be empty"),
  port: z.number().int().min(1, "Port must be between 1 and 65535").max(65535, "Port must be between 1 and 65535"),
  tlsX509: z.string().min(1, "TlsX509 certificate cannot be empty"),
});

export const GuardianSchema = z.object({
  guardianAddress: z.string().min(1, "Guardian address cannot be empty"),
  guardianIndex: z.number().int().min(0, "Guardian index must be non-negative"),
});

export const PeerSignatureSchema = z.object({
  signature: z.string().min(1, "Signature cannot be empty"),
});

export const PeerSchema = z.intersection(z.intersection(BasePeerSchema, GuardianSchema), PeerSignatureSchema);

export const PeerRegistrationSchema = z.intersection(
  z.object({ peer: BasePeerSchema }),
  PeerSignatureSchema
);

// Config schema that reads from file paths and transforms to runtime values
export const SelfConfigSchema = z.object({
  guardianPrivateKeyPath: z.string().min(1, "Guardian private key path cannot be empty"),
  serverUrl: z.string().url("Server URL must be a valid HTTP(S) URL"),
  peer: BasePeerSchema,
}).transform((data) => {
  // Load and validate guardian private key
  let guardianPrivateKey: string;
  try {
    const keyContents = readFileSync(data.guardianPrivateKeyPath, 'utf-8');
    const keyBytes = parseGuardianKey(keyContents); // Extract and validate the key
    // Convert to hex format for ethers.Wallet
    guardianPrivateKey = '0x' + Buffer.from(keyBytes).toString('hex');
  } catch (error: any) {
    throw new Error(`Failed to read or validate guardian private key from ${data.guardianPrivateKeyPath}: ${error.message}`);
  }

  // Load and validate TLS certificate
  let tlsX509: string;
  try {
    const certContents = readFileSync(data.peer.tlsX509, 'utf-8');
    if (!checkTlsCertificate(certContents)) {
      throw new Error("Invalid TLS X509 certificate format");
    }
    tlsX509 = certContents;
  } catch (error: any) {
    throw new Error(`Failed to read or validate TLS X509 certificate from ${data.peer.tlsX509}: ${error.message}`);
  }

  return {
    guardianPrivateKey,
    serverUrl: data.serverUrl,
    peer: {
      hostname: data.peer.hostname,
      port: data.peer.port,
      tlsX509
    }
  };
});

export const BaseServerConfigSchema = z.object({
  port: z.number().int().min(1, "Port must be between 1 and 65535").max(65535, "Port must be between 1 and 65535"),
  threshold: z.number().int().min(1, "Threshold must be a positive integer")
});

export const WormholeConfigSchema = z.object({
  ethereum: z.object({
    rpcUrl: z.string().url("Ethereum RPC URL must be a valid URL"),
    chainId: z.number().int().min(1).optional()
  }),
  wormholeContractAddress: z.string().min(1, "Wormhole contract address cannot be empty"),
});

export const ServerConfigSchema = z.intersection(BaseServerConfigSchema, WormholeConfigSchema);

export const WormholeGuardianDataSchema = z.object({
  guardians: z.array(z.string().min(1, "Guardian addresses cannot be empty"))
});

export const UploadResponseSchema = z.object({
  peer: PeerSchema,
  threshold: z.number().int().min(1, "Threshold must be a positive integer")
});

export const PeerArraySchema = z.array(PeerSchema);

export const PeersResponseSchema = z.object({
  peers: PeerArraySchema,
  threshold: z.number().int().min(1, "Threshold must be a positive integer"),
  totalExpectedGuardians: z.number().int().min(1, "Total expected guardians must be a positive integer")
});

// Type definitions inferred from Zod schemas
export type Peer = z.infer<typeof PeerSchema>;
export type BasePeer = z.infer<typeof BasePeerSchema>;
export type Guardian = z.infer<typeof GuardianSchema>;
export type PeerSignature = z.infer<typeof PeerSignatureSchema>;
export type PeerRegistration = z.infer<typeof PeerRegistrationSchema>;
export type SelfConfig = z.infer<typeof SelfConfigSchema>;
export type BaseServerConfig = z.infer<typeof BaseServerConfigSchema>;
export type WormholeConfig = z.infer<typeof WormholeConfigSchema>;
export type ServerConfig = z.infer<typeof ServerConfigSchema>;
export type WormholeGuardianData = z.infer<typeof WormholeGuardianDataSchema>;
export type UploadResponse = z.infer<typeof UploadResponseSchema>;
export type PeersResponse = z.infer<typeof PeersResponseSchema>;

export type ValidationError<T> = {
  success: true;
  data: T;
} | {
  success: false;
  error: string;
};

// Validation helper function
export function validate<IN, OUT>(
  schema: z.ZodSchema<OUT, z.ZodTypeDef, IN>,
  data: IN,
  errorMessage: string,
): ValidationError<OUT> {
  const validationResult = schema.safeParse(data);
  if (!validationResult.success) {
    let fullMessage = errorMessage + '\n';
    validationResult.error.errors.forEach((error) => {
      fullMessage += `  - ${error.path.flat().join('.')}: ${error.message}\n`;
    });
    return { success: false, error: fullMessage };
  }
  return { success: true, data: validationResult.data };
}

export function validateOrFail<IN, OUT>(schema: z.ZodSchema<OUT, z.ZodTypeDef, IN>, data: IN, errorMessage: string): OUT {
  const validationResult = validate(schema, data, errorMessage);
  if (!validationResult.success) {
    console.error(`[ERROR] ${validationResult.error}`);
    throw new Error(validationResult.error);
  }
  return validationResult.data;
}