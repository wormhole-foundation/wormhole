import * as solanaWeb3 from "@solana/web3.js";
import {PublicKey, TransactionInstruction} from "@solana/web3.js";
import BN from 'bn.js';
import assert from "assert";
// @ts-ignore
import * as BufferLayout from 'buffer-layout'
import {Token} from "@solana/spl-token";
import {TOKEN_PROGRAM} from "../config";
import * as bs58 from "bs58";

export interface AssetMeta {
    chain: number,
    address: Buffer
}

export interface Lockup {
    amount: BN,
    toChain: number,
    sourceAddress: PublicKey,
    targetAddress: Uint8Array,
    assetAddress: Uint8Array,
    assetChain: number,
    nonce: number,
    vaa: Uint8Array,
    vaaTime: number,
    initialized: boolean,
}

export const CHAIN_ID_SOLANA = 1;

class SolanaBridge {
    connection: solanaWeb3.Connection;
    programID: PublicKey;
    tokenProgram: PublicKey;

    constructor(connection: solanaWeb3.Connection, programID: PublicKey, tokenProgram: PublicKey) {
        this.programID = programID;
        this.tokenProgram = tokenProgram;
        this.connection = connection;
    }

    async createLockAssetInstruction(
        payer: PublicKey,
        tokenAccount: PublicKey,
        mint: PublicKey,
        amount: BN,
        targetChain: number,
        targetAddress: Buffer,
        asset: AssetMeta,
        nonce: number,
    ): Promise<TransactionInstruction> {
        const dataLayout = BufferLayout.struct([
            BufferLayout.u8('instruction'),
            uint256('amount'),
            BufferLayout.u8('targetChain'),
            BufferLayout.blob(32, 'assetAddress'),
            BufferLayout.u8('assetChain'),
            BufferLayout.u8('assetDecimals'),
            BufferLayout.blob(32, 'targetAddress'),
            BufferLayout.seq(BufferLayout.u8(), 1),
            BufferLayout.u32('nonce'),
        ]);

        let nonceBuffer = Buffer.alloc(4);
        nonceBuffer.writeUInt32LE(nonce, 0);

        // @ts-ignore
        let configKey = await this.getConfigKey();
        let seeds: Array<Buffer> = [Buffer.from("transfer"), configKey.toBuffer(), new Buffer([asset.chain]),
            padBuffer(asset.address, 32), new Buffer([targetChain]), padBuffer(targetAddress, 32), tokenAccount.toBuffer(),
            nonceBuffer,
        ];
        // @ts-ignore
        let transferKey = (await solanaWeb3.PublicKey.findProgramAddress(seeds, this.programID))[0];

        const data = Buffer.alloc(dataLayout.span);
        dataLayout.encode(
            {
                instruction: 1, // TransferOut instruction
                amount: padBuffer(new Buffer(amount.toArray()), 32),
                targetChain: targetChain,
                assetAddress: padBuffer(asset.address, 32),
                assetChain: asset.chain,
                assetDecimals: 0, // This is fetched on chain
                targetAddress: padBuffer(targetAddress, 32),
                nonce: nonce,
            },
            data,
        );

        const keys = [
            {pubkey: this.programID, isSigner: false, isWritable: false},
            {pubkey: solanaWeb3.SystemProgram.programId, isSigner: false, isWritable: false},
            {pubkey: this.tokenProgram, isSigner: false, isWritable: false},
            {pubkey: tokenAccount, isSigner: false, isWritable: true},
            {pubkey: configKey, isSigner: false, isWritable: false},

            {pubkey: transferKey, isSigner: false, isWritable: true},
            {pubkey: mint, isSigner: false, isWritable: false},
            {pubkey: payer, isSigner: true, isWritable: true},
        ];

        if (asset.chain == CHAIN_ID_SOLANA) {
            // @ts-ignore
            let custodyKey = (await solanaWeb3.PublicKey.findProgramAddress([Buffer.from("custody"), configKey.toBuffer(), mint.toBuffer()], this.programID))[0];
            keys.push({pubkey: custodyKey, isSigner: false, isWritable: true})
        }


        return new TransactionInstruction({
            keys,
            programId: this.programID,
            data,
        });
    }

