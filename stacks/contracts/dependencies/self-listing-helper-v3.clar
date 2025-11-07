;; SPDX-License-Identifier: BUSL-1.1

;; =================================
;; Self Listing Helper V3
;; =================================
;;
;; This contract enables permissionless creation of AMM pools through two main paths:
;;
;; 1. Standard Pool Creation (create):
;;    - For tokens that are already approved in the system
;;    - Requires token-x to be whitelisted and have sufficient balance
;;    - Enforces standard pool parameters and security checks
;;
;; 2. Permissionless Pool Creation (create2):
;;    - Allows creation of pools with new, unregistered tokens
;;    - Uses a verification system to ensure token contract legitimacy:
;;      a. Verifies the token contract deployment on Stacks blockchain
;;      b. Checks contract code matches an approved template
;;      c. Validates deployment proof using Stacks block headers
;;    - The verify-deploy mechanism works by:
;;      1. Taking deployment transaction proof from Stacks blockchain
;;      2. Verifying the contract code matches a whitelisted template
;;      3. Confirming the deployment transaction was properly mined
;;      This ensures only legitimate token contracts can be used
;;
;; Additional Features:
;; - Liquidity locking/burning mechanisms
;; - Fee rebate management
;; - Token approval governance
;; - Pool parameter configuration
;;
;; The contract combines permissioned and permissionless approaches to enable
;; safe, flexible AMM pool creation while maintaining system security.

(use-trait ft-trait 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.trait-sip-010.sip-010-trait)

(define-constant err-not-authorised (err u1000))
(define-constant err-token-not-approved (err u1002))
(define-constant err-insufficient-balance (err u1003))
(define-constant err-pool-exists (err u1004))
(define-constant err-invalid-lock-parameter (err u1005))

(define-constant ONE_8 u100000000)
(define-constant MAX_UINT u340282366920938463463374607431768211455)

(define-constant NONE 0x00)
(define-constant LOCK 0x01)
(define-constant BURN 0x02)

(define-constant tx-version (if is-in-mainnet 0x00 0x80))
(define-constant curr-chain-id (if is-in-mainnet 0x00000001 0x80000000))
(define-constant standard-auth-type 0x04)
(define-constant p2pkh-hash-mode 0x00)
(define-constant pub-key-encoding 0x00)
(define-constant anchor-mode 0x03)
(define-constant post-conditions-mode-allow 0x01)
(define-constant post-conditions 0x00000000)
(define-constant versioned-smart-contract 0x06)
(define-constant clarity-version 0x03)

(define-data-var wrapped-token-template (list 20 (string-ascii 5000)) (list))

(define-map approved-token-x principal { approved: bool, min-x: uint })

(define-data-var fee-rebate uint u50000000)

(define-map wrap-token-map principal principal)

;; read-only calls

(define-read-only (is-dao-or-extension)
  (ok (asserts! (or (is-eq tx-sender 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.executor-dao) (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.executor-dao is-extension contract-caller)) err-not-authorised)))

(define-read-only (get-approved-token-x-or-default (token-x principal))
    (default-to { approved: false, min-x: MAX_UINT } (map-get? approved-token-x token-x)))

(define-read-only (get-fee-rebate)
	(var-get fee-rebate))

(define-read-only (get-lock-period)
	(contract-call? .liquidity-locker get-lock-period))

(define-read-only (get-locked-liquidity-or-default (owner principal) (pool-id uint))
	(contract-call? .liquidity-locker get-locked-liquidity-or-default owner pool-id))

(define-read-only (get-locked-liquidity-for-pool-or-default (pool-id uint))
  (contract-call? .liquidity-locker get-locked-liquidity-for-pool-or-default pool-id))

(define-read-only (get-burnt-liquidity-or-default (pool-id uint))
	(contract-call? .liquidity-locker get-burnt-liquidity-or-default pool-id))

(define-read-only (get-wrapped-token-contract-code (token principal))
  (let ((token-str (unwrap-panic (as-max-len? (principal-to-string token) u100)))
        (template (var-get wrapped-token-template)))
    (get result (fold join-template-parts-iter 
      template 
      {token: token-str, result: ""}))))

(define-read-only (principal-to-string (p principal))
	(let (
			(destructed (match (principal-destruct? p) ok-value ok-value err-value err-value))
			(checksum (unwrap-panic (slice? (sha256 (sha256 (concat (get version destructed) (get hash-bytes destructed)))) u0 u4)))
			(data (unwrap-panic (as-max-len? (concat (get hash-bytes destructed) checksum) u24)))
			(result (concat (concat "S" (unwrap-panic (element-at? C32 (buff-to-uint-be (get version destructed))))) (append-leading-0 data (trim-leading-0 (hash-bytes-to-string data))))))
		(match (get name destructed) n (concat (concat result ".") n) result)))

