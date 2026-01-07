;; Title: hiro-kit-cursor
;; Version: v1

(define-read-only (read-buff-1 (cursor { bytes: (buff 8192), pos: uint }))
    (ok { 
        value: (unwrap! (as-max-len? (unwrap! (slice? (get bytes cursor) (get pos cursor) (+ (get pos cursor) u1)) (err u1)) u1) (err u1)), 
        next: { bytes: (get bytes cursor), pos: (+ (get pos cursor) u1) }
    }))

(define-read-only (read-buff-2 (cursor { bytes: (buff 8192), pos: uint }))
    (ok { 
        value: (unwrap! (as-max-len? (unwrap! (slice? (get bytes cursor) (get pos cursor) (+ (get pos cursor) u2)) (err u1)) u2) (err u1)), 
        next: { bytes: (get bytes cursor), pos: (+ (get pos cursor) u2) }
    }))

(define-read-only (read-buff-4 (cursor { bytes: (buff 8192), pos: uint }))
    (ok { 
        value: (unwrap! (as-max-len? (unwrap! (slice? (get bytes cursor) (get pos cursor) (+ (get pos cursor) u4)) (err u1)) u4) (err u1)), 
        next: { bytes: (get bytes cursor), pos: (+ (get pos cursor) u4) }
    }))

(define-read-only (read-buff-8 (cursor { bytes: (buff 8192), pos: uint }))
    (ok { 
        value: (unwrap! (as-max-len? (unwrap! (slice? (get bytes cursor) (get pos cursor) (+ (get pos cursor) u8)) (err u1)) u8) (err u1)), 
        next: { bytes: (get bytes cursor), pos: (+ (get pos cursor) u8) }
    }))

(define-read-only (read-buff-16 (cursor { bytes: (buff 8192), pos: uint }))
    (ok { 
        value: (unwrap! (as-max-len? (unwrap! (slice? (get bytes cursor) (get pos cursor) (+ (get pos cursor) u16)) (err u1)) u16) (err u1)), 
        next: { bytes: (get bytes cursor), pos: (+ (get pos cursor) u16) }
    }))

(define-read-only (read-buff-20 (cursor { bytes: (buff 8192), pos: uint }))
    (ok { 
        value: (unwrap! (as-max-len? (unwrap! (slice? (get bytes cursor) (get pos cursor) (+ (get pos cursor) u20)) (err u1)) u20) (err u1)), 
        next: { bytes: (get bytes cursor), pos: (+ (get pos cursor) u20) }
    }))

(define-read-only (read-buff-32 (cursor { bytes: (buff 8192), pos: uint }))
    (ok { 
        value: (unwrap! (as-max-len? (unwrap! (slice? (get bytes cursor) (get pos cursor) (+ (get pos cursor) u32)) (err u1)) u32) (err u1)), 
        next: { bytes: (get bytes cursor), pos: (+ (get pos cursor) u32) }
    }))

(define-read-only (read-buff-64 (cursor { bytes: (buff 8192), pos: uint }))
    (ok { 
        value: (unwrap! (as-max-len? (unwrap! (slice? (get bytes cursor) (get pos cursor) (+ (get pos cursor) u64)) (err u1)) u64) (err u1)), 
        next: { bytes: (get bytes cursor), pos: (+ (get pos cursor) u64) }
    }))

(define-read-only (read-buff-65 (cursor { bytes: (buff 8192), pos: uint }))
    (ok { 
        value: (unwrap! (as-max-len? (unwrap! (slice? (get bytes cursor) (get pos cursor) (+ (get pos cursor) u65)) (err u1)) u65) (err u1)), 
        next: { bytes: (get bytes cursor), pos: (+ (get pos cursor) u65) }
    }))

(define-read-only (read-buff-8192-max (cursor { bytes: (buff 8192), pos: uint }) (size (optional uint)))
    (let ((min (get pos cursor))
          (max (match size value 
            (+ value (get pos cursor))
            (len (get bytes cursor)))))
      (ok { 
          value: (unwrap! (as-max-len? (unwrap! (slice? (get bytes cursor) min max) (err u1)) u8192) (err u1)), 
          next: { bytes: (get bytes cursor), pos: max }
      })))

