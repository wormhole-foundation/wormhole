import {
    Address,
    beginCell,
    Builder,
    Cell,
    Contract,
    contractAddress,
    ContractProvider,
    Dictionary,
    Sender,
    SendMode,
    Slice,
    TupleItem,
} from '@ton/core';
import { Opcodes } from './Constants';

export type GuardianSet = {
    keys: Buffer[];
    expirationTime: number;
};

export type Signature = {
    signature: Buffer; // 65 bytes
    guardianIndex: number;
};

export type WormholeConfig = {
    messageFee: bigint;
    sequences: Dictionary<Address, number>;
    guardianSets: Dictionary<number, GuardianSet>;
    guardianSetIndex: number;
    guardianSetExpiry: number;
    chainId: number;
    governanceChainId: number;
    governanceContract: Buffer;
    id: number; // unique contract ID
};

export const GuardianSetDictionaryValue = {
    serialize: (src: GuardianSet, builder: Builder) => {
        const keysDict = Dictionary.empty(Dictionary.Keys.Uint(8), Dictionary.Values.Buffer(32));
        src.keys.forEach((key, index) => {
            keysDict.set(index, key);
        });
        builder.storeDict(keysDict).storeUint(src.keys.length, 8).storeUint(src.expirationTime, 32);
    },
    parse: (src: Slice): GuardianSet => {
        const keysDict = src.loadDict(Dictionary.Keys.Uint(8), Dictionary.Values.Buffer(32));
        const keys = keysDict.keys().map((key) => {
            return keysDict.get(key)!;
        });
        const count = src.loadUint(8);
        if (count !== keys.length) {
            throw new Error('Invalid guardian set count: parsed ' + keys.length + ' keys, got ' + count);
        }
        const expirationTime = src.loadUint(32);
        return { keys, expirationTime };
    },
};

export const SignatureDictionaryValue = {
    serialize: (src: Signature, builder: Builder) => {
        builder.storeBuffer(src.signature, 65).storeUint(src.guardianIndex, 8);
    },
    parse: (src: Slice): Signature => {
        const signature = src.loadBuffer(65);
        const guardianIndex = src.loadUint(8);
        return { signature, guardianIndex };
    },
};

export function wormholeConfigToCell(config: WormholeConfig): Cell {
    return beginCell()
        .storeUint(config.messageFee, 64)
        .storeDict(config.sequences, Dictionary.Keys.Address(), Dictionary.Values.Uint(64))
        .storeDict(config.guardianSets, Dictionary.Keys.Uint(32), GuardianSetDictionaryValue)
        .storeUint(config.guardianSetIndex, 32)
        .storeUint(config.guardianSetExpiry, 32)
        .storeUint(config.chainId, 16)
        .storeUint(config.governanceChainId, 16)
        .storeBuffer(config.governanceContract, 32)
        .storeUint(config.id, 16)
        .endCell();
}

export class Wormhole implements Contract {
    constructor(
        readonly address: Address,
        readonly init?: { code: Cell; data: Cell },
    ) {}

    static createFromAddress(address: Address) {
        return new Wormhole(address);
    }

    static createFromConfig(config: WormholeConfig, code: Cell, workchain = 0) {
        const data = wormholeConfigToCell(config);
        const init = { code, data };
        return new Wormhole(contractAddress(workchain, init), init);
    }

    async sendDeploy(provider: ContractProvider, via: Sender, value: bigint) {
        await provider.internal(via, {
            value,
            sendMode: SendMode.PAY_GAS_SEPARATELY,
            body: beginCell().endCell(),
        });
    }

    async sendPublishMessage(
        provider: ContractProvider,
        via: Sender,
        opts: {
            value: bigint;
            queryId?: bigint | number;
            nonce: number;
            consistencyLevel: number;
            payload: Cell;
            tail?: Cell;
        },
    ) {
        await provider.internal(via, {
            value: opts.value,
            sendMode: SendMode.PAY_GAS_SEPARATELY,
            body: beginCell()
                .storeUint(Opcodes.OP_PUBLISH_MESSAGE, 32)
                .storeUint(BigInt(opts.queryId ?? 0), 64)
                .storeUint(opts.nonce, 32)
                .storeUint(opts.consistencyLevel, 8)
                .storeRef(opts.payload)
                .storeRef(opts.tail ?? beginCell().endCell())
                .endCell(),
        });
    }

    async sendParseAndVerifyVM(
        provider: ContractProvider,
        via: Sender,
        opts: {
            value: bigint;
            queryId?: bigint | number;
            encodedVM: Cell;
            tail: Cell;
        },
    ) {
        await provider.internal(via, {
            value: opts.value,
            sendMode: SendMode.PAY_GAS_SEPARATELY,
            body: beginCell()
                .storeUint(Opcodes.OP_PARSE_AND_VERIFY_VM, 32)
                .storeUint(BigInt(opts.queryId ?? 0), 64)
                .storeRef(opts.encodedVM)
                .storeRef(opts.tail)
                .endCell(),
        });
    }

    async getMessageFee(provider: ContractProvider): Promise<bigint> {
        const result = await provider.get('messageFee', []);
        return result.stack.readBigNumber();
    }

    async getVerifyVM(provider: ContractProvider, vmCell: Cell): Promise<boolean> {
        const args: TupleItem[] = [{ type: 'cell', cell: vmCell }];
        const result = await provider.get('verifyVM', args);
        return result.stack.readBoolean();
    }

    async getSequence(provider: ContractProvider, sender: Address): Promise<number> {
        const args: TupleItem[] = [{ type: 'slice', cell: beginCell().storeAddress(sender).endCell() }];
        const result = await provider.get('getSequence', args);
        return result.stack.readNumber();
    }
}