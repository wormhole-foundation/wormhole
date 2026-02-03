import { ethers, resolveAddress, TypedDataDomain, TypedDataEncoder, TypedDataField } from "ethers";
import { GetPublicKeyCommand, KMSClient, SignCommand } from "@aws-sdk/client-kms";

export class KmsSigner extends ethers.AbstractSigner {
    private readonly kmsClient: KMSClient
    private readonly region: string;
    private address: string | undefined;

    constructor(private readonly arn: string, provider: ethers.Provider | null) {
        super(provider);
        this.region = regionFromKmsArn(arn);
        this.kmsClient = new KMSClient({ region: this.region });
    }

    connect(provider: null | ethers.Provider): KmsSigner {
        return new KmsSigner(this.arn, provider);
    }

    signTransaction(): Promise<string> {
        throw new Error("signTransaction not implemented");
    }

    async signTypedData(domain: TypedDataDomain,
        types: Record<string, Array<TypedDataField>>,
        value: Record<string, unknown>,): Promise<string> {
        // Populate any ENS names
        const populated = await TypedDataEncoder.resolveNames(domain, types, value, (name: string) => {
            return resolveAddress(name, this.provider) as Promise<string>;
        });

        const digestHex = TypedDataEncoder.hash(populated.domain, types, populated.value as Record<string, unknown>);

        let sig = await this.signWithKms(digestHex);

        // Serialize the signature
        return sig.serialized;
    }

    async signMessage(message: string | Uint8Array): Promise<string> {
        // EIP-191 digest that ethers.signMessage(bytes) signs
        const digestHex = ethers.hashMessage(message);

        const sig = await this.signWithKms(digestHex);

        return sig.serialized;
    }


    async getAddress(): Promise<string> {
        if (this.address === undefined) {
            // cache address to avoid multiple calls to kms
            this.address = await this.getAddressFromKms();
        }
        return this.address;
    }

    private async getAddressFromKms(): Promise<string> {
        let out;
        try {
            out = await this.kmsClient.send(
                new GetPublicKeyCommand({ KeyId: this.arn })
            );
        } catch (error) {
            console.error(`[KMS] GetPublicKeyCommand failed:`, error);
            throw error;
        }

        if (!out.PublicKey) {
            throw new Error("KMS GetPublicKey returned no public key");
        }

        // KMS returns SubjectPublicKeyInfo DER
        const spkiDer = new Uint8Array(out.PublicKey);

        // ---- extract uncompressed EC public key (0x04 + 64 bytes) ----
        let i = 0;
        const read = () => spkiDer[i++];
        const readLen = () => {
            const b = read();
            if ((b & 0x80) === 0) return b;
            const n = b & 0x7f;
            let len = 0;
            for (let j = 0; j < n; j++) len = (len << 8) | read();
            return len;
        };

        // SEQUENCE
        if (read() !== 0x30) throw new Error("Invalid SPKI");
        readLen();

        while (i < spkiDer.length) {
            const tag = read();
            const len = readLen();

            if (tag === 0x03) {
                // BIT STRING
                if (spkiDer[i] !== 0x00) {
                    throw new Error("Unexpected unused bits");
                }
                const pubkey = spkiDer.slice(i + 1, i + len);
                if (pubkey[0] !== 0x04) {
                    throw new Error("Expected uncompressed public key");
                }

                const address = ethers.computeAddress(ethers.hexlify(pubkey));
                return address;
            }

            i += len;
        }

        throw new Error("Public key not found in SPKI");
    }

    private async signWithKms(digestHex: string): Promise<ethers.Signature> {
        const expectedAddress = await this.getAddress();

        const digestBytes = ethers.getBytes(digestHex);
        try {
            const signed = await this.kmsClient.send(new SignCommand({
                KeyId: this.arn,
                MessageType: "DIGEST",
                Message: digestBytes,
                SigningAlgorithm: "ECDSA_SHA_256",
            }));

            if (!signed.Signature) throw new Error("KMS Sign returned no Signature");

            // DER -> r,s ; normalize low-s ; determine v by recovery
            const { r, s } = parseEcdsaDerSignature(new Uint8Array(signed.Signature));
            const sLow = normalizeLowS(s);

            const rHex = bigintToHex32(r);
            const sHex = bigintToHex32(sLow);

            for (const v of [27, 28] as const) {
                const sig = ethers.Signature.from({ r: rHex, s: sHex, v });
                const recovered = ethers.recoverAddress(digestHex, sig);

                if (recovered.toLowerCase() === expectedAddress.toLowerCase()) {
                    return sig; // 0x + 65 bytes
                }
            }

            throw new Error("Failed to determine recovery id (v) for KMS signature");
        } catch (error) {
            console.error(`[KMS] SignCommand failed:`, error);
            throw error;
        }
    }
}

const SECP256K1_N = BigInt("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141");
const SECP256K1_HALF_N = SECP256K1_N / 2n;

function regionFromKmsArn(arn: string): string {
    // arn:aws:kms:us-east-2:ACCOUNT:key/...
    const parts = arn.split(":");
    if (parts.length < 4) throw new Error(`Invalid KMS ARN: ${arn}`);
    return parts[3];
}

function bigintToHex32(x: bigint): string {
    let h = x.toString(16);
    if (h.length > 64) throw new Error("Value does not fit into 32 bytes");
    h = h.padStart(64, "0");
    return "0x" + h;
}

function normalizeLowS(s: bigint): bigint {
    return s > SECP256K1_HALF_N ? (SECP256K1_N - s) : s;
}

/**
 * Minimal DER parser for ECDSA signature:
 *   SEQUENCE { INTEGER r, INTEGER s }
 */
function parseEcdsaDerSignature(der: Uint8Array): { r: bigint; s: bigint } {
    let i = 0;
    const readByte = () => {
        if (i >= der.length) throw new Error("DER parse: out of bounds");
        return der[i++];
    };
    const readLen = (): number => {
        const b = readByte();
        if ((b & 0x80) === 0) return b;
        const n = b & 0x7f;
        if (n === 0 || n > 4) throw new Error("DER parse: invalid length");
        let len = 0;
        for (let k = 0; k < n; k++) len = (len << 8) | readByte();
        return len;
    };
    const expectTag = (tag: number) => {
        const t = readByte();
        if (t !== tag) throw new Error(`DER parse: expected tag 0x${tag.toString(16)}, got 0x${t.toString(16)}`);
    };
    const readInt = (): bigint => {
        expectTag(0x02); // INTEGER
        const len = readLen();
        const bytes = der.slice(i, i + len);
        i += len;
        // INTEGER is signed; ECDSA r/s are positive. Strip leading 0x00 if present.
        let start = 0;
        while (start < bytes.length - 1 && bytes[start] === 0x00) start++;
        let x = 0n;
        for (let j = start; j < bytes.length; j++) x = (x << 8n) | BigInt(bytes[j]);
        return x;
    };

    expectTag(0x30); // SEQUENCE
    readLen(); // total length (not strictly needed)
    const r = readInt();
    const s = readInt();
    return { r, s };
}