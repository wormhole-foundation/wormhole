;; SPDX-License-Identifier: BUSL-1.1

(define-constant err-not-authorised (err u1000))
(define-constant err-no-locked-liquidity (err u1100))
(define-constant err-end-burn-block (err u1101))

(define-constant MAX_UINT u340282366920938463463374607431768211455)

(define-data-var lock-period uint u26280) ;; c. 6 months
(define-map locked-liquidity { owner: principal, pool-id: uint } { amount: uint, end-burn-block: uint })
(define-map locked-liquidity-owners uint (list 200 principal))
(define-map burnt-liquidity uint uint)

;; read-only calls

(define-read-only (is-dao-or-extension)
  (ok (asserts! (or (is-eq tx-sender 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.executor-dao) (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.executor-dao is-extension contract-caller)) err-not-authorised)))

(define-read-only (get-lock-period)
	(var-get lock-period))

(define-read-only (get-locked-liquidity-or-default (owner principal) (pool-id uint))
	(default-to { amount: u0, end-burn-block: MAX_UINT } (map-get? locked-liquidity { owner: owner, pool-id: pool-id })))

(define-read-only (get-locked-liquidity-for-pool-or-default (pool-id uint))
  (let (
      (owners (default-to (list) (map-get? locked-liquidity-owners pool-id)))
      (initial-state { pool-id: pool-id, result: (list) }))
    (get result (fold accumulate-locked-liquidity owners initial-state))))

(define-read-only (get-burnt-liquidity-or-default (pool-id uint))
	(default-to u0 (map-get? burnt-liquidity pool-id)))

;; public calls

(define-public (lock-liquidity (amount uint) (pool-id uint))
	(begin
		(try! (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.token-amm-pool-v2-01 transfer-fixed pool-id amount tx-sender (as-contract tx-sender)))
		(map-set locked-liquidity { owner: tx-sender, pool-id: pool-id } { amount: (+ (get amount (get-locked-liquidity-or-default tx-sender pool-id)) amount), end-burn-block: (+ burn-block-height (get-lock-period)) })
		(add-owner-to-locked-liquidity pool-id tx-sender)
		(print { notification: "lock-liquidity", payload: { sender: tx-sender, amount: amount, pool-id: pool-id, end-burn-block: (+ burn-block-height (get-lock-period)) }})
		(ok true)))

(define-public (burn-liquidity (amount uint) (pool-id uint))
 (let (
		(sender tx-sender)
		(current-burnt-amount (get-burnt-liquidity-or-default pool-id)))
 	(as-contract (try! (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.token-amm-pool-v2-01 burn-fixed pool-id amount sender)))
  (map-set burnt-liquidity pool-id (+ current-burnt-amount amount))
	(print { notification: "burn-liquidity", payload: { sender: sender, amount: amount, pool-id: pool-id, total-burnt: (+ current-burnt-amount amount) }})
	(ok true)))

(define-public (claim-liquidity (pool-id uint))
	(let (
			(sender tx-sender)
			(liquidity-details (get-locked-liquidity-or-default sender pool-id)))
		(asserts! (> (get amount liquidity-details) u0) err-no-locked-liquidity)
		(asserts! (> burn-block-height (get end-burn-block liquidity-details)) err-end-burn-block)
		(as-contract (try! (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.token-amm-pool-v2-01 transfer-fixed pool-id (get amount liquidity-details) tx-sender sender)))
		(map-delete locked-liquidity { owner: sender, pool-id: pool-id })
		(print { notification: "claim-liquidity", payload: liquidity-details, sender: sender, pool-id: pool-id })
		(ok true)))

;; governance calls

(define-public (set-lock-period (new-lock-period uint))
	(begin
		(try! (is-dao-or-extension))
		(ok (var-set lock-period new-lock-period))))

(define-public (set-locked-liquidity (updates (list 200 { owner: principal, pool-id: uint, amount: uint, end-burn-block: uint })))
	(begin
		(try! (is-dao-or-extension))
		(ok (fold update-locked-liquidity updates true))))

(define-public (set-burnt-liquidity (updates (list 200 { pool-id: uint, burnt-liquidity: uint })))
  (begin
    (try! (is-dao-or-extension))
    (ok (fold update-burnt-liquidity updates true))))

;; private calls

(define-private (update-locked-liquidity (entry { owner: principal, pool-id: uint, amount: uint, end-burn-block: uint }) (previous-result bool))
  (begin
    (map-set locked-liquidity { owner: (get owner entry), pool-id: (get pool-id entry) } { amount: (get amount entry), end-burn-block: (get end-burn-block entry) })
    (add-owner-to-locked-liquidity (get pool-id entry) (get owner entry))
		(print { notification: "manual-set-locked-liquidity", payload: { owner: (get owner entry), pool-id: (get pool-id entry), amount: (get amount entry), end-burn-block: (get end-burn-block entry) }})
    true))

(define-private (update-burnt-liquidity (entry { pool-id: uint, burnt-liquidity: uint }) (previous-result bool))
  (begin
      (map-set burnt-liquidity (get pool-id entry) (get burnt-liquidity entry))
      (print { notification: "manual-set-burn-liquidity", payload: { amount: (get burnt-liquidity entry), pool-id: (get pool-id entry) }})
      true))

(define-private (add-owner-to-locked-liquidity (pool-id uint) (owner principal))
	(let (
			(current-owners (default-to (list) (map-get? locked-liquidity-owners pool-id))))
		(and (is-none (index-of current-owners owner)) (map-set locked-liquidity-owners pool-id (unwrap-panic (as-max-len? (append current-owners owner) u200))))))

(define-private (accumulate-locked-liquidity (owner principal) (state { pool-id: uint, result: (list 200 { owner: principal, amount: uint, end-burn-block: uint }) }))
  (let (
      (locked-info (get-locked-liquidity-or-default owner (get pool-id state)))
      (amount (get amount locked-info))
      (end-burn-block (get end-burn-block locked-info))
      (new-entry { owner: owner, amount: amount, end-burn-block: end-burn-block }))
    { pool-id: (get pool-id state), result: (unwrap-panic (as-max-len? (append (get result state) new-entry) u200)) }))