    // fetchAssetMeta fetches the AssetMeta for an SPL token
    async fetchAssetMeta(
        mint: PublicKey,
    ): Promise<AssetMeta> {
        // @ts-ignore
        let configKey = await this.getConfigKey();
        let seeds: Array<Buffer> = [Buffer.from("meta"), configKey.toBuffer(), mint.toBuffer()];
        // @ts-ignore
        let metaKey = (await solanaWeb3.PublicKey.findProgramAddress(seeds, this.programID))[0];
        let metaInfo = await this.connection.getAccountInfo(metaKey);
        if (metaInfo == null || metaInfo.lamports == 0) {
            return {
                address: mint.toBuffer(),
                chain: CHAIN_ID_SOLANA,
            }
        } else {
            const dataLayout = BufferLayout.struct([
                BufferLayout.u8('assetChain'),
                BufferLayout.blob(32, 'assetAddress'),
            ]);
            let wrappedMeta = dataLayout.decode(metaInfo?.data);

            return {
                address: wrappedMeta.assetAddress,
                chain: wrappedMeta.assetChain
            }
        }
    }

    // fetchAssetMeta fetches the AssetMeta for an SPL token
    async fetchTransferProposals(
        tokenAccount: PublicKey,
    ): Promise<Lockup[]> {
        let accountRes = await fetch("http://localhost:8899", {
            method: "POST",
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                "jsonrpc": "2.0",
                "id": 1,
                "method": "getProgramAccounts",
                "params": [this.programID.toString(), {
                    "filters": [{"dataSize": 1152}, {
                        "memcmp": {
                            "offset": 33,
                            "bytes": tokenAccount.toString()
                        }
                    }]
                }]
            }),
        })
        let raw_accounts = (await accountRes.json())["result"];

        const dataLayout = BufferLayout.struct([
            uint256('amount'),
            BufferLayout.u8('toChain'),
            BufferLayout.blob(32, 'sourceAddress'),
            BufferLayout.blob(32, 'targetAddress'),
            BufferLayout.blob(32, 'assetAddress'),
            BufferLayout.u8('assetChain'),
            BufferLayout.u8('assetDecimals'),
            BufferLayout.u32('nonce'),
            BufferLayout.blob(1001, 'vaa'),
            BufferLayout.u32('vaaTime'),
            BufferLayout.u8('initialized'),
        ]);

        let accounts: Lockup[] = [];
        for (let acc of raw_accounts) {
            acc = acc.account;
            let parsedAccount = dataLayout.decode(bs58.decode(acc.data))
            console.log(parsedAccount);
            accounts.push({
                amount: new BN(parsedAccount.amount, 2, "le"),
                assetAddress: parsedAccount.assetAddress,
                assetChain: acc.assetChain,
                initialized: acc.initialized == 1,
                nonce: acc.nonce,
                sourceAddress: new PublicKey(parsedAccount.sourceAddress),
                targetAddress: parsedAccount.targetAddress,
                toChain: acc.toChain,
                vaa: acc.vaa,
                vaaTime: acc.vaaTime
            })
        }

        return accounts
    }

    AccountLayout = BufferLayout.struct([publicKey('mint'), publicKey('owner'), uint64('amount'), BufferLayout.u32('option'), publicKey('delegate'), BufferLayout.u8('is_initialized'), BufferLayout.u8('is_native'), BufferLayout.u16('padding'), uint64('delegatedAmount')]);

    async createWrappedAssetAndAccountInstructions(owner: PublicKey, mint: PublicKey): Promise<[TransactionInstruction[], solanaWeb3.Account]> {
        const newAccount = new solanaWeb3.Account();

        // @ts-ignore
        const balanceNeeded = await Token.getMinBalanceRentForExemptAccount(this.connection);
        let transaction = solanaWeb3.SystemProgram.createAccount({
            fromPubkey: owner,
            newAccountPubkey: newAccount.publicKey,
            lamports: balanceNeeded,
            space: this.AccountLayout.span,
            programId: TOKEN_PROGRAM,
        }); // create the new account

        const keys = [{
            pubkey: newAccount.publicKey,
            isSigner: false,
            isWritable: true
        }, {
            pubkey: mint,
            isSigner: false,
            isWritable: false
        }, {
            pubkey: owner,
            isSigner: false,
            isWritable: false
        }];
        const dataLayout = BufferLayout.struct([BufferLayout.u8('instruction')]);
        const data = Buffer.alloc(dataLayout.span);
        dataLayout.encode({
            instruction: 1 // InitializeAccount instruction

        }, data);
        let ix_init = {
            keys,
            programId: TOKEN_PROGRAM,
            data
        }

        return [[transaction.instructions[0], ix_init], newAccount]
    }

    async getConfigKey(): Promise<PublicKey> {
        // @ts-ignore
        return (await solanaWeb3.PublicKey.findProgramAddress([Buffer.from("bridge")], this.programID))[0]
    }

    async getWrappedAssetMint(asset: AssetMeta): Promise<PublicKey> {
        if (asset.chain === 1) {
            return new PublicKey(asset.address)
        }

        let configKey = await this.getConfigKey();
        let seeds: Array<Buffer> = [Buffer.from("wrapped"), configKey.toBuffer(), Buffer.of(asset.chain),
            padBuffer(asset.address, 32)];
        // @ts-ignore
        return (await solanaWeb3.PublicKey.findProgramAddress(seeds, this.programID))[0];
    }
}

