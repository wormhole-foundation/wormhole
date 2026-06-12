//! Message fee integration test.
//!
//! Note: On localnet this test will fail at "Posting message with fee paid"
//! because the wormhole contract uses a hardcoded native token address
//! (testnet/mainnet). The native asset contract ID differs per network
//! (localnet has a different ID). To pass on localnet the contract would need a
//! configurable native token address at deployment.

use integration_tests::{
    TestContext, assemble_vaa, craft_governance_payload, craft_vaa_body, eth_address_from_privkey,
    rpc_call, sign_vaa_body,
};

#[test]
#[ignore]
fn integration_set_message_fee_and_required_fee() {
    let ctx = TestContext::new();

    println!("Deploying contract...");
    ctx.deploy_native_asset();

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

    println!("Crafting SetMessageFee VAA...");
    let new_fee: u64 = 1000;

    // Payload
    let mut action_payload = Vec::new();
    action_payload.extend_from_slice(&[0u8; 24]);
    action_payload.extend_from_slice(&new_fee.to_be_bytes());

    let payload = craft_governance_payload(3, &action_payload); // Action: SetMessageFee

    let body = craft_vaa_body(1, gov_emitter_arr, 0, 0, &payload);
    let (recid, compact) = sign_vaa_body(&body, guardian_privkey);

    let vaa = assemble_vaa(0, vec![(0, compact, recid)], &body);
    let vaa_hex = hex::encode(vaa);
    println!("VAA crafted: {}", vaa_hex);

    println!("Submitting SetMessageFee VAA...");
    ctx.invoke(
        &ctx.admin_identity,
        &contract_id,
        "submit_set_message_fee",
        &["--vaa_bytes", &vaa_hex],
    );

    // Verify fee updated
    let fee_out = ctx.invoke(&ctx.admin_identity, &contract_id, "get_message_fee", &[]);
    let current_fee = fee_out.trim().parse::<u64>().expect("Failed to parse fee");
    assert_eq!(current_fee, new_fee, "Fee should be updated to {}", new_fee);
    println!("Fee updated successfully.");

    // Create emitter_identity
    let emitter_identity = "fee_emitter";
    let emitter_addr = ctx.setup_identity(emitter_identity);

    println!("Funding emitter address: {}", emitter_addr);

    println!("Attempting post_message without fee...");
    let post_fail_out = std::process::Command::new("stellar")
        .args([
            "contract",
            "invoke",
            "--network",
            &ctx.network,
            "--source",
            emitter_identity,
            "--id",
            &contract_id,
            "--",
            "post_message",
            "--emitter",
            &emitter_addr,
            "--nonce",
            "1",
            "--payload",
            "00",
            "--consistency_level",
            "1",
        ])
        .output()
        .expect("Failed to execute post_message");

    assert!(
        !post_fail_out.status.success(),
        "post_message should fail without fee"
    );
    let stderr = String::from_utf8_lossy(&post_fail_out.stderr);
    println!("Stderr from failed post: {}", stderr);
    assert!(
        stderr.contains("InsufficientFeePaid") || stderr.contains("Error(Contract, #50)"),
        "Error should be InsufficientFeePaid (50)"
    );
    println!("post_message failed as expected without fee.");

    println!("Approving fee payment...");
    let native_token = ctx.get_native_asset_id();
    let latest = rpc_call(
        &ctx.rpc_url,
        r#"{"jsonrpc":"2.0","id":1,"method":"getLatestLedger","params":{}}"#,
    );
    let latest_ledger = latest["result"]["sequence"].as_u64().unwrap() as u32;
    let expiration = latest_ledger + 100;

    ctx.invoke(
        emitter_identity,
        &native_token,
        "approve",
        &[
            "--from",
            &emitter_addr,
            "--spender",
            &contract_id,
            "--amount",
            &new_fee.to_string(),
            "--expiration_ledger",
            &expiration.to_string(),
        ],
    );
    println!("Fee payment approved.");

    println!("Posting message with fee paid...");
    let post_success_out = ctx.invoke(
        emitter_identity,
        &contract_id,
        "post_message",
        &[
            "--emitter",
            &emitter_addr,
            "--nonce",
            "2",
            "--payload",
            "00",
            "--consistency_level",
            "1",
        ],
    );
    println!(
        "post_message succeeded with fee. Result: {}",
        post_success_out.trim()
    );

    println!("Verifying balance changes...");
    let contract_balance = ctx.get_balance(&native_token, &contract_id);
    assert!(
        contract_balance >= new_fee as i128,
        "Contract balance should have increased"
    );
    println!("Contract balance: {}", contract_balance);

    println!("Integration test passed!");
}
