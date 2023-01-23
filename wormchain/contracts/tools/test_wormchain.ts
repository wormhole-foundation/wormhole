import "dotenv/config";
import * as os from "os"
import { SigningCosmWasmClient, toBinary, ExecuteResult } from "@cosmjs/cosmwasm-stargate";
import { GasPrice } from "@cosmjs/stargate"
import { fromBase64 } from "cosmwasm";
import { MsgExecuteContract } from "cosmjs-types/cosmwasm/wasm/v1/tx";
import { StdFee, Secp256k1HdWallet, coins, Coin } from "@cosmjs/amino";
import { fromHex, toUtf8 } from "@cosmjs/encoding";
import { zeroPad } from "ethers/lib/utils.js";
import { keccak256 } from "@cosmjs/crypto"

import * as elliptic from "elliptic"
import { concatArrays, encodeUint8, signPayload } from "./utils";

import * as devnetConsts from "./devnet-consts.json"



// interface for logs that return from transactions
interface Attribute {
  key: string
  value: string
}
interface Event {
  type: string
  attributes: Array<Attribute>
}
interface Log {
  events: Array<Event>
}


function signBinary(key: elliptic.ec.KeyPair, binary: string): Uint8Array {
  // base64 string to Uint8Array,
  // so we have bytes to work with for signing, though not sure 100% that's correct.
  const bytes = fromBase64(binary);

  // create the "digest" for signing.
  // The contract will calculate the digest of the "data",
  // then use that with the signature to ec recover the publickey that signed.
  const digest = keccak256(keccak256(bytes));

  // sign the digest
  const signature = key.sign(digest, { canonical: true });

  // create 65 byte signature (64 + 1)
  const signedParts = [
    zeroPad(signature.r.toBuffer(), 32),
    zeroPad(signature.s.toBuffer(), 32),
    encodeUint8(signature.recoveryParam || 0),
  ];

  // combine parts to be Uint8Array with length 65
  const signed = concatArrays(signedParts);

  return signed
}


