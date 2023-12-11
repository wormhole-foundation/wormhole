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
          "terra14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssrc8au",
          codeIds["wormhole.wasm"],
          {
            action: "",
          },
          { uluna: 1000 }
        ),
      ],
      memo: "",
    })
    .then((tx) => terra.tx.broadcast(tx))
    .then((rs) => console.log(rs));

  await wallet
    .createAndSignTx({
      msgs: [
        new MsgMigrateContract(
          wallet.key.accAddress,
          "terra1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrquka9l6",
          codeIds["token_bridge.wasm"],
          {
            action: "",
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
          "terra14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssrc8au",
          "terra14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssrc8au"
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

  // Perform a Guardian Set Upgrade to check the following
  // flow with six guardians rather than the default one.
  const guardianUpgradeVAA =
    "01000000000100f8547caf1d1263e6b4742aef05691a9e2a7aa082bb2f1deb3850e43b801a87044cf786924d8adff5553f31b41149f94a32b568321390450f12c31aa15c2f941101000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000227cc370000000000000000000000000000000000000000000000000000000000436f72650200000000000106befa429d57cd18b7f8a4d91a2da9ab4af05d0fbe4ba0c2db9a26208b3bb1a50b01b16941c10d76db4ba0c2db9a26208b3bb1a50b01b16941c10d76db4ba0c2db9a26208b3bb1a50b01b16941c10d76db4ba0c2db9a26208b3bb1a50b01b16941c10d76db4ba0c2db9a26208b3bb1a50b01b16941c10d76db";

  await wallet
    .createAndSignTx({
      msgs: [
        new MsgExecuteContract(
          wallet.key.accAddress,
          "terra14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssrc8au",
          {
            submit_v_a_a: {
              vaa: Buffer.from(guardianUpgradeVAA, "hex").toString("base64"),
            },
          },
          { uluna: 1000 }
        ),
      ],
      memo: "",
    })
    .then((tx) => terra.tx.broadcast(tx))
    .then((rs) => console.log(rs));

  // Upgrace VAA with 5 signatures to test qurom threshold.
  const upgradeVAA =
    "0100000001050058f5e6a55261e137b12405eb5acf3e4670101c3b7561c6694d7116b6afec85b153f90992fb5e0d6d5a79506f524324fb21894ef655367cc37a572b07a9bfe43301011dba8dca119605dcd30efaf7c4f6980afdf5d58f9625648b652288505abe19be11eabe7424e69d3dae682a84c58208237a975c5ed7757613f546763e14db621200021dba8dca119605dcd30efaf7c4f6980afdf5d58f9625648b652288505abe19be11eabe7424e69d3dae682a84c58208237a975c5ed7757613f546763e14db621200031dba8dca119605dcd30efaf7c4f6980afdf5d58f9625648b652288505abe19be11eabe7424e69d3dae682a84c58208237a975c5ed7757613f546763e14db621200041dba8dca119605dcd30efaf7c4f6980afdf5d58f9625648b652288505abe19be11eabe7424e69d3dae682a84c58208237a975c5ed7757613f546763e14db6212000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000000a653200000000000000000000000000000000000000000000000000000000000436f72650100030000000000000000000000000000000000000000000000000000000000000005";

  // Perform a decentralised update with a signed VAA.
  await wallet
    .createAndSignTx({
      msgs: [
        new MsgExecuteContract(
          wallet.key.accAddress,
          "terra14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssrc8au",
          {
            submit_v_a_a: {
              vaa: Buffer.from(upgradeVAA, "hex").toString("base64"),
            },
          },
          { uluna: 1000 }
        ),
      ],
      memo: "",
    })
    .then((tx) => terra.tx.broadcast(tx))
    .then((rs) => console.log(rs));

  // Set the Admin of the Token Bridge to itself.
  await wallet
    .createAndSignTx({
      msgs: [
        new MsgUpdateContractAdmin(
          wallet.key.accAddress,
          "terra1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrquka9l6",
          "terra1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrquka9l6"
        ),
      ],
      memo: "",
    })
    .then((tx) => terra.tx.broadcast(tx))
    .then((rs) => console.log(rs));

  // Upgrade VAA for the Token Bridge.
  const upgradeTokenVAA =
    "01000000010500088c284fe2adf0976511290902cbb1dd29239dcd9cb343936c8e76825777db0912eecb7d1be70ddc8b15091834bc0626ea52cc82a202c71f1dc2ff6acffa111b0101b9c36107b2fa1ad413ec6a71aca58d4cd44dea28b692c242805ff0c6df7ce0cb5648f92f5a17a1e1cd2e6df89abb236716d9556a03e6ec5d2ad463cd326d1b830102b9c36107b2fa1ad413ec6a71aca58d4cd44dea28b692c242805ff0c6df7ce0cb5648f92f5a17a1e1cd2e6df89abb236716d9556a03e6ec5d2ad463cd326d1b830103b9c36107b2fa1ad413ec6a71aca58d4cd44dea28b692c242805ff0c6df7ce0cb5648f92f5a17a1e1cd2e6df89abb236716d9556a03e6ec5d2ad463cd326d1b830104b9c36107b2fa1ad413ec6a71aca58d4cd44dea28b692c242805ff0c6df7ce0cb5648f92f5a17a1e1cd2e6df89abb236716d9556a03e6ec5d2ad463cd326d1b8301000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000441f94100000000000000000000000000000000000000000000546f6b656e4272696467650200030000000000000000000000000000000000000000000000000000000000000005";

  // Perform a decentralised update with a signed VAA.
  await wallet
    .createAndSignTx({
      msgs: [
        new MsgExecuteContract(
          wallet.key.accAddress,
          "terra1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrquka9l6",
          {
            submit_vaa: {
              data: Buffer.from(upgradeTokenVAA, "hex").toString("base64"),
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
