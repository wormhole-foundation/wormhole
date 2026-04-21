//! Wormhole Executor contract implementation for Stellar/Soroban.
//!
//! The Executor is a simple on-chain payment rail: a `payer` prepays a relayer
//! (identified by the `quoter`/`payee` pair inside an off-chain-signed quote)
//! to execute a cross-chain delivery request on a destination chain. The
//! actual relaying happens off-chain; this contract only records the payment
//! and emits the event that off-chain relayers consume.
//!
//! # On-chain responsibilities
//!
//! The contract deliberately does very little. On a call to
//! [`ExecutorInterface::request_execution`] it:
//!
//! 1. Accepts a [`SignedQuote`] and a delivery request from the payer.
//! 2. Validates basic sanity: `amount >= 0`, the quote's `src_chain` matches
//!    the chain id configured at construction, the quote's `dst_chain` matches
//!    the `dst_chain` argument, and the quote has not expired relative to the
//!    current ledger timestamp.
//! 3. Requires the `payer`'s authorization, then transfers `amount` of the
//!    native token (XLM, via the Stellar Asset Contract at
//!    `NATIVE_TOKEN_ADDRESS`) from `payer` to `signed_quote.payee`.
//! 4. Emits a [`RequestForExecution`] event that off-chain relayers consume to
//!    fulfil the delivery on the destination chain.
//!
//! # Quote authentication is NOT performed on-chain
//!
//! Despite its name, [`SignedQuote`] is **not** verified by this contract.
//! Neither the [`SignedQuote::prefix`] domain tag (e.g. `EQ01`) nor the
//! `quoter`'s cryptographic signature over the quote are checked here. This
//! is intentional and matches the reference EVM implementation: quote
//! authentication is a caller-side responsibility, performed off-chain by
//! the relayer SDK before the transaction is submitted. The on-chain
//! contract only enforces chain-id, expiry and amount checks plus the
//! `payer`'s auth for the token transfer.
//!
//! # Architecture
//!
//! - [`Executor`] - Main contract struct implementing [`ExecutorInterface`]
//! - [`ExecutorInterface`] - Public interface (re-exported from
//!   `wormhole-soroban-client`)
//! - [`SignedQuote`] - Off-chain quote payload, NOT verified on-chain
//!   (re-exported from `wormhole-soroban-client`)
//! - [`RequestForExecution`] - Event emitted on successful requests
//! - [`ExecutorError`] - Error codes returned by `request_execution`

#![no_std]

use soroban_sdk::{
    Address, Bytes, BytesN, Env, String, contract, contractevent, contractimpl, contracttype, token,
};
use wormhole_soroban_client::{
    ExecutorError, ExecutorInterface, NATIVE_TOKEN_ADDRESS, SignedQuote,
};

#[cfg(test)]
mod tests;

const EXECUTOR_VERSION: &str = "Executor-0.0.1";

/// Instance storage keys for the [`Executor`] contract.
#[contracttype]
#[derive(Clone)]
pub enum DataKey {
    /// Wormhole chain id of this deployment, set once by
    /// [`Executor::__constructor`] and read on every call to
    /// [`ExecutorInterface::request_execution`] to validate
    /// [`SignedQuote::src_chain`].
    ChainId,
}

