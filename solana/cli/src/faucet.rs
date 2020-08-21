use crate::CommmandResult;
use solana_client::rpc_client::RpcClient;
use solana_faucet::faucet::request_airdrop_transaction;
use solana_sdk::hash::Hash;
use solana_sdk::pubkey::Pubkey;
use solana_sdk::signature::{Signature, Signer, SignerError};
use solana_sdk::transaction::Transaction;
use std::error;
use std::net::SocketAddr;
use std::thread::sleep;
use std::time::Duration;

// Quick and dirty Keypair that assumes the client will do retries but not update the
// blockhash. If the client updates the blockhash, the signature will be invalid.
struct FaucetKeypair {
    transaction: Transaction,
}

impl FaucetKeypair {
    fn new_keypair(
        faucet_addr: &SocketAddr,
        to_pubkey: &Pubkey,
        lamports: u64,
        blockhash: Hash,
    ) -> Result<Self, Box<dyn error::Error>> {
        let transaction = request_airdrop_transaction(faucet_addr, to_pubkey, lamports, blockhash)?;
        Ok(Self { transaction })
    }

    fn airdrop_transaction(&self) -> Transaction {
        self.transaction.clone()
    }
}

impl Signer for FaucetKeypair {
    /// Return the public key of the keypair used to sign votes
    fn pubkey(&self) -> Pubkey {
        self.transaction.message().account_keys[0]
    }

    fn try_pubkey(&self) -> Result<Pubkey, SignerError> {
        Ok(self.pubkey())
    }

    fn sign_message(&self, _msg: &[u8]) -> Signature {
        self.transaction.signatures[0]
    }

    fn try_sign_message(&self, message: &[u8]) -> Result<Signature, SignerError> {
        Ok(self.sign_message(message))
    }
}

pub fn request_and_confirm_airdrop(
    rpc_client: &RpcClient,
    faucet_addr: &SocketAddr,
    to_pubkey: &Pubkey,
    lamports: u64,
) -> CommmandResult {
    let (blockhash, _fee_calculator) = rpc_client.get_recent_blockhash()?;
    let keypair = {
        let mut retries = 5;
        loop {
            let result = FaucetKeypair::new_keypair(faucet_addr, to_pubkey, lamports, blockhash);
            if result.is_ok() || retries == 0 {
                break result;
            }
            retries -= 1;
            sleep(Duration::from_secs(1));
        }
    }?;
    Ok(Some(keypair.airdrop_transaction()))
}
