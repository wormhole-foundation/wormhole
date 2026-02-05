import { PeerClientConfig } from "../types.js";
import { KmsSigner } from "./kms.js";
import { ethers } from "ethers";

export type CreateSignerConfig = Pick<PeerClientConfig, "guardianKey">;

export async function createSigner(config: CreateSignerConfig): Promise<ethers.Signer> {
    if (config.guardianKey === undefined) {
        throw new Error("Guardian private key or ARN is required");
    }

    if (config.guardianKey.type === "key") {
        return new ethers.Wallet(config.guardianKey.key);
    }

    return KmsSigner.create(config.guardianKey.arn);
}