async function main() {

  /* Set up cosmos client & wallet */

  const WORMCHAIN_ID = 3104

  let host = devnetConsts.chains[3104].tendermintUrlLocal
  if (os.hostname().includes("wormchain-deploy")) {
    // running in tilt devnet
    host = devnetConsts.chains[3104].tendermintUrlTilt
  }
  const denom = devnetConsts.chains[WORMCHAIN_ID].addresses.native.denom
  const mnemonic = devnetConsts.chains[WORMCHAIN_ID].accounts.wormchainNodeOfGuardian0.mnemonic
  const addressPrefix = "wormhole"
  const signerPk = devnetConsts.devnetGuardians[0].private
  const guardianKeysCSV = String(process.env.INIT_SIGNERS_KEYS_CSV)
  if (!guardianKeysCSV || guardianKeysCSV === "undefined") {
    throw new Error("could not find process.env.process.env.INIT_SIGNERS_KEYS_CSV, exiting.")
  }
  const guardianKeys = guardianKeysCSV.split(",")


  const tokenBridgeAddress = devnetConsts.chains[WORMCHAIN_ID].contracts.tokenBridgeNativeAddress

  const accountingAddress = devnetConsts.chains[WORMCHAIN_ID].contracts.accountingNativeAddress
  const mockTokenAddress = devnetConsts.chains[WORMCHAIN_ID].addresses.testToken.address

  const transferChain = 2

  const transferAmount = "3000000"
  const transferRelayFee = "0"
  const transferRecipient = "0000000000000000000000004206942069420694206942069420694206942069"

  const foreignAssetAddr = "0x2D8BE6BF0baA74e0A907016679CaE9190e80dD0A"
  const foreignAssetAddrHex = "0000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a"
  const foreignAssetRegisterVaa = "01000000000100efccb8a5d54162691095c88369b873d93ed4ba9365ed0f94adcf39743bb034be56ce0a94d9b163cb0c63be24b94e31b75e1a5889e3de03db147278d9d3eb7d260100000992abb6000000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16000000000000000d0f020000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a000212544b4e0000000000000000000000000000000000000000000000000000000000457468657265756d205465737420546f6b656e00000000000000000000000000"
  const foreignAssetUpdateVaa = "01000000000100d1389731568d9816267accba90abb9db37dcd09738750fae421067f2f7f33f014c2d862e288e9a3149e2d8bcd2e53ffe2ed72dfc5e8eb50c740a0df34c60103f01000011ac2076010000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16000000000000000e0f020000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a000212544b4e0000000000000000000000000000000000000000000000000000000000457468657265756d205465737420546f6b656e00000000000000000000000000";
  const ethTokenBridge = "0000000000000000000000000290FB167208Af455bB137780163b7B7a9a10C16"
  const ethChainId = 2


  const redeemCW20TokenPayload = "01000000000100938620996f3161182d6b3767a26c2e7e0d38e3a4014a4d2ce5aefb55e3e975e9276717520d5567d761af8f7e93e1372cd0b84501c90d008f46fb3ce4de4aeb1f00000000000000000000020000000000000000000000000290FB167208Af455bB137780163b7B7a9a10C160000000000000000000100000000000000000000000000000000000000000000000000000000000f4240003f822e9066cfea09b9ce1247e8f79a86a24dda2d8b3d76a608ae75832204110c20000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d0c200000000000000000000000000000000000000000000000000000000000000000"
  const redeemCW20Token = signPayload(
    ethChainId,
    ethTokenBridge,
    guardianKeys,
    redeemCW20TokenPayload
  )
  console.log({ redeemCW20Token })

  const redeemWormTokenPayload = "010000000001004b953d51b2d7219c72687ffb048988fe5f2b882c7d8d2be510bc615f4013fb8b5149fc8c297f077a3c2f2379cbaeb6fd19ae92e53e29885fc9a1252018e22d3200000000000000000000020000000000000000000000000290FB167208Af455bB137780163b7B7a9a10C160000000000000000000100000000000000000000000000000000000000000000000000000000000f4240010c0ded78f1b69ec7b79b9ee592fbbcacebc97db1c695220a833135bfa748240c20000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d0c200000000000000000000000000000000000000000000000000000000000000000"
  const redeemWormToken = signPayload(
    ethChainId,
    ethTokenBridge,
    guardianKeys,
    redeemWormTokenPayload,
  )

  // this VAA could be used to test redeeming a VAA for more tokens than are locked in the contract (10 000 000).
  const redeem10WormToken = "01000000000100c190e93fe411b69a7869fef9abe3d7399ce403e7555165b123ea36ceebee57144b9b145436aab4c2f5afcef74cc94f06a6d50dc6046aaedb13564d0c4f55ed9201000000000000000000020000000000000000000000000290FB167208Af455bB137780163b7B7a9a10C16000000000000000000010000000000000000000000000000000000000000000000000000000000989680010c0ded78f1b69ec7b79b9ee592fbbcacebc97db1c695220a833135bfa748240c20000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d0c200000000000000000000000000000000000000000000000000000000000000000"

  const attestERC20TokenPayload = "01000000000100efccb8a5d54162691095c88369b873d93ed4ba9365ed0f94adcf39743bb034be56ce0a94d9b163cb0c63be24b94e31b75e1a5889e3de03db147278d9d3eb7d260100000992abb6000000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16000000000000000d0f020000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a000212544b4e0000000000000000000000000000000000000000000000000000000000457468657265756d205465737420546f6b656e00000000000000000000000000"
  const attestERC20Token = signPayload(
    ethChainId,
    ethTokenBridge,
    guardianKeys,
    attestERC20TokenPayload
  )

  // another attest message for the same token
  const attestERC20TokenUpdate = "01000000000100d1389731568d9816267accba90abb9db37dcd09738750fae421067f2f7f33f014c2d862e288e9a3149e2d8bcd2e53ffe2ed72dfc5e8eb50c740a0df34c60103f01000011ac2076010000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16000000000000000e0f020000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a000212544b4e0000000000000000000000000000000000000000000000000000000000457468657265756d205465737420546f6b656e00000000000000000000000000"
  const redeemERC20TokenPayload = "0100000000010006aedc647506dfffbe54b18bd346d817dac709db418b29dfc3e8322d745970123b8162ce9be73a737e95e2932929570f80ca7abcdc350c1b6464a3689634170c01000000000000000000020000000000000000000000000290FB167208Af455bB137780163b7B7a9a10C16000000000000000000010000000000000000000000000000000000000000000000000de0b6b3a76400000000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a0002000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d0c200000000000000000000000000000000000000000000000000000000000000000"
  const redeemERC20Token = signPayload(
    ethChainId,
    ethTokenBridge,
    guardianKeys,
    redeemERC20TokenPayload
  )

  const w = await Secp256k1HdWallet.fromMnemonic(mnemonic, { prefix: addressPrefix })

  const gas = GasPrice.fromString(`0${denom}`)
  let cwc = await SigningCosmWasmClient.connectWithSigner(host, w, { prefix: addressPrefix, gasPrice: gas })

  // there is no danger here, just several Cosmos chains in devnet, so check for config issues
  let id = await cwc.getChainId()
  if (id !== "wormchain") {
    throw new Error(`Wormchain CosmWasmClient connection produced an unexpected chainID: ${id}`)
  }

  const signers = await w.getAccounts()
  const signer = signers[0].address
  console.log("wormchain wallet pubkey: ", signer)


  const createSender = (
    client: SigningCosmWasmClient,
    signerAddress: string,
    contractAddress: string,
    fee: number | StdFee | "auto") =>
    (message: object, funds?: Coin[]): Promise<ExecuteResult> =>
      client.execute(signerAddress, contractAddress, message, fee, undefined, funds)

  const tbSender = createSender(cwc, signer, tokenBridgeAddress, "auto")


  const nativeBalance = await cwc.getBalance(signer, denom)
  console.log("nativeBalance ", nativeBalance.amount)

  const utestBalance = await cwc.getBalance(signer, "utest")
  console.log("utest balance ", utestBalance.amount)


  const registerNativeAssetMsg = {
    create_asset_meta: {
      asset_info: {
        native_token: {
          denom
        }
      },
      nonce: 1
    }
  }
  const registerNativeAssetRes = await tbSender(registerNativeAssetMsg)
  console.log("registerNativeAssetRes ", registerNativeAssetRes.transactionHash)


  const nativeDepositRes = await tbSender(
    { deposit_tokens: {} },
    coins(transferAmount, denom))
  console.log("nativeDepositRes ", nativeDepositRes.transactionHash)

  const nativeTransferRes = await tbSender({
    initiate_transfer: {
      asset: {
        amount: transferAmount,
        info: {
          native_token: { denom }
        }
      },
      fee: transferRelayFee,
      nonce: 1,
      recipient: Buffer.from(transferRecipient, 'hex').toString('base64'),
      recipient_chain: transferChain,
    }
  })
  console.log("nativeTransferRes ", nativeTransferRes.transactionHash)




  // CW20 transfer out of WC
  const registerCW20AssetMsg = {
    create_asset_meta: {
      asset_info: {
        token: {
          contract_addr: mockTokenAddress
        }
      },
      nonce: 1
    }
  }
  const registerCW20AssetRes = await tbSender(registerCW20AssetMsg)
  console.log("registerCW20AssetRes ", registerCW20AssetRes.transactionHash)


  const cw20AllowanceRes = await cwc.execute(signer, mockTokenAddress, {
    increase_allowance: {
      amount: transferAmount,
      spender: tokenBridgeAddress
    }
  }, "auto")
  console.log("cw20AllowanceRes ", cw20AllowanceRes.transactionHash)

  const preCW20TransferBalance = await cwc.queryContractSmart(mockTokenAddress, {
    balance: {
      address: signer
    }
  })
  console.log("preCW20TransferBalance.balance ", preCW20TransferBalance.balance)
  const cw20TransferRes = await tbSender({
    initiate_transfer: {
      asset: {
        amount: transferAmount,
        info: {
          token: {
            contract_addr: mockTokenAddress
          }
        }
      },
      fee: transferRelayFee,
      nonce: 1,
      recipient: Buffer.from(transferRecipient, 'hex').toString('base64'),
      recipient_chain: transferChain,
    }
  })
  console.log("cw20TransferRes ", cw20TransferRes.transactionHash)

  const postCW20Balance = await cwc.queryContractSmart(mockTokenAddress, {
    balance: {
      address: signer
    }
  })
  console.log("postCW20Balance.balance ", postCW20Balance.balance)


  // Redeem an ERC20 transfer from from Eth
  const registerForeignAssetRes = await tbSender({
    submit_vaa: {
      data: Buffer.from(attestERC20Token, "hex").toString("base64")
    }
  })


  const { logs } = registerForeignAssetRes
  const { events } = logs[0]
  const ExternalTokenId = events.reduce((accum, event) => {
    if (event.type === "wasm") {
      const found = event.attributes.find((item) => item.key === "token_address")
      return found?.value ? found.value : accum
    }
    return accum
  }, String())
  console.log("wrappedForeign ExternalTokenId ", ExternalTokenId)

  const wrappedForeignRes = await cwc.queryContractSmart(tokenBridgeAddress, {
    wrapped_registry: {
      chain: transferChain,
      address: Buffer.from(foreignAssetAddrHex, "hex").toString('base64')
    }
  })
  console.log("wrappedForeignRes: ", wrappedForeignRes)

  const preRedeemBalance = await cwc.queryContractSmart(wrappedForeignRes.address, { balance: { address: signer } })
  console.log("preRedeemBalance.balance ", preRedeemBalance.balance)

  const redeemRes = await tbSender({
    submit_vaa: {
      data: Buffer.from(redeemERC20Token, "hex").toString("base64")
    }
  })
  console.log("redeemRes ", redeemRes.transactionHash)

  const postRedeemBalance = await cwc.queryContractSmart(wrappedForeignRes.address, { balance: { address: signer } })
  console.log("postRedeemBalance.balance ", postRedeemBalance.balance)



  // redeem native uworm returning from Eth
  const preRedeemWormBalance = await cwc.getBalance(signer, denom)
  console.log("preRedeemWormBalance.balance ", preRedeemWormBalance.amount)

  const redeemWormRes = await tbSender({
    submit_vaa: {
      data: Buffer.from(redeemWormToken, "hex").toString("base64")
    }
  })
  console.log("redeemWormRes ", redeemWormRes.transactionHash)

  const postRedeemWormBalance = await cwc.getBalance(signer, denom)
  console.log("postRedeemWormBalance.balance ", postRedeemWormBalance.amount)



  // redeem a CW20 returning from Eth
  const preRedeemCW20Balance = await cwc.queryContractSmart(mockTokenAddress, { balance: { address: signer } })
  console.log("preRedeemCW20Balance.balance ", preRedeemCW20Balance.balance)

  const redeemCW20Res = await tbSender({
    submit_vaa: {
      data: Buffer.from(redeemCW20Token, "hex").toString("base64")
    }
  })
  console.log("redeemCW20Res ", redeemCW20Res.transactionHash)

  const postRedeemCW20Balance = await cwc.queryContractSmart(mockTokenAddress, { balance: { address: signer } })
  console.log("postRedeemCW20Balance.balance ", postRedeemCW20Balance.balance)


  // create key for guardian0
  const ec = new elliptic.ec("secp256k1");
  // create key from the devnet guardian0's private key
  const key = ec.keyFromPrivate(Buffer.from(signerPk, "hex"));


  // Test empty observation

  // object to json string, then to base64 (serde binary)
  const arrayBinaryString = toBinary([]);

  // combine parts to be Uint8Array with length 65
  const signedEmptyArray = signBinary(key, arrayBinaryString)

  const observeEmptyArray = {
    submit_observations: {
      observations: arrayBinaryString,
      guardian_set_index: 0,
      signature: {
        index: 0,
        signature: Array.from(signedEmptyArray),
      },
    },
  };

  let emptyArrayObsRes = await cwc.execute(signer, accountingAddress, observeEmptyArray, "auto");
  console.log(`emptyArrayObsRes.transactionHash: ${emptyArrayObsRes.transactionHash}`);


  // Test (fake) observation
  const emitter_address = "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
  const observations = [
    {
      emitter_chain: 2,
      emitter_address: emitter_address,
      sequence: 2,
      nonce: 1,
      consistency_level: 0,
      timestamp: 1,
      payload:
        Buffer.from("030000000000000000000000000000000000000000000000000000000005f5e1000000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a0002000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d0c2000000000000000000000000000000000000000000000000000000000000f4240", "hex").toString("base64"),

      tx_hash:
        Buffer.from("9fc68fb0ee735d45c9074a20adef1747b0593803f33b9f3f2252c8e2df567f41", "hex").toString("base64")
    },
  ];

  // object to json string, then to base64 (serde binary)
  const observationsBinaryString = toBinary(observations);

  const signed = signBinary(key, observationsBinaryString)

  const executeMsg = {
    submit_observations: {
      observations: observationsBinaryString,
      guardian_set_index: 0,
      signature: {
        index: 0,
        signature: Array.from(signed),
      },
    },
  };
  console.log(executeMsg);

  let inst = await cwc.execute(
    signer,
    accountingAddress,
    executeMsg,
    "auto"
  );
  let txHash = inst.transactionHash;
  console.log(`executed submit_observation! txHash: ${txHash}`);



  console.log("done, exiting success.")
}

try {
  main()
} catch (e) {
  console.error(e)
  throw e
}
