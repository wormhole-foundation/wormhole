(use-trait ft-trait .trait-sip-010.sip-010-trait)
(define-constant ERR-NOT-AUTHORIZED (err u1000))
(define-constant ERR-INVALID-POOL (err u2001))
(define-constant ERR-INVALID-LIQUIDITY (err u2003))
(define-constant ERR-POOL-ALREADY-EXISTS (err u2000))
(define-constant ERR-PERCENT-GREATER-THAN-ONE (err u5000))
(define-constant ERR-EXCEEDS-MAX-SLIPPAGE (err u2020))
(define-constant ERR-ORACLE-NOT-ENABLED (err u7002))
(define-constant ERR-ORACLE-AVERAGE-BIGGER-THAN-ONE (err u7004))
(define-constant ERR-PAUSED (err u1001))
(define-constant ERR-SWITCH-THRESHOLD-BIGGER-THAN-ONE (err u7005))
(define-constant ERR-NO-LIQUIDITY (err u2002))
(define-constant ERR-MAX-IN-RATIO (err u4001))
(define-constant ERR-MAX-OUT-RATIO (err u4002))
(define-constant ONE_8 u100000000) ;; 8 decimal places
(define-data-var pool-nonce uint u0)
(define-data-var switch-threshold uint u80000000)
(define-data-var max-ratio-limit uint ONE_8)
(define-map blocklist principal bool)
(define-map pools-id-map uint { token-x: principal, token-y: principal, factor: uint })
(define-map pools-data-map
  {
    token-x: principal,
    token-y: principal,
    factor: uint
  }
  {
    pool-id: uint,
    total-supply: uint,
    balance-x: uint,
    balance-y: uint,
    pool-owner: principal,    
    fee-rate-x: uint,
    fee-rate-y: uint,
    fee-rebate: uint,
    oracle-enabled: bool,
    oracle-average: uint,
    oracle-resilient: uint,
    start-block: uint,
    end-block: uint,
    threshold-x: uint,
    threshold-y: uint,
    max-in-ratio: uint,
    max-out-ratio: uint
  }
)
(define-read-only (is-dao-or-extension)
  (ok (asserts! (or (is-eq tx-sender .executor-dao) (contract-call? .executor-dao is-extension contract-caller)) ERR-NOT-AUTHORIZED)))
(define-read-only (is-blocklisted-or-default (sender principal))
	(default-to false (map-get? blocklist sender)))
(define-read-only (get-switch-threshold)
    (var-get switch-threshold))
(define-read-only (get-max-ratio-limit)
    (var-get max-ratio-limit))    
(define-read-only (get-pool-details-by-id (pool-id uint))
    (ok (unwrap! (map-get? pools-id-map pool-id) ERR-INVALID-POOL)))
(define-read-only (get-pool-details (token-x principal) (token-y principal) (factor uint))
    (ok (unwrap! (get-pool-exists token-x token-y factor) ERR-INVALID-POOL)))
(define-read-only (get-pool-exists (token-x principal) (token-y principal) (factor uint))
    (map-get? pools-data-map { token-x: token-x, token-y: token-y, factor: factor }) )
(define-public (set-blocklist-many (blocked-many (list 1000 { sender: principal, blocked: bool })))
	(begin 
		(try! (is-dao-or-extension))
		(ok (map set-blocklist blocked-many))))
(define-public (set-switch-threshold (new-threshold uint))
    (begin 
        (try! (is-dao-or-extension))
        (asserts! (<= new-threshold ONE_8) ERR-SWITCH-THRESHOLD-BIGGER-THAN-ONE)
        (ok (var-set switch-threshold new-threshold))))
(define-public (set-max-ratio-limit (new-limit uint))
    (begin 
        (try! (is-dao-or-extension))
        (ok (var-set max-ratio-limit new-limit))))
(define-public (create-pool (token-x-trait <ft-trait>) (token-y-trait <ft-trait>) (factor uint) (pool-owner principal)) 
    (let (
            (pool-id (+ (var-get pool-nonce) u1))
            (token-x (contract-of token-x-trait))
            (token-y (contract-of token-y-trait))
            (pool-data {
                pool-id: pool-id,
                total-supply: u0,
                balance-x: u0,
                balance-y: u0,
                pool-owner: pool-owner,
                fee-rate-x: u0,
                fee-rate-y: u0,
                fee-rebate: u0,
                oracle-enabled: false,
                oracle-average: u0,
                oracle-resilient: u0,
                start-block: u340282366920938463463374607431768211455,
                end-block: u340282366920938463463374607431768211455,
                threshold-x: u0,
                threshold-y: u0,
                max-in-ratio: u0,
                max-out-ratio: u0
            }))
        (try! (is-dao-or-extension))
        (asserts! (and (is-none (map-get? pools-data-map { token-x: token-x, token-y: token-y, factor: factor })) (is-none (map-get? pools-data-map { token-x: token-y, token-y: token-x, factor: factor }))) ERR-POOL-ALREADY-EXISTS)             
        (map-set pools-data-map { token-x: token-x, token-y: token-y, factor: factor } pool-data)
        (map-set pools-id-map pool-id { token-x: token-x, token-y: token-y, factor: factor })
        (var-set pool-nonce pool-id)
        (print { object: "pool", action: "created", data: pool-data, token-x: token-x, token-y: token-y, factor: factor })
        (ok true)))        