/// Event emitted when a payer successfully requests a cross-chain delivery.
///
/// Published by [`ExecutorInterface::request_execution`] after the native
/// token transfer has completed. Off-chain relayers subscribe to this event
/// to learn that they have been paid and must fulfil the associated delivery
/// on the destination chain.
///
/// The event is published with topics `["Executor", "RequestForExecution"]`.
#[contractevent(topics = ["Executor", "RequestForExecution"])]
#[derive(Clone)]
pub struct RequestForExecution {
    /// Address of the quoter that authored the associated [`SignedQuote`].
    /// Mirrors `signed_quote.quoter` for off-chain indexers that only
    /// inspect top-level event fields.
    pub quoter: Address,
    /// Amount of native token (stroops) paid by the payer to
    /// `signed_quote.payee` for this request.
    pub amt_paid: i128,
    /// Wormhole chain id of the destination chain, as passed to
    /// [`ExecutorInterface::request_execution`].
    pub dst_chain: u32,
    /// 32-byte destination address on the destination chain, in Wormhole's
    /// left-zero-padded 32-byte encoding. Pass-through; not validated on
    /// chain.
    pub dst_addr_wa32: BytesN<32>,
    /// Address that the off-chain relayer should refund if the delivery
    /// cannot be completed. Pass-through metadata; this contract never
    /// reads it.
    pub refund: Address,
    /// The full [`SignedQuote`] submitted by the payer, recorded verbatim
    /// in the event for off-chain consumption.
    pub signed_quote: SignedQuote,
    /// Opaque delivery request payload whose format is defined by the
    /// off-chain relayer protocol. Not interpreted on chain.
    pub request: Bytes,
    /// Opaque relaying instructions whose format is defined by the
    /// off-chain relayer protocol. Not interpreted on chain.
    pub relay_instructions: Bytes,
}

fn get_native_token_address(env: &Env) -> Address {
    Address::from_string(&String::from_str(env, NATIVE_TOKEN_ADDRESS))
}

/// Wormhole Executor contract for Stellar/Soroban.
///
/// Implements [`ExecutorInterface`]. See the crate-level documentation for
/// the Executor's role as a prepaid cross-chain delivery payment rail and
/// for the important note that quote authentication is **not** performed on
/// chain.
#[contract]
pub struct Executor;

#[contractimpl]
impl Executor {
    /// Constructor called atomically during contract deployment.
    ///
    /// Stores the Wormhole `chain_id` of this deployment in instance
    /// storage. The value is read on every call to
    /// [`ExecutorInterface::request_execution`] to validate
    /// [`SignedQuote::src_chain`].
    ///
    /// # Arguments
    ///
    /// * `chain_id` - Wormhole chain id this Executor instance runs on.
    pub fn __constructor(env: Env, chain_id: u32) {
        env.storage().instance().set(&DataKey::ChainId, &chain_id);
    }
}

#[contractimpl]
impl ExecutorInterface for Executor {
    fn chain_id(env: Env) -> u32 {
        env.storage().instance().get(&DataKey::ChainId).unwrap()
    }

    fn executor_version(env: Env) -> String {
        String::from_str(&env, EXECUTOR_VERSION)
    }

    #[allow(clippy::too_many_arguments)]
    fn request_execution(
        env: Env,
        dst_chain: u32,
        dst_addr_wa32: BytesN<32>,
        refund: Address,
        payer: Address,
        amount: i128,
        signed_quote: SignedQuote,
        request: Bytes,
        relay_instructions: Bytes,
    ) -> Result<(), ExecutorError> {
        if amount < 0 {
            return Err(ExecutorError::InvalidAmount);
        }

        let this_chain = Self::chain_id(env.clone());

        if signed_quote.src_chain != this_chain {
            return Err(ExecutorError::QuoteSrcChainMismatch);
        }
        if signed_quote.dst_chain != dst_chain {
            return Err(ExecutorError::QuoteDstChainMismatch);
        }
        if signed_quote.expiry <= env.ledger().timestamp() {
            return Err(ExecutorError::QuoteExpired);
        }

        payer.require_auth();

        let event = RequestForExecution {
            quoter: signed_quote.quoter.clone(),
            amt_paid: amount,
            dst_chain,
            dst_addr_wa32,
            refund,
            signed_quote,
            request,
            relay_instructions,
        };

        let native_token = get_native_token_address(&env);
        let token_client = token::TokenClient::new(&env, &native_token);
        token_client.transfer(&payer, &event.signed_quote.payee, &amount);
        event.publish(&env);

        Ok(())
    }
}
