requires "evm.md"
requires "foundry.md"

module WORMHOLE-LEMMAS
    imports FOUNDRY
    imports INFINITE-GAS
    imports SET-SYMBOLIC

    syntax StepSort ::= Int
                      | Bool
                      | Bytes
                      | Set
 // ------------------------

    syntax KItem ::= runLemma ( StepSort )
                   | doneLemma( StepSort )
 // --------------------------------------

    syntax Int ::= "notMaxUInt8"   [alias] /* FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF00 */
                 | "notMaxUInt16"  [alias] /* FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF0000 */
                 | "notMaxUInt32"  [alias] /* FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF00000000 */
                 | "notMaxUInt64"  [alias] /* FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF0000000000000000 */
                 | "notMaxUInt240" [alias] /* FFFF000000000000000000000000000000000000000000000000000000000000 */
                 | "notMaxUInt248" [alias] /* FF00000000000000000000000000000000000000000000000000000000000000 */
 // ------------------------------------------------------------------------------------------------------------

    rule notMaxUInt8 => 115792089237316195423570985008687907853269984665640564039457584007913129639680

    rule notMaxUInt16 => 115792089237316195423570985008687907853269984665640564039457584007913129574400

    rule notMaxUInt32 => 115792089237316195423570985008687907853269984665640564039457584007908834672640

    rule notMaxUInt64 => 115792089237316195423570985008687907853269984665640564039439137263839420088320

    rule notMaxUInt240 => 115790322390251417039241401711187164934754157181743688420499462401711837020160

    rule notMaxUInt248 => 115339776388732929035197660848497720713218148788040405586178452820382218977280

    rule <k> runLemma(T) => doneLemma(T) ... </k>

    rule notMaxUInt248 &Int ( X <<Int 248 ) => X <<Int 248
        requires #rangeUInt ( 8 , X )
        [simplification]

    rule notMaxUInt240 &Int ( X <<Int 240 ) => X <<Int 240
        requires #rangeUInt ( 16 , X )
        [simplification]

    rule 4294901760 &Int ( X <<Int 16 ) => X <<Int 16
        requires #rangeUInt ( 16 , X )
        [simplification]

    rule maxUInt16 &Int ( ( ( X *Int pow16 ) |Int ( 115792089237316195423570985008687907853269984665640564039457584007908834738175 &Int _ ) ) >>Int 16 ) => X
        requires #rangeUInt ( 16 , X )
        [simplification]

    rule maxUInt16 &Int ( ( ( X <<Int 16 ) |Int ( 115792089237316195423570985008687907853269984665640564039457584007908834738175 &Int _ ) ) /Int pow16 ) => X
        requires #rangeUInt ( 16 , X )
        [simplification]

    rule maxUInt8 &Int ( X |Int ( notMaxUInt8 &Int _ ) ) => X
        requires #rangeUInt ( 8 , X )
        [simplification]

    rule maxUInt16 &Int ( X |Int ( notMaxUInt16 &Int _ ) ) => X
        requires #rangeUInt ( 16 , X )
        [simplification]

    rule maxUInt32 &Int ( X |Int ( notMaxUInt32 &Int _ ) ) => X
        requires #rangeUInt ( 32 , X )
        [simplification]

    rule maxUInt64 &Int ( X |Int ( notMaxUInt64 &Int _ ) ) => X
        requires #rangeUInt ( 64 , X )
        [simplification]

    rule notMaxUInt16 &Int X => 0
        requires #rangeUInt ( 16 , X )
        [simplification]

    rule notMaxUInt32 &Int X => 0
        requires #rangeUInt ( 32 , X )
        [simplification]

    rule notMaxUInt64 &Int X => 0
        requires #rangeUInt ( 64 , X )
        [simplification]

    rule maxUInt32 &Int X => X
        requires #rangeUInt ( 32 , X )
        [simplification]

    rule #lookup ( MAP [ KEY1 <- _ ] , KEY2 ) => #lookup ( MAP , KEY2 )
        requires KEY1 =/=Int KEY2
        [simplification]

    rule chop ( I:Int ) => I modInt pow256 [simplification, smt-lemma]

    rule A &Int B => B &Int A
        [symbolic(A), concrete(B), simplification]

    // Not being able to generalize this for some reason
    rule (X <<Int 248) <Int pow256 => true
        requires #rangeUInt ( 8 , X )
        [simplification]

    rule (X <<Int 240) <Int pow256 => true
        requires #rangeUInt ( 16 , X )
        [simplification]

    rule (X <<Int 16) <Int pow256 => true
        requires #rangeUInt ( 240 , X )
        [simplification]

    rule 0 <=Int (X <<Int Y) => true
        requires 0 <=Int X andBool 0 <=Int Y
        [simplification]

    rule ( ( X modInt N ) +Int Y ) modInt N => (X +Int Y) modInt N
        [simplification]

    rule X xorInt maxUInt256 => maxUInt256 -Int X
        requires #rangeUInt ( 256 , X )
        [simplification]

    rule X <=Int 115792089237316195423570985008687907853269984665640564039457584007913129639934 => X =/=Int maxUInt256
        requires #rangeUInt ( 256 , X )
        [simplification]

    rule keccak ( _ ) ==Int maxUInt256 => false
        [simplification]

    rule 1 |Int _ ==Int 0 => false
        [simplification]

    rule (A |Int B) ==Int (B |Int A) => true
        [simplification, smt-lemma]

    rule maxUInt8 &Int #asWord ( X ) => #asWord ( #range ( X , 31 , 1 ) )
        requires lengthBytes ( X ) ==Int 32
        [simplification]

    rule maxUInt16 &Int #asWord ( X ) => #asWord ( #range ( X , 30 , 2 ) )
        requires lengthBytes ( X ) ==Int 32
        [simplification]

    rule #buf ( 32 , #asWord ( X ) ) => X
        requires lengthBytes ( X ) ==Int 32
        [simplification]

    rule #buf ( 32 , X <<Int 248 ) => #buf ( 1 , X ) +Bytes #buf ( 31 , 0 )
        requires #rangeUInt ( 8 , X )
        [simplification]

    rule #buf ( 32 , X <<Int 240 ) => #buf ( 2 , X ) +Bytes #buf ( 30 , 0 )
        requires #rangeUInt ( 16 , X )
        [simplification]

    rule #range ( X , S1 , W1 ) +Bytes #range ( X , S2 , W2 ) => #range ( X , S1 , W1 +Int W2 )
        requires S1 +Int W1 ==Int S2
         andBool S2 +Int W2 <=Int lengthBytes ( X )
         [simplification]

    rule ( X <<Int 16 ) ==Int ( X *Int pow16 ) => true
        requires 0 <=Int X
        [simplification, smt-lemma]
