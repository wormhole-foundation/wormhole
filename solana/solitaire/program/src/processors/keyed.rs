use solana_program::sysvar::Sysvar as SolanaSysvar;

use crate::{
    processors::seeded::Owned,
    AccountState,
    Data,
    Derive,
    Info,
    Mut,
    Signer,
    System,
    Sysvar,
};

pub trait Keyed<'a, 'b: 'a> {
    fn info(&'a self) -> &Info<'b>;
}

impl<'a, 'b: 'a, T: Owned + Default, const IS_INITIALIZED: AccountState> Keyed<'a, 'b>
    for Data<'b, T, IS_INITIALIZED>
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

impl<'a, 'b: 'a, Var: SolanaSysvar> Keyed<'a, 'b> for Sysvar<'b, Var> {
    fn info(&'a self) -> &'a Info<'b> {
        &self.0
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

impl<'a, 'b: 'a, T, const SEED: &'static str> Keyed<'a, 'b> for Derive<T, SEED>
where
    T: Keyed<'a, 'b>,
{
    fn info(&'a self) -> &'a Info<'b> {
        self.0.info()
    }
}

impl<'a, 'b: 'a, T> Keyed<'a, 'b> for Mut<T>
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
