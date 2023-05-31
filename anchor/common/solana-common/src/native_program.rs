use anchor_lang::solana_program;

#[derive(Debug, Clone)]
pub struct BpfLoaderUpgradeable;

impl anchor_lang::Id for BpfLoaderUpgradeable {
    fn id() -> solana_program::pubkey::Pubkey {
        solana_program::bpf_loader_upgradeable::id()
    }
}