(define-read-only (verify-deploy
	(verify-params {
		nonce: (buff 8),
		fee-rate: (buff 8),
		signature: (buff 65),
		contract: principal,
		token-y: principal,
		proof: { tx-index: uint, hashes: (list 14 (buff 32)), tree-depth: uint},
		tx-block-height: uint,
		block-header-without-signer-signatures: (buff 712) }))
	(contract-call? .code-body-prover is-contract-deployed 
		(get nonce verify-params)
		(get fee-rate verify-params)
		(get signature verify-params)
		(get contract verify-params)
		(contract-call? .clarity-stacks-helper string-ascii-to-buffer (get-wrapped-token-contract-code (get token-y verify-params)))
		(get proof verify-params)
		(get tx-block-height verify-params)
		(get block-header-without-signer-signatures verify-params)))

;; public calls			

(define-public (create
  (request-details {
		token-x-trait: <ft-trait>, token-y-trait: <ft-trait>,
		factor: uint,
		bal-x: uint, bal-y: uint,
		fee-rate-x: uint, fee-rate-y: uint,
		max-in-ratio: uint, max-out-ratio: uint,
		threshold-x: uint, threshold-y: uint,
		oracle-enabled: bool, oracle-average: uint,
		start-block: uint,
		lock: (buff 1) }))
  (let (
			(token-y-trait (get token-y-trait request-details)))
		(try! (pre-check request-details))
		(asserts! (< u0 (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.amm-vault-v2-01 get-reserve (contract-of token-y-trait))) err-token-not-approved)
		(print { type: "create", request: request-details })
		(post-check request-details)))
		
(define-public (create2
    (request-details {
        token-x-trait: <ft-trait>, token-y-trait: <ft-trait>,
				factor: uint,
        bal-x: uint, bal-y: uint,
        fee-rate-x: uint, fee-rate-y: uint,
        max-in-ratio: uint, max-out-ratio: uint,
        threshold-x: uint, threshold-y: uint,
        oracle-enabled: bool, oracle-average: uint,
        start-block: uint,
				lock: (buff 1) })
		(verify-params {
			nonce: (buff 8),
			fee-rate: (buff 8),
			signature: (buff 65),
			contract: principal,
			token-y: principal,
			proof: { tx-index: uint, hashes: (list 14 (buff 32)), tree-depth: uint},
			tx-block-height: uint,
			block-header-without-signer-signatures: (buff 712) }))
  (let ((token-y-trait (get token-y-trait request-details)))
		(asserts! (is-eq (contract-of token-y-trait) (get contract verify-params)) err-token-not-approved)
		(try! (pre-check request-details))
		(try! (verify-deploy verify-params))
		(try! (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.amm-vault-v2-01 set-approved-token (contract-of token-y-trait) true))
		(map-set wrap-token-map (get token-y verify-params) (get contract verify-params))
		(print { type: "create2", request: request-details, verify: verify-params })
		(post-check request-details)))

(define-public (lock-liquidity (amount uint) (pool-id uint))
	(contract-call? .liquidity-locker lock-liquidity amount pool-id))

(define-public (burn-liquidity (amount uint) (pool-id uint))
	(contract-call? .liquidity-locker burn-liquidity amount pool-id))

(define-public (claim-liquidity (pool-id uint))
	(contract-call? .liquidity-locker claim-liquidity pool-id))

 ;; governance calls

(define-public (approve-token-x (token principal) (approved bool) (min-x uint))
    (begin
        (try! (is-dao-or-extension))
        (ok (map-set approved-token-x token { approved: approved, min-x: min-x }))))

(define-public (set-fee-rebate (new-fee-rebate uint))
	(begin 
		(try! (is-dao-or-extension))
		(ok (var-set fee-rebate new-fee-rebate))))

(define-public (set-wrapped-token-template (new-template (list 20 (string-ascii 5000))))
  (begin
    (try! (is-dao-or-extension))
    (ok (var-set wrapped-token-template new-template))))

;; private calls

(define-private (pre-check 
	(request-details {
		token-x-trait: <ft-trait>, token-y-trait: <ft-trait>,
		factor: uint,
		bal-x: uint, bal-y: uint,
		fee-rate-x: uint, fee-rate-y: uint,
		max-in-ratio: uint, max-out-ratio: uint,
		threshold-x: uint, threshold-y: uint,
		oracle-enabled: bool, oracle-average: uint,
		start-block: uint,
		lock: (buff 1) }))
	(let (
			(token-x-trait (get token-x-trait request-details))
			(token-y-trait (get token-y-trait request-details))
			(token-x-details (get-approved-token-x-or-default (contract-of token-x-trait))))
		(asserts! (get approved token-x-details) err-token-not-approved)
    (asserts! (>= (get bal-x request-details) (get min-x token-x-details)) err-insufficient-balance)
    (asserts! (and 
      (is-none (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.amm-pool-v2-01 get-pool-exists (contract-of token-x-trait) (contract-of token-y-trait) (get factor request-details)))
      (is-none (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.amm-pool-v2-01 get-pool-exists (contract-of token-y-trait) (contract-of token-x-trait) (get factor request-details))))
        err-pool-exists)
    (asserts! (or (is-eq (get lock request-details) LOCK) (is-eq (get lock request-details) BURN) (is-eq (get lock request-details) NONE)) err-invalid-lock-parameter)
		(ok true)))

