import { deploy_contract, instantiate_contract, query_contract } from './utils';

async function script() {
    // Deploy cw20-wrapped
    let wrapped_code_id = await deploy_contract('../artifacts/cw20_wrapped.wasm');
    if (wrapped_code_id == -1) return;
    console.log(`CW20 Wrapped program deployed with code id ${wrapped_code_id}`);
    // Deploy wormhole
    let wormhole_code_id = await deploy_contract('../artifacts/wormhole.wasm');
    if (wormhole_code_id == -1) return;
    console.log(`Wormhole program deployed with code id ${wormhole_code_id}`);
    // Instantiate wormhole
    let contract_address = await instantiate_contract(wormhole_code_id, {
        initial_guardian_set: {
            addresses: [
                { bytes: [0xbe, 0xfa, 0x42, 0x9d, 0x57, 0xcd, 0x18, 0xb7, 0xf8, 0xa4, 0xd9, 0x1a, 0x2d, 0xa9, 0xab, 0x4a, 0xf0, 0x5d, 0x0f, 0xbe] }
            ],
            expiration_time: 1000 * 60 * 60
        },
        guardian_set_expirity: 0,
        wrapped_asset_code_id: wrapped_code_id,
    });
    console.log(`Wormhole instance created at ${contract_address}`);
}

script();