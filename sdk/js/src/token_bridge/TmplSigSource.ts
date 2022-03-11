export const tealSource = `
#pragma version 6
intcblock 1
pushint TMPL_ADDR_IDX // TMPL_ADDR_IDX
pop
pushbytes TMPL_EMITTER_ID // TMPL_EMITTER_ID
pop
callsub init_0
return

// init
init_0:
global GroupSize
pushint 3 // 3
==
gtxn 0 TypeEnum
intc_0 // pay
==
&&
gtxn 0 Amount
pushint TMPL_SEED_AMT // TMPL_SEED_AMT
==
&&
gtxn 0 RekeyTo
global ZeroAddress
==
&&
gtxn 0 CloseRemainderTo
global ZeroAddress
==
&&
gtxn 1 TypeEnum
pushint 6 // appl
==
&&
gtxn 1 OnCompletion
intc_0 // OptIn
==
&&
gtxn 1 ApplicationID
pushint TMPL_APP_ID // TMPL_APP_ID
==
&&
gtxn 1 RekeyTo
global ZeroAddress
==
&&
gtxn 2 TypeEnum
intc_0 // pay
==
&&
gtxn 2 Amount
pushint 0 // 0
==
&&
gtxn 2 RekeyTo
pushbytes TMPL_APP_ADDRESS // TMPL_APP_ADDRESS
==
&&
gtxn 2 CloseRemainderTo
global ZeroAddress
==
&&
retsub
`