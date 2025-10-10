import { Address, beginCell, Cell, Contract, contractAddress, ContractProvider, Sender, SendMode } from '@ton/core';
import { Opcodes } from './Constants';

export type IntegratorConfig = {
    wormholeAddress: Address;
    nonce: number;
    id: number;
};

export function integratorConfigToCell(config: IntegratorConfig): Cell {
    return beginCell()
        .storeAddress(config.wormholeAddress)
        .storeUint(config.nonce, 32)
        .storeUint(config.id, 16)
        .endCell();
}

export type CommentOpts = {
    queryId: number;
    consistencyLevel: number;
    chainId: number;
    to: Buffer;
    comment: string;
};

export type RelayCommentOpts = {
    queryId: number;
    encodedVaa: Cell;
};
export class Integrator implements Contract {
    constructor(
        readonly address: Address,
        readonly init?: { code: Cell; data: Cell },
    ) {}

    static createFromAddress(address: Address) {
        return new Integrator(address);
    }

    static createFromConfig(config: IntegratorConfig, code: Cell, workchain = 0) {
        const data = integratorConfigToCell(config);
        const init = { code, data };
        return new Integrator(contractAddress(workchain, init), init);
    }

    async sendDeploy(provider: ContractProvider, via: Sender, value: bigint) {
        await provider.internal(via, {
            value,
            sendMode: SendMode.PAY_GAS_SEPARATELY,
            body: beginCell().endCell(),
        });
    }

    async sendComment(provider: ContractProvider, via: Sender, value: bigint, opts: CommentOpts) {
        if (opts.to.length !== 32) {
            throw new Error('address must be 32 bytes');
        }
        await provider.internal(via, {
            value,
            sendMode: SendMode.PAY_GAS_SEPARATELY,
            body: beginCell()
                .storeUint(Opcodes.OP_SEND_COMMENT, 32)
                .storeUint(opts.queryId, 64)
                .storeUint(opts.consistencyLevel, 8)
                .storeUint(opts.chainId, 16)
                .storeBuffer(opts.to, 32)
                .storeStringRefTail(opts.comment)
                .endCell(),
        });
    }

    async sendRelayComment(provider: ContractProvider, via: Sender, value: bigint, opts: RelayCommentOpts) {
        await provider.internal(via, {
            value,
            sendMode: SendMode.PAY_GAS_SEPARATELY,
            body: beginCell()
                .storeUint(Opcodes.OP_RELAY_COMMENT, 32)
                .storeUint(opts.queryId, 64)
                .storeRef(opts.encodedVaa)
                .endCell(),
        });
    }
}
