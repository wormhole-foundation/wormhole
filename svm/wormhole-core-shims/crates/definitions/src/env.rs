#[macro_export]
macro_rules! env_pubkey {
    ($name: literal) => {
        const_crypto::bs58::decode_pubkey(env!($name))
    };
}
