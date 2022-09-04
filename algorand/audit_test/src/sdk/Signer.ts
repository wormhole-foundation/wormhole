import algosdk, { Transaction } from 'algosdk'
import { Address } from './AlgorandTypes'
import { SignCallback, TealSignCallback } from './Deployer'

export class Signer {
    private signatures: Map<Address, Uint8Array> = new Map()
    readonly callback: SignCallback
    readonly tealCallback: TealSignCallback

    constructor() {
        this.callback = this.sign.bind(this)
        this.tealCallback = this.tealSign.bind(this)
    }

    private getPrivateKey(addr: Address): Uint8Array {
        const pk = this.signatures.get(addr)
        if (pk === undefined)
            throw new Error("Couldn't find account " + addr + " for signing")
        return pk
    }

    addFromMnemonic(mnemonic: string): Address {
        const account = algosdk.mnemonicToSecretKey(mnemonic)
        this.signatures.set(account.addr, account.sk)
        return account.addr
    }

    addFromSecretKey(secretKey: Uint8Array): Address {
        const mnemonic = algosdk.secretKeyToMnemonic(secretKey)
        return this.addFromMnemonic(mnemonic)
    }

    createAccount(): Address {
        const { sk: secretKey, addr: address } = algosdk.generateAccount();
        this.signatures.set(address, secretKey)
        return address
    }

    async sign(txs: Transaction[]): Promise<Uint8Array[]> {
        return Promise.all(txs.map(async tx => {
            const sender = algosdk.encodeAddress(tx.from.publicKey)
            return tx.signTxn(this.getPrivateKey(sender))
        }))
    }

    rawSign(txs: Transaction[]): Uint8Array[] {
        return txs.map(tx => {
            const sender = algosdk.encodeAddress(tx.from.publicKey)
            return tx.rawSignTxn(this.getPrivateKey(sender))
        })
    }

    async tealSign(data: Uint8Array, from: Address, to: Address): Promise<Uint8Array> {
        return algosdk.tealSign(this.getPrivateKey(from), data, to)
    }
}
