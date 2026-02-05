import { ethers, resolveAddress, TypedDataDomain, TypedDataEncoder, TypedDataField } from "ethers";
import { GetPublicKeyCommand, KMSClient, SignCommand } from "@aws-sdk/client-kms";
import { defaultProvider } from "@aws-sdk/credential-provider-node";

const sequenceTag = 0x30;
const integerTag = 0x02;
const bitStringTag = 0x03;

type AwsCredentialIdentity = Awaited<ReturnType<ReturnType<typeof defaultProvider>>>;

export class KmsSigner extends ethers.AbstractSigner {
    private readonly kmsClient: KMSClient;
    private readonly region: string;
    private address: string | undefined;
    static credentials: AwsCredentialIdentity | undefined;

    private constructor(private readonly arn: string, provider: ethers.Provider | null = null) {
        super(provider);

        this.region = regionFromKmsArn(arn);
        this.kmsClient = new KMSClient({ region: this.region, credentials: KmsSigner.credentials });
    }

    static async create(arn: string, provider: ethers.Provider | null = null) {
        if (this.credentials === undefined) {
            const credentialsProvider = defaultProvider();
            this.credentials = await credentialsProvider();
        }

        return new this(arn, provider);
    }

    connect(provider: null | ethers.Provider): KmsSigner {
        return new KmsSigner(this.arn, provider);
    }

    signTransaction(): Promise<string> {
        throw new Error("signTransaction not implemented");
    }

    async signTypedData(
        domain: TypedDataDomain,
        types: Record<string, Array<TypedDataField>>,
        value: Record<string, unknown>,
    ): Promise<string> {
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
        const out = await this.kmsClient.send(
            new GetPublicKeyCommand({ KeyId: this.arn })
        );

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
            if (b > 86) throw new Error(`DER parse: Invalid length for SubjectPublicKeyInfo ${b} at offset ${i - 1}`);
            return b;
        };
        const readAlgoLen = () => {
            const b = read();
            if (b > 16) throw new Error(`DER parse: Invalid length for AlgorithmIdentifier ${b} at offset ${i - 1}`);
            return b;
        };
        const readPubKeyLen = () => {
            const b = read();
            if (b > 66) throw new Error(`DER parse: Invalid length for public key ${b} at offset ${i - 1}`);
            return b;
        };

        if (read() !== sequenceTag) throw new Error("DER parse: Invalid SPKI");
        const derLength = 2 + readLen();
        if (derLength > spkiDer.length)
            throw new Error(`DER parse: out of bounds buffer read.
Expected DER object length: ${derLength}, actual buffer size: ${spkiDer.length}`);

        // Read AlgorithmIdentifier
        const algoTag = read();
        if (algoTag !== sequenceTag) throw new Error(`DER parse: Invalid SPKI. Expected sequence at ${i - 1}`);
        // Skip the algorithm and parameters here
        const algoLen = readAlgoLen();
        i += algoLen;

        const pubKeytag = read();
        if (pubKeytag !== bitStringTag) throw new Error(`DER parse: Invalid SPKI. Expected bitsting at ${i - 1}`);
        const pubKeyLen = readPubKeyLen();

        const pubkey = spkiDer.subarray(i, i + pubKeyLen);
        if (pubkey[0] !== 0x04) {
            throw new Error("DER parse: Expected uncompressed public key");
        }

        const address = ethers.computeAddress(ethers.hexlify(pubkey));
        return address;
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

const SECP256K1_N = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141n;
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
    let derLength: number;

    const readByte = () => {
        if (i >= der.length) throw new Error("DER parse: out of bounds");
        return der[i++];
    };
    const readLen = (): number => {
        const b = readByte();
        // Note that the highest bit indicates whether this is multibyte but we're going to ignore that here
        if (b > 72) throw new Error(`DER parse: Invalid length for ECDSA signature ${b} at offset ${i}`);
        return b;
    };
    const readLenInt = (): number => {
        const b = readByte();
        // Note that the highest bit indicates whether this is multibyte but we're going to ignore that here
        if (b > 33) throw new Error(`DER parse: Invalid length for integer in ECDSA signature ${b}, at offset ${i}`);
        return b;
    };
    const expectTag = (tag: number) => {
        const t = readByte();
        if (t !== tag) throw new Error(`DER parse: expected tag 0x${tag.toString(16)}, got 0x${t.toString(16)}`);
    };
    const readInt = (): bigint => {
        expectTag(integerTag);
        const len = readLenInt();
        // Is this out of bounds?
        if (derLength < i + len) throw new Error(`DER parse: out of bounds integer read`);
        const bytes = der.subarray(i, i + len);
        // INTEGER is signed; ECDSA r/s are positive.
        if ((bytes[0] & 0x80) > 0) throw new Error(`DER parse: expected positive integer at ${i}`);
        i += len;
        let x = 0n;
        for (let j = 0; j < bytes.length; j++) x = (x << 8n) | BigInt(bytes[j]);
        return x;
    };

    expectTag(sequenceTag);
    derLength = 2 + readLen();
    if (derLength > der.length)
        throw new Error(`DER parse: out of bounds buffer read.
Expected DER object length: ${derLength}, actual buffer size: ${der.length}`);
    const r = readInt();
    const s = readInt();
    return { r, s };
}