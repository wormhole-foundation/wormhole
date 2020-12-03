import { LCDClient, MsgStoreCode, MsgInstantiateContract, MsgExecuteContract, MnemonicKey, isTxError } from '@terra-money/terra.js';
import * as fs from 'fs';

// test1 key from localterra accounts
const mk = new MnemonicKey({
    mnemonic: 'notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius'
})

let terra: LCDClient;
let wallet;

export function init_lcd(host_name='http://localhost:1317') {
    // connect to localterra
    terra = new LCDClient({
        URL: host_name,
        chainID: 'localterra'
    });
    wallet = terra.wallet(mk);
}

function delay(ms: number) {
    return new Promise( resolve => setTimeout(resolve, ms) );
}

export async function deploy_contract(wasm_file) : Promise<number> {
  
    const storeCode = new MsgStoreCode(
        wallet.key.accAddress,
        fs.readFileSync(wasm_file).toString('base64')
    );
    for (;;) {
        try {
            const storeCodeTx = await wallet.createAndSignTx({
                msgs: [storeCode],
            });
            const storeCodeTxResult = await terra.tx.broadcast(storeCodeTx);

            //console.log(storeCodeTxResult);

            if (isTxError(storeCodeTxResult)) {
                throw new Error(
                `store code failed. code: ${storeCodeTxResult.code}, codespace: ${storeCodeTxResult.codespace}, raw_log: ${storeCodeTxResult.raw_log}`
                );
            }

            const {
                store_code: { code_id },
            } = storeCodeTxResult.logs[0].eventsByType;

            return parseInt(code_id[0], 10);
        } catch (err) {
            console.log(`Error ${err}`);
            if (err.response) {
                console.log(err.response.data);
            }
            await delay(5000);
        }
    }
}

export async function instantiate_contract(code_id: number, initMsg: object) : Promise<string> {
    try {
        const instantiate = new MsgInstantiateContract(
            wallet.key.accAddress,
            code_id,
            initMsg,
            {},
            false
        );

        const instantiateTx = await wallet.createAndSignTx({
            msgs: [instantiate],
        });
        const instantiateTxResult = await terra.tx.broadcast(instantiateTx);

        if (isTxError(instantiateTxResult)) {
            throw new Error(
            `instantiate failed. code: ${instantiateTxResult.code}, codespace: ${instantiateTxResult.codespace}, raw_log: ${instantiateTxResult.raw_log}`
            );
            return null;
        }

        const {
            instantiate_contract: { contract_address },
        } = instantiateTxResult.logs[0].eventsByType;

        return contract_address[0];

    } catch (err) {
        console.log(`Error ${err}`);
        if (err.response) {
            console.log(err.response.data);
        }
        return null;
    }
}

export async function execute_contract(contract_address: string, msg: object) : Promise<any> {
    try {
        const execute = new MsgExecuteContract(
            wallet.key.accAddress,
            contract_address,
            { ...msg }, { } 
        );

        const executeTx = await wallet.createAndSignTx({
            msgs: [execute]
        });

        const result = await terra.tx.broadcast(executeTx);
        return result;
    } catch (err) {
        console.log(`Error ${err}`);
        if (err.response) {
            console.log(err.response.data);
        }
        return null;
    }
}

export async function query_contract(contract_address: string, query: object) : Promise<any> {
    const result = await terra.wasm.contractQuery(
        contract_address, query
    );
    return result;
}