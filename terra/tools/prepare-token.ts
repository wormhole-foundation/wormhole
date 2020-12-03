import { init_lcd, deploy_contract, instantiate_contract, query_contract } from './utils';

async function script() {
    const TEST_ADDRESS: string = 'terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v';
    // cw20_base.wasm is a binary artifact built from https://github.com/CosmWasm/cosmwasm-plus repository at v0.2.0
    // and is a standard base cw20 contract. Is it used for testing only.
    let code_id = await deploy_contract('../artifacts/cw20_base.wasm');
    if (code_id == -1) return;
    console.log(`Program deployed with code id ${code_id}`);
    let contract_address = await instantiate_contract(code_id, {
        name: 'Test token',
        symbol: 'TST',
        decimals: 8,
        initial_balances: [{
            address: TEST_ADDRESS,
            amount: '100000000000000',
        }],
        mint: null,
    });
    console.log(`Contract instance created at ${contract_address}`);

    // Verify if token was minted to the test address
    let result = await query_contract(contract_address, {balance: { address : TEST_ADDRESS}});
    console.log(`${TEST_ADDRESS} balance is ${result.balance}`);
}

init_lcd(process.argv[2]);
script();