(define-public (update-pool (token-x principal) (token-y principal) (factor uint)
    (pool-data {
        pool-id: uint,
        total-supply: uint,
        balance-x: uint,
        balance-y: uint,
        pool-owner: principal,    
        fee-rate-x: uint,
        fee-rate-y: uint,
        fee-rebate: uint,
        oracle-enabled: bool,
        oracle-average: uint,
        oracle-resilient: uint,
        start-block: uint,
        end-block: uint,
        threshold-x: uint,
        threshold-y: uint,
        max-in-ratio: uint,
        max-out-ratio: uint }))
    (begin
        (try! (is-dao-or-extension))
        (ok (map-set pools-data-map { token-x: token-x, token-y: token-y, factor: factor } pool-data))))
(define-public (set-fee-rebate (token-x principal) (token-y principal) (factor uint) (fee-rebate uint))
    (let  (            
            (pool (try! (get-pool-details token-x token-y factor))))
        (try! (is-dao-or-extension))
        (ok (map-set pools-data-map { token-x: token-x, token-y: token-y, factor: factor } (merge pool { fee-rebate: fee-rebate })))))
(define-public (set-pool-owner (token-x principal) (token-y principal) (factor uint) (pool-owner principal))
    (let (
            (pool (try! (get-pool-details token-x token-y factor))))
        (try! (is-dao-or-extension))
        (ok (map-set pools-data-map { token-x: token-x, token-y: token-y, factor: factor } (merge pool { pool-owner: pool-owner })))))
(define-public (set-start-block (token-x principal) (token-y principal) (factor uint) (new-start-block uint))
    (let (
            (pool (try! (get-pool-details token-x token-y factor))))
        (try! (is-dao-or-extension))
        (ok (map-set  pools-data-map { token-x: token-x, token-y: token-y, factor: factor } (merge pool {start-block: new-start-block})))))    
(define-public (set-end-block (token-x principal) (token-y principal) (factor uint) (new-end-block uint))
    (let (
            (pool (try! (get-pool-details token-x token-y factor))))
        (try! (is-dao-or-extension))
        (ok (map-set pools-data-map { token-x: token-x, token-y: token-y, factor: factor } (merge pool {end-block: new-end-block})))))
(define-public (set-max-in-ratio (token-x principal) (token-y principal) (factor uint) (new-max-in-ratio uint))
    (let (
            (pool (try! (get-pool-details token-x token-y factor))))
        (try! (is-dao-or-extension))
        (asserts! (<= new-max-in-ratio (var-get max-ratio-limit)) ERR-MAX-IN-RATIO)
        (ok (map-set pools-data-map { token-x: token-x, token-y: token-y, factor: factor } (merge pool {max-in-ratio: new-max-in-ratio})))))
(define-public (set-max-out-ratio (token-x principal) (token-y principal) (factor uint) (new-max-out-ratio uint))
    (let (
            (pool (try! (get-pool-details token-x token-y factor))))
        (try! (is-dao-or-extension))
        (asserts! (<= new-max-out-ratio (var-get max-ratio-limit)) ERR-MAX-OUT-RATIO)
        (ok (map-set pools-data-map { token-x: token-x, token-y: token-y, factor: factor } (merge pool {max-out-ratio: new-max-out-ratio})))))
(define-public (set-oracle-enabled (token-x principal) (token-y principal) (factor uint) (enabled bool))
    (let (
            (pool (try! (get-pool-details token-x token-y factor))))
        (try! (is-dao-or-extension))
        (ok (map-set pools-data-map { token-x: token-x, token-y: token-y, factor: factor } (merge pool {oracle-enabled: enabled})))))
(define-public (set-oracle-average (token-x principal) (token-y principal) (factor uint) (new-oracle-average uint))
    (let (
            (pool (try! (get-pool-details token-x token-y factor))))
        (try! (is-dao-or-extension))
        (asserts! (get oracle-enabled pool) ERR-ORACLE-NOT-ENABLED)
        (asserts! (< new-oracle-average ONE_8) ERR-ORACLE-AVERAGE-BIGGER-THAN-ONE)
        (ok (map-set pools-data-map { token-x: token-x, token-y: token-y, factor: factor } (merge pool { oracle-average: new-oracle-average, oracle-resilient: u0 })))))
(define-public (set-threshold-x (token-x principal) (token-y principal) (factor uint) (new-threshold uint))
    (let (
            (pool (try! (get-pool-details token-x token-y factor))))
        (try! (is-dao-or-extension))
        (ok (map-set pools-data-map { token-x: token-x, token-y: token-y, factor: factor } (merge pool { threshold-x: new-threshold })))))
(define-public (set-threshold-y (token-x principal) (token-y principal) (factor uint) (new-threshold uint))
    (let (
            (pool (try! (get-pool-details token-x token-y factor))))
        (try! (is-dao-or-extension))
        (ok (map-set pools-data-map { token-x: token-x, token-y: token-y, factor: factor } (merge pool { threshold-y: new-threshold })))))
(define-public (set-fee-rate-x (token-x principal) (token-y principal) (factor uint) (fee-rate-x uint))
    (let (        
            (pool (try! (get-pool-details token-x token-y factor))))
        (try! (is-dao-or-extension))
        (ok (map-set pools-data-map { token-x: token-x, token-y: token-y, factor: factor } (merge pool { fee-rate-x: fee-rate-x })))))
(define-public (set-fee-rate-y (token-x principal) (token-y principal) (factor uint) (fee-rate-y uint))
    (let (    
            (pool (try! (get-pool-details token-x token-y factor))))
        (try! (is-dao-or-extension))
        (ok (map-set pools-data-map { token-x: token-x, token-y: token-y, factor: factor } (merge pool { fee-rate-y: fee-rate-y })))))
(define-private (set-blocklist (blocked { sender: principal, blocked: bool }))
	(begin
		(print { object: "amm-registry", action: "set-blocklist", payload: blocked }) 
		(ok (map-set blocklist (get sender blocked) (get blocked blocked)))))