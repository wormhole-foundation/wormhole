// npx pretty-quick

const nearAPI = require("near-api-js");
const BN = require("bn.js");
const fs = require("fs");
const fetch = require("node-fetch");
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";

function getConfig(env: any) {
  switch (env) {
    case "sandbox":
    case "local":
      return {
        networkId: "sandbox",
        nodeUrl: "http://localhost:3030",
        masterAccount: "test.near",
        wormholeAccount: "wormhole.test.near",
        tokenAccount: "token.test.near",
        nftAccount: "nft.test.near",
        testAccount: "test.test.near",
      };
    case "testnet":
      return {
        networkId: "testnet",
        nodeUrl: "https://rpc.testnet.near.org",
        masterAccount: "wormhole.testnet",
        wormholeAccount: "wormhole.wormhole.testnet",
        tokenAccount: "token.wormhole.testnet",
        nftAccount: "nft.wormhole.testnet",
        testAccount: "test.wormhole.testnet",
      };
  }
  return {};
}

async function initNear() {
  let e = process.env.NEAR_ENV || "sandbox";

  let config = getConfig(e);

  let masterKey: any;

  if (e === "sandbox") {
    // Retrieve the validator key directly in the Tilt environment
    const response = await fetch("http://localhost:3031/validator_key.json");

    const keyFile = await response.json();

    masterKey = nearAPI.utils.KeyPair.fromString(
      keyFile.secret_key || keyFile.private_key
    );
  } else {
    masterKey = nearAPI.utils.KeyPair.fromString(process.env.NEAR_PK);
  }
  let masterPubKey = masterKey.getPublicKey();

  let keyStore = new nearAPI.keyStores.InMemoryKeyStore();
  keyStore.setKey(config.networkId, config.masterAccount, masterKey);

  let near = await nearAPI.connect({
    deps: {
      keyStore,
    },
    networkId: config.networkId,
    nodeUrl: config.nodeUrl,
  });
  let masterAccount = new nearAPI.Account(
    near.connection,
    config.masterAccount
  );

  console.log(
    "Finish init NEAR masterAccount: " +
      JSON.stringify(await masterAccount.getAccountBalance())
  );

  const wormholeContract = await fs.readFileSync("./near_wormhole.wasm");
  const tokenContract = await fs.readFileSync("./near_token_bridge.wasm");
  const nftContract = await fs.readFileSync("./near_nft_bridge.wasm");
  const testContract = await fs.readFileSync("./near_mock_bridge_integration.wasm");

  let wormholeAccount: any;

  console.log("setting key for new wormhole contract");
  keyStore.setKey(config.networkId, config.wormholeAccount, masterKey);
  keyStore.setKey(config.networkId, config.tokenAccount, masterKey);
  keyStore.setKey(config.networkId, config.nftAccount, masterKey);

  if (e === "sandbox") {
    console.log("Deploying core/wormhole contract: " + config.wormholeAccount);
    wormholeAccount = await masterAccount.createAndDeployContract(
      config.wormholeAccount,
      masterKey.getPublicKey(),
      wormholeContract,
      new BN("20000000000000000000000000")
    );

      await wormholeAccount.functionCall({
        contractId: config.wormholeAccount,
        methodName: "register_emitter",
        args: {emitter: config.tokenAccount},
        attachedDeposit: new BN("30000000000000000000000"),
        gas: new BN("100000000000000"),
      })
      await wormholeAccount.functionCall({
        contractId: config.wormholeAccount,
        methodName: "register_emitter",
        args: {emitter: config.testAccount},
        attachedDeposit: new BN("30000000000000000000000"),
        gas: new BN("100000000000000"),
      })
      await wormholeAccount.functionCall({
        contractId: config.wormholeAccount,
        methodName: "register_emitter",
        args: {emitter: config.nftAccount},
        attachedDeposit: new BN("30000000000000000000000"),
        gas: new BN("100000000000000"),
      })
  } else {
    // This uses the standard API to redeploy ... we can migrate over to the vaa's later
    console.log(
      "redeploying core/wormhole contract: " + config.wormholeAccount
    );
    wormholeAccount = new nearAPI.Account(
      near.connection,
      config.wormholeAccount
    );
    await wormholeAccount.deployContract(wormholeContract);

//    console.log("migrating " + config.wormholeAccount);
//    console.log(
//      await wormholeAccount.functionCall({
//        contractId: config.wormholeAccount,
//        methodName: "migrate",
//        args: {},
//        attachedDeposit: new BN(1),
//        gas: new BN("100000000000000"),
//      })
//    );
//    console.log("done migrating " + config.tokenAccount);

  }

  let tokenAccount: any;

  if (e === "sandbox") {
    console.log("Deploying token bridgecontract: " + config.tokenAccount);
    tokenAccount = await masterAccount.createAndDeployContract(
      config.tokenAccount,
      masterKey.getPublicKey(),
      tokenContract,
      new BN("20000000000000000000000000")
    );
  } else {
    // This uses the standard API to redeploy ... we can migrate over to the vaa's later
    console.log("redeploying token bridge contract: " + config.tokenAccount);
    tokenAccount = new nearAPI.Account(near.connection, config.tokenAccount);
    await tokenAccount.deployContract(tokenContract);

//    console.log("migrating " + config.tokenAccount);
//    console.log(
//      await tokenAccount.functionCall({
//        contractId: config.tokenAccount,
//        methodName: "migrate",
//        args: {},
//        attachedDeposit: new BN(1),
//        gas: new BN("100000000000000"),
//      })
//    );
//    console.log("done migrating " + config.tokenAccount);
  }

  let nftAccount: any;

  if (e === "sandbox") {
    console.log("Deploying nft bridge contract: " + config.nftAccount);
    let nftAccount = await masterAccount.createAndDeployContract(
      config.nftAccount,
      masterKey.getPublicKey(),
      nftContract,
      new BN("20000000000000000000000000")
    );
  } else {
    // This uses the standard API to redeploy ... we can migrate over to the vaa's later
    console.log("redeploying nft contract: " + config.nftAccount);
    nftAccount = new nearAPI.Account(near.connection, config.nftAccount);
    await nftAccount.deployContract(nftContract);
  }

  let lines: any;

  if (e === "sandbox") {
    console.log("Deploying mach contract to " + config.testAccount);
    let testAccount = await masterAccount.createAndDeployContract(
      config.testAccount,
      masterKey.getPublicKey(),
      testContract,
      new BN("25000000000000000000000000")
    );

    console.log("booting wormhole to devnet keys");

    lines = fs.readFileSync(".env", "utf-8").split("\n");
  } else {
    console.log("booting wormhole to testnet keys");

    lines = fs.readFileSync("/home/jsiegel/testnet-env", "utf-8").split("\n");
  }

  //  console.log(lines);
  let signers: any[] = [];

  let vaasToken: any[] = [];
  let vaasNFT: any[] = [];

  lines.forEach((line: any) => {
    let f = line.split("=");
    if (f[0] === "INIT_SIGNERS") {
      signers = eval(f[1]);
    }
    if (f[0].startsWith("REGISTER_") && f[0].endsWith("TOKEN_BRIDGE_VAA")) {
      vaasToken.push(f[1]);
    } else if (f[0].endsWith("TOKEN_BRIDGE_VAA_REGISTER")) {
      vaasToken.push(f[1]);
    }

    if (f[0].startsWith("REGISTER_") && f[0].endsWith("NFT_BRIDGE_VAA")) {
      vaasNFT.push(f[1]);
    } else if (
      f[0].endsWith("NFT_BRIDGE_VAA") ||
      f[0].endsWith("NFT_BRIDGE_VAA_REGISTER")
    ) {
      vaasNFT.push(f[1]);
    }
  });

  console.log(vaasToken);
  console.log(vaasNFT);

  if (e === "sandbox") {
    let result = await masterAccount.functionCall({
      contractId: config.wormholeAccount,
      methodName: "boot_wormhole",
      args: {
        gset: 0,
        addresses: signers,
      },
      gas: 100000000000000,
    });

    console.log("Booting up the token bridge");

    result = await masterAccount.functionCall({
      contractId: config.tokenAccount,
      methodName: "boot_portal",
      args: {
        core: config.wormholeAccount,
      },
      gas: 100000000000000,
    });

    console.log("Booting up the nft bridge");

    result = await masterAccount.functionCall({
      contractId: config.nftAccount,
      methodName: "boot_portal",
      args: {
        core: config.wormholeAccount,
      },
      gas: 100000000000000,
    });
  }

  for (const line of vaasNFT) {
    console.log("Submitting to " + config.nftAccount + ": " + line);

    try {
      await masterAccount.functionCall({
        contractId: config.nftAccount,
        methodName: "submit_vaa",
        args: {
          vaa: line,
        },
        attachedDeposit: new BN("30000000000000000000000"),
        gas: new BN("300000000000000"),
      });

      await masterAccount.functionCall({
        contractId: config.nftAccount,
        methodName: "submit_vaa",
        args: {
          vaa: line,
        },
        attachedDeposit: new BN("30000000000000000000000"),
        gas: new BN("300000000000000"),
      });
    } catch {
      console.log("Exception thrown.. ");
    }
  }

  console.log("nft bridge booted");

  for (const line of vaasToken) {
    console.log("Submitting to " + config.tokenAccount + ": " + line);

    try {
      await masterAccount.functionCall({
        contractId: config.tokenAccount,
        methodName: "submit_vaa",
        args: {
          vaa: line,
        },
        attachedDeposit: new BN("30000000000000000000001"),
        gas: new BN("300000000000000"),
      });

      await masterAccount.functionCall({
        contractId: config.tokenAccount,
        methodName: "submit_vaa",
        args: {
          vaa: line,
        },
        attachedDeposit: new BN("30000000000000000000001"),
        gas: new BN("300000000000000"),
      });
    } catch {
      console.log("Exception thrown.. ");
    }
  }

  console.log("token bridge booted");

  //  console.log("deleting the master key from the token contract");
  //  await tokenAccount.deleteKey(masterKey.getPublicKey());

  //  console.log("deleting the master key from the nft contract");
  //  await nftAccount.deleteKey(masterKey.getPublicKey());

  //  console.log("deleting the master key from the wormhole contract");
  //  await wormholeAccount.deleteKey(masterKey.getPublicKey());
}

initNear();
