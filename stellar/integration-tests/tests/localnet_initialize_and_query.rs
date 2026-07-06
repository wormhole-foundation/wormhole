use integration_tests::{TestContext, find_event};

#[test]
#[ignore]
fn testnet_initialize_and_query() {
    let ctx = TestContext::new();

    ctx.fund_identity(&ctx.admin_identity);

    // Deploy with constructor (initializes in one step)
    let guardian1 = "0101010101010101010101010101010101010101";
    let guardian2 = "0202020202020202020202020202020202020202";
    let emitter = "0404040404040404040404040404040404040404040404040404040404040404";
    let contract_id = ctx.deploy_contract(&[guardian1.to_string(), guardian2.to_string()], emitter);

    // Query state (no is_initialized; verify via guardian set and gov emitter)
    assert!(
        ctx.invoke(
            &ctx.admin_identity,
            &contract_id,
            "get_current_guardian_set_index",
            &[]
        )
        .trim()
        .contains("0")
    );

    let gset = ctx.invoke(
        &ctx.admin_identity,
        &contract_id,
        "get_guardian_set",
        &["--index", "0"],
    );
    assert!(gset.contains("010101"));
    assert!(gset.contains("020202"));

    let gov = ctx.invoke(
        &ctx.admin_identity,
        &contract_id,
        "get_governance_emitter",
        &[],
    );
    assert!(gov.contains("040404"));

    // Fetch events through RPC
    let found_init = find_event(
        &ctx.rpc_url,
        &contract_id,
        &[
            vec!["wormhole_core", "AAAADwAAAA13b3JtaG9sZV9jb3JlAAAA"],
            vec!["init", "AAAADwAAAARpbml0"],
        ],
    );

    assert!(found_init, "Expected init event not found");
}
