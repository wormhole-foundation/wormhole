# Algorand memory allocation

# Table of Contents

1. [Background](about:blank#orgea5c5c2)
2. [The “allocator” program](about:blank#org85bc975)
   1. [Instantiating template variables](about:blank#orgf176818)
      1. [Instantiation, off-chain](about:blank#org2091dfe)
      2. [Instantiation, on-chain](about:blank#orga6fa146)
   2. [Allocating, client-side](about:blank#org74c4227)

<a id="orgea5c5c2"></a>

# Background

The Algorand blockchain has a completely different virtual machine from the other chains Wormhole currently supports. The assembly language of Algorand is called TEAL, which runs on the Algorand VM (AVM). This means that understanding the Algorand contracts will require understanding a whole new set of platform-specific features and constraints.

The purpose of this post is to investigate the way the Wormhole contracts handle (or rather, implement) memory management on Algorand. This is particularly interesting because of the unique memory constraints on this platform which require a fair amount of creativity to overcome. This code is critical, and highly non-trivial.

Like EVM bytecode, TEAL is a purpose-designed instruction set, but unlike EVM, there is currently no high-level language (like Solidity) with a compiler targeting TEAL. There is an in-between solution, called pyTEAL. pyTEAL is **not** a compiler from Python to TEAL, instead, it is an embedded domain-specific language for generating TEAL code in Python. This means that each pyTEAL program is a code generator (it’s a heterogeneous two-stage programming language). The thing about multistage programming languages is that you always have to think about when a piece of code will execute - compile time or runtime? We’ll discuss this in detail.

A pyTEAL program essentially constructs TEAL abstract syntax during its execution. The pyTEAL library provides a function called `compileTeal` which turns this abstract syntax into concrete syntax (not sure how much compilation is happening, but we’ll roll with this name). Finally, the TEAL file has to be compiled to binary before it can be uploaded to the Algorand blockchain. Somewhat frustratingly, this part of the process requires connecting to a running Algorand node. It’s unclear to me why this is the case, as we will see, this compilation step might as well be just called an assembler step. There’s pretty much a 1-to-1 mapping from TEAL to the binary.

Algorand contracts can only store a fixed amount of state. This is a major limitation for us, as in order to support some of the key wormhole features like replay protection, we need an unbounded amount of storage.

<a id="org85bc975"></a>

# The “allocator” program

The main allocator code resides in [/algorand/TmplSig.py](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/TmplSig.py). Let’s work backwards in this file:

```python
if __name__ == '__main__':
    core = TmplSig("sig")

    with open("sig.tmpl.teal", "w") as f:
        f.write(core.get_sig_tmpl())
```

[/algorand/TmplSig.py#L136-L142](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/TmplSig.py#L136-L142)

If we run this program, it will generate a file called `sig.tmpl.teal` based on whatever `get_sig_tmpl()` returns from the `TmplSig` class. The first few lines of `sig.tmpl.teal` look like this:

<!-- cspell:disable -->
```
#pragma version 6
intcblock 1
pushint TMPL_ADDR_IDX // TMPL_ADDR_IDX
pop
pushbytes TMPL_EMITTER_ID // TMPL_EMITTER_ID
pop
txn TypeEnum
pushint 6 // appl
==
assert
txn OnCompletion
intc_0 // OptIn
==
assert
txn ApplicationID
pushint TMPL_APP_ID // TMPL_APP_ID
==
assert
txn RekeyTo
pushbytes TMPL_APP_ADDRESS // TMPL_APP_ADDRESS
==
assert
txn Fee
pushint 0 // 0
==
assert
txn CloseRemainderTo
global ZeroAddress
==
assert
txn AssetCloseTo
global ZeroAddress
==
assert
intc_0 // 1
return
```
<!-- cspell:enable -->

We’ll examine this file more carefully soon. For now, the key takeaway is that it contains TEAL bytecode, which is a stack-based programming language. What’s curious is that right at the beginning, we push `TMPL_ADDR_IDX` and `TMPL_EMITTER_ID`, only to immediately pop them from the stack, which seems redundant. Indeed, as we will see, these four lines of code are here just for the sake of being here, and we don’t actually expect them to do anything useful. This will make more sense soon.

Let’s look at the `get_sig_tmpl` function.

```python
    def get_sig_tmpl(self):
```

[/algorand/TmplSig.py#L111](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/TmplSig.py#L111)

its return statement is a call to `compileTeal`:

```python
        return compileTeal(sig_tmpl(), mode=Mode.Signature, version=6, assembleConstants=True)
```

[/algorand/TmplSig.py#L134](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/TmplSig.py#L134)

From the `mode=Mode.Signature` argument in `compileTeal` we can see that this program is a signature program, also known as a LogicSig. LogicSigs are special programs that belong to an account, and their purpose is to authorise transactions from that account (or more generally, to act as a signing authority for the account). If the LogicSig program executes successfully (without reverting), then the transaction is authorised. Algorand allows running such programs as a way of implementing domain-specific account authorisation, such as for escrow systems, etc. The address of the LogicSig’s account is deterministically derived from the hash of the LogicSig’s bytecode (this will be very important very soon).

The `sig_tmpl` function returns a sequence of TEAL instructions, the first two of which are

```python
                # Just putting adding this as a tmpl var to make the address unique and deterministic
                # We don't actually care what the value is, pop it
                Pop(Tmpl.Int("TMPL_ADDR_IDX")),
                Pop(Tmpl.Bytes("TMPL_EMITTER_ID")),
```

[/algorand/TmplSig.py#L117-L120](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/TmplSig.py#L117-L120)

Here are the two pop statements we looked at in the TEAL code. The pyTEAL compiler knows to generate push instructions for arguments whenever necessary, so we don’t explicitly push `TMPL_ADDR_IDX` and `TMPL_EMITTER_ID`. These variables immediately get popped, because the LogicSig doesn’t actually make use of them, and they’re only there so when replaced with different values in the bytecode, we get an operationally equivalent, yet distinct bytecode.

`Tmpl.Int("TMPL_ADDR_IDX")` and `Tmpl.Bytes("TMPL_EMITTER_ID")` are _template variables_. Normally, they can be thought of as variables in a TEAL program that get replaced at compile time by the compiler, sort of like CPP macros. In fact, this already hints at how the LogicSig is going to be used: these variables will be programmatically replaced (albeit not just at compile time, but more on that later) with distinct values to generate distinct LogicSigs, with deterministic addresses. The wormhole contract will then be able to use the memory of the associated accounts of these LogicSigs. To see how, we’ll first go through what the LogicSig does in the first place.

When using a LogicSig to sign a transaction (like in our case), the LogicSig program can query information about the transaction from the Algorand runtime. If the LogicSig doesn't revert, then the transaction will be executed on-chain. It is the LogicSig’s responsibility to decide whether it wants to approve this transaction, so it will perform a number of checks to ensure the transaction does what’s expected. Importantly, anyone can pass in their own transactions and use the LogicSig to sign it, so forgetting a check here could result in hijacking the LogicSig’s associated account. That's because (by default), transactions that are signed (that is, approved) by the LogicSig can access the LogicSig’s account. In fact, that's what LogicSigs were designed for in the first place: to implement arbitrary logic for deciding who can spend money out of some account.

The first instruction after the two `Pop`s above is

```python
                Assert(Txn.type_enum() == TxnType.ApplicationCall),
```

[/algorand/TmplSig.py#L122](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/TmplSig.py#L122)

Which asserts that the transaction being signed is an application call.
Note that `==` here is an overloaded operator, and it doesn’t compare two python values. Instead, it generates a piece of pyTEAL abstract syntax that represents an equality operation. In TEAL’s concrete syntax, this looks like:

<!-- cspell:disable -->
```
txn TypeEnum
pushint 6 // appl
==
assert
```
<!-- cspell:enable -->

The `txn` opcode pushes a transaction field variable to the stack, in this case its type, which is made available by the AVM runtime.
`pushint` pushes an integer to the stack, here the number 6, which corresponds to application call. `==` pops the top two elements from the stack and pushes 1 if they are equal, or 0 if they are not. Finally, `assert` pops the top of the stack, and reverts the transaction if it’s 0 (or if the stack is empty).

Application calls are one of the built-in transaction types defined by Algorand, another one is Payment. We require that this one is an application call, because of the next check, opting in:

```python
                Assert(Txn.on_completion() == OnComplete.OptIn),
                Assert(Txn.application_id() == Tmpl.Int("TMPL_APP_ID")),
```

[/algorand/TmplSig.py#L123-L124](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/TmplSig.py#L123-L124)

Opting in means that the current application (which is the `"TMPL_APP_ID"`) can allocate local storage into the _sender_'s account data. When the sender is the LogicSig's associated account, then the transaction opts into the LogicSig's account data. **This is the memory allocation mechanism**. By opting the LogicSig’s associated account into the wormhole contract, wormhole can now use it to store memory. Since these accounts are also limited in size, we need multiple of them, and at deterministic locations. This is why `TMPL_ADDR_IDX` and `TMPL_EMITTER_ID` are used in the program. The wormhole contracts populate these templates with actual values which will allow deriving a deterministic address (the LogicSig’s associated account address) which will be known to be a writeable account by the wormhole contract (since the LogicSig opted into the wormhole contract here). This mechanism is similar to Solana’s Program-Derived Addresses (PDAs). The difference is that in Solana, PDAs are always owned by the program they’re derived from, whereas in Algorand, ownership of an account can be transferred using the `rekey` mechanism. At this point, the LogicSig's account is owned by the LogicSig. Next, we make sure that the transaction transfers ownership of the account to our application:

```python
                Assert(Txn.rekey_to() == Tmpl.Bytes("TMPL_APP_ADDRESS")),
```

[/algorand/TmplSig.py#L125](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/TmplSig.py#L125)

Rekeying is a feature that allows changing an account’s key so that someone else becomes the signing authority for it. Only the current signing authority can authorize rekeying, and in the process it loses authorization. Since this transaction is signed by a LogicSig, the account in question is the LogicSig’s associated account, and the current signing authority is the LogicSig itself. Once it approves a rekey on its associated account, then further transactions from the associated account do not require running the LogicSig logic, and could just use whatever the new key is (in this case, the wormhole application will be able to use this memory freely).

This means that at this stage, the LogicSig transfers ownership of its associated account to the wormhole program. This has two functions. First, a safety mechanism, because it means that the LogicSig is no longer able to sign any further transactions, the wormhole program owns it now. With this ownership assignment, the LogicSic’s account truly behaves like a Solana PDA: it’s an account at a deterministic address that’s owned (and thus only writeable) by the program. Second, we can allow assets to get created into this account (which gets over the asset limitation issue) but the main wormhole contract can still sign for transactions against it.

Finally, 3 more checks:

```python
                Assert(Txn.fee() == Int(0)),
                Assert(Txn.close_remainder_to() == Global.zero_address()),
                Assert(Txn.asset_close_to() == Global.zero_address()),
```

[/algorand/TmplSig.py#L127-L129](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/TmplSig.py#L127-L129)

We check that the transaction fee is 0. `close_remainder_to` could request the account to be closed and the funds to be sent to another account. This has to be zero, otherwise the account would be deleted. Similarly, `asset_close_to` is also the zero address.

Finally, if all the checks succeeded, the LogicSig succeeds:

```python
Approve()
```

[/algorand/TmplSig.py#L152](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/TmplSig.py#L152)

To summarize, this is how allocation works: a special program opts in to the wormhole contract, thereby allowing wormhole to write memory into the program’s account, then transfers ownership of the account to wormhole. Since the address of this account is derived from the program’s bytecode, the addresses are reconstructible deterministically, and a given account with such an address cannot have been created any other way than by executing that program with the expected arguments.

<a id="orgf176818"></a>

## Instantiating template variables

Now we understand how the allocator contract works. In this section, we’ll review how it can be used. The placeholder variables serve as a way to generate multiple distinct programs in a deterministic way. Let’s see how this works.

The construction of the bytecode has to happen both off-chain and on-chain. It needs to happen off-chain because the client has to deploy these programs (since an on-chain program cannot deploy another program, this must be done via a transaction from an off-chain entity). It also needs to happen on-chain, because the wormhole program needs to be able to derive (and validate) the addresses of thusly allocated accounts, and the address derivation works by hashing the bytecode, so the bytecode needs to be constructed in the smart contract from runtime information. The off-chain element could be done easily: we can just compile the allocator LogicSig with the template variables filled in with the appropriate variables. The TEAL compiler supports template variable substitution in this way. However, the on-chain component is more complicated, because the smart contract has no access to the compiler, so there’s no way to instantiate the template variables using the standard mechanism.

Instead, we turn to programmatically patching the generated binary. The LogicSig is compiled once and for all with default values standing in for the template variables, and there’s some code (both off-chain and on-chain) that knows where in the bytecode the template variables are, and replaces them with the appropriate values. The off-chain counterpart can just deploy this patched bytecode, while the on-chain code can hash it to derive the address.

The constructor of the `TmplSig` class starts by initializing the following data structure:

<!-- cspell:disable -->
```python
        self.map = {"name":"lsig.teal","version":6,"source":"","bytecode":"BiABAYEASIAASDEQgQYSRDEZIhJEMRiBABJEMSCAABJEMQGBABJEMQkyAxJEMRUyAxJEIg==",
                    "template_labels":{
                        "TMPL_ADDR_IDX":{"source_line":3,"position":5,"bytes":False},
                        "TMPL_EMITTER_ID":{"source_line":5,"position":8,"bytes":True},
                        "TMPL_APP_ID":{"source_line":16,"position":24,"bytes":False},
                        "TMPL_APP_ADDRESS":{"source_line":20,"position":30,"bytes":True}
                    },
        }
```
<!-- cspell:enable -->

[/algorand/TmplSig.py#L39-L47](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/TmplSig.py#L39-L47)

`bytecode` is a base64 encoded binary blob, the assembled binary of the `sig.tmpl.teal` program. The `template_labels` then encodes offsets into the binary where occurrences of the template variables are. In addition to `position`, they all store a `bytes` flag too. This stores whether the template variable is a byte array or not. The importance of this will be that byte arrays can be arbitrary length, and they have an additional byte at the beginning that describes the length of the byte array. Ints on the other hand, are encoded as varints, which are variable width integers that do not contain an additional length byte (see [https://www.sqlite.org/src4/doc/trunk/www/varint.wiki](https://www.sqlite.org/src4/doc/trunk/www/varint.wiki)). The code that patches the binary will need to make sure to write an additional length byte for byte arrays, hence the flag.

To see what the layout looks like, let’s decode the first few bytes of the bytecode by hand. The TEAL opcodes are documented on the Algorand website here: [https://developer.algorand.org/docs/get-details/dapps/avm/teal/opcodes/](https://developer.algorand.org/docs/get-details/dapps/avm/teal/opcodes/). If we decode the bytecode from base64, we get the following (hex):

```
06 20 01 01 81 00 48 80 00 48 31 10 81 06 12 44 31 19 22 12 44 31 18 81 00 12 44
31 20 80 00 12 44 31 01 81 00 12 44 31 09 32 03 12 44 31 15 32 03 12 44 22
```

and the first few lines of the TEAL code as a reminder:

<!-- cspell:disable -->
```
#pragma version 6
intcblock 1
pushint TMPL_ADDR_IDX // TMPL_ADDR_IDX
pop
pushbytes TMPL_EMITTER_ID // TMPL_EMITTER_ID
pop
```
<!-- cspell:enable -->

The first byte (`0x06`) is the version identifier. This matches `#pragma version 6` in the TEAL file. `0x20` is the `intcblock` instruction. It takes a byte that represents how many ints are stored (1 here) in this section, and then a list of ints (here, it’s just 1). `0x81` is the `pushint` instruction, and here we push `0x0`. This means that that this program was compiled with the template variables filled with zeros. This 0 is at offset 5 in the bytecode, which agrees with the `'position': 5'` field of the above data structure for `TMPL_ADDR_IDX`. The `0x48` opcode next is the pop instruction. Next, `0x80` is a `pushbytes` instruction, which first takes the a varint for the length of the byte array, then the byte array. Here, since the length is 0, there are no bytes following, instead `0x48` pops immediately. This byte array is at position 8, which corresponds to `TMPL_EMITTER_ID` above.

<a id="org2091dfe"></a>

### Instantiation, off-chain

The python code that constructs the bytecode is defined as

<!-- cspell:disable -->
```python
    def populate(self, values: Dict[str, Union[str, int]]) -> LogicSigAccount:
        """populate uses the map to fill in the variable of the bytecode and returns a logic sig with the populated bytecode"""
        # Get the template source
        contract = list(base64.b64decode(self.map["bytecode"]))

        shift = 0
        for k, v in self.sorted.items():
            if k in values:
                pos = v["position"] + shift
                if v["bytes"]:
                    val = bytes.fromhex(values[k])
                    lbyte = uvarint.encode(len(val))
                    # -1 to account for the existing 00 byte for length
                    shift += (len(lbyte) - 1) + len(val)
                    # +1 to overwrite the existing 00 byte for length
                    contract[pos : pos + 1] = lbyte + val
                else:
                    val = uvarint.encode(values[k])
                    # -1 to account for existing 00 byte
                    shift += len(val) - 1
                    # +1 to overwrite existing 00 byte
                    contract[pos : pos + 1] = val

        # Create a new LogicSigAccount given the populated bytecode,
        return LogicSigAccount(bytes(contract))
```
<!-- cspell:enable -->

[/algorand/TmplSig.py#L58-L85](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/TmplSig.py#L58-L85)

It loops through the template variables, and replaces them with values defined in the `values` dictionary. For byte arrays, it inserts the length byte first. The `shift` variable maintains the number of extra bytes inserted so far, as the subsequent byte offsets all shift by this amount.

<a id="orga6fa146"></a>

### Instantiation, on-chain

The on-chain program is similar to the above, but it just concatenates the byte chunks together:

```python
        @Subroutine(TealType.bytes)
        def get_sig_address(acct_seq_start: Expr, emitter: Expr):
            # We could iterate over N items and encode them for a more general interface
            # but we inline them directly here

            return Sha512_256(
                Concat(
                Bytes("Program"),
                # ADDR_IDX aka sequence start
                tmpl_sig.get_bytecode_chunk(0),
                encode_uvarint(acct_seq_start, Bytes("")),

                # EMITTER_ID
                tmpl_sig.get_bytecode_chunk(1),
                encode_uvarint(Len(emitter), Bytes("")),
                emitter,

                # APP_ID
                tmpl_sig.get_bytecode_chunk(2),
                encode_uvarint(Global.current_application_id(), Bytes("")),

                # TMPL_APP_ADDRESS
                tmpl_sig.get_bytecode_chunk(3),
                encode_uvarint(Len(Global.current_application_address()), Bytes("")),
                Global.current_application_address(),


                tmpl_sig.get_bytecode_chunk(4),
                )
            )
```

[/algorand/wormhole_core.py#L86-L115](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/wormhole_core.py#L86-L115)

It writes the string “Program” first. That’s because program addresses are derived by hashing the string “Program” appended to the program’s bytecode. The `get_sig_address` function generates exactly this hash. Notice that the arguments it takes are both of type `Expr`. That’s again because `get_sig_address` is a python program that operates on TEAL expressions to construct a TEAL expression. The bytecode chunks are constructed at compile time, but the concatenation happens at runtime (since the template variables are TEAL expressions, whose values are only available at runtime). This works similarly to the off-chain typescript code.

<a id="org74c4227"></a>

## Allocating, client-side

Finally, let us look at how the client-side code actually allocates these accounts. The main idea is that it constructs the allocator LogicSig with the appropriate template variables substituted, then constructs the three transactions required by the allocator.

The function that does this is `optin`:

```python
    def optin(self, client, sender, app_id, idx, emitter, doCreate=True):
```

[/algorand/admin.py#L485](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/admin.py#L485)

First, construct the bytecode with the variables filled in

```python
        lsa = self.tsig.populate(
            {
                "TMPL_APP_ID": app_id,
                "TMPL_APP_ADDRESS": aa,
                "TMPL_ADDR_IDX": idx,
                "TMPL_EMITTER_ID": emitter,
            }
        )
```

[/algorand/admin.py#L488-L495](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/admin.py#L488-L495)

Then grab the address of the associated account

```python
        sig_addr = lsa.address()
```

[/algorand/admin.py#L497](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/admin.py#L497)

Then check if we’ve already allocated this account

```python
        if sig_addr not in self.cache and not self.account_exists(client, app_id, sig_addr):
```

[/algorand/admin.py#L499](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/admin.py#L499)

if not, construct the optin transaction which will also rekey the LogicSig's account to our application.
First, we construct a "seed" transaction, which will pay enough money from the user's wallet into the LogicSig's account to cover for the execution cost:

```python
                seed_txn = transaction.PaymentTxn(sender = sender.getAddress(),
                                                  sp = sp,
                                                  receiver = sig_addr,
                                                  amt = self.seed_amt)
                seed_txn.fee = seed_txn.fee * 2
```

[/algorand/admin.py#L506-L510](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/admin.py#L506-L510)

Next, the actual opt-in transaction. The sender (the first argument to `ApplicationOptInTxn`) is the `sig_address`, so our application will allocate memory into it via opting in.

```python
                optin_txn = transaction.ApplicationOptInTxn(sig_addr, sp, app_id, rekey_to=get_application_address(app_id))
                optin_txn.fee = 0
```

[/algorand/admin.py#L512-L513](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/admin.py#L512-L513)

remember that this code is not trusted, and the LogicSig will verify this transaction is doing the correct thing.

Next, sign the transactions:

```python
                signed_seed = seed_txn.sign(sender.getPrivateKey())
                signed_optin = transaction.LogicSigTransaction(optin_txn, lsa)
```

[/algorand/admin.py#L517-L518](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/admin.py#L517-L518)

The first one is signed by the user's wallet, as it's only used to send money from the user's account. The transaction is signed by the logic sig (so it has signing authority over the associated account). Next, send the transactions

```python
                client.send_transactions([signed_seed, signed_optin])
                self.waitForTransaction(client, signed_optin.get_txid())
```

[/algorand/admin.py#L520-L521](https://github.com/wormhole-foundation/wormhole/blob/0af600ddde4f507b30ea043de66033d7383f53af/algorand/admin.py#L520-L521)

With that, an account is allocated. The client can now pass this account to wormhole, which, after validating that the address is right, will be able to use it to read and write values to.
