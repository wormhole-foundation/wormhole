import { execute_contract, query_contract } from './utils';

async function script() {
    if (process.argv.length < 5) {
        console.log('Required 3 params TOKEN_CONTRACT, WORMHOLE_CONTRACT, integer AMOUNT');
    }
    let token_contract = process.argv[2];
    let wormhole_contract = process.argv[3];
    let amount = process.argv[4];

    let allowanceResult = await execute_contract(token_contract, {increase_allowance: {spender: wormhole_contract, amount}});
    if (allowanceResult == null) return;
    console.log('Allowance increased');
    let lockResult = await execute_contract(wormhole_contract, {lock_assets: {
            asset: token_contract, 
            amount,
            recipient: [...Buffer.from('00000000000000000000000019a4437E2BA06bF1FA42C56Fb269Ca0d30f60716', 'hex')],
            target_chain: 2, // Ethereum
            nonce: Date.now() % 1000000
        }});
    if (lockResult == null) return;
    console.log('Tokens locked');
}

script();