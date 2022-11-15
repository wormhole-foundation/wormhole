mod helpers;

use helpers::*;

#[test]
fn instantiate_contract() {
    const COUNT: usize = 5;
    let accounts = create_accounts(COUNT);
    let transfers = create_transfers(COUNT);
    let modifications = create_modifications(COUNT);

    let (_, contract) =
        proper_instantiate(accounts.clone(), transfers.clone(), modifications.clone());

    for a in accounts {
        let balance = contract.query_balance(a.key).unwrap();
        assert_eq!(a.balance, balance);
    }

    for t in transfers {
        let data = contract.query_transfer(t.key).unwrap();
        assert_eq!(t.data, data);
    }

    for m in modifications {
        let data = contract.query_modification(m.sequence).unwrap();
        assert_eq!(m, data);
    }
}
