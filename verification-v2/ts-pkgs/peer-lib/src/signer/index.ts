import { PeerClientConfig } from "../types.js";
import { KmsSigner } from "./kms.js";
import { ethers } from "ethers";

export function isPrivateKey(guardianPrivateKeyOrArn: string): boolean {
    return guardianPrivateKeyOrArn.startsWith("0x");
}

export type CreateSignerConfig = Pick<PeerClientConfig, "guardianKey">;

export function createSigner(config: CreateSignerConfig): ethers.Signer {
    if (config.guardianKey === undefined) {
        throw new Error("Guardian private key or ARN is required");
    }

    if (config.guardianKey.type === "key") {
        return new ethers.Wallet(config.guardianKey.key);
    }

    return new KmsSigner(config.guardianKey.arn);
}