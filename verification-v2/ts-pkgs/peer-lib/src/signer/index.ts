import { PeerClientConfig } from "../types.js";
import { KmsSigner } from "./kms.js";
import { ethers } from "ethers";

export function isPrivateKey(guardianPrivateKeyOrArn: string): boolean {
    return guardianPrivateKeyOrArn.startsWith("0x");
}

export type CreateSignerConfig = Pick<PeerClientConfig, "guardianPrivateKeyOrArn" | "wormhole">;

export function createSigner(config: CreateSignerConfig): ethers.Signer {
    if (config.guardianPrivateKeyOrArn === undefined) {
        throw new Error("Guardian private key or ARN is required");
    }

    if (isPrivateKey(config.guardianPrivateKeyOrArn)) {
        return new ethers.Wallet(config.guardianPrivateKeyOrArn);
    }

    const provider = new ethers.JsonRpcProvider(config.wormhole?.ethereum.rpcUrl, undefined, { staticNetwork: true });
    return new KmsSigner(config.guardianPrivateKeyOrArn, provider);
}