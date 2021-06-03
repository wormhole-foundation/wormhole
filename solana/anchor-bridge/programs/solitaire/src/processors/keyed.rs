use solana_program::pubkey::Pubkey;

use crate::{
    processors::seeded::Owned,
    Data,
    Derive,
    Info,
    Signer,
    System,
    Sysvar,
};

pub trait Keyed<'a, 'b: 'a> {
    fn info(&'a self) -> &Info<'b>;
}

impl<'a, 'b: 'a, T: Owned, const IsInitialized: bool, const Lazy: bool> Keyed<'a, 'b>
    for Data<'b, T, IsInitialized, Lazy>
{
    fn info(&'a self) -> &'a Info<'b> {
        &self.0
    }
}

impl<'a, 'b: 'a, T> Keyed<'a, 'b> for Signer<T>
where
    T: Keyed<'a, 'b>,
{
    fn info(&'a self) -> &'a Info<'b> {
        self.0.info()
    }
}

impl<'a, 'b: 'a, T, Var> Keyed<'a, 'b> for Sysvar<T, Var>
where
    T: Keyed<'a, 'b>,
{
    fn info(&'a self) -> &'a Info<'b> {
        self.0.info()
    }
}

impl<'a, 'b: 'a, T> Keyed<'a, 'b> for System<T>
where
    T: Keyed<'a, 'b>,
{
    fn info(&'a self) -> &'a Info<'b> {
        self.0.info()
    }
}

impl<'a, 'b: 'a, T, const Seed: &'static str> Keyed<'a, 'b> for Derive<T, Seed>
where
    T: Keyed<'a, 'b>,
{
    fn info(&'a self) -> &'a Info<'b> {
        self.0.info()
    }
}

impl<'a, 'b: 'a> Keyed<'a, 'b> for Info<'b> {
    fn info(&'a self) -> &'a Info<'b> {
        self
    }
}
