#!/bin/bash -f

. env.sh

cd coin
sui client publish --gas-budget 20000 | tee publish.log
grep ID: publish.log  | head -2 > ids.log
witness_container="`grep "Account Address" ids.log  | sed -e 's/^.*: \(.*\) ,.*/\1/'`"
coin_package="`grep "Immutable" ids.log  | sed -e 's/^.*: \(.*\) ,.*/\1/'`"
echo "export WITNESS_CONTAINER=\"$witness_container\"" >> ../env.sh
echo "export COIN_PACKAGE=\"$coin_package\"" >> ../env.sh

----- Certificate ----
Transaction Hash: a2HGeJrOuCLF22ISRaHj4QJd4kPUaYrKEV9I7nw8Rrk=
Transaction Signature: AA==@b7JZHePb7vJF9ruRLS1x9Nhinb7Jyc75tpWYNVgE6oR6d9ac05ogt7m6UB8GuG3Zs2fCOODwEzKMot4JOsLKBA==@govcpBD2KSn7WdvLWL21kcxXS+bPPMd1I6S5MxR2/Mg=
Signed Authorities Bitmap: RoaringBitmap<[0, 1, 3]>
Transaction Kind : Publish
----- Transaction Effects ----
Status : Failure { error: "MoveAbort(ModuleId { address: c2de1fd3e0f924a1af8c9b5d36fa5532910fd504, name: Identifier(\"myvaa\") }, 6)" }
Mutated Objects:
  - ID: 0x2495020ea547b76d326dc3ec9875cc4e6d655335 , Owner: Account Address ( 0xef02d13e211fceca9c93ccd7c7b4931aaec954bd )