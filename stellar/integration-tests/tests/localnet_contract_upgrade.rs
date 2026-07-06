use integration_tests::{
    TestContext, assemble_vaa, craft_governance_payload, craft_vaa_body, eth_address_from_privkey,
    find_event, sign_vaa_body,
};

#[test]
#[ignore]
fn integration_contract_upgrade_flow() {
    let ctx = TestContext::new();
    let upgrade_wasm_path =
        std::env::var("WORMHOLE_UPGRADE_WASM_PATH").expect("WORMHOLE_UPGRADE_WASM_PATH not set");

    println!("Deploying initial contract...");
    let guardian_privkey = [1u8; 32];
    let guardian_addr = eth_address_from_privkey(&guardian_privkey);
    let guardian_addr_hex = hex::encode(guardian_addr);

    let mut gov_emitter_arr = [0u8; 32];
    gov_emitter_arr[31] = 4;
    let gov_emitter_hex = hex::encode(gov_emitter_arr);

    let contract_id =
        ctx.deploy_contract(std::slice::from_ref(&guardian_addr_hex), &gov_emitter_hex);
    println!("Contract deployed at: {}", contract_id);

    // Verify initial chain ID
    let initial_chain_id = ctx.invoke(&ctx.admin_identity, &contract_id, "get_chain_id", &[]);
    assert_eq!(initial_chain_id.trim(), "61");
    println!("Initial Chain ID: {}", initial_chain_id.trim());

    println!("Installing upgrade WASM...");
    let hash_out = integration_tests::run(std::process::Command::new("stellar").args([
        "contract",
        "install",
        "--network",
        &ctx.network,
        "--source",
        &ctx.admin_identity,
        "--wasm",
        &upgrade_wasm_path,
    ]));
    let wasm_hash_hex = hash_out.trim().to_string();
    println!("Upgrade WASM hash: {}", wasm_hash_hex);
    let wasm_hash = hex::decode(&wasm_hash_hex).expect("Failed to decode WASM hash");

    println!("Crafting ContractUpgrade VAA...");

    // Payload
    let payload = craft_governance_payload(1, &wasm_hash); // Action: ContractUpgrade

    let body = craft_vaa_body(1, gov_emitter_arr, 0, 1, &payload);
    let (recid, compact) = sign_vaa_body(&body, guardian_privkey);

    let vaa = assemble_vaa(0, vec![(0, compact, recid)], &body);
    let vaa_hex = hex::encode(vaa);

    println!("Submitting ContractUpgrade VAA...");
    ctx.invoke(
        &ctx.admin_identity,
        &contract_id,
        "submit_contract_upgrade",
        &["--vaa_bytes", &vaa_hex],
    );
    println!("ContractUpgrade VAA submitted.");

    println!("Verifying event through RPC...");
    let found_event = find_event(
        &ctx.rpc_url,
        &contract_id,
        &[
            vec!["upgrade", "AAAADwAAAAd1cGdyYWRl"],
            vec!["wormhole_core", "AAAADwAAAA13b3JtaG9sZV9jb3JlAAAA"],
        ],
    );
    assert!(found_event, "Contract upgrade event not found");
    println!("Contract upgrade event verified.");

    // Verify contract behavior HAS CHANGED
    let new_chain_id_out = ctx.invoke(&ctx.admin_identity, &contract_id, "get_chain_id", &[]);
    assert_eq!(new_chain_id_out.trim(), "999");
    println!(
        "Contract behavior changed after upgrade! New Chain ID: {}",
        new_chain_id_out.trim()
    );

    println!("Integration test passed!");
}