(define-private (post-check
	(request-details {
		token-x-trait: <ft-trait>, token-y-trait: <ft-trait>,
		factor: uint,
		bal-x: uint, bal-y: uint,
		fee-rate-x: uint, fee-rate-y: uint,
		max-in-ratio: uint, max-out-ratio: uint,
		threshold-x: uint, threshold-y: uint,
		oracle-enabled: bool, oracle-average: uint,
		start-block: uint,
		lock: (buff 1) }))
	(let (
			(token-x-trait (get token-x-trait request-details))
			(token-y-trait (get token-y-trait request-details))
			(token-x (contract-of token-x-trait))
			(token-y (contract-of token-y-trait))
			(factor (get factor request-details))
			(supply (get supply (try! (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.amm-pool-v2-01 create-pool token-x-trait token-y-trait factor tx-sender (get bal-x request-details) (get bal-y request-details)))))
			(pool-id (get pool-id (try! (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.amm-pool-v2-01 get-pool-details token-x token-y factor)))))						
		(and (is-eq (get lock request-details) LOCK) (try! (lock-liquidity supply pool-id)))
		(and (is-eq (get lock request-details) BURN) (try! (burn-liquidity supply pool-id)))
		(try! (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.amm-pool-v2-01 set-fee-rate-x token-x token-y factor (get fee-rate-x request-details)))
		(try! (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.amm-pool-v2-01 set-fee-rate-y token-x token-y factor (get fee-rate-y request-details)))
		(try! (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.amm-pool-v2-01 set-max-in-ratio token-x token-y factor (get max-in-ratio request-details)))
		(try! (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.amm-pool-v2-01 set-max-out-ratio token-x token-y factor (get max-out-ratio request-details)))
		(try! (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.amm-pool-v2-01 set-threshold-x token-x token-y factor (get threshold-x request-details)))
		(try! (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.amm-pool-v2-01 set-threshold-y token-x token-y factor (get threshold-y request-details)))
		(try! (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.amm-pool-v2-01 set-oracle-enabled token-x token-y factor (get oracle-enabled request-details)))
		(try! (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.amm-pool-v2-01 set-oracle-average token-x token-y factor (get oracle-average request-details)))
		(try! (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.amm-pool-v2-01 set-start-block token-x token-y factor (get start-block request-details)))
		(try! (contract-call? 'SP102V8P0F7JX67ARQ77WEA3D3CFB5XW39REDT0AM.amm-registry-v2-01 set-fee-rebate token-x token-y factor (var-get fee-rebate)))
    (ok pool-id)))	

(define-constant C32 "0123456789ABCDEFGHJKMNPQRSTVWXYZ")
(define-constant LIST_15 (list 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0))
(define-constant LIST_24 (list 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0))
(define-constant LIST_39 (concat LIST_24 LIST_15))

(define-private (c32-to-string-iter (idx int) (it { s: (string-ascii 39), r: uint }))
	{ s: (unwrap-panic (as-max-len? (concat (unwrap-panic (element-at? C32 (mod (get r it) u32))) (get s it)) u39)), r: (/ (get r it) u32) })

(define-private (hash-bytes-to-string (data (buff 24)))
	(let (
			;; fixed-length: 8 * 15 / 5 = 24
			(low-part (get s (fold c32-to-string-iter LIST_24 { s: "", r: (buff-to-uint-be (unwrap-panic (as-max-len? (unwrap-panic (slice? data u9 u24)) u16)))})))
			;; fixed-length: ceil(8 * 9 / 5) = 15
			(high-part (get s (fold c32-to-string-iter LIST_15 { s: "", r: (buff-to-uint-be (unwrap-panic (as-max-len? (unwrap-panic (slice? data u0 u9)) u16)))}))))
		(unwrap-panic (as-max-len? (concat high-part low-part) u39))))

(define-private (trim-leading-0-iter (idx int) (it (string-ascii 39)))
	(if (is-eq (element-at? it u0) (some "0")) (unwrap-panic (slice? it u1 (len it))) it))

(define-private (trim-leading-0 (s (string-ascii 39)))
	(fold trim-leading-0-iter LIST_39 s))

(define-private (append-leading-0-iter (idx int) (it { hash-bytes: (buff 24), address: (string-ascii 39)}))
	(if (is-eq (element-at? (get hash-bytes it) u0) (some 0x00))
		{ hash-bytes: (unwrap-panic (slice? (get hash-bytes it) u1 (len (get hash-bytes it)))), address: (unwrap-panic (as-max-len? (concat "0" (get address it)) u39)) }
		it))

(define-private (append-leading-0 (hash-bytes (buff 24)) (s (string-ascii 39)))
	(get address (fold append-leading-0-iter LIST_24 { hash-bytes: hash-bytes, address: s })))

(define-private (join-template-parts-iter (part (string-ascii 5000)) (state {token: (string-ascii 100), result: (string-ascii 10000)}))
  {
    token: (get token state),
    result: (unwrap-panic (as-max-len? 
      (concat 
        (get result state) 
        (if (is-eq (len (get result state)) u0) part (concat (get token state) part)))
      u10000))
  })
