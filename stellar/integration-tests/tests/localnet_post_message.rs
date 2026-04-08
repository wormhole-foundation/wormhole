use integration_tests::{TestContext, find_event};

#[test]
#[ignore]
fn integration_post_message_no_fee_flow() {
    let ctx = TestContext::new();

    println!("Deploying contract...");
    let guardian = "0101010101010101010101010101010101010101";
    let gov_emitter = "0404040404040404040404040404040404040404040404040404040404040404";
    let contract_id = ctx.deploy_contract(&[guardian.to_string()], gov_emitter);
    println!("Contract deployed at: {}", contract_id);

    let emitter_identity = "test_emitter";
    println!("Setting up emitter identity: {}", emitter_identity);
    let emitter_addr = ctx.setup_identity(emitter_identity);
    println!("Emitter address: {}", emitter_addr);

    println!("Posting first message...");
    let seq_out = ctx.invoke(
        emitter_identity,
        &contract_id,
        "post_message",
        &[
            "--emitter",
            &emitter_addr,
            "--nonce",
            "123",
            "--payload",
            "abcdef",
            "--consistency_level",
            "1",
        ],
    );
    let sequence = seq_out
        .trim()
        .parse::<u64>()
        .expect("Failed to parse sequence number");
    assert_eq!(sequence, 0, "First sequence should be 0");
    println!("First message posted. Sequence: {}", sequence);

    println!("Verifying first message events and state...");
    let next_seq_out = ctx.invoke(
        &ctx.admin_identity,
        &contract_id,
        "get_emitter_sequence",
        &["--emitter", &emitter_addr],
    );
    let next_sequence = next_seq_out
        .trim()
        .parse::<u64>()
        .expect("Failed to parse next sequence");
    assert_eq!(next_sequence, 1, "Next sequence should be 1");

    // Fetch events via RPC
    let found_event = find_event(
        &ctx.rpc_url,
        &contract_id,
        &[
            vec![
                "message_published",
                "AAAADwAAABFtZXNzYWdlX3B1Ymxpc2hlZAAAAA==",
            ],
            vec!["wormhole", "AAAADwAAAAh3b3JtaG9sZQ=="],
        ],
    );
    assert!(found_event, "MessagePublished event not found");

    println!("Posting second message...");
    let seq_out2 = ctx.invoke(
        emitter_identity,
        &contract_id,
        "post_message",
        &[
            "--emitter",
            &emitter_addr,
            "--nonce",
            "124",
            "--payload",
            "010203",
            "--consistency_level",
            "1",
        ],
    );
    let sequence2 = seq_out2
        .trim()
        .parse::<u64>()
        .expect("Failed to parse second sequence number");
    assert_eq!(sequence2, 1, "Second sequence should be 1");
    println!("Second message posted. Sequence: {}", sequence2);

    let next_seq_out2 = ctx.invoke(
        &ctx.admin_identity,
        &contract_id,
        "get_emitter_sequence",
        &["--emitter", &emitter_addr],
    );
    let next_sequence2 = next_seq_out2
        .trim()
        .parse::<u64>()
        .expect("Failed to parse next sequence 2");
    assert_eq!(
        next_sequence2, 2,
        "Next sequence should be 2 after second post"
    );
    println!("Integration test passed!");
}
