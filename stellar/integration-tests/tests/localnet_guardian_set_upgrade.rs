pub(crate) use integration_tests::{
    TestContext, assemble_vaa, craft_governance_payload, craft_vaa_body, eth_address_from_privkey,
    find_event, sign_vaa_body,
};

#[test]
#[ignore]
fn integration_guardian_set_upgrade_flow() {
    let ctx = TestContext::new();

    println!("Deploying contract...");
    let guardian_privkey = [1u8; 32];
    let guardian_addr = eth_address_from_privkey(&guardian_privkey);
    let guardian_addr_hex = hex::encode(guardian_addr);

    let mut gov_emitter_arr = [0u8; 32];
    gov_emitter_arr[31] = 4;
    let gov_emitter_hex = hex::encode(gov_emitter_arr);

    let contract_id =
        ctx.deploy_contract(std::slice::from_ref(&guardian_addr_hex), &gov_emitter_hex);
    println!("Contract deployed at: {}", contract_id);
    println!("Contract initialized with guardian: {}", guardian_addr_hex);

    println!("Crafting GuardianSetUpgrade VAA...");
    let new_guardian_privkey = [2u8; 32];
    let new_guardian_addr = eth_address_from_privkey(&new_guardian_privkey);
    let new_guardian_addr_hex = hex::encode(new_guardian_addr);

    // Payload
    let mut action_payload = Vec::new();
    action_payload.extend_from_slice(&1u32.to_be_bytes()); // New Guardian Set Index: 1
    action_payload.push(1); // Guardian count: 1
    action_payload.extend_from_slice(&new_guardian_addr);

    let payload = craft_governance_payload(2, &action_payload); // Action: GuardianSetUpgrade

    let body = craft_vaa_body(1, gov_emitter_arr, 0, 1, &payload);
    let (recid, compact) = sign_vaa_body(&body, guardian_privkey);

    let vaa = assemble_vaa(0, vec![(0, compact, recid)], &body);
    let vaa_hex = hex::encode(vaa);
    println!("VAA crafted: {}", vaa_hex);

    println!("Submitting GuardianSetUpgrade VAA...");
    ctx.invoke(
        &ctx.admin_identity,
        &contract_id,
        "submit_guardian_set_upgrade",
        &["--vaa_bytes", &vaa_hex],
    );

    println!("Verifying state updated...");
    let index_out = ctx.invoke(
        &ctx.admin_identity,
        &contract_id,
        "get_current_guardian_set_index",
        &[],
    );
    let current_index = index_out
        .trim()
        .parse::<u32>()
        .expect("Failed to parse index");
    assert_eq!(current_index, 1, "Guardian set index should be 1");

    let gset_out = ctx.invoke(
        &ctx.admin_identity,
        &contract_id,
        "get_guardian_set",
        &["--index", "1"],
    );
    assert!(
        gset_out.contains(&new_guardian_addr_hex),
        "New guardian set should contain the new guardian address"
    );
    println!("New guardian set verified.");

    let expiry_out = ctx.invoke(
        &ctx.admin_identity,
        &contract_id,
        "get_guardian_set_expiry",
        &["--index", "0"],
    );
    assert!(
        !expiry_out.contains("void") && !expiry_out.trim().is_empty(),
        "Old guardian set should have an expiry set"
    );
    println!("Old set expiry: {}", expiry_out.trim());

    println!("Verifying event via RPC...");
    let found_event = find_event(
        &ctx.rpc_url,
        &contract_id,
        &[
            vec![
                "guardian_set_upgrade",
                "AAAADwAAABRndWFyZGlhbl9zZXRfdXBncmFkZQ==",
            ],
            vec!["wormhole_core", "AAAADwAAAA13b3JtaG9sZV9jb3JlAAAA"],
        ],
    );
    assert!(found_event, "GuardianSetUpgrade event not found");
    println!("GuardianSetUpgrade event verified.");

    println!("Verifying a message using the NEW guardian set...");
    let mut msg_body = Vec::new();
    msg_body.extend_from_slice(&0u32.to_be_bytes()); // Timestamp
    msg_body.extend_from_slice(&123u32.to_be_bytes()); // Nonce
    msg_body.extend_from_slice(&61u16.to_be_bytes()); // Emitter Chain
    msg_body.extend_from_slice(&[0u8; 32]); // Emitter Address
    msg_body.extend_from_slice(&0u64.to_be_bytes()); // Sequence
    msg_body.push(1); // Consistency Level
    msg_body.extend_from_slice(b"Hello Wormhole"); // Payload

    let (msg_recid, msg_compact) = sign_vaa_body(&msg_body, new_guardian_privkey);

    let msg_vaa = assemble_vaa(1, vec![(0, msg_compact, msg_recid)], &msg_body);
    let msg_vaa_hex = hex::encode(msg_vaa);

    ctx.invoke(
        &ctx.admin_identity,
        &contract_id,
        "verify_vaa",
        &["--vaa_bytes", &msg_vaa_hex],
    );
    println!("VAA verification with new guardian set succeeded.");

    println!("Integration test passed!");
}