// Taken from https://github.com/solana-labs/solana-program-library
// Licensed under Apache 2.0

export class u64 extends BN {
    /**
     * Convert to Buffer representation
     */
    toBuffer(): Buffer {
        const a = super.toArray().reverse();
        const b = Buffer.from(a);
        if (b.length === 8) {
            return b;
        }
        assert(b.length < 8, 'u64 too large');

        const zeroPad = Buffer.alloc(8);
        b.copy(zeroPad);
        return zeroPad;
    }

    /**
     * Construct a u64 from Buffer representation
     */
    static fromBuffer(buffer: Buffer): u64 {
        assert(buffer.length === 8, `Invalid buffer length: ${buffer.length}`);
        return new BN(
            // @ts-ignore
            [...buffer]
                .reverse()
                .map(i => `00${i.toString(16)}`.slice(-2))
                .join(''),
            16,
        );
    }
}

function padBuffer(b: Buffer, len: number): Buffer {
    const zeroPad = Buffer.alloc(len);
    b.copy(zeroPad, len - b.length);
    return zeroPad;
}

export class u256 extends BN {
    /**
     * Convert to Buffer representation
     */
    toBuffer(): Buffer {
        const a = super.toArray().reverse();
        const b = Buffer.from(a);
        if (b.length === 32) {
            return b;
        }
        assert(b.length < 32, 'u256 too large');

        const zeroPad = Buffer.alloc(32);
        b.copy(zeroPad);
        return zeroPad;
    }

    /**
     * Construct a u256 from Buffer representation
     */
    static fromBuffer(buffer: number[]): u256 {
        assert(buffer.length === 32, `Invalid buffer length: ${buffer.length}`);
        return new BN(
            // @ts-ignore
            [...buffer]
                .reverse()
                .map(i => `00${i.toString(16)}`.slice(-2))
                .join(''),
            16,
        );
    }
}

/**
 * Layout for a public key
 */
export const publicKey = (property: string = 'publicKey'): Object => {
    return BufferLayout.blob(32, property);
};

/**
 * Layout for a 64bit unsigned value
 */
export const uint64 = (property: string = 'uint64'): Object => {
    return BufferLayout.blob(8, property);
};

/**
 * Layout for a 256-bit unsigned value
 */
export const uint256 = (property: string = 'uint256'): Object => {
    return BufferLayout.blob(32, property);
};

/**
 * Layout for a Rust String type
 */
export const rustString = (property: string = 'string') => {
    const rsl = BufferLayout.struct(
        [
            BufferLayout.u32('length'),
            BufferLayout.u32('lengthPadding'),
            BufferLayout.blob(BufferLayout.offset(BufferLayout.u32(), -8), 'chars'),
        ],
        property,
    );
    const _decode = rsl.decode.bind(rsl);
    const _encode = rsl.encode.bind(rsl);

    rsl.decode = (buffer: Buffer, offset: number) => {
        const data = _decode(buffer, offset);
        return data.chars.toString('utf8');
    };

    rsl.encode = (str: string, buffer: Buffer, offset: number) => {
        const data = {
            chars: Buffer.from(str, 'utf8'),
        };
        return _encode(data, buffer, offset);
    };

    return rsl;
};

export {SolanaBridge}
