import {
    Signer,
    ChainName,
    SignOnlySigner,
    SignedTx,
    UnsignedTransaction,
    RpcConnection,
} from '@wormhole-foundation/connect-sdk';
import { EvmPlatform } from '@wormhole-foundation/connect-sdk-evm';
import { ethers } from 'ethers';

// Get a SignOnlySigner for the EVM platform
export async function getEvmSigner(
    rpc: RpcConnection<'Evm'>,
    privateKey: string,
): Promise<Signer> {
    const [_, chain] = await EvmPlatform.chainFromRpc(rpc);
    return new EvmSigner(chain, rpc, privateKey);
}

// EvmSigner implements SignOnlySender
export class EvmSigner implements SignOnlySigner {
    _wallet: ethers.Wallet;

    constructor(
        private _chain: ChainName,
        private provider: ethers.Provider,
        privateKey: string,
    ) {
        this._wallet = new ethers.Wallet(privateKey, provider);
    }

    chain(): ChainName {
        return this._chain;
    }

    address(): string {
        return this._wallet.address;
    }

    async sign(tx: UnsignedTransaction[]): Promise<SignedTx[]> {
        const signed = [];

        let nonce = await this.provider.getTransactionCount(this.address());

        let gasLimit = 1_000_000n;
        let maxFeePerGas = 1_500_000_000n; // 1.5gwei
        let maxPriorityFeePerGas = 100_000_000n; // 0.1gwei

        for (const txn of tx) {
            const { transaction, description } = txn;
            console.log(`Signing: ${description} for ${this.address()}`);

            const t: ethers.TransactionRequest = {
                ...transaction,
                ...{
                    gasLimit,
                    maxFeePerGas,
                    maxPriorityFeePerGas,
                    nonce,
                },
            };

            // TODO
            // const estimate = await this.provider.estimateGas(t)
            // t.gasLimit = estimate

            signed.push(await this._wallet.signTransaction(t));

            nonce += 1;
        }
        return signed;
    }
}