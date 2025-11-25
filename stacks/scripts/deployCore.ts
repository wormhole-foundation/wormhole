import { StacksNetworks, type StacksNetworkName } from "@stacks/network";
import path from "path";
import fs from "fs";
import { broadcastTransaction, Cl, fetchFeeEstimateTransaction, fetchNonce, makeContractCall, makeContractDeploy, serializePayloadBytes } from '@stacks/transactions';
import { getKeys, waitForTransactionSuccess } from './utils';
import { createVAA, secp256k1, serialize, UniversalAddress, type VAA } from '@wormhole-foundation/sdk';
import { mocks } from "@wormhole-foundation/sdk-definitions/testing";
import { bytesToHex } from '@stacks/common';

/**
 * .env configuration:
 * 
 * - STACKS_API_URL: Stacks node URL, for example: https://api.testnet.hiro.so
 * - DEPLOYER_MNEMONIC: Deployer mnemonic (only one of DEPLOYER_MNEMONIC or DEPLOYER_PRIVATE_KEY must be set)
 * - DEPLOYER_ACCOUNT_INDEX: Deployer account index, for example: 0 for the first account (only used with DEPLOYER_MNEMONIC)
 * - DEPLOYER_PRIVATE_KEY: Deployer private key (only one of DEPLOYER_MNEMONIC or DEPLOYER_PRIVATE_KEY must be set)
 * - NETWORK_NAME: Stacks network name, for example: testnet. Valid values: mainnet, testnet, devnet, mocknet
 */

interface Guardian {
  index: number;
  address: string;
  pubKey: string;
}

