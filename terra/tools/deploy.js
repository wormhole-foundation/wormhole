import { Wallet, LCDClient, MnemonicKey } from "@terra-money/terra.js";
import { StdFee, MsgInstantiateContract, MsgExecuteContract, MsgStoreCode } from "@terra-money/terra.js";
import { readFileSync, readdirSync } from "fs";

// TODO: Workaround /tx/estimate_fee errors.

const gas_prices = {
  uluna: "0.15",
  usdr: "0.1018",
  uusd: "0.15",
  ukrw: "178.05",
  umnt: "431.6259",
  ueur: "0.125",
  ucny: "0.97",
  ujpy: "16",
  ugbp: "0.11",
  uinr: "11",
  ucad: "0.19",
  uchf: "0.13",
  uaud: "0.19",
  usgd: "0.2",
};

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

  await wallet.sequence();

  // Deploy WASM blobs.
  const artifacts = readdirSync('../artifacts/');
  artifacts.sort();
  for (const artifact in artifacts) {
    console.log(artifact);
    console.log(artifacts.hasOwnProperty(artifact));
    if(artifacts.hasOwnProperty(artifact) && artifacts[artifact].includes('.wasm')) {
        const file = artifacts[artifact];
        const contract_bytes = readFileSync(`../artifacts/${file}`);
        console.log(`Storing Bytes, ${contract_bytes.length}, for ${file}`);
        const store_code = new MsgStoreCode(
            wallet.key.accAddress,
            contract_bytes.toString('base64'),
        );

        try {
            const tx = await wallet.createAndSignTx({
                msgs: [store_code],
                memo: '',
                fee: new StdFee(
                    3000000,
                    { uluna: "100000" }
                )
            });

            const rs = await terra.tx.broadcast(tx);

            console.log(JSON.stringify(rs, null, 2));
            await wallet.sequence();
        } catch (e) {
            console.log('Failed to Execute');
        }
    }
  }

  const govChain = 1;
  const govAddress = "0000000000000000000000000000000000000000000000000000000000000004";

  //Instantiate Contracts
  wallet.createAndSignTx({
      msgs: [
        new MsgInstantiateContract(
            wallet.key.accAddress,
            undefined,
            2,
            {
                gov_chain: govChain,
                gov_address: Buffer.from(govAddress, 'hex').toString('base64'),
                guardian_set_expirity: 86400,
                initial_guardian_set: {
                    addresses: [
                        {
                            bytes: Buffer.from('beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe', 'hex').toString('base64'),
                        }
                    ],
                    expiration_time: 0
                },
            },
        )
    ],
    memo:'',
  })
  .then(tx => terra.tx.broadcast(tx))
  .then(rs => console.log(rs));

  wallet.createAndSignTx({
      msgs: [
        new MsgInstantiateContract(
            wallet.key.accAddress,
            undefined,
            4,
            {
                owner: deployer.key.accAddress,
                gov_chain: govChain,
                gov_address: Buffer.from(govAddress, 'hex').toString('base64'),
                wormhole_contract: "",
                wrapped_asset_code_id: 2,
            },
        )
    ],
    memo:'',
  })
  .then(tx => terra.tx.broadcast(tx))
  .then(rs => console.log(rs));

  wallet.createAndSignTx({
      msgs: [
        new MsgInstantiateContract(
            wallet.key.accAddress,
            undefined,
            3,
            {
                name: "MOCK",
                symbol: "MCK",
                decimals: 6,
                initial_balances: [
                    {
                        "address": deployer.key.acc_address,
                        "amount": "100000000"
                    }
                ],
                mint: null,
            },
        )
    ],
    memo:'',
  })
  .then(tx => terra.tx.broadcast(tx))
  .then(rs => console.log(rs));

  const registrations = [
    '01000000000100c9f4230109e378f7efc0605fb40f0e1869f2d82fda5b1dfad8a5a2dafee85e033d155c18641165a77a2db6a7afbf2745b458616cb59347e89ae0c7aa3e7cc2d400000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000546f6b656e4272696467650100000001c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f',
    '01000000000100e2e1975d14734206e7a23d90db48a6b5b6696df72675443293c6057dcb936bf224b5df67d32967adeb220d4fe3cb28be515be5608c74aab6adb31099a478db5c01000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000546f6b656e42726964676501000000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16'
  ];

  registrations.forEach(registration => {
    wallet.createAndSignTx({
      msgs: [
          new MsgExecuteContract(
              wallet.key.accAddress,
              "",
              {
                  submit_vaa: {
                      data: Buffer.from(registration, 'hex'),
                  },
              },
              { uluna: 1000 }
          ),
      ],
      memo: '',
    })
    .then(tx => terra.tx.broadcast(tx))
    .then(rs => console.log(rs));
  });
}

main()
