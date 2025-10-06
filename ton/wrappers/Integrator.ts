import { Address, beginCell, Cell, Contract, contractAddress, ContractProvider, Sender, SendMode } from '@ton/core';
import { Opcodes } from './Constants';

export type IntegratorConfig = {
    id: number;
    wormholeAddress: Address;
};

export function integratorConfigToCell(config: IntegratorConfig): Cell {
    return beginCell().storeAddress(config.wormholeAddress).storeUint(config.id, 16).endCell();
}

export type CommentOpts = {
    queryId: number;
    nonce: number;
    consistencyLevel: number;
    to: Address;
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
        await provider.internal(via, {
            value,
            sendMode: SendMode.PAY_GAS_SEPARATELY,
            body: beginCell()
                .storeUint(Opcodes.OP_SEND_COMMENT, 32)
                .storeUint(opts.queryId, 64)
                .storeUint(opts.nonce, 32)
                .storeUint(opts.consistencyLevel, 8)
                .storeAddress(opts.to)
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
