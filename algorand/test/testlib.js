
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
    constructor() {
        this.zeroBytes = "0000000000000000000000000000000000000000000000000000000000000000";
    }
    encoder(type, val) {
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

    ord(c) {
        return c.charCodeAt(0)
    }

    genGuardianSetUpgrade(signers, guardianSet, targetSet, nonce, seq, guardianKeys) {
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

        var seconds = Math.floor(new Date().getTime() / 1000.0);

//        console.log(b.join(''));
        return this.createSignedVAA(guardianSet, signers, seconds, nonce, 1, emitter, seq, 32, b.join(''))
    }

    genGSetFee( signers, guardianSet, nonce, seq, amt) {
//        b  = self.zeroPadBytes[0:(28*2)]
//        b += self.encoder("uint8", ord("C"))
//        b += self.encoder("uint8", ord("o"))
//        b += self.encoder("uint8", ord("r"))
//        b += self.encoder("uint8", ord("e"))
//        b += self.encoder("uint8", 3)
//        b += self.encoder("uint16", 8)
//        b += self.encoder("uint256", int(amt))  # a whole algo!
//
//        emitter = bytes.fromhex(.zeroPadBytes[0:(31*2)] + "04")
//        return self.createSignedVAA(guardianSet, signers, int(time.time()), nonce, 1, emitter, seq, 32, 0, b)
    }

    genGFeePayout( signers, guardianSet, targetSet, nonce, seq, amt, dest) {
//        b  = self.zeroPadBytes[0:(28*2)]
//        b += self.encoder("uint8", ord("C"))
//        b += self.encoder("uint8", ord("o"))
//        b += self.encoder("uint8", ord("r"))
//        b += self.encoder("uint8", ord("e"))
//        b += self.encoder("uint8", 4)
//        b += self.encoder("uint16", 8)
//        b += self.encoder("uint256", int(amt * 1000000))
//        b += decode_address(dest).hex()
//
//        emitter = bytes.fromhex(.zeroPadBytes[0:(31*2)] + "04")
//        return self.createSignedVAA(guardianSet, signers, int(time.time()), nonce, 1, emitter, seq, 32, 0, b)
    }

    getEmitter( chain) {
//        if chain == 1:
//            return "ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5"
//        if chain == 2:
//            return "0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585"
//        if chain == 3:
//            return "0000000000000000000000007cf7b764e38a0a5e967972c1df77d432510564e2"
//        if chain == 4:
//            return "000000000000000000000000b6f6d86a8f9879a9c87f643768d9efc38c1da6e7"
//        if chain == 5:
//            return "0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde"
        }
        
    genRegisterChain( signers, guardianSet, nonce, seq, chain) {
//        b  = self.zeroPadBytes[0:((32 -11)*2)]
//        b += self.encoder("uint8", ord("T"))
//        b += self.encoder("uint8", ord("o"))
//        b += self.encoder("uint8", ord("k"))
//        b += self.encoder("uint8", ord("e"))
//        b += self.encoder("uint8", ord("n"))
//        b += self.encoder("uint8", ord("B"))
//        b += self.encoder("uint8", ord("r"))
//        b += self.encoder("uint8", ord("i"))
//        b += self.encoder("uint8", ord("d"))
//        b += self.encoder("uint8", ord("g"))
//        b += self.encoder("uint8", ord("e"))
//
//        b += self.encoder("uint8", 1)  # action
//        b += self.encoder("uint16", 0) # target chain
//        b += self.encoder("uint16", chain)
//        b += self.getEmitter(chain)
//        emitter = bytes.fromhex(.zeroPadBytes[0:(31*2)] + "04")
//        return self.createSignedVAA(guardianSet, signers, int(time.time()), nonce, 1, emitter, seq, 32, 0, b)
    }

    genAssetMeta( signers, guardianSet, nonce, seq, tokenAddress, chain, decimals, symbol, name) {
//        b  = self.encoder("uint8", 2)
//        b += self.zeroPadBytes[0:((32-len(tokenAddress))*2)]
//        b += tokenAddress.hex()
//        b += self.encoder("uint16", chain)
//        b += self.encoder("uint8", decimals)
//        b += symbol.hex()
//        b += self.zeroPadBytes[0:((32-len(symbol))*2)]
//        b += name.hex()
//        b += self.zeroPadBytes[0:((32-len(name))*2)]
//        emitter = bytes.fromhex(.getEmitter(chain))
//        return self.createSignedVAA(guardianSet, signers, int(time.time()), nonce, 1, emitter, seq, 32, 0, b)
    }

    genTransfer( signers, guardianSet, nonce, seq, amount, tokenAddress, tokenChain, toAddress, toChain, fee) {
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
    }

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
    createSignedVAA (guardianSetIndex,
                     signers,
                     timestamp,
                     nonce,
                     emitterChainId,
                     emitterAddress,
                     sequence,
                     consistencyLevel,
                     payload) {

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

  zeroPadBytes (value, length) {
    while (value.length < 2 * length) {
      value = '0' + value
    }
    return value
  }

  shuffle (array) {
    let currentIndex = array.length; let randomIndex

    // While there remain elements to shuffle...
    while (currentIndex !== 0) {
      // Pick a remaining element...
      randomIndex = Math.floor(Math.random() * currentIndex)
      currentIndex--;

      // And swap it with the current element.
      [array[currentIndex], array[randomIndex]] = [
        array[randomIndex], array[currentIndex]]
    }

    return array
  }
}

module.exports = {
  TestLib
}
