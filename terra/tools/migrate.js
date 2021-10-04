import { Wallet, LCDClient, MnemonicKey } from "@terra-money/terra.js";
import {
  StdFee,
  MsgExecuteContract,
  MsgInstantiateContract,
  MsgMigrateContract,
  MsgStoreCode,
  MsgUpdateContractAdmin,
} from "@terra-money/terra.js";
import { readFileSync, readdirSync } from "fs";

async function main() {
  const terra = new LCDClient({
    URL: "http://localhost:1317",
    chainID: "localterra",
  });

  const wallet = terra.wallet(
    new MnemonicKey({
      mnemonic:
        "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius",
    })
  );

  const hardcodedGas = {
    "wormhole.wasm": 5000000,
  };

  // Deploy Wormhole alone.
  const file = "wormhole.wasm";
  const contract_bytes = readFileSync(`../artifacts/${file}`);
  console.log(`Storing WASM: ${file} (${contract_bytes.length} bytes)`);

  // Get new code id.
  const store_code = new MsgStoreCode(
    wallet.key.accAddress,
    contract_bytes.toString("base64")
  );

  const codeIds = {};
  try {
    const tx = await wallet.createAndSignTx({
      msgs: [store_code],
      memo: "",
      fee: new StdFee(hardcodedGas["wormhole.wasm"], {
        uluna: "100000",
      }),
    });

    const rs = await terra.tx.broadcast(tx);
    const ci = /"code_id","value":"([^"]+)/gm.exec(rs.raw_log)[1];
    codeIds[file] = parseInt(ci);
  } catch (e) {
    console.log("Failed to Execute");
  }

  // Perform a Centralised update.
  await wallet
    .createAndSignTx({
      msgs: [
        new MsgMigrateContract(
          wallet.key.accAddress,
          "terra18vd8fpwxzck93qlwghaj6arh4p7c5n896xzem5",
          codeIds["wormhole.wasm"],
          {
              "action": ""
          },
          { uluna: 1000 }
        ),
      ],
      memo: "",
    })
    .then((tx) => terra.tx.broadcast(tx))
    .then((rs) => console.log(rs));

  // Set the Admin to the contract.
  await wallet
    .createAndSignTx({
      msgs: [
        new MsgUpdateContractAdmin(
          wallet.key.accAddress,
          "terra18vd8fpwxzck93qlwghaj6arh4p7c5n896xzem5",
          "terra18vd8fpwxzck93qlwghaj6arh4p7c5n896xzem5"
        ),
      ],
      memo: "",
    })
    .then((tx) => terra.tx.broadcast(tx))
    .then((rs) => console.log(rs));

  // Deploy a new CodeID.
  try {
    const tx = await wallet.createAndSignTx({
      msgs: [store_code],
      memo: "",
      fee: new StdFee(hardcodedGas["wormhole.wasm"], {
        uluna: "100000",
      }),
    });

    const rs = await terra.tx.broadcast(tx);
    const ci = /"code_id","value":"([^"]+)/gm.exec(rs.raw_log)[1];
    codeIds[file] = parseInt(ci);
  } catch (e) {
    console.log("Failed to Execute");
  }

  const upgradeVAA = '010000000001008928c70a029a924d334a24587e9d2ddbcfa7250d7ba61200e86b16966ef2bbd675fb759aa7a47c6392482ef073e9a6d7c4980dc53ed6f90fc84331486e284912000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000004e78c580000000000000000000000000000000000000000000000000000000000436f72650100030000000000000000000000000000000000000000000000000000000000000005';

  // Perform a decentralised update with a signed VAA.
  await wallet
    .createAndSignTx({
      msgs: [
        new MsgExecuteContract(
          wallet.key.accAddress,
          "terra18vd8fpwxzck93qlwghaj6arh4p7c5n896xzem5",
          {
            submit_v_a_a: {
              vaa: Buffer.from(upgradeVAA, "hex").toString(
                "base64"
              ),
            },
          },
          { uluna: 1000 }
        ),
      ],
      memo: "",
    })
    .then((tx) => terra.tx.broadcast(tx))
    .then((rs) => console.log(rs));
}

main();
