import {
  Connection,
  Ed25519Keypair,
  fromB64,
  JsonRpcProvider,
  normalizeSuiObjectId,
  RawSigner,
  TransactionBlock,
} from "@mysten/sui.js";
import { execSync } from "child_process";

const deploy = async (signer: RawSigner) => {
  // Build contracts
  const build = JSON.parse(
    execSync(`sui move build --dump-bytecode-as-base64 --path .`, {
      encoding: "utf-8",
    })
  );

  // Publish contracts
  const tx = new TransactionBlock();
  const [upgradeCap] = tx.publish(
    build.modules.map((m: string) => Array.from(fromB64(m))),
    build.dependencies.map((d: string) => normalizeSuiObjectId(d))
  );

  // Transfer upgrade capability to deployer
  tx.transferObjects([upgradeCap], tx.pure(await signer.getAddress()));

  // Execute transactions
  return signer.signAndExecuteTransactionBlock({
    transactionBlock: tx,
    options: {
      showObjectChanges: true,
    },
  });
};

const printOwnedObjects = async (provider: JsonRpcProvider, owner: string) => {
  const res = await provider.getOwnedObjects({ owner });
  const objects = res.data.map(async (e) => {
    const object = await provider.getObject({ id: e.data.objectId });
    return {
      objectId: object.data.objectId,
      type: object.data.type,
    };
  });
  console.log(JSON.stringify(objects, null, 2));
};

const getOwnedObjectId = async (
  provider: JsonRpcProvider,
  owner: string,
  type: string
): Promise<string | null> => {
  const res = await provider.getOwnedObjects({
    owner,
    filter: { StructType: type },
  });
  return res.data.length > 0 ? res.data[0].data.objectId : null;
};

const main = async () => {
  const provider = new JsonRpcProvider(
    new Connection({ fullnode: "http://0.0.0.0:9000" })
  );
  const privateKey = "AGA20wtGcwbcNAG4nwapbQ5wIuXwkYQEWFUoSVAxctHb";
  const signer = new RawSigner(
    Ed25519Keypair.fromSecretKey(fromB64(privateKey).slice(1)),
    provider
  );

  console.log("Deploying contracts...");
  const deployRes = await deploy(signer);

  const packageId = deployRes.objectChanges.find((e) => e.type === "published")[
    "packageId"
  ];
  console.log("Package id:", packageId);

  const upgradeCapObjectId = await getOwnedObjectId(
    provider,
    await signer.getAddress(),
    "0x2::package::UpgradeCap"
  );
  console.log("Upgrade cap object id:", upgradeCapObjectId);

  const initTx = new TransactionBlock();
  initTx.moveCall({
    target: `${packageId}::example::init_with_params`,
    arguments: [initTx.object(upgradeCapObjectId)],
  });
  const initRes = await signer.signAndExecuteTransactionBlock({
    transactionBlock: initTx,
    options: {
      showObjectChanges: true,
    },
  });

  const stateObjectId: string = initRes.objectChanges.find(
    (e) =>
      e.type === "created" && e.objectType === `${packageId}::example::State`
  )["objectId"];
  console.log("State object id:", stateObjectId);

  // Looking at owned objects of State
  // ERROR!
  await printOwnedObjects(provider, stateObjectId);

  // const msgTx = new TransactionBlock();
  // msgTx.moveCall({
  //   target: `${packageId}::example::send_message_entry`,
  //   arguments: [msgTx.object(upgradeCapObjectId)],
  // });

  // // ERROR!
  // // Error: The following input objects are not invalid: {{upgradeCapObjectId}}
  // const msgRes = await signer.signAndExecuteTransactionBlock({
  //   transactionBlock: msgTx,
  //   options: {
  //     showObjectChanges: true,
  //   },
  // });
  // console.log(JSON.stringify(msgRes, null, 2));
};

main();
