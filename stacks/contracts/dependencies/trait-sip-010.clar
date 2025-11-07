(define-trait sip-010-trait
  (
    ;; Transfer from the caller to a new principal
    (transfer (uint principal principal (optional (buff 34))) (response bool uint))
    ;; the human readable name of the token
    (get-name () (response (string-ascii 32) uint))
    ;; the ticker symbol, or empty if none
    (get-symbol () (response (string-ascii 32) uint))
    ;; the number of decimals used, e.g. 6 would mean 1_000_000 represents 1 token
    (get-decimals () (response uint uint))
    ;; the balance of the passed principal
    (get-balance (principal) (response uint uint))
    ;; the current total supply (which does not need to be a constant)
    (get-total-supply () (response uint uint))
    ;; an optional URI that represents metadata of this token
    (get-token-uri () (response (optional (string-utf8 256)) uint))
    ;; helper functions for 8-digit fixed notation
    (transfer-fixed (uint principal principal (optional (buff 34))) (response bool uint))
    (get-balance-fixed (principal) (response uint uint))
    (get-total-supply-fixed () (response uint uint))    
    (mint (uint principal) (response bool uint))
    (burn (uint principal) (response bool uint))  
    (mint-fixed (uint principal) (response bool uint))
		(burn-fixed (uint principal) (response bool uint))      
 )
)