const guardians: Record<StacksNetworkName, Guardian[]> = {
  devnet: [/** fill if needed */],
  mocknet: [/** fill if needed */],
  testnet: [{
    index: 0,
    address: "0x13947Bd48b18E53fdAeEe77F3473391aC727C638",
    pubKey: "0x04fa9d6b47043b15b4b33cf05bb1de0f13da703136b2cd157324eb615ec9ee951238c67b54d03e3c182e0e77b85f883ccd74dddfc59010fbf475a0bee6593ce2bd"
  }],
  mainnet: [
    {
      index: 0,
      address: "0x5893B5A76c3f739645648885bDCcC06cd70a3Cd3",
      pubKey: "0x049a1e801daa25d9808e70aae9981353086f958955cc94ef33a461b0e596feaef90a8474dd10cf6ae967143f86105c16d6304a3d268ea952fda9389139d4bb9da1",
    },
    {
      index: 1,
      address: "0xfF6CB952589BDE862c25Ef4392132fb9D4A42157",
      pubKey: "0x042766db08820e311b22e109801ab8ea505b12e3df3d91ebc87c999ffb6929d1abb0ade987c74aa37db26eea4086ee738a2f34a5594edb8760da0eac5be356b731",
    },
    {
      index: 2,
      address: "0x114De8460193bdf3A2fCf81f86a09765F4762fD1",
      pubKey: "0x0454177ff4a8329520b76efd86f8bfce5c942554db16e673267dc1133b3f5e230b2d8cbf90fe274946045d4491de288d736680edc2ee9ee5b1b15416b0a34806c4",
    },
    {
      index: 3,
      address: "0x107A0086b32d7A0977926A205131d8731D39cbEB",
      pubKey: "0x047fa3e98fcc2621337b217b61408a98facaabd25bad2b158438728ce863c14708cfcda1f3b50a16ca0211199079fb338d479a54546ec3c5f775af23a7d7f4fb24",
    },
    {
      index: 4,
      address: "0x8C82B2fd82FaeD2711d59AF0F2499D16e726f6b2",
      pubKey: "0x040bdcbccc0297c2a4f92a7c39358c42f22a8ed700a78bd05c39c8b61aaf2338e825b6c0d26d1f2a2ae4129cd751201f73d7234c753bd0735212a5288b19748fd2",
    },
    {
      index: 5,
      address: "0x11b39756C042441BE6D8650b69b54EbE715E2343",
      pubKey: "0x04cfd90084be68de514fe14a7c281f492223f045566f859ea5c166d6e60bc650c23940909a8e96c2fbffbd15a598b4e6a5b5aa14c126bf58cc1a9e396fe7771965",
    },
    {
      index: 6,
      address: "0x54Ce5B4D348fb74B958e8966e2ec3dBd4958a7cd",
      pubKey: "0x048edf3f9d997357a0e2c916ee090392c3a645ebac4f6cd8f826d3ecc0173b33bf06b7c14e8002fc9a5d01af9824a5cb3778472cd477e0ab378091448bca6f0417",
    },
    {
      index: 7,
      address: "0x15e7cAF07C4e3DC8e7C469f92C8Cd88FB8005a20",
      pubKey: "0x0447b15c5039dcb2850b59bea323db662cc597dd7d48fe6b8dbb6cd8704c45854bf0e92fa267c844ba1a700105e157c8099d55c82316cb5e50c56a5d0920ff91c2",
    },
    {
      index: 8,
      address: "0x74a3bf913953D695260D88BC1aA25A4eeE363ef0",
      pubKey: "0x04d5225476d7849b362226952ffc561bab99832f3f8b99741f6d81bbeaffa8e7f6e54a85e5029a3b510707eaa9684df496e4b1268075ad0328693a30bf1b1e0033",
    },
    {
      index: 9,
      address: "0x000aC0076727b35FBea2dAc28fEE5cCB0fEA768e",
      pubKey: "0x04d9fa78b5b958bea1929080b8ad96dc555d34b051a27aebf711eb1186b807b0448316d994606ac807121838d6c41a58f308bc6307acdf69491fa4b17282f3e66f",
    },
    {
      index: 10,
      address: "0xAF45Ced136b9D9e24903464AE889F5C8a723FC14",
      pubKey: "0x04cc64af75ec2e2741fb9af9f6191cb9ee187d6d26af4d1e96d7bab47e6ec09be12d3192030dc4bbf54d1da319a7a2acfc7a9dd4c644af6646a4aaa02b1024bbab",
    },
    {
      index: 12,
      address: "0xD2CC37A4dc036a8D232b48f62cDD4731412f4890",
      pubKey: "0x040cfc9d5b5dcf702a1525f9d4ed1841e8eb8b34434cc82470dd35435f1dbdc73ffb51544b7500394eac9c7fa567868b495326075147a2d809ebbfd43273eeec91",
    },
    {
      index: 13,
      address: "0xDA798F6896A3331F64b48c12D1D57Fd9cbe70811",
      pubKey: "0x040aa78894d894a15933969f5826347439e2c309f2049277a10066c9197840499498ad19ee3d1b291f932ec0890bbdafcec292c4f02a446670cd0084f997e25e2f",
    },
    {
      index: 14,
      address: "0x71AA1BE1D36CaFE3867910F99C09e347899C19C3",
      pubKey: "0x0400f400e3fe40f64032485aad9240ead45a8e1fc83ec08c96db861c0eca155ac898df8673e778e3ccaae8a0f9e6af415fe40e99b0cbc88d7610e536b6041b07fb",
    },
    {
      index: 15,
      address: "0x8192b6E7387CCd768277c17DAb1b7a5027c0b3Cf",
      pubKey: "0x04604f384174c7ed3a0dc5f476569a978266a7943bd775449d1b8b27f4eb8beb99cdf095f9200a2dabb1bc5d68c3d96ea3d47f4d34499d59953669b6c8c093d578",
    },
    {
      index: 16,
      address: "0x178e21ad2E77AE06711549CFBB1f9c7a9d8096e8",
      pubKey: "0x044881345cbb299fa7c60ab2d16cb7fe7bf8d14675506ef6eb6037038b5b7092ea0a9e4d0b53ba3904edd99f86717d6ba81dffe44eb5b23c6fd22c91ab73c33021",
    },
    {
      index: 17,
      address: "0x5E1487F35515d02A92753504a8D75471b9f49EdB",
      pubKey: "0x04ee3d4cc17633afe7e1794fcfd728e0643325e3d130eb1daa39c0c5cb05a200b43876117a182cabdcc3795632aa529473a0c8245f9e4f6e43e54c3f1da28bcb82",
    },
    {
      index: 18,
      address: "0x6FbEBc898F403E4773E95feB15E80C9A99c8348d",
      pubKey: "0x0421f338444e96af31cf44958acf5764844efbddace3b823ed761c340c59ed2685d829818c83eebe8f00f783f1048a53515845536668a9e0c059ade7579a0f4204",
    }
  ]
};