(define-read-only (read-uint-8 (cursor { bytes: (buff 8192), pos: uint }))
    (let ((cursor-bytes (try! (read-buff-1 cursor))))
        (ok (merge cursor-bytes { value: (buff-to-uint-be (get value cursor-bytes)) }))))

(define-read-only (read-uint-16 (cursor { bytes: (buff 8192), pos: uint }))
    (let ((cursor-bytes (try! (read-buff-2 cursor))))
        (ok (merge cursor-bytes { value: (buff-to-uint-be (get value cursor-bytes)) }))))

(define-read-only (read-uint-32 (cursor { bytes: (buff 8192), pos: uint }))
    (let ((cursor-bytes (try! (read-buff-4 cursor))))
        (ok (merge cursor-bytes { value: (buff-to-uint-be (get value cursor-bytes)) }))))

(define-read-only (read-uint-64 (cursor { bytes: (buff 8192), pos: uint }))
    (let ((cursor-bytes (try! (read-buff-8 cursor))))
        (ok (merge cursor-bytes { value: (buff-to-uint-be (get value cursor-bytes)) }))))

(define-read-only (read-uint-128 (cursor { bytes: (buff 8192), pos: uint }))
    (let ((cursor-bytes (try! (read-buff-16 cursor))))
        (ok (merge cursor-bytes { value: (buff-to-uint-be (get value cursor-bytes)) }))))

(define-read-only (read-int-8 (cursor { bytes: (buff 8192), pos: uint }))
    (let ((cursor-bytes (try! (read-buff-1 cursor))))
        (ok (merge 
            cursor-bytes 
            { value: (bit-shift-right (bit-shift-left (buff-to-int-be (get value cursor-bytes)) u120) u120) }))))

(define-read-only (read-int-16 (cursor { bytes: (buff 8192), pos: uint }))
    (let ((cursor-bytes (try! (read-buff-2 cursor))))
        (ok (merge 
            cursor-bytes 
            { value: (bit-shift-right (bit-shift-left (buff-to-int-be (get value cursor-bytes)) u112) u112) }))))

(define-read-only (read-int-32 (cursor { bytes: (buff 8192), pos: uint }))
    (let ((cursor-bytes (try! (read-buff-4 cursor))))
        (ok (merge 
            cursor-bytes 
            { value: (bit-shift-right (bit-shift-left (buff-to-int-be (get value cursor-bytes)) u96) u96) }))))

(define-read-only (read-int-64 (cursor { bytes: (buff 8192), pos: uint }))
    (let ((cursor-bytes (try! (read-buff-8 cursor))))
        (ok (merge 
            cursor-bytes 
            { value: (bit-shift-right (bit-shift-left (buff-to-int-be (get value cursor-bytes)) u64) u64) }))))

(define-read-only (read-int-128 (cursor { bytes: (buff 8192), pos: uint }))
    (let ((cursor-bytes (try! (read-buff-16 cursor))))
        (ok (merge 
            cursor-bytes 
            { value: (buff-to-int-be (get value cursor-bytes)) }))))

(define-read-only (new (bytes (buff 8192)) (offset (optional uint)))
    { 
        value: none, 
        next: { bytes: bytes, pos: (match offset value value u0) }
    })

(define-read-only (advance (cursor { bytes: (buff 8192), pos: uint }) (offset uint))
     { bytes: (get bytes cursor), pos: (+ (get pos cursor) offset) })

(define-read-only (slice (cursor { bytes: (buff 8192), pos: uint }) (size (optional uint)))
    (match (slice? (get bytes cursor) 
                   (get pos cursor) 
                   (match size value 
                   (+ (get pos cursor) value)    
                      (len (get bytes cursor))))
        bytes bytes 0x))
