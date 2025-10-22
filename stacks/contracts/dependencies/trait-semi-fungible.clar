(define-trait semi-fungible-trait
	(
		;; Get a token type balance of the passed principal.
		(get-balance (uint principal) (response uint uint))
		;; Get the total SFT balance of the passed principal.
		(get-overall-balance (principal) (response uint uint))
		;; Get the current total supply of a token type.
		(get-total-supply (uint) (response uint uint))
		;; Get the overall SFT supply.
		(get-overall-supply () (response uint uint))
		;; Get the number of decimal places of a token type.
		(get-decimals (uint) (response uint uint))
		;; Get an optional token URI that represents metadata for a specific token.
		(get-token-uri (uint) (response (optional (string-utf8 256)) uint))
		;; Transfer from one principal to another.
		(transfer (uint uint principal principal) (response bool uint))
		;; Transfer from one principal to another with a memo.
		(transfer-memo (uint uint principal principal (buff 34)) (response bool uint))
		;; helper functions for fixed notation
		(transfer-fixed (uint uint principal principal) (response bool uint))
		(transfer-memo-fixed (uint uint principal principal (buff 34)) (response bool uint))
    	(get-balance-fixed (uint principal) (response uint uint))
		(get-overall-balance-fixed (principal) (response uint uint))				
    	(get-total-supply-fixed (uint) (response uint uint))		
		(get-overall-supply-fixed () (response uint uint))	
        (mint (uint uint principal) (response bool uint))
        (burn (uint uint principal) (response bool uint))  
        (mint-fixed (uint uint principal) (response bool uint))
		(burn-fixed (uint uint principal) (response bool uint))  					
	)
)