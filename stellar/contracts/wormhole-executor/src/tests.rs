use super::*;
use soroban_sdk::{
    Address, Bytes, BytesN, Env, Event, String, contract, contractimpl, contracttype,
    testutils::{Address as _, Events, Ledger},
};

const SIGNED_QUOTE_PREFIX: [u8; 4] = *b"EQ01";

#[contracttype]
#[derive(Clone)]
enum MockTokenStorage {
    Balance(Address),
}

#[contract]
struct MockNativeToken;

#[contractimpl]
impl MockNativeToken {
    pub fn set_balance(env: Env, id: Address, amount: i128) {
        env.storage()
            .persistent()
            .set(&MockTokenStorage::Balance(id), &amount);
    }

    pub fn balance(env: Env, id: Address) -> i128 {
        env.storage()
            .persistent()
            .get(&MockTokenStorage::Balance(id))
            .unwrap_or(0)
    }

    pub fn transfer(env: Env, from: Address, to: Address, amount: i128) {
        let from_balance = Self::balance(env.clone(), from.clone());
        assert!(amount >= 0, "negative transfer");
        assert!(from_balance >= amount, "insufficient");

        let to_balance = Self::balance(env.clone(), to.clone());
        Self::set_balance(env.clone(), from, from_balance - amount);
        Self::set_balance(env, to, to_balance + amount);
    }
}

fn install_native_token_mock(env: &Env) -> MockNativeTokenClient<'_> {
    let native = Address::from_string(&String::from_str(env, NATIVE_TOKEN_ADDRESS));
    env.register_at(&native, MockNativeToken, ());
    MockNativeTokenClient::new(env, &native)
}

fn register_executor(env: &Env, chain_id: u32) -> ExecutorClient<'_> {
    let exec_addr = env.register(Executor, (&chain_id,));
    ExecutorClient::new(env, &exec_addr)
}

fn mk_quote(env: &Env, payee: Address, src_chain: u16, dst_chain: u16, expiry: u64) -> SignedQuote {
    SignedQuote {
        prefix: BytesN::from_array(env, &SIGNED_QUOTE_PREFIX),
        quoter: Address::generate(env),
        payee,
        src_chain: u32::from(src_chain),
        dst_chain: u32::from(dst_chain),
        expiry,
    }
}

#[test]
fn init_roundtrip_and_version() {
    let env = Env::default();
    env.mock_all_auths();

    let client = register_executor(&env, 1234);

    assert_eq!(client.chain_id(), 1234);
    assert_eq!(
        client.executor_version(),
        String::from_str(&env, EXECUTOR_VERSION)
    );
}

#[test]
fn request_happy_path_pays_quote_payee_and_emits_event() {
    let env = Env::default();
    env.mock_all_auths();
    let native = install_native_token_mock(&env);

    let src_chain = 1234u16;
    let dst_chain = 4321u16;
    let payee = Address::generate(&env);
    let payer = Address::generate(&env);
    let refund = Address::generate(&env);
    let dst_addr_wa32 = BytesN::<32>::from_array(&env, &[9u8; 32]);
    let amount = 250i128;

    native.set_balance(&payer, &1_000);

    let exec_addr = env.register(Executor, (&(src_chain as u32),));
    let client = ExecutorClient::new(&env, &exec_addr);
    let signed_quote = mk_quote(
        &env,
        payee.clone(),
        src_chain,
        dst_chain,
        env.ledger().timestamp() + 600,
    );
    let request = Bytes::from_slice(&env, b"any-request-bytes");
    let relay_instructions = Bytes::from_slice(&env, &[0xCA, 0xFE]);

    client.request_execution(
        &(dst_chain as u32),
        &dst_addr_wa32,
        &refund,
        &payer,
        &amount,
        &signed_quote,
        &request,
        &relay_instructions,
    );

    let expected = RequestForExecution {
        quoter: signed_quote.quoter.clone(),
        amt_paid: amount,
        dst_chain: dst_chain as u32,
        dst_addr_wa32: dst_addr_wa32.clone(),
        refund: refund.clone(),
        signed_quote: signed_quote.clone(),
        request: request.clone(),
        relay_instructions: relay_instructions.clone(),
    };

    assert_eq!(env.events().all(), [expected.to_xdr(&env, &exec_addr)]);
    assert_eq!(native.balance(&payer), 750);
    assert_eq!(native.balance(&payee), amount);
}

