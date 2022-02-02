
/**
 *
 * Pricecaster Testing Library.
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
    const body = [
      web3EthAbi.encodeParameter('uint32', timestamp).substring(2 + (64 - 8)),
      web3EthAbi.encodeParameter('uint32', nonce).substring(2 + (64 - 8)),
      web3EthAbi.encodeParameter('uint16', emitterChainId).substring(2 + (64 - 4)),
      web3EthAbi.encodeParameter('bytes32', emitterAddress).substring(2),
      web3EthAbi.encodeParameter('uint64', sequence).substring(2 + (64 - 16)),
      web3EthAbi.encodeParameter('uint8', consistencyLevel).substring(2 + (64 - 2)),
      payload.substr(2)
    ]

    const hash = web3Utils.keccak256(web3Utils.keccak256('0x' + body.join('')))

    // console.log('VAA body Hash: ', hash)

    let signatures = ''

    for (const i in signers) {
      // eslint-disable-next-line new-cap
      const ec = new elliptic.ec('secp256k1')
      const key = ec.keyFromPrivate(signers[i])
      const signature = key.sign(hash.substr(2), { canonical: true })

      const packSig = [
        web3EthAbi.encodeParameter('uint8', i).substring(2 + (64 - 2)),
        this.zeroPadBytes(signature.r.toString(16), 32),
        this.zeroPadBytes(signature.s.toString(16), 32),
        web3EthAbi.encodeParameter('uint8', signature.recoveryParam).substr(2 + (64 - 2))
      ]

      signatures += packSig.join('')
    }

    const vm = [
      web3EthAbi.encodeParameter('uint8', 1).substring(2 + (64 - 2)),
      web3EthAbi.encodeParameter('uint32', guardianSetIndex).substring(2 + (64 - 8)),
      web3EthAbi.encodeParameter('uint8', signers.length).substring(2 + (64 - 2)),

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
