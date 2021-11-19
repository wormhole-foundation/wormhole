#!/bin/bash
export SIGNATURES64=`node -e "console.log(Buffer.from('25a2cec435380f6413e8b5d5531cd8789322a1d8bc488309bb868c33a26cc9492947b48895460d8d2261d669bcef210987cc5eeb9fa21504c3f5a9b5a0ff32df00ee4d1a5e589b5aa0d4787eaf57ba4b41e6a54e35e8ca60a028a0f1e35db3a8ed5d901a9831272fc117f472fcd0115d31365efd575a19a28eaf5ad9be7cf5f0d901a671b7c2af66aa6bff3337adaa7e5f196630508f85491e650cfdaccd2f67d2a605faf1c267eddaa50ac6de8d35894afce7974f14982f6173b0d020e0567f2a4a01','hex').toString('base64'))"`
export GUARDIAN_KEYS='52A26Ce40F8CAa8D36155d37ef0D5D783fc614d2389A74E8FFa224aeAD0778c786163a7A2150768CB4459EA6482D4aE574305B239B4f2264239e7599'
export GKEYSBASE64=`node -e "console.log(Buffer.from('$GUARDIAN_KEYS',  'hex').toString('base64'))"`


export VAABODY='0000000100000001000171f8dcb863d176e2c420ad6610cf687359612b6fb392e0642b0ca6b1f186aa3b00000000000000010012345678'
export VAABODY64=`node -e "console.log(Buffer.from('$VAABODY',  'hex').toString('base64'))"`

goal app call --app-id $1 --from "$STATELESS_ADDR" --app-arg "str:verify" --app-arg "b64:$GKEYSBASE64" --app-arg "int:3" --noteb64 "$VAABODY64" -o verify.txn