endmodule

module WORMHOLE-LEMMAS-SPEC
    imports WORMHOLE-LEMMAS

    claim [lowest-32-bits]: <k> runLemma ( VV0_index_114b9705:Int ==Int ( maxUInt32 &Int ( VV0_index_114b9705:Int |Int ( notMaxUInt32 &Int #lookup ( _STORAGE:Map , 11 ) ) ) ) )
           => doneLemma ( true ) ... </k>
           requires #rangeUInt ( 32 , VV0_index_114b9705 )

    claim [chop-successor-key]: <k> runLemma ( #lookup ( STORAGE [ chop ( KEY +Int 1 ) <- _ ] , KEY ) )
           => doneLemma ( #lookup ( STORAGE , KEY ) ) ... </k>

    claim [chop-successor]: <k> runLemma ( chop ( KEY +Int 1 ) =/=Int KEY )
        => doneLemma ( true ) ... </k>

    claim [bits-16-to-31]: <k> runLemma ( ( 65535 &Int ( ( ( ( ( VV0_newChainId_114b9705:Int <<Int 16 ) modInt 115792089237316195423570985008687907853269984665640564039457584007913129639936 ) &Int 4294901760 ) /Int 65536 ) &Int 65535 ) ) ==Int VV0_newChainId_114b9705:Int )
        => doneLemma ( true ) ... </k>
        requires #rangeUInt ( 16 , VV0_newChainId_114b9705 )

    claim [bits-16-to-31-2]: <k> runLemma ( ( maxUInt16 &Int ( ( ( 4294901760 &Int ( VV0_newChainId_114b9705:Int <<Int 16 ) ) |Int ( 115792089237316195423570985008687907853269984665640564039457584007908834738175 &Int #lookup ( _STORAGE:Map , 8 ) ) ) /Int pow16 ) ) ==Int VV0_newChainId_114b9705:Int ) => doneLemma ( true ) ... </k>
        requires #rangeUInt ( 16 , VV0_newChainId_114b9705 )

    claim [bits-16-to-31-3]: <k> runLemma ( maxUInt16 &Int ( ( ( ( ( X <<Int 16 ) modInt pow256 ) &Int 4294901760 ) /Int pow16 ) &Int maxUInt16 ) ) => doneLemma ( X ) ... </k>
        requires #rangeUInt ( 16 , X )

    claim [bits-16-to-31-4]: <k> runLemma ( ( X *Int pow16 ) |Int Y ==Int ( ( ( ( X <<Int 16 ) modInt pow256 ) &Int 4294901760 ) |Int Y ) ) => doneLemma ( true ) ... </k>
        requires #rangeUInt ( 16 , X )

    claim [bits-16-to-31-5]: <k> runLemma ( ( ( X *Int pow16 ) |Int ( 115792089237316195423570985008687907853269984665640564039457584007908834738175 &Int Y ) ) ==Int ( ( 115792089237316195423570985008687907853269984665640564039457584007908834738175 &Int Y ) |Int ( 4294901760 &Int ( ( X <<Int 16 ) modInt pow256 ) ) ) ) => doneLemma ( true ) ... </k>
        requires #rangeUInt ( 16 , X )
         andBool #rangeUInt ( 256 , Y )

    claim [shift-mask]: <k> runLemma ( ( ( VV0_newChainId_114b9705:Int *Int pow16 ) |Int Y ==Int ( ( ( ( VV0_newChainId_114b9705:Int <<Int 16 ) modInt pow256 ) &Int 4294901760 ) |Int Y ) ) )
        => doneLemma ( true ) ... </k>
        requires #rangeUInt ( 16 , VV0_newChainId_114b9705 )

    claim [shift-range]: <k> runLemma ( #rangeUInt ( 256 , X <<Int 16 ) ) => doneLemma ( true ) ... </k>
        requires #rangeUInt ( 16 , X )

    claim [shift-mod]: <k> runLemma ( ( X <<Int 16 ) modInt pow256 ) => doneLemma ( X <<Int 16 ) ... </k>
        requires #rangeUInt ( 16 , X )

    claim [lowest-8-bits]: <k> runLemma ( maxUInt8 &Int #asWord ( #range ( #buf ( 32 , X ) , 1 , 31 ) +Bytes #range ( #buf ( 32 , #asWord ( b"\x01" +Bytes #range ( #buf ( 32 , Y ) , 0 , 2 ) +Bytes #range ( #buf ( 32 , Z ) , 0 , 29 ) ) ) , 0 , 1 ) ) )
        => doneLemma ( 1 ) ... </k>
        requires #rangeUInt ( 256 , X )
         andBool #rangeUInt ( 256 , Y )
         andBool #rangeUInt ( 256 , Z )

    claim [highest-8-bits]: <k> runLemma ( maxUInt8 &Int #asWord ( #range ( #buf ( 32 , ( notMaxUInt248 &Int ( ( X <<Int 248 ) modInt pow256 ) ) ) , 0 , 1 ) ) )
        => doneLemma ( X ) ... </k>
        requires #rangeUInt ( 8 , X )

    claim [highest-16-bits]: <k> runLemma ( maxUInt16 &Int #asWord ( #range ( #buf ( 32 , X ) , 3 , 29 ) +Bytes b"\x01" +Bytes #range ( #buf ( 32 , ( notMaxUInt240 &Int ( ( Y <<Int 240 ) modInt pow256 ) ) ) , 0 , 2 ) ) )
        => doneLemma ( Y ) ... </k>
        requires #rangeUInt ( 256 , X )
         andBool #rangeUInt ( 16 , Y )

    claim [one-byte-range]: <k> runLemma ( #asWord ( #range ( #buf ( 32 , notMaxUInt248 &Int ( X <<Int 248 ) modInt pow256 ) , 0 , 1 ) ) <Int pow8 )
        => doneLemma ( true ) ... </k>
        requires #rangeUInt ( 8 , X )

    claim [shift-248]: <k> runLemma ( ( X <<Int 248 ) <Int pow256 )
        => doneLemma ( true ) ... </k>
        requires #rangeUInt ( 8 , X )

    claim [length-concat]: <k> runLemma ( lengthBytes ( #range ( #buf ( 32 , _X ) , 3 , 29 ) +Bytes b"\x01" +Bytes #buf ( 2 , Y ) ) )
        => doneLemma ( 32 ) ... </k>
        requires #rangeUInt ( 16 , Y )

    claim [concat-ranges]: <k> runLemma ( #asWord ( #range ( #buf ( 32 , X ) , 0 , 29 ) +Bytes #range ( #buf ( 32 , X ) , 29 , 3 ) ) )
        => doneLemma ( X ) ... </k>
        requires #rangeUInt ( 256 , X )

endmodule
