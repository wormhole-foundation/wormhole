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

import {
  ChainId,
  CHAIN_ID_AVAX,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_FANTOM,
  CHAIN_ID_OASIS,
  CHAIN_ID_POLYGON,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
} from "../utils";

const web3EthAbi = require("web3-eth-abi");
const web3Utils = require("web3-utils");
const elliptic = require("elliptic");

export class TestLib {
  zeroBytes: string;

  singleGuardianKey: string[] = ["beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"];

  singleGuardianPrivKey: string[] = [
    "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0",
  ];

  constructor() {
    this.zeroBytes =
      "0000000000000000000000000000000000000000000000000000000000000000";
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

  guardianKeys: string[] = [
    "52A26Ce40F8CAa8D36155d37ef0D5D783fc614d2",
    "389A74E8FFa224aeAD0778c786163a7A2150768C",
    "B4459EA6482D4aE574305B239B4f2264239e7599",
    "072491bd66F63356090C11Aae8114F5372aBf12B",
    "51280eA1fd2B0A1c76Ae29a7d54dda68860A2bfF",
    "fa9Aa60CfF05e20E2CcAA784eE89A0A16C2057CB",
    "e42d59F8FCd86a1c5c4bA351bD251A5c5B05DF6A",
    "4B07fF9D5cE1A6ed58b6e9e7d6974d1baBEc087e",
    "c8306B84235D7b0478c61783C50F990bfC44cFc0",
    "C8C1035110a13fe788259A4148F871b52bAbcb1B",
    "58A2508A20A7198E131503ce26bBE119aA8c62b2",
    "8390820f04ddA22AFe03be1c3bb10f4ba6CF94A0",
    "1FD6e97387C34a1F36DE0f8341E9D409E06ec45b",
    "255a41fC2792209CB998A8287204D40996df9E54",
    "bA663B12DD23fbF4FbAC618Be140727986B3BBd0",
    "79040E577aC50486d0F6930e160A5C75FD1203C6",
    "3580D2F00309A9A85efFAf02564Fc183C0183A96",
    "3869795913D3B6dBF3B24a1C7654672c69A23c35",
    "1c0Cc52D7673c52DE99785741344662F5b2308a0",
  ];

  guardianPrivKeys: string[] = [
    "563d8d2fd4e701901d3846dee7ae7a92c18f1975195264d676f8407ac5976757",
    "8d97f25916a755df1d9ef74eb4dbebc5f868cb07830527731e94478cdc2b9d5f",
    "9bd728ad7617c05c31382053b57658d4a8125684c0098f740a054d87ddc0e93b",
    "5a02c4cd110d20a83a7ce8d1a2b2ae5df252b4e5f6781c7855db5cc28ed2d1b4",
    "93d4e3b443bf11f99a00901222c032bd5f63cf73fc1bcfa40829824d121be9b2",
    "ea40e40c63c6ff155230da64a2c44fcd1f1c9e50cacb752c230f77771ce1d856",
    "87eaabe9c27a82198e618bca20f48f9679c0f239948dbd094005e262da33fe6a",
    "61ffed2bff38648a6d36d6ed560b741b1ca53d45391441124f27e1e48ca04770",
    "bd12a242c6da318fef8f98002efb98efbf434218a78730a197d981bebaee826e",
    "20d3597bb16525b6d09e5fb56feb91b053d961ab156f4807e37d980f50e71aff",
    "344b313ffbc0199ff6ca08cacdaf5dc1d85221e2f2dc156a84245bd49b981673",
    "848b93264edd3f1a521274ca4da4632989eb5303fd15b14e5ec6bcaa91172b05",
    "c6f2046c1e6c172497fc23bd362104e2f4460d0f61984938fa16ef43f27d93f6",
    "693b256b1ee6b6fb353ba23274280e7166ab3be8c23c203cc76d716ba4bc32bf",
    "13c41508c0da03018d61427910b9922345ced25e2bbce50652e939ee6e5ea56d",
    "460ee0ee403be7a4f1eb1c63dd1edaa815fbaa6cf0cf2344dcba4a8acf9aca74",
    "b25148579b99b18c8994b0b86e4dd586975a78fa6e7ad6ec89478d7fbafd2683",
    "90d7ac6a82166c908b8cf1b352f3c9340a8d1f2907d7146fb7cd6354a5436cca",
    "b71d23908e4cf5d6cd973394f3a4b6b164eb1065785feee612efdfd8d30005ed",
  ];

  encoder(type: string, val: any) {
    if (type == "uint8")
      return web3EthAbi.encodeParameter("uint8", val).substring(2 + (64 - 2));
    if (type == "uint16")
      return web3EthAbi.encodeParameter("uint16", val).substring(2 + (64 - 4));
    if (type == "uint32")
      return web3EthAbi.encodeParameter("uint32", val).substring(2 + (64 - 8));
    if (type == "uint64")
      return web3EthAbi.encodeParameter("uint64", val).substring(2 + (64 - 16));
    if (type == "uint128")
      return web3EthAbi
        .encodeParameter("uint128", val)
        .substring(2 + (64 - 32));
    if (type == "uint256" || type == "bytes32")
      return web3EthAbi.encodeParameter(type, val).substring(2 + (64 - 64));
  }

  ord(c: any) {
    return c.charCodeAt(0);
  }

  genGuardianSetUpgrade(
    signers: any,
    guardianSet: number,
    targetSet: number,
    nonce: number,
    seq: number,
    guardianKeys: string[]
  ): string {
    const b = [
      "0x",
      this.zeroBytes.slice(0, 28 * 2),
      this.encoder("uint8", this.ord("C")),
      this.encoder("uint8", this.ord("o")),
      this.encoder("uint8", this.ord("r")),
      this.encoder("uint8", this.ord("e")),
      this.encoder("uint8", 2),
      this.encoder("uint16", 0),
      this.encoder("uint32", targetSet),
      this.encoder("uint8", guardianKeys.length),
    ];

    guardianKeys.forEach((x) => {
      b.push(x);
    });

    let emitter = "0x" + this.zeroBytes.slice(0, 31 * 2) + "04";
    let seconds = Math.floor(new Date().getTime() / 1000.0);

    return this.createSignedVAA(
      guardianSet,
      signers,
      seconds,
      nonce,
      1,
      emitter,
      seq,
      32,
      b.join("")
    );
  }

  genGSetFee(
    signers: any,
    guardianSet: number,
    nonce: number,
    seq: number,
    amt: number
  ) {
    const b = [
      "0x",
      this.zeroBytes.slice(0, 28 * 2),
      this.encoder("uint8", this.ord("C")),
      this.encoder("uint8", this.ord("o")),
      this.encoder("uint8", this.ord("r")),
      this.encoder("uint8", this.ord("e")),
      this.encoder("uint8", 3),
      this.encoder("uint16", 8),
      this.encoder("uint256", Math.floor(amt)),
    ];

    let emitter = "0x" + this.zeroBytes.slice(0, 31 * 2) + "04";

    var seconds = Math.floor(new Date().getTime() / 1000.0);

    return this.createSignedVAA(
      guardianSet,
      signers,
      seconds,
      nonce,
      1,
      emitter,
      seq,
      32,
      b.join("")
    );
  }

  genGFeePayout(
    signers: any,
    guardianSet: number,
    nonce: number,
    seq: number,
    amt: number,
    dest: Uint8Array
  ) {
    const b = [
      "0x",
      this.zeroBytes.slice(0, 28 * 2),
      this.encoder("uint8", this.ord("C")),
      this.encoder("uint8", this.ord("o")),
      this.encoder("uint8", this.ord("r")),
      this.encoder("uint8", this.ord("e")),
      this.encoder("uint8", 4),
      this.encoder("uint16", 8),
      this.encoder("uint256", Math.floor(amt)),
      this.uint8ArrayToHexString(dest, false),
    ];

    let emitter = "0x" + this.zeroBytes.slice(0, 31 * 2) + "04";

    var seconds = Math.floor(new Date().getTime() / 1000.0);

    return this.createSignedVAA(
      guardianSet,
      signers,
      seconds,
      nonce,
      1,
      emitter,
      seq,
      32,
      b.join("")
    );
  }

  getTokenEmitter(chain: number): string {
    if (chain === CHAIN_ID_SOLANA) {
      return "ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5";
    }
    if (chain === CHAIN_ID_ETH) {
      return "0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585";
    }
    if (chain === CHAIN_ID_TERRA) {
      return "0000000000000000000000007cf7b764e38a0a5e967972c1df77d432510564e2";
    }
    if (chain === CHAIN_ID_BSC) {
      return "000000000000000000000000b6f6d86a8f9879a9c87f643768d9efc38c1da6e7";
    }
    if (chain === CHAIN_ID_POLYGON) {
      return "0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde";
    }
    if (chain === CHAIN_ID_AVAX) {
      return "0000000000000000000000000e082f06ff657d94310cb8ce8b0d9a04541d8052";
    }
    if (chain === CHAIN_ID_OASIS) {
      return "0000000000000000000000005848c791e09901b40a9ef749f2a6735b418d7564";
    }
    if (chain === CHAIN_ID_FANTOM) {
      return "0000000000000000000000007C9Fc5741288cDFdD83CeB07f3ea7e22618D79D2";
    }
    return "";
  }

  getNftEmitter(chain: ChainId): string {
    if (chain === CHAIN_ID_SOLANA) {
      return "0def15a24423e1edd1a5ab16f557b9060303ddbab8c803d2ee48f4b78a1cfd6b";
    }
    if (chain === CHAIN_ID_ETH) {
      return "0000000000000000000000006ffd7ede62328b3af38fcd61461bbfc52f5651fe";
    }
    if (chain === CHAIN_ID_BSC) {
      return "0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde";
    }
    if (chain === CHAIN_ID_POLYGON) {
      return "00000000000000000000000090bbd86a6fe93d3bc3ed6335935447e75fab7fcf";
    }
    if (chain === CHAIN_ID_AVAX) {
      return "000000000000000000000000f7b6737ca9c4e08ae573f75a97b73d7a813f5de5";
    }
    if (chain === CHAIN_ID_OASIS) {
      return "00000000000000000000000004952D522Ff217f40B5Ef3cbF659EcA7b952a6c1";
    }
    if (chain === CHAIN_ID_FANTOM) {
      return "000000000000000000000000A9c7119aBDa80d4a4E0C06C8F4d8cF5893234535";
    }
    return "";
  }

  genRegisterChain(
    signers: any,
    guardianSet: number,
    nonce: number,
    seq: number,
    chain: string
  ) {
    const b = [
      "0x",
      this.zeroBytes.slice(0, (32 - 11) * 2),
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
      this.getTokenEmitter(parseInt(chain)),
    ];
    let emitter = "0x" + this.zeroBytes.slice(0, 31 * 2) + "04";

    var seconds = Math.floor(new Date().getTime() / 1000.0);

    return this.createSignedVAA(
      guardianSet,
      signers,
      seconds,
      nonce,
      1,
      emitter,
      seq,
      32,
      b.join("")
    );
  }

  genAssetMeta(
    signers: any,
    guardianSet: number,
    nonce: number,
    seq: number,
    tokenAddress: string,
    chain: number,
    decimals: number,
    symbol: string,
    name: string
  ) {
    const b = [
      "0x",
      this.encoder("uint8", 2),
      this.zeroBytes.slice(0, 64 - tokenAddress.length),
      tokenAddress,
      this.encoder("uint16", chain),
      this.encoder("uint8", decimals),
      Buffer.from(symbol).toString("hex"),
      this.zeroBytes.slice(0, (32 - symbol.length) * 2),
      Buffer.from(name).toString("hex"),
      this.zeroBytes.slice(0, (32 - name.length) * 2),
    ];

    let emitter = "0x" + this.getTokenEmitter(chain);
    let seconds = Math.floor(new Date().getTime() / 1000.0);

    return this.createSignedVAA(
      guardianSet,
      signers,
      seconds,
      nonce,
      chain,
      emitter,
      seq,
      32,
      b.join("")
    );
  }

  genTransfer(
    signers: any,
    guardianSet: number,
    nonce: number,
    seq: number,
    amount: number,
    tokenAddress: string,
    tokenChain: number,
    toAddress: string,
    toChain: number,
    fee: number
  ) {
    const b = [
      "0x",
      this.encoder("uint8", 1),
      this.encoder("uint256", Math.floor(amount * 100000000)),
      this.zeroBytes.slice(0, 64 - tokenAddress.length),
      tokenAddress,
      this.encoder("uint16", tokenChain),
      this.zeroBytes.slice(0, 64 - toAddress.length),
      toAddress,
      this.encoder("uint16", toChain),
      this.encoder("uint256", Math.floor(fee * 100000000)),
    ];

    let emitter = "0x" + this.getTokenEmitter(tokenChain);
    let seconds = Math.floor(new Date().getTime() / 1000.0);

    return this.createSignedVAA(
      guardianSet,
      signers,
      seconds,
      nonce,
      tokenChain,
      emitter,
      seq,
      32,
      b.join("")
    );
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
  createSignedVAA(
    guardianSetIndex: number,
    signers: any,
    timestamp: number,
    nonce: number,
    emitterChainId: number,
    emitterAddress: string,
    sequence: number,
    consistencyLevel: number,
    payload: string
  ) {
    const body = [
      this.encoder("uint32", timestamp),
      this.encoder("uint32", nonce),
      this.encoder("uint16", emitterChainId),
      this.encoder("bytes32", emitterAddress),
      this.encoder("uint64", sequence),
      this.encoder("uint8", consistencyLevel),
      payload.substring(2),
    ];

    const hash = web3Utils.keccak256(web3Utils.keccak256("0x" + body.join("")));

    let signatures = "";

    for (const i in signers) {
      // eslint-disable-next-line new-cap
      const ec = new elliptic.ec("secp256k1");
      const key = ec.keyFromPrivate(signers[i]);
      const signature = key.sign(hash.substr(2), { canonical: true });

      const packSig = [
        this.encoder("uint8", i),
        this.zeroPadBytes(signature.r.toString(16), 32),
        this.zeroPadBytes(signature.s.toString(16), 32),
        this.encoder("uint8", signature.recoveryParam),
      ];

      signatures += packSig.join("");
    }

    const vm = [
      this.encoder("uint8", 1),
      this.encoder("uint32", guardianSetIndex),
      this.encoder("uint8", signers.length),

      signatures,
      body.join(""),
    ].join("");

    return vm;
  }

  zeroPadBytes(value: string, length: number) {
    while (value.length < 2 * length) {
      value = "0" + value;
    }
    return value;
  }
}

module.exports = {
  TestLib,
};
