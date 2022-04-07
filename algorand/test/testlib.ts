
/**
 *
 * Copyright 2022 Wormhole Project Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

const web3EthAbi = require('web3-eth-abi')
const web3Utils = require('web3-utils')
const elliptic = require('elliptic')

class TestLib {
    zeroBytes: string;

    constructor() {
        this.zeroBytes = "0000000000000000000000000000000000000000000000000000000000000000";
    }

    hexStringToUint8Array(hs: string): Uint8Array {
        if (hs.length % 2 === 1) {
            // prepend a 0
            hs = "0" + hs;
        }
        const buf = Buffer.from(hs, "hex");
        const retval = Uint8Array.from(buf);
        return retval;
    }

    uint8ArrayToHexString(arr: Uint8Array, add0x: boolean) {
        const ret: string = Buffer.from(arr).toString("hex");
        if (!add0x) {
            return ret;
        }
        return "0x" + ret;
    }

    encoder(type:string, val: any) {
        if (type == 'uint8') 
            return web3EthAbi.encodeParameter('uint8', val).substring(2 + (64 - 2));
        if (type == 'uint16')
            return web3EthAbi.encodeParameter('uint16', val).substring(2 + (64 - 4));
        if (type == 'uint32')
            return web3EthAbi.encodeParameter('uint32', val).substring(2 + (64 - 8));
        if (type == 'uint64')
            return web3EthAbi.encodeParameter('uint64', val).substring(2 + (64 - 16));
        if (type == 'uint128')
            return web3EthAbi.encodeParameter('uint128', val).substring(2 + (64 - 32));
        if (type == 'uint256' || type == 'bytes32')
            return web3EthAbi.encodeParameter(type, val).substring(2 + (64 - 64));
    }

    ord(c:any) {
        return c.charCodeAt(0)
    }

    genGuardianSetUpgrade(signers: any, guardianSet: number, targetSet: number, nonce: number, seq: number, guardianKeys:string[]) {
        const b = [
            "0x",
            this.zeroBytes.slice(0, 28*2),
            this.encoder("uint8", this.ord("C")),
            this.encoder("uint8", this.ord("o")),
            this.encoder("uint8", this.ord("r")),
            this.encoder("uint8", this.ord("e")),
            this.encoder("uint8", 2),
            this.encoder("uint16", 0),
            this.encoder("uint32", targetSet),
            this.encoder("uint8", guardianKeys.length)
        ];

        guardianKeys.forEach(x => {
            b.push(x);
        });

        let emitter = "0x" + this.zeroBytes.slice(0, 31*2) + "04"
        let seconds = Math.floor(new Date().getTime() / 1000.0);

        return this.createSignedVAA(guardianSet, signers, seconds, nonce, 1, emitter, seq, 32, b.join(''))
    }

    genGSetFee( signers: any, guardianSet:number , nonce:number, seq:number, amt:number) {
        const b = [
            "0x",
            this.zeroBytes.slice(0, 28*2),
            this.encoder("uint8", this.ord("C")),
            this.encoder("uint8", this.ord("o")),
            this.encoder("uint8", this.ord("r")),
            this.encoder("uint8", this.ord("e")),
            this.encoder("uint8", 3),
            this.encoder("uint16", 8),
            this.encoder("uint256", Math.floor(amt)),
        ];

        let emitter = "0x" + this.zeroBytes.slice(0, 31*2) + "04"

        var seconds = Math.floor(new Date().getTime() / 1000.0);

        return this.createSignedVAA(guardianSet, signers, seconds, nonce, 1, emitter, seq, 32, b.join(''))
    }

    genGFeePayout( signers: any, guardianSet: number, nonce: number, seq: number, amt: number, dest: Uint8Array) {
        const b = [
            "0x",
            this.zeroBytes.slice(0, 28*2),
            this.encoder("uint8", this.ord("C")),
            this.encoder("uint8", this.ord("o")),
            this.encoder("uint8", this.ord("r")),
            this.encoder("uint8", this.ord("e")),
            this.encoder("uint8", 4),
            this.encoder("uint16", 8),
            this.encoder("uint256", Math.floor(amt)),
            this.uint8ArrayToHexString(dest, false)
        ];

        let emitter = "0x" + this.zeroBytes.slice(0, 31*2) + "04"

        var seconds = Math.floor(new Date().getTime() / 1000.0);

        return this.createSignedVAA(guardianSet, signers, seconds, nonce, 1, emitter, seq, 32, b.join(''))
    }

    getEmitter( chain: any) : string {
        if (chain == 1) {
            return "ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5";
        }
        if (chain == 2) {
            return "0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585";
        }
        if (chain == 3) {
            return "0000000000000000000000007cf7b764e38a0a5e967972c1df77d432510564e2";
        }
        if (chain == 4) {
            return "000000000000000000000000b6f6d86a8f9879a9c87f643768d9efc38c1da6e7";
        }
        if (chain == 5) {
            return "0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde";
        }
        return ""
    }
        
    genRegisterChain( signers: any, guardianSet: number, nonce: number, seq: number, chain: string) {
        const b = [
            "0x",
            this.zeroBytes.slice(0, (32-11)*2),
            this.encoder("uint8", this.ord("T")),
            this.encoder("uint8", this.ord("o")),
            this.encoder("uint8", this.ord("k")),
            this.encoder("uint8", this.ord("e")),
            this.encoder("uint8", this.ord("n")),
            this.encoder("uint8", this.ord("B")),
            this.encoder("uint8", this.ord("r")),
            this.encoder("uint8", this.ord("i")),
            this.encoder("uint8", this.ord("d")),
            this.encoder("uint8", this.ord("g")),
            this.encoder("uint8", this.ord("e")),
            this.encoder("uint8", 1),
            this.encoder("uint16", 0),
            this.encoder("uint16", chain),
            this.getEmitter(chain)
        ]
        let emitter = "0x" + this.zeroBytes.slice(0, 31*2) + "04"

        var seconds = Math.floor(new Date().getTime() / 1000.0);

        return this.createSignedVAA(guardianSet, signers, seconds, nonce, 1, emitter, seq, 32, b.join(''))
    }

    genAssetMeta( signers:any, guardianSet:number, nonce:number, seq:number, tokenAddress:string, chain:number, decimals:number, symbol:string, name:string) {
        const b = [
            "0x",
            this.encoder("uint8", 2),
            this.zeroBytes.slice(0, 64 - tokenAddress.length),
            tokenAddress,
            this.encoder("uint16", chain),
            this.encoder("uint8", decimals),
            Buffer.from(symbol).toString("hex"),
            this.zeroBytes.slice(0, (32 - symbol.length)*2),
            Buffer.from(name).toString("hex"),
            this.zeroBytes.slice(0, (32 - name.length)*2)
        ]

//        console.log(b.join())
//        console.log(b.join('').length)

        let emitter = "0x" + this.getEmitter(chain);
        let seconds = Math.floor(new Date().getTime() / 1000.0);

        return this.createSignedVAA(guardianSet, signers, seconds, nonce, 1, emitter, seq, 32, b.join(''))
    }

//    genTransfer( signers, guardianSet, nonce, seq, amount, tokenAddress, tokenChain, toAddress, toChain, fee) {
//        b  = self.encoder("uint8", 1)
//        b += self.encoder("uint256", int(amount * 100000000))
//
//        b += self.zeroPadBytes[0:((32-len(tokenAddress))*2)]
//        b += tokenAddress.hex()
//
//        b += self.encoder("uint16", tokenChain)
//
//        b += self.zeroPadBytes[0:((32-len(toAddress))*2)]
//        b += toAddress.hex()
//
//        b += self.encoder("uint16", toChain)
//
//        b += self.encoder("uint256", int(fee * 100000000))
//
//        emitter = bytes.fromhex(.getEmitter(tokenChain))
//        return self.createSignedVAA(guardianSet, signers, int(time.time()), nonce, 1, emitter, seq, 32, 0, b)
//    }

  /**
      * Create a packed and signed VAA for testing.
      * See https://github.com/certusone/wormhole/blob/dev.v2/design/0001_generic_message_passing.md
      *
      * @param {} guardianSetIndex  The guardian set index
      * @param {*} signers The list of private keys for signing the VAA
      * @param {*} timestamp The timestamp of VAA
      * @param {*} nonce The nonce.
      * @param {*} emitterChainId The emitter chain identifier
      * @param {*} emitterAddress The emitter chain address, prefixed with 0x
      * @param {*} sequence The sequence.
      * @param {*} consistencyLevel  The reported consistency level
      * @param {*} payload This VAA Payload hex string, prefixed with 0x
      */
    createSignedVAA (guardianSetIndex: number,
                     signers: any,
                     timestamp : number,
                     nonce: number,
                     emitterChainId: number,
                     emitterAddress: string,
                     sequence: number,
                     consistencyLevel: number,
                     payload: string) {

        console.log(typeof payload);

        const body = [
            this.encoder('uint32', timestamp),
            this.encoder('uint32', nonce),
            this.encoder('uint16', emitterChainId),
            this.encoder('bytes32', emitterAddress),
            this.encoder('uint64', sequence),
            this.encoder('uint8', consistencyLevel),
            payload.substring(2)
        ]

        const hash = web3Utils.keccak256(web3Utils.keccak256('0x' + body.join('')))

        let signatures = ''

        for (const i in signers) {
            // eslint-disable-next-line new-cap
            const ec = new elliptic.ec('secp256k1')
            const key = ec.keyFromPrivate(signers[i])
            const signature = key.sign(hash.substr(2), { canonical: true })

            const packSig = [
                this.encoder('uint8', i),
                this.zeroPadBytes(signature.r.toString(16), 32),
                this.zeroPadBytes(signature.s.toString(16), 32),
                this.encoder('uint8', signature.recoveryParam)
            ]

            signatures += packSig.join('')
        }

        const vm = [
            this.encoder('uint8', 1),
            this.encoder('uint32', guardianSetIndex),
            this.encoder('uint8', signers.length),

            signatures,
            body.join('')
        ].join('')

    return vm
  }

  zeroPadBytes (value: string, length: number) {
    while (value.length < 2 * length) {
      value = '0' + value
    }
    return value
  }
}

module.exports = {
    TestLib
}
