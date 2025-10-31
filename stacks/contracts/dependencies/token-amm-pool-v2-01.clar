(impl-trait .trait-semi-fungible.semi-fungible-trait)
(define-constant ERR-NOT-AUTHORIZED (err u1000))
(define-constant ERR-TOO-MANY-POOLS (err u2004))
(define-constant ERR-INVALID-BALANCE (err u1001))
(define-constant ERR-TRANSFER-FAILED (err u3000))
(define-constant ONE_8 u100000000)
(define-fungible-token amm-pool-v2-01-token)
(define-map token-balances {token-id: uint, owner: principal} uint)
(define-map token-supplies uint uint)
(define-map token-owned principal (list 200 uint))
(define-data-var token-name (string-ascii 32) "amm-pool-v2-01-token")
(define-data-var token-symbol (string-ascii 32) "amm-pool-v2-01-token")
(define-data-var token-uri (optional (string-utf8 256)) (some u"https://cdn.alexlab.co/metadata/token-amm-pool-v2-01.json"))
(define-data-var token-decimals uint u8)
(define-data-var transferrable bool true)
(define-read-only (is-dao-or-extension)
	(ok (asserts! (or (is-eq tx-sender .executor-dao) (contract-call? .executor-dao is-extension contract-caller)) ERR-NOT-AUTHORIZED)))
(define-read-only (get-transferrable)
	(ok (var-get transferrable)))
(define-read-only (get-token-owned (owner principal))
    (default-to (list) (map-get? token-owned owner)))
(define-read-only (get-balance (token-id uint) (who principal))
	(ok (get-balance-or-default token-id who)))
(define-read-only (get-overall-balance (who principal))
	(ok (ft-get-balance amm-pool-v2-01-token who)))
(define-read-only (get-total-supply (token-id uint))
	(ok (default-to u0 (map-get? token-supplies token-id))))
(define-read-only (get-overall-supply)
	(ok (ft-get-supply amm-pool-v2-01-token)))
(define-read-only (get-decimals (token-id uint))
	(ok (var-get token-decimals)))
(define-read-only (get-token-uri (token-id uint))
	(ok (var-get token-uri)))
(define-read-only (get-name (token-id uint))
	(ok (var-get token-name)))
(define-read-only (get-symbol (token-id uint))
	(ok (var-get token-symbol)))
(define-read-only (get-total-supply-fixed (token-id uint))
  	(ok (decimals-to-fixed (default-to u0 (map-get? token-supplies token-id)))))
(define-read-only (get-balance-fixed (token-id uint) (who principal))
  	(ok (decimals-to-fixed (get-balance-or-default token-id who))))
(define-read-only (get-overall-supply-fixed)
	(ok (decimals-to-fixed (ft-get-supply amm-pool-v2-01-token))))
(define-read-only (get-overall-balance-fixed (who principal))
	(ok (decimals-to-fixed (ft-get-balance amm-pool-v2-01-token who))))
(define-read-only (get-token-balance-owned-in-fixed (owner principal))
	(begin 
		(match (map-get? token-owned owner)
			token-ids
			(map 
				create-tuple-token-balance 
				token-ids 
				(map 
					get-balance-or-default
					token-ids
					(list 
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
						owner	owner	owner	owner	owner	owner	owner	owner	owner	owner
					)
				)
			)
			(list)
		)
	)
)
(define-public (set-transferrable (new-transferrable bool))
	(begin 
		(try! (is-dao-or-extension))
		(ok (var-set transferrable new-transferrable))))
(define-public (set-decimals (new-decimals uint))
	(begin
		(try! (is-dao-or-extension))
		(ok (var-set token-decimals new-decimals))))
(define-public (set-token-uri (new-uri (optional (string-utf8 256))))
	(begin
		(try! (is-dao-or-extension))
		(ok (var-set token-uri new-uri))))
(define-public (set-name (new-name (string-ascii 32)))
	(begin
		(try! (is-dao-or-extension))
		(ok (var-set token-name new-name))))
(define-public (set-symbol (new-symbol (string-ascii 10)))
	(begin
		(try! (is-dao-or-extension))
		(ok (var-set token-symbol new-symbol))))
(define-public (mint (token-id uint) (amount uint) (recipient principal))
	(begin
		(try! (is-dao-or-extension))
		(try! (ft-mint? amm-pool-v2-01-token amount recipient))
		(try! (set-balance token-id (+ (get-balance-or-default token-id recipient) amount) recipient))
		(map-set token-supplies token-id (+ (unwrap-panic (get-total-supply token-id)) amount))
		(print {type: "sft_mint", token-id: token-id, amount: amount, recipient: recipient})
		(ok true)))
(define-public (burn (token-id uint) (amount uint) (sender principal))
	(begin
		(try! (is-dao-or-extension))
		(try! (ft-burn? amm-pool-v2-01-token amount sender))
		(try! (set-balance token-id (- (get-balance-or-default token-id sender) amount) sender))
		(map-set token-supplies token-id (- (unwrap-panic (get-total-supply token-id)) amount))
		(print {type: "sft_burn", token-id: token-id, amount: amount, sender: sender})
		(ok true)))
(define-public (mint-fixed (token-id uint) (amount uint) (recipient principal))
  	(mint token-id (fixed-to-decimals amount) recipient))
