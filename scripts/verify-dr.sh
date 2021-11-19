#!/bin/bash
export SIGNATURES64=`node -e "console.log(Buffer.from('$2','hex').toString('base64'))"`
export GUARDIAN_KEYS='52A26Ce40F8CAa8D36155d37ef0D5D783fc614d2389A74E8FFa224aeAD0778c786163a7A2150768CB4459EA6482D4aE574305B239B4f2264239e7599'
export GKEYSBASE64=`node -e "console.log(Buffer.from('$GUARDIAN_KEYS',  'hex').toString('base64'))"`
export VAABODY=$3
export VAABODY64=`node -e "console.log(Buffer.from('$VAABODY',  'hex').toString('base64'))"`
rm verify.txn verify.stxn
goal app call --app-id $1 --from "$STATELESS_ADDR" --app-arg "str:verify" --app-arg "b64:$GKEYSBASE64" --app-arg "int:3" --noteb64 "$VAABODY64" -o verify.txn 
goal clerk sign --program vaa-verify.teal --argb64 "$SIGNATURES64" --infile verify.txn --outfile verify.stxn
goal clerk dryrun -t verify.stxn