#[test]
fn request_accepts_empty_request_like_the_solidity_reference() {
    let env = Env::default();
    env.mock_all_auths();
    let native = install_native_token_mock(&env);

    let src_chain = 10u16;
    let dst_chain = 20u16;
    let payee = Address::generate(&env);
    let payer = Address::generate(&env);
    let refund = Address::generate(&env);
    let dst_addr_wa32 = BytesN::<32>::from_array(&env, &[1u8; 32]);

    native.set_balance(&payer, &99);

    let client = register_executor(&env, src_chain as u32);
    let signed_quote = mk_quote(
        &env,
        payee.clone(),
        src_chain,
        dst_chain,
        env.ledger().timestamp() + 60,
    );

    let res = client.try_request_execution(
        &(dst_chain as u32),
        &dst_addr_wa32,
        &refund,
        &payer,
        &42,
        &signed_quote,
        &Bytes::new(&env),
        &Bytes::new(&env),
    );

    assert_eq!(res, Ok(Ok(())));
    assert_eq!(native.balance(&payee), 42);
}

#[test]
#[should_panic(expected = "Error(Contract, #11)")] // QuoteExpired
fn request_rejects_expired_quote() {
    let env = Env::default();
    env.mock_all_auths();
    env.ledger().with_mut(|li| li.timestamp = 1000);
    install_native_token_mock(&env);

    let src_chain = 111u16;
    let dst_chain = 222u16;
    let payer = Address::generate(&env);
    let refund = Address::generate(&env);
    let dst_addr_wa32 = BytesN::<32>::from_array(&env, &[3u8; 32]);

    let client = register_executor(&env, src_chain as u32);
    let signed_quote = mk_quote(&env, Address::generate(&env), src_chain, dst_chain, 1000);

    client.request_execution(
        &(dst_chain as u32),
        &dst_addr_wa32,
        &refund,
        &payer,
        &1,
        &signed_quote,
        &Bytes::from_slice(&env, b"request"),
        &Bytes::new(&env),
    );
}

#[test]
#[should_panic(expected = "Error(Contract, #12)")] // QuoteSrcChainMismatch
fn request_rejects_source_chain_mismatch() {
    let env = Env::default();
    env.mock_all_auths();
    install_native_token_mock(&env);

    let src_chain = 77u16;
    let dst_chain = 88u16;
    let payer = Address::generate(&env);
    let refund = Address::generate(&env);
    let dst_addr_wa32 = BytesN::<32>::from_array(&env, &[4u8; 32]);

    let client = register_executor(&env, 9999);
    let signed_quote = mk_quote(
        &env,
        Address::generate(&env),
        src_chain,
        dst_chain,
        env.ledger().timestamp() + 600,
    );

    client.request_execution(
        &(dst_chain as u32),
        &dst_addr_wa32,
        &refund,
        &payer,
        &1,
        &signed_quote,
        &Bytes::from_slice(&env, b"request"),
        &Bytes::new(&env),
    );
}

#[test]
#[should_panic(expected = "Error(Contract, #13)")] // QuoteDstChainMismatch
fn request_rejects_dst_chain_mismatch() {
    let env = Env::default();
    env.mock_all_auths();
    install_native_token_mock(&env);

    let src_chain = 55u16;
    let dst_chain = 66u16;
    let payer = Address::generate(&env);
    let refund = Address::generate(&env);
    let dst_addr_wa32 = BytesN::<32>::from_array(&env, &[5u8; 32]);

    let client = register_executor(&env, src_chain as u32);
    let signed_quote = mk_quote(
        &env,
        Address::generate(&env),
        src_chain,
        dst_chain,
        env.ledger().timestamp() + 600,
    );

    client.request_execution(
        &99u32,
        &dst_addr_wa32,
        &refund,
        &payer,
        &1,
        &signed_quote,
        &Bytes::from_slice(&env, b"request"),
        &Bytes::new(&env),
    );
}

#[test]
#[should_panic(expected = "Error(Contract, #14)")] // InvalidAmount
fn request_rejects_negative_amount() {
    let env = Env::default();
    env.mock_all_auths();
    install_native_token_mock(&env);

    let src_chain = 20u16;
    let dst_chain = 30u16;
    let payer = Address::generate(&env);
    let refund = Address::generate(&env);
    let dst_addr_wa32 = BytesN::<32>::from_array(&env, &[9u8; 32]);

    let client = register_executor(&env, src_chain as u32);
    let signed_quote = mk_quote(
        &env,
        Address::generate(&env),
        src_chain,
        dst_chain,
        env.ledger().timestamp() + 600,
    );

    client.request_execution(
        &(dst_chain as u32),
        &dst_addr_wa32,
        &refund,
        &payer,
        &-1,
        &signed_quote,
        &Bytes::from_slice(&env, b"request"),
        &Bytes::new(&env),
    );
}
