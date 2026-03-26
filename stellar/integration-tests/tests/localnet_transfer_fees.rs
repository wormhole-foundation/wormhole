use integration_tests::{
    TestContext, assemble_vaa, craft_governance_payload, craft_vaa_body, eth_address_from_privkey,
    find_event, sign_vaa_body,
};
use stellar_strkey::Strkey;

#[test]
#[ignore]
fn integration_transfer_fees_flow() {
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

    println!("Funding contract with XLM...");
    let native_token = ctx.get_native_asset_id();
    let admin_addr = ctx.get_identity_address(&ctx.admin_identity);

    let fund_amount: i128 = 500_000_000;
    ctx.invoke(
        &ctx.admin_identity,
        &native_token,
        "transfer",
        &[
            "--from",
            &admin_addr,
            "--to",
            &contract_id,
            "--amount",
            &fund_amount.to_string(),
        ],
    );

    let contract_balance = ctx.invoke(
        &ctx.admin_identity,
        &contract_id,
        "get_contract_balance",
        &[],
    );
    let contract_balance = contract_balance
        .trim()
        .trim_matches('"')
        .parse::<i128>()
        .expect("Failed to parse contract balance");
    assert!(contract_balance >= fund_amount, "Contract should be funded");
    println!("Contract balance: {} stroops", contract_balance);

    let recipient_identity = "fee_recipient";
    let recipient_addr = ctx.setup_identity(recipient_identity);

    println!("Funding recipient address: {}", recipient_addr);

    // Get recipient's initial balance
    let initial_recipient_balance = ctx.get_balance(&native_token, &recipient_addr);
    println!(
        "Initial recipient balance: {} stroops",
        initial_recipient_balance
    );

    println!("Crafting TransferFees VAA...");
    let transfer_amount: u64 = 200_000_000; // 20 XLM
    let strkey = Strkey::from_string(&recipient_addr).expect("Invalid recipient address");
    let recipient_pk = match strkey {
        Strkey::PublicKeyEd25519(pk) => pk.0,
        _ => panic!("Expected ED25519 public key"),
    };

    // Payload
    let mut action_payload = Vec::new();
    action_payload.extend_from_slice(&[0u8; 24]);
    action_payload.extend_from_slice(&transfer_amount.to_be_bytes());
    action_payload.extend_from_slice(&recipient_pk);

    let payload = craft_governance_payload(4, &action_payload); // Action: TransferFees

    let body = craft_vaa_body(1, gov_emitter_arr, 0, 0, &payload);
    let (recid, compact) = sign_vaa_body(&body, guardian_privkey);

    let vaa = assemble_vaa(0, vec![(0, compact, recid)], &body);
    let vaa_hex = hex::encode(vaa);

    println!("Submitting TransferFees VAA...");
    ctx.invoke(
        &ctx.admin_identity,
        &contract_id,
        "submit_transfer_fees",
        &["--vaa_bytes", &vaa_hex],
    );

    println!("Verifying balance changes...");
    let new_contract_balance = ctx.invoke(
        &ctx.admin_identity,
        &contract_id,
        "get_contract_balance",
        &[],
    );
    let new_contract_balance = new_contract_balance
        .trim()
        .trim_matches('"')
        .parse::<i128>()
        .expect("Failed to parse new contract balance");
    assert_eq!(
        new_contract_balance,
        contract_balance - transfer_amount as i128,
        "Contract balance should have decreased"
    );

    let new_recipient_balance = ctx.get_balance(&native_token, &recipient_addr);
    assert_eq!(
        new_recipient_balance,
        initial_recipient_balance + transfer_amount as i128,
        "Recipient balance should have increased"
    );
    println!("Balances verified successfully.");

    // Verify get_last_fee_transfer and event
    let last_transfer_out = ctx.invoke(
        &ctx.admin_identity,
        &contract_id,
        "get_last_fee_transfer",
        &[],
    );
    assert!(
        !last_transfer_out.contains("void") && !last_transfer_out.trim().is_empty(),
        "Last fee transfer should have a timestamp"
    );

    println!("Verifying event via RPC...");
    let found_event = find_event(
        &ctx.rpc_url,
        &contract_id,
        &[
            vec!["fee_transfer", "AAAADwAAAAxmZWVfdHJhbnNmZXI="],
            vec!["wormhole_core", "AAAADwAAAA13b3JtaG9sZV9jb3JlAAAA"],
        ],
    );
    assert!(found_event, "FeeTransfer event not found");
    println!("FeeTransfer event verified.");

    println!("Testing failure case: Transfer more than available...");
    let too_much_amount: u64 = 1_000_000_000;
    let mut too_much_action_payload = action_payload.clone();
    too_much_action_payload[24..32].copy_from_slice(&too_much_amount.to_be_bytes());

    let too_much_payload = craft_governance_payload(4, &too_much_action_payload);
    let too_much_body = craft_vaa_body(1, gov_emitter_arr, 0, 1, &too_much_payload);
    let (tm_recid, tm_compact) = sign_vaa_body(&too_much_body, guardian_privkey);

    let too_much_vaa = assemble_vaa(0, vec![(0, tm_compact, tm_recid)], &too_much_body);
    let too_much_vaa_hex = hex::encode(too_much_vaa);

    let tm_res = std::process::Command::new("stellar")
        .args([
            "contract",
            "invoke",
            "--network",
            &ctx.network,
            "--source",
            &ctx.admin_identity,
            "--id",
            &contract_id,
            "--",
            "submit_transfer_fees",
            "--vaa_bytes",
            &too_much_vaa_hex,
        ])
        .output()
        .expect("Failed to execute submit_transfer_fees");

    assert!(
        !tm_res.status.success(),
        "Transfer should fail when amount is too high"
    );
    let tm_stderr = String::from_utf8_lossy(&tm_res.stderr);
    assert!(
        tm_stderr.contains("InsufficientFees") || tm_stderr.contains("Error(Contract, #51)"),
        "Error should be InsufficientFees (51)"
    );
    println!("Transfer failed as expected for high amount.");

    println!("Testing failure case: Leaving less than 1 XLM...");
    let leave_too_little_amount: u64 = 495_000_000; // Contract has 500, needs to keep 10. 500-495 = 5 (too little)
    let mut little_action_payload = action_payload.clone();
    little_action_payload[24..32].copy_from_slice(&leave_too_little_amount.to_be_bytes());

    let little_payload = craft_governance_payload(4, &little_action_payload);
    let little_body = craft_vaa_body(1, gov_emitter_arr, 0, 2, &little_payload);
    let (l_recid, l_compact) = sign_vaa_body(&little_body, guardian_privkey);

    let little_vaa = assemble_vaa(0, vec![(0, l_compact, l_recid)], &little_body);
    let little_vaa_hex = hex::encode(little_vaa);

    let l_res = std::process::Command::new("stellar")
        .args([
            "contract",
            "invoke",
            "--network",
            &ctx.network,
            "--source",
            &ctx.admin_identity,
            "--id",
            &contract_id,
            "--",
            "submit_transfer_fees",
            "--vaa_bytes",
            &little_vaa_hex,
        ])
        .output()
        .expect("Failed to execute submit_transfer_fees");

    assert!(
        !l_res.status.success(),
        "Transfer should fail when leaving too little balance"
    );
    let l_stderr = String::from_utf8_lossy(&l_res.stderr);
    assert!(
        l_stderr.contains("InsufficientFees") || l_stderr.contains("Error(Contract, #51)"),
        "Error should be InsufficientFees (51)"
    );
    println!("Transfer failed as expected when leaving too little balance.");

    println!("Integration test passed!");
}
