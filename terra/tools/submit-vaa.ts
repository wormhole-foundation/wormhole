import { execute_contract } from './utils';

async function script() {
    if (process.argv.length < 3) {
        console.log('Required 1 param WORMHOLE_CONTRACT');
    }
    let wormhole_contract = process.argv[2];

    // Test VAA built using bridge/cmd/vaa-test
    let vaaResult = await execute_contract(wormhole_contract, {submit_v_a_a: {
            vaa: Buffer.from('010000000001001063f503dd308134e0f158537f54c5799719f4fa2687dd276c72ef60ae0c82c47d4fb560545afaabdf60c15918e221763fd1892c75f2098c0ffd5db4af254a4501000007d01000000038010302010400000000000000000000000000000000000000000000000000000000000101010101010101010101010101010101010101000000000000000000000000010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e9988080000000000000000000000000000000000000000000000000de0b6b3a7640000', 'hex').toString('base64')
        }});
    if (vaaResult == null) return;
    console.log('Vaa submitted');
}

script();