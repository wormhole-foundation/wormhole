use cw_multi_test::App;

use crate::interface::CounterContract;

#[test]
fn proper_initialization() {
    let mut app = App::default();
    let code_id = CounterContract::store_code(&mut app);

    let contract = CounterContract::instantiate(&mut app, code_id, "owner", "Counter 1").unwrap();

    let count = contract.query_count(&app).unwrap();
    assert_eq!(0, count);
}

#[test]
fn increment_works() {
    let mut app = App::default();
    let code_id = CounterContract::store_code(&mut app);

    // Instantiate with count 0
    let contract = CounterContract::instantiate(&mut app, code_id, "owner", "Counter 1").unwrap();

    // Anyone can increment
    contract.increment(&mut app, "anyone").unwrap();
    let count = contract.query_count(&app).unwrap();
    assert_eq!(1, count);

    // Multiple increments
    contract.increment(&mut app, "anyone").unwrap();
    contract.increment(&mut app, "anyone").unwrap();
    let count = contract.query_count(&app).unwrap();
    assert_eq!(3, count);
}

#[test]
fn reset_works() {
    let mut app = App::default();
    let code_id = CounterContract::store_code(&mut app);

    // Instantiate with count 5
    let contract = CounterContract::instantiate(&mut app, code_id, "owner", "Counter 1").unwrap();

    // Increment the counter
    contract.increment(&mut app, "anyone").unwrap();
    contract.increment(&mut app, "anyone").unwrap();

    let count = contract.query_count(&app).unwrap();
    assert_eq!(2, count);

    // Can reset to same value
    contract.reset(&mut app, "anyone").unwrap();
    let count = contract.query_count(&app).unwrap();
    assert_eq!(0, count);

    // Can reset to lower value
    contract.reset(&mut app, "anyone").unwrap();
    let count = contract.query_count(&app).unwrap();
    assert_eq!(0, count);
}

#[test]
fn multiple_counters() {
    let mut app = App::default();
    let code_id = CounterContract::store_code(&mut app);

    // Create two counters
    let contract1 = CounterContract::instantiate(&mut app, code_id, "owner", "Counter 1").unwrap();

    let contract2 = CounterContract::instantiate(&mut app, code_id, "owner", "Counter 2").unwrap();

    // increment contract1 5 times
    for _ in 0..5 {
        contract1.increment(&mut app, "anyone").unwrap();
    }

    // increment contract2 10 times
    for _ in 0..10 {
        contract2.increment(&mut app, "anyone").unwrap();
    }

    // Check initial values
    assert_eq!(5, contract1.query_count(&app).unwrap());
    assert_eq!(10, contract2.query_count(&app).unwrap());

    // Increment first counter
    contract1.increment(&mut app, "anyone").unwrap();
    assert_eq!(6, contract1.query_count(&app).unwrap());
    assert_eq!(10, contract2.query_count(&app).unwrap());

    // Reset second counter
    contract2.reset(&mut app, "anyone").unwrap();
    assert_eq!(6, contract1.query_count(&app).unwrap());
    assert_eq!(0, contract2.query_count(&app).unwrap());
}
