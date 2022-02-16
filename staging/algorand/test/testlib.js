
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
      * Create a packed and signed Pyth VAA for testing.
      * This uses the V2 Pyth VAA (batched prices scheme).
      *
      * @param {} guardianSetIndex  The guardian set index
      * @param {*} signers The list of private keys for signing the VAA
      * @param {*} pythChainId The chainId of Pyth emitter.
      * @param {*} pythEmitterAddress The address of Pyth contract.
      * @param {*} numOfAttest The number of attestations to generate in the payload.
      */
  createSignedPythVAA (guardianSetIndex,
    signers,
    pythChainId,
    pythEmitterAddress,
    numOfAttest) {
    // Payload is:
    //
    // 50325748    P2W_MAGIC (p2wh)
    // 0002        P2W_FORMAT_VERSION
    // nn          Payload ID
    // nnnn        # of Price attestations in this batch
    // nnnn        Size in bytes of each attestation
    //
    // Attestation(s) follows...
    //
    // 50325748       P2W_MAGIC
    // 0002           format version
    // 01             id
    // 230abfe0ec3b460bd55fc4fb36356716329915145497202b8eb8bf1af6a0a3b9      product_id
    // fe650f0367d4a7ef9815a593ea15d36593f0643aaaf0149bb04be67ab851decd      price_id
    // 01                price_type
    // 0000002f17254388  price
    // fffffff7          exponent
    // 0000002eed73d900  twap value
    // 0000000070d3b43f  twap numerator for next upd
    // 0000000037faa03d  twap denom for next upd
    // 000000000e9e5551  twac value
    // 00000000894af11c  twac numerator for next upd
    // 0000000037faa03d  twac denom for next upd
    // 000000000dda6eb8  confidence
    // 01                status
    // 00                corporate_act
    // 0000000061a5ff9a  timestamp (based on Solana contract call time)

    const SAMPLE_PRODUCT_ID = this.hexToBytes('cafeefaccafeefaccafeefaccafeefaccafeefaccafeefaccafeefaccafeefac')
    const SAMPLE_PRICE_ID = this.hexToBytes('deaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddead')
    const SAMPLE_PRICE = this.hexToBytes('0000001100220033')
    const SAMPLE_EXPONENT = this.hexToBytes('fffffff7')
    const SAMPLE_TWAP = this.hexToBytes('1111111111111111')
    const SAMPLE_TWAP_NUM = this.hexToBytes('2222222222222222')
    const SAMPLE_TWAP_DENOM = this.hexToBytes('3333333333333333')
    const SAMPLE_TWAC = this.hexToBytes('4444444444444444')
    const SAMPLE_TWAC_NUM = this.hexToBytes('5555555555555555')
    const SAMPLE_TWAC_DENOM = this.hexToBytes('6666666666666666')
    const SAMPLE_CONFIDENCE = this.hexToBytes('cccccccccccccccc')
    const SAMPLE_STATUS = 0x01
    const SAMPLE_CORP_ACT = 0x00
    const SAMPLE_TIMESTAMP = this.hexToBytes('eeeeeeeeeeeeeeee')

    const payload = []
    payload.push(0x50, 0x32, 0x57, 0x48,
      0x00, 0x02,
      0x01,
      numOfAttest >> 8, numOfAttest & 0xFF,
      0x00, 0x96)

    for (let i = 0; i < numOfAttest; ++i) {
      payload.push(0x50, 0x32, 0x57, 0x48,
        0x00, 0x02,
        0x01,
        SAMPLE_PRODUCT_ID,
        SAMPLE_PRICE_ID,
        0x01,
        SAMPLE_PRICE,
        SAMPLE_EXPONENT, SAMPLE_TWAP, SAMPLE_TWAP_NUM, SAMPLE_TWAP_DENOM,
        SAMPLE_TWAC, SAMPLE_TWAC_NUM, SAMPLE_TWAC_DENOM,
        SAMPLE_CONFIDENCE, SAMPLE_STATUS, SAMPLE_CORP_ACT, SAMPLE_TIMESTAMP)
    }

    const payloadHex = Buffer.from(payload.flat()).toString('hex')
    const body = [
      web3EthAbi.encodeParameter('uint32', (Date.now() & 0xFFFFFFFF).toString()).substring(2 + (64 - 8)),
      web3EthAbi.encodeParameter('uint32', Math.ceil(Math.random() * 999999)).substring(2 + (64 - 8)),
      web3EthAbi.encodeParameter('uint16', pythChainId).substring(2 + (64 - 4)),
      web3EthAbi.encodeParameter('bytes32', pythEmitterAddress).substring(2),
      web3EthAbi.encodeParameter('uint64', Math.ceil(Math.random() * 999999999)).substring(2 + (64 - 16)),
      web3EthAbi.encodeParameter('uint8', 1).substring(2 + (64 - 2)),
      payloadHex
    ]

    const hash = web3Utils.keccak256(web3Utils.keccak256('0x' + body.join('')))

    console.log('VAA body Hash: ', hash)

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

  hexToBytes (hex) {
    // eslint-disable-next-line no-var
    for (var bytes = [], c = 0; c < hex.length; c += 2) { bytes.push(parseInt(hex.substr(c, 2), 16)) }
    return bytes
  }
}

module.exports = {
  TestLib
}