(async() => {

  const STACKS_API_URL = process.env.STACKS_API_URL

  if(!STACKS_API_URL) {
    throw new Error("RPC_URL is required")
  }

  const DEPLOYER_MNEMONIC = process.env.DEPLOYER_MNEMONIC
  const DEPLOYER_PRIVATE_KEY = process.env.DEPLOYER_PRIVATE_KEY

  if((!!DEPLOYER_MNEMONIC && !!DEPLOYER_PRIVATE_KEY) || (!DEPLOYER_MNEMONIC && !DEPLOYER_PRIVATE_KEY)) {
    throw new Error("Only one of DEPLOYER_MNEMONIC or DEPLOYER_PRIVATE_KEY must be set")
  }

  const NETWORK_NAME = process.env.NETWORK_NAME as StacksNetworkName

  if(!NETWORK_NAME) {
    throw new Error("NETWORK_NAME is required")
  }

  if (!StacksNetworks.includes(NETWORK_NAME)) {
    throw new Error(`Invalid NETWORK_NAME: ${NETWORK_NAME} | Valid networks: ${StacksNetworks.join(", ")}`)
  }

  const {privateKey: deployerPrivateKey, address: deployerAddress} = await getKeys(NETWORK_NAME, DEPLOYER_MNEMONIC, DEPLOYER_PRIVATE_KEY)

  const balance = await getPrincipalStxBalance(STACKS_API_URL, deployerAddress)
  console.log(`Using deployer address: ${deployerAddress} , STX balance: ${balance}`)

  const stacksContractsDir = path.resolve(process.cwd(), "../contracts")
  const coreaddr32ContractName = "addr32"
  const coreStateContractName = "wormhole-core-state"
  const coreTraitContractName = "wormhole-trait-core-v2"
  const coreProxyContractName = "wormhole-core-proxy-v2"
  const exportTraitContractName = "wormhole-trait-export-v1"
  const governanceTraitContractName = "wormhole-trait-governance-v1"
  const coreContractName = "wormhole-core-v4"
  const contractFiles = [
    `${coreaddr32ContractName}.clar`,
    `${coreStateContractName}.clar`,
    `${coreTraitContractName}.clar`,
    `${coreProxyContractName}.clar`,
    `${exportTraitContractName}.clar`,
    `${governanceTraitContractName}.clar`,
    `${coreContractName}.clar`,
  ].map((filename) => path.join(stacksContractsDir, filename))
  const contracts = [...contractFiles].map(
    (filePath) => {
      let code = fs.readFileSync(filePath, "utf8")
      return {
        name: path.basename(filePath).replace(".clar", ""),
        filename: path.basename(filePath),
        code: code
      }
    }
  )

  let nonce = await fetchNonce({
    address: deployerAddress,
    client: { baseUrl: STACKS_API_URL },
  })

  console.log(`Using nonce ${nonce} for deployer address: ${deployerAddress} , deploying: ${contracts.length} contracts`)
  for(const contract of contracts) {
    console.log(`Deploying ${contract.name}`)
    const txParams = {
      contractName: contract.name,
      codeBody: contract.code,
      clarityVersion: 4,
      senderKey: deployerPrivateKey,
      nonce,
      network: NETWORK_NAME,
      client: { baseUrl: STACKS_API_URL },
    }
    // TODO: check why/when testnet estimations are wrong
    
    // const transactionToEstimate = await makeContractDeploy(
    //   txParams
    // )

    // const estimation = await fetchFeeEstimateTransaction({
    //   payload: bytesToHex(serializePayloadBytes(transactionToEstimate.payload)),
    //   network: NETWORK_NAME,
    //   client: {
    //     baseUrl: STACKS_API_URL
    //   },
    // })

    const transactionToSend = await makeContractDeploy({
      ...txParams,
      // fee: estimation[0].fee,
      fee: 6000000,
    })
    
    const response = await broadcastTransaction({
      transaction: transactionToSend,
      network: NETWORK_NAME,
      client: { baseUrl: STACKS_API_URL },
    });
    console.log(`Deployment tx:`, response)
    if("error" in response && response.reason === "ContractAlreadyExists") {
      console.log(`Contract ${contract.name} already exists, skipping deployment`)
      continue
    }
    await waitForTransactionSuccess(STACKS_API_URL, response.txid);
    console.log(`[x] ${contract.name} deployed`)
    nonce += 1n;
  }

  console.log(`All contracts deployed`)
  console.log(`Initialize core contract...`)
  const initializeTransaction = await makeContractCall({
    contractAddress: deployerAddress,
    contractName: coreContractName,
    functionName: "initialize",
    functionArgs: [Cl.none()],
    senderKey: deployerPrivateKey,
    nonce,
    network: NETWORK_NAME,
    client: { baseUrl: STACKS_API_URL },
  })

  const initializeResponse = await broadcastTransaction({
    transaction: initializeTransaction,
    network: NETWORK_NAME,
    client: { baseUrl: STACKS_API_URL },
  })

  console.log(`Initialize txid: ${initializeResponse.txid}`)
  await waitForTransactionSuccess(STACKS_API_URL, initializeResponse.txid);
  console.log(`[x] Core initialized`)
  nonce += 1n;

  const upgradeGuardianSetVaa = createUpgradeGuardianSetVaa(0, guardians[NETWORK_NAME].map(g => g.address))

    const upgradeGuardianSetTransaction = await makeContractCall({
      contractAddress: deployerAddress,
      contractName: coreContractName,
      functionName: "guardian-set-upgrade",
      functionArgs: [
        Cl.buffer(serialize(upgradeGuardianSetVaa)),
        // slice(4) to remove the 0x04 prefix
        Cl.list(guardians[NETWORK_NAME].map(g => Cl.bufferFromHex(g.pubKey.slice(4))))
      ],
      senderKey: deployerPrivateKey,
      nonce,
      network: NETWORK_NAME,
      client: { baseUrl: STACKS_API_URL },
    })

    const upgradeGuardianSetResponse = await broadcastTransaction({
      transaction: upgradeGuardianSetTransaction,
      network: NETWORK_NAME,
      client: { baseUrl: STACKS_API_URL },
    })
    console.log(`Upgrade guardian set txid: ${upgradeGuardianSetResponse.txid}`)
    await waitForTransactionSuccess(STACKS_API_URL, upgradeGuardianSetResponse.txid);
    console.log(`[x] Guardian set upgraded`)
    nonce += 1n;

    const balanceAfter = await getPrincipalStxBalance(STACKS_API_URL, deployerAddress)
    const stxSpent = balance - balanceAfter
    console.log(`STX balance after: ${balanceAfter}`)
    console.log(`STX spent: ${stxSpent}`)
    console.log(`Deployed`, {
      addr32: `${deployerAddress}.${coreaddr32ContractName}`,
      state: `${deployerAddress}.${coreStateContractName}`,
      traitCore: `${deployerAddress}.${coreTraitContractName}`,
      proxy: `${deployerAddress}.${coreProxyContractName}`,
      traitExport: `${deployerAddress}.${exportTraitContractName}`,
      traitGovernance: `${deployerAddress}.${governanceTraitContractName}`,
      core: `${deployerAddress}.${coreContractName}`,
    })
})()



function createUpgradeGuardianSetVaa(guardianSetId: number, providedEthKeys: string[]): VAA {
  const vaa = createVAA(
    "WormholeCore:GuardianSetUpgrade",
    {
      guardianSet: guardianSetId,
      timestamp: 1784985530,
      nonce: 0,
      emitterChain: "Solana",
      emitterAddress: new UniversalAddress('0000000000000000000000000000000000000000000000000000000000000004'),
      sequence: 1n,
      consistencyLevel: 0,
      signatures: [],
      payload: {
        chain: "Stacks",
        actionArgs: {
          guardianSet: guardianSetId,
          guardians: providedEthKeys
        }
      }
    }
  )
  const mockPkeys = Array.from({ length: 1 }, () => secp256k1.utils.randomPrivateKey())
  const mockGuardians = new mocks.MockGuardians(1, mockPkeys.map(k => Buffer.from(k).toString('hex')));
  mockGuardians.addSignatures(vaa)
  return vaa
}

async function getPrincipalStxBalance(stacksApiUrl: string, principal: string): Promise<number> {
  const balance = await fetch(`${stacksApiUrl}/extended/v1/address/${principal}/balances`)
  return Number((await balance.json()).stx.balance) / 1e6
}

