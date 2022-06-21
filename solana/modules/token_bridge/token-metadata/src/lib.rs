// The solana_program::declare_id! macro generates spurious import statements.
#[allow(unused_imports)]
pub mod instruction;
pub mod state;
pub mod utils;

solana_program::declare_id!("metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s");
