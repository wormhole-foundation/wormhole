import { execute_contract } from './utils';

async function script() {
    if (process.argv.length < 3) {
        console.log('Required 1 param WORMHOLE_CONTRACT');
    }
    let wormhole_contract = process.argv[2];

    // Test VAA built using bridge/cmd/vaa-test
    let vaaResult = await execute_contract(wormhole_contract, {submit_v_a_a: {
            vaa: [...Buffer.from('010000000001005468beb21caff68710b2af2d60a986245bf85099509b6babe990a6c32456b44b3e2e9493e3056b7d5892957e14beab24be02dab77ed6c8915000e4a1267f78f400000007d01000000038018002010400000000000000000000000000000000000000000000000000000000000101010101010101010101010101010101010101000000000000000000000000010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e9988080000000000000000000000000000000000000000000000000de0b6b3a7640000', 'hex')]
        }});
    if (vaaResult == null) return;
    console.log('Vaa submitted');
}

script();