(define-public (burn-fixed (token-id uint) (amount uint) (sender principal))
  	(burn token-id (fixed-to-decimals amount) sender))
(define-public (transfer (token-id uint) (amount uint) (sender principal) (recipient principal))
	(let (
			(sender-balance (get-balance-or-default token-id sender)))
		(asserts! (var-get transferrable) ERR-TRANSFER-FAILED)
		(asserts! (is-eq tx-sender sender) ERR-NOT-AUTHORIZED)
		(asserts! (<= amount sender-balance) ERR-INVALID-BALANCE)
		(try! (ft-transfer? amm-pool-v2-01-token amount sender recipient))
		(try! (set-balance token-id (- sender-balance amount) sender))
		(try! (set-balance token-id (+ (get-balance-or-default token-id recipient) amount) recipient))
		(print {type: "sft_transfer", token-id: token-id, amount: amount, sender: sender, recipient: recipient})
		(ok true)))
(define-public (transfer-memo (token-id uint) (amount uint) (sender principal) (recipient principal) (memo (buff 34)))
	(let (
			(sender-balance (get-balance-or-default token-id sender)))
		(asserts! (var-get transferrable) ERR-TRANSFER-FAILED)
		(asserts! (is-eq tx-sender sender) ERR-NOT-AUTHORIZED)
		(asserts! (<= amount sender-balance) ERR-INVALID-BALANCE)
		(try! (ft-transfer? amm-pool-v2-01-token amount sender recipient))
		(try! (set-balance token-id (- sender-balance amount) sender))
		(try! (set-balance token-id (+ (get-balance-or-default token-id recipient) amount) recipient))
		(print {type: "sft_transfer", token-id: token-id, amount: amount, sender: sender, recipient: recipient, memo: memo})
		(ok true)))
(define-public (transfer-fixed (token-id uint) (amount uint) (sender principal) (recipient principal))
  	(transfer token-id (fixed-to-decimals amount) sender recipient))
(define-public (transfer-memo-fixed (token-id uint) (amount uint) (sender principal) (recipient principal) (memo (buff 34)))
  	(transfer-memo token-id (fixed-to-decimals amount) sender recipient memo))
(define-public (transfer-many (transfers (list 200 {token-id: uint, amount: uint, sender: principal, recipient: principal})))
	(fold transfer-many-iter transfers (ok true)))
(define-public (transfer-many-memo (transfers (list 200 {token-id: uint, amount: uint, sender: principal, recipient: principal, memo: (buff 34)})))
	(fold transfer-many-memo-iter transfers (ok true)))
(define-public (transfer-many-fixed (transfers (list 200 {token-id: uint, amount: uint, sender: principal, recipient: principal})))
	(fold transfer-many-fixed-iter transfers (ok true)))
(define-public (transfer-many-memo-fixed (transfers (list 200 {token-id: uint, amount: uint, sender: principal, recipient: principal, memo: (buff 34)})))
	(fold transfer-many-memo-fixed-iter transfers (ok true)))
(define-private (pow-decimals)
  	(pow u10 (unwrap-panic (get-decimals u0))))
(define-private (fixed-to-decimals (amount uint))
  	(/ (* amount (pow-decimals)) ONE_8))
(define-private (decimals-to-fixed (amount uint))
  	(/ (* amount ONE_8) (pow-decimals)))
(define-private (transfer-many-iter (item {token-id: uint, amount: uint, sender: principal, recipient: principal}) (previous-response (response bool uint)))
	(match previous-response prev-ok (transfer (get token-id item) (get amount item) (get sender item) (get recipient item)) prev-err previous-response))
(define-private (transfer-many-memo-iter (item {token-id: uint, amount: uint, sender: principal, recipient: principal, memo: (buff 34)}) (previous-response (response bool uint)))
	(match previous-response prev-ok (transfer-memo (get token-id item) (get amount item) (get sender item) (get recipient item) (get memo item)) prev-err previous-response))
(define-private (transfer-many-fixed-iter (item {token-id: uint, amount: uint, sender: principal, recipient: principal}) (previous-response (response bool uint)))
	(match previous-response prev-ok (transfer-fixed (get token-id item) (get amount item) (get sender item) (get recipient item)) prev-err previous-response))
(define-private (transfer-many-memo-fixed-iter (item {token-id: uint, amount: uint, sender: principal, recipient: principal, memo: (buff 34)}) (previous-response (response bool uint)))
	(match previous-response prev-ok (transfer-memo-fixed (get token-id item) (get amount item) (get sender item) (get recipient item) (get memo item)) prev-err previous-response))
(define-private (create-tuple-token-balance (token-id uint) (balance uint))
	{ token-id: token-id, balance: (decimals-to-fixed balance) })
(define-private (set-balance (token-id uint) (balance uint) (owner principal))
    (begin
		(and 
			(is-none (index-of (get-token-owned owner) token-id))
			(map-set token-owned owner (unwrap! (as-max-len? (append (get-token-owned owner) token-id) u200) ERR-TOO-MANY-POOLS)))	
	    (map-set token-balances {token-id: token-id, owner: owner} balance)
        (ok true)))
(define-private (get-balance-or-default (token-id uint) (who principal))
	(default-to u0 (map-get? token-balances {token-id: token-id, owner: who})))