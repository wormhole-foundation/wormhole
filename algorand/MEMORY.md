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

The main allocator code resides in [/algorand/TmplSig.py](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/TmplSig.py). Let’s work backwards in this file:

```python
if __name__ == '__main__':
    core = TmplSig("sig")

    with open("sig.tmpl.teal", "w") as f:
        f.write(core.get_sig_tmpl())
```
[/algorand/TmplSig.py#L165-L171](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/TmplSig.py#L165-L171)

If we run this program, it will generate a file called `sig.tmpl.teal` based on whatever `get_sig_tmpl()` returns from the `TmplSig` class. The first few lines of `sig.tmpl.teal` look like this:

```
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
assert
...
```

We’ll examine this file more carefully soon. For now, the key takeaway is that it contains TEAL bytecode, which is a stack-based programming language. What’s curious is that right at the beginning, we push `TMPL_ADDR_IDX` and `TMPL_EMITTER_ID`, only to immediately pop them from the stack, which seems redundant. Indeed, as we will see, these four lines of code are here just for the sake of being here, and we don’t actually expect them to do anything useful. This will make more sense soon.

Let’s look at the `get_sig_tmpl` function.

```python
def get_sig_tmpl(self):
```
[/algorand/TmplSig.py#L120](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/TmplSig.py#L120)

its return statement is a call to `compileTeal`:

```python
return compileTeal(sig_tmpl(), mode=Mode.Signature, version=6, assembleConstants=True)
```
[/algorand/TmplSig.py#L163](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/TmplSig.py#L163)

From the `mode=Mode.Signature` argument in `compileTeal` we can see that this program is a signature program, also known as a LogicSig. LogicSigs are special programs that belong to an account, and their purpose is to authorise transactions from that account (or more generally, to act as a signing authority for the account). If the LogicSig program executes successfully (without reverting), then the transaction is authorised. Algorand allows running such programs as a way of implementing domain-specific account authorisation, such as for escrow systems, etc. The address of the LogicSig’s account is deterministically derived from the hash of the LogicSig’s bytecode (this will be very important very soon).

The `sig_tmpl` function first assigns the following variables:

```python
admin_app_id = Tmpl.Int("TMPL_APP_ID")
admin_address = Tmpl.Bytes("TMPL_APP_ADDRESS")
seed_amt = Tmpl.Int("TMPL_SEED_AMT")
```
[/algorand/TmplSig.py#L125-L127](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/TmplSig.py#L125-L127)

These are *template variables*. Normally, they can be thought of as variables in a TEAL program that get replaced at compile time by the compiler, sort of like CPP macros. In fact, this already hints at how the LogicSig is going to be used: these variables will be programmatically replaced (albeit not just at compile time, but more on that later) with distinct values to generate distinct LogicSigs, with deterministic addresses. The wormhole contract will then be able to use the memory of the associated accounts of these LogicSigs. To see how, we’ll first go through what the LogicSig does in the first place.

Let’s look at the return statement of the `get_sig_tmpl` next, which is a sequence of TEAL instructions:

```python
return Seq(
    # Just putting adding this as a tmpl var to make the address unique and deterministic
    # We don't actually care what the value is, pop it
    Pop(Tmpl.Int("TMPL_ADDR_IDX")),
    Pop(Tmpl.Bytes("TMPL_EMITTER_ID")),
    init(),
)
```
[/algorand/TmplSig.py#L155-L161](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/TmplSig.py#L155-L161)

Here are the two pop statements we looked at in the TEAL code. The pyTEAL compiler knows to generate push instructions for arguments whenever necessary, so we don’t explicitly push `TMPL_ADDR_IDX` and `TMPL_EMITTER_ID`. These variables immediately get popped, because the LogicSig doesn’t actually make use of them, and they’re only there so when replaced with different values in the bytecode, we get an operationally equivalent, yet distinct bytecode.

Next, we call `init()`. This is actually a python function (defined just above) which returns a series of TEAL instructions. Since python functions are executed at compile time, it would be natural to think that the TEAL instructions that `init()` returns simply get inlined into the instructions returned by `sig_tmpl()`. However, in the assembly, there is a jump to the `init_0` label:

```
callsub init_0
```

this is thanks to the following decorator on `init()`:

```python
@Subroutine(TealType.uint64)
```
[/algorand/TmplSig.py#L129](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/TmplSig.py#L129)

This tells pyTEAL that the `init()` function should not be inlined, but instead a subroutine should be generated from it. Whenever `init()` is then referenced elsewhere, it will be compiled to a `callsub init_0` instruction. This mechanism allows for example to do recursive calls in pyTEAL. If we omit the decorator, then a recursive function call would just recursively be inlined, and the resulting TEAL code would become quite large. In this particular case, I think the decorator is redundant, and we would actually save a few bytes in the bytecode by removing it.

Next, let’s look at `init()`. First we create aliases for `Gtxn[0]`, `Gtxn[1]`, and `Gtxn[2]`:

```python
algo_seed = Gtxn[0]
optin = Gtxn[1]
rekey = Gtxn[2]
```
[/algorand/TmplSig.py#L131-L133](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/TmplSig.py#L131-L133)

These aliases each refer to a transaction in the transaction group, by index. Algorand allows multiple transactions to be included in one atomic group. This way, a user can compose several transactions in an atomic way (this is similar to Solana’s instructions). When using a LogicSig to sign a transaction (like in our case), the LogicSig program has access to each tx in the group and can query information about them from the Algorand runtime. If the LogicSig doesn't revert, then the transactions will be executed on-chain. `algo_seed` is the first, `optin` is the second, and `rekey` is the third transaction in the group. It is then the LogicSig’s responsibility to decide whether it wants to approve this transaction group, so it will perform a number of checks to ensure the transactions do what’s expected. Importantly, anyone can pass in their own transactions and call the LogicSig, so forgetting a check here could result in hijacking the LogicSig’s associated account. That's because (by default), transactions that are signed (that is, approved) by the LogicSig can acces the the LogicSig’s account.

The first instruction in `init()` is

```python
Assert(Global.group_size() == Int(3)),
```
[/algorand/TmplSig.py#L136](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/TmplSig.py#L136)

which asserts that there are 3 instructions in the transaction group. Note that `==` here is an overloaded operator, and it doesn’t compare two python values. Instead, it generates a piece of pyTEAL abstract syntax that represents an equality operation. In TEAL’s concrete syntax, this looks like:

```
global GroupSize
pushint 3 // 3
==
assert
```

The `global` instruction pushes a global variable to the stack, in this case `GroupSize`, which is made available by the AVM runtime. `pushint` pushes an integer to the stack, here the number 3. `==` pops the top two elements from the stack and pushes 1 if they are equal, or 0 if they are not. Finally, `assert` pops the top of the stack, and reverts the transaction if it’s 0 (or if the stack is empty).

Next, we will make sure that those three transactions are what we expect them to be. The first one, `algo_seed`:

```python
Assert(algo_seed.type_enum() == TxnType.Payment),
Assert(algo_seed.amount() == seed_amt),
Assert(algo_seed.rekey_to() == Global.zero_address()),
Assert(algo_seed.close_remainder_to() == Global.zero_address()),
```
[/algorand/TmplSig.py#L138-L141](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/TmplSig.py#L138-L141)

Its type (`type_enum()`) we expect to be a payment, and the amount paid has to be `seed_amt`, which is the `TMPL_SEED_AMT` template variable. This payment is necessary to cover the fees of the transactions and pay for the long term storage associated with this account. The `rekey_to()` is checked to be the zero address, which means that this transaction must not rekey the LogicSig’s account. Rekeying is a feature that allows changing an account’s key so that someone else becomes the signing authority for it. Only the current signing authority can authorize rekeying, and in the process it loses authorization. Since this transaction is signed by a LogicSig, the account in question is the LogicSig’s associated account, and the current signing authority is the LogicSig itself. If it approved a rekey on its associated account, then further transactions from the associated account would not require running the LogicSig logic, and could just use whatever the new key is. Therefore it must check that none of these transactions (which, remember, are untrusted, they’re passed in by the client) rekey the account to something unexpected. `close_remainder_to` could request the account to be closed and the funds to be sent to another account. This also has to be zero, otherwise the account would be deleted.

Next, `optin`:

```python
Assert(optin.type_enum() == TxnType.ApplicationCall),
Assert(optin.on_completion() == OnComplete.OptIn),
Assert(optin.application_id() == admin_app_id),
Assert(optin.rekey_to() == Global.zero_address()),
```
[/algorand/TmplSig.py#L143-L146](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/TmplSig.py#L143-L146)

Here we check that this is an `ApplicationCall` which, when finished, opts the current account (the LogicSig’s associated account) into the `admin_app_id`. Opting an account in to a smart contract allows the smart contract to use some local memory within the account. **This is the memory allocation mechanism**. By opting the LogicSig’s associated account into the wormhole contract, wormhole can now use it to store memory. Since these accounts are also limited in size, we need multiple of them, and at deterministic locations. This is why `TMPL_ADDR_IDX` and `TMPL_EMITTER_ID` are used in the program. The wormhole contracts populate these templates with actual values which will allow deriving a deterministic address (the LogicSig’s associated account address) which will be known to be a writeable account by the wormhole contract (since the LogicSig opted into the wormhole contract here). Rekeying is disabled in this step too. This mechanism is similar to Solana’s Program-Derived Addresses (PDAs). The difference is that in Solana, PDAs are always owned by the program they’re derived from, whereas in Algorand, the rekeying mechanism allows transferring ownership, which is an additional attack surface.

Finally, the `rekey` transaction:

```python
Assert(rekey.type_enum() == TxnType.Payment),
Assert(rekey.amount() == Int(0)),
Assert(rekey.rekey_to() == admin_address),
Assert(rekey.close_remainder_to() == Global.zero_address()),
```
[/algorand/TmplSig.py#L148-L151](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/TmplSig.py#L148-L151)

It’s a payment instruction again, which we expect to send 0 money. The purpose of this instruction is not to pay money (thus it's just a dummy payment), but instead to rekey the account to `admin_address`, which will be populated to be the wormhole contract. This means that at this stage, the LogicSig transfer ownership of its associated account to the wormhole program. This has two functions.  First, a safety mechanism, because it means that the LogicSig is no longer able to sign any further transactions, the wormhole program owns it now. With this ownership assignment, the LogicSic’s account truly behaves like a Solana PDA: it’s an account at a deterministic address that’s owned (and thus only writeable) by the program.  Second, we can allow assets to get created into this account (which gets over the asset limitation issue) but the main wormhole contract can still sign for transactions against it.

Finally, if all the checks succeeded, the LogicSig succeeds:

```python
Approve()
```
[/algorand/TmplSig.py#L152](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/TmplSig.py#L152)

<aside>
☝ The rekey checks on the first two transactions aren’t actually necessary, because the third transaction rekeys to the wormhole program. If any of the first two transactions did a rekey, then the third transaction’s rekey would fail because the LogicSig would have lost authority over the account, and wouldn’t be able to authorize the final rekey, so the transaction would revert. We still keep these checks in though, because they don’t hurt.

</aside>

To summarize, this is how allocation works: a special program opts in to the wormhole contract, thereby allowing wormhole to write memory into the program’s account, then transfers ownership of the account to wormhole. Since the address of this account is derived from the program’s bytecode, the addresses are reconstructable deterministically, and a given account with such an address cannot have been created any other way than by executing that program with the expected arguments.

<a id="orgf176818"></a>

## Instantiating template variables

Now we understand how the allocator contract works. In this section, we’ll review how it can be used. The placeholder variables serve as a way to generate multiple distinct programs in a deterministic way. Let’s see how this works.

The construction of the bytecode has to happen both off-chain and on-chain. It needs to happen off-chain because the client has to deploy these programs (since an on-chain program cannot deploy another program, this must be done via a transaction from an off-chain entity). It also needs to happen on-chain, because the wormhole program needs to be able to derive (and validate) the addresses of thusly allocated accounts, and the address derivation works by hashing the bytecode, so the bytecode needs to be constructed in the smart contract from runtime information. The off-chain element could be done easily: we can just compile the allocator LogicSig with the template variables filled in with the appropriate variables. The TEAL compiler supports template variable substitution in this way. However, the on-chain component is more complicated, because the smart contract has no access to the compiler, so there’s no way to instantiate the template variables using the standard mechanism.

Instead, we turn to programmatically patching the generated binary. The LogicSig is compiled once and for all with default values standing in for the template variables, and there’s some code (both off-chain and on-chain) that knows where in the bytecode the template variables are, and replaces them with the appropriate values. The off-chain counterpart can just deploy this patched bytecode, while the on-chain code can hash it to derive the address.

The constructor of the `TmplSig` class starts by initializing the following data structure:

```python
self.map = {
    'bytecode': 'BiABAYEASIAASIgAAUMyBIEDEkQzABAiEkQzAAiBABJEMwAgMgMSRDMACTIDEkQzARCBBhJEMwEZIhJEMwEYgQASRDMBIDIDEkQzAhAiEkQzAgiBABJEMwIggAASRDMCCTIDEkQiQw==',
    'label_map': {'init_0': 9},
    'name': 'sig.teal',
    'template_labels': {'TMPL_ADDR_IDX': {'bytes': False,
                                          'position': 5,
                                          'source_line': 3},
                        'TMPL_APP_ADDRESS': {'bytes': True,
                                             'position': 91,
                                             'source_line': 57},
                        'TMPL_APP_ID': {'bytes': False,
                                        'position': 64,
                                        'source_line': 41},
                        'TMPL_EMITTER_ID': {'bytes': True,
                                            'position': 8,
                                            'source_line': 5},
                        'TMPL_SEED_AMT': {'bytes': False,
                                          'position': 30,
                                          'source_line': 21}},
    'version': 6}
```
[/algorand/TmplSig.py#L38-L57](https://github.com/certusone/wormhole/blob/7e13a65ede8247ed64db5627cb3bf50e4e21c8a7/algorand/TmplSig.py#L38-L57)

`bytecode` is a base64 encoded binary blob, presumably the assembled binary of the `sig.tmpl.teal` program. The `template_labels` then seems to encode offsets into the binary where occurrences of the template variables are. In addition to `position`, they all store a `bytes` flag too. This stores whether the template variable is a byte array or not. The importance of this will be that byte arrays can be arbitrary length, and they have an additional byte at the beginning that describes the length of the byte array. Ints on the other hand, are encoded as varints, which are variable width integers that do not contain an additional length byte (see [https://www.sqlite.org/src4/doc/trunk/www/varint.wiki](https://www.sqlite.org/src4/doc/trunk/www/varint.wiki)). The code that patches the binary will need to make sure to write an additional length byte for byte arrays, hence the flag.

To see what the layout looks like, let’s decode the first few bytes of the bytecode by hand. The TEAL opcodes are documented on the Algorand website here: [https://developer.algorand.org/docs/get-details/dapps/avm/teal/opcodes/](https://developer.algorand.org/docs/get-details/dapps/avm/teal/opcodes/). If we decode the bytecode from base64, we get the following (hex):

```
06 20 01 01 81 00 48 80 00 48 88 00 01 43 32 04 81 03 12 44 33 00 10 22 12 44 33
00 08 81 00 12 44 33 00 20 32 03 12 44 33 00 09 32 03 12 44 33 01 10 81 06 12 44
33 01 19 22 12 44 33 01 18 81 00 12 44 33 01 20 32 03 12 44 33 02 10 22 12 44 33
02 08 81 00 12 44 33 02 20 80 00 12 44 33 02 09 32 03 12 44 22 43
```

and the first few lines of the TEAL code as a reminder:

```
#pragma version 6
intcblock 1
pushint TMPL_ADDR_IDX // TMPL_ADDR_IDX
pop
pushbytes TMPL_EMITTER_ID // TMPL_EMITTER_ID
pop
callsub init_0
return
```

The first byte (`0x06`) is the version identifier. This matches `#pragma version 6` in the TEAL file. `0x20` is the `intcblock` instruction. It takes a byte that represents how many ints are stored (1 here) in this section, and then a list of ints (here, it’s just 1). `0x81` is the `pushint` instruction, and here we push `0x0`. This means that that this program was compiled with the template variables filled with zeros. This 0 is at offset 5 in the bytecode, which agrees with the `'position': 5'` field of the above data structure for `TMPL_ADDR_IDX`. The `0x48` opcode next is the pop instruction. Next, `0x80` is a `pushbytes` instruction, which first takes the a varint for the length of the byte array, then the byte array. Here, since the length is 0, there are no bytes following, instead `0x48` pops immediately. This byte array is at position 8, which corresponds to `TMPL_EMITTER_ID` above.

<a id="org2091dfe"></a>

### Instantiation, off-chain

The python code that constructs the bytecode is defined as

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

        # Create a new LogicSigAccount given the populated bytecode
        return LogicSigAccount(bytes(contract))
```
[/algorand/TmplSig.py#L67-L94](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/TmplSig.py#L67-L94)

It loops through the template variables, and replaces them with values defined in the `values` dictionary. For byte arrays, it inserts the length byte first. The `shift` variable maintains the number of extra bytes inserted so far, as the subsequent byte offsets all shift by this amount.

In comparison, the typescript code does it in a more straightforward way:

```tsx
async populate(data: PopulateData): Promise<LogicSigAccount> {
        const byteString: string = [
            "0620010181",
            varint
                .encode(data.addrIdx)
                .map((n: number) => properHex(n))
                .join(''),
            "4880",
            varint
                .encode(data.emitterId.length / 2)
                .map((n: number) => properHex(n))
                .join(''),
            data.emitterId,
            "488800014332048103124433001022124433000881",
            varint
                .encode(data.seedAmt)
                .map((n: number) => properHex(n))
                .join(''),
            "124433002032031244330009320312443301108106124433011922124433011881",
            varint
                .encode(data.appId)
                .map((n: number) => properHex(n))
                .join(''),
            "1244330120320312443302102212443302088100124433022080",
            varint
                .encode(data.appAddress.length / 2)
                .map((n: number) => properHex(n))
                .join(''),
            data.appAddress,
            "1244330209320312442243",
        ].join('');
        this.bytecode = hexStringToUint8Array(byteString);
        console.log(
            "This is the final product:",
            Buffer.from(this.bytecode).toString("hex")
        );
        return new LogicSigAccount(this.bytecode);
    }
```

Here we just directly construct the binary by concatenating the bytes together, interleaving the appropriate template variables. Arguably this is harder to change if the LogicSig ever changes, but we don’t expect it to change anyway. Integration testing can eliminate any refactoring hazard here.

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
        # EMMITTER_ID
        tmpl_sig.get_bytecode_chunk(1),
        encode_uvarint(Len(emitter), Bytes("")),
        emitter,
        # SEED_AMT
        tmpl_sig.get_bytecode_chunk(2),
        encode_uvarint(Int(seed_amt), Bytes("")),
        # APP_ID
        tmpl_sig.get_bytecode_chunk(3),
        encode_uvarint(Global.current_application_id(), Bytes("")),
        # TMPL_APP_ADDRESS
        tmpl_sig.get_bytecode_chunk(4),
        encode_uvarint(Len(Global.current_application_address()), Bytes("")),
        Global.current_application_address(),
        tmpl_sig.get_bytecode_chunk(5),
        )
    )
```
[/algorand/portal/core.py#L78-L105](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/wormhole_core.py#L78-L105)

It writes the string “Program” first. That’s because program addresses are derived by hashing the string “Program” appended to the program’s bytecode. The `get_sig_address` function generates exactly this hash. Notice that the arguments it takes are both of type `Expr`. That’s again because `get_sig_address` is a python program that operates on TEAL expressions to construct a TEAL expression. The bytecode chunks are constructed at compile time, but the concatenation happens at runtime (since the template variables are TEAL expressions, whose values are only available at runtime). This works similarly to the off-chain typescript code.

<a id="org74c4227"></a>

## Allocating, client-side

Finally, let us look at how the client-side code actually allocates these accounts. The main idea is that it constructs the allocator LogicSig with the appropriate template variables substituted, then constructs the three transactions required by the allocator.

The function that does this is `optin`:

```
def optin(self, client, sender, app_id, idx, emitter, doCreate=True):
```
[/algorand/admin.py#L467](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/admin.py#L467)

First, construct the bytecode with the variables filled in

```python
lsa = self.tsig.populate(
    {
        "TMPL_SEED_AMT": self.seed_amt,
        "TMPL_APP_ID": app_id,
        "TMPL_APP_ADDRESS": aa,
        "TMPL_ADDR_IDX": idx,
        "TMPL_EMITTER_ID": emitter,
    }
)
```
[/algorand/admin.py#L470-L478](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/admin.py#L470-L478)

Then grab the address of the associated account

```python
sig_addr = lsa.address()
```
[/algorand/admin.py#L480](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/admin.py#L480)

Then check if we’ve already allocated this account

```python
if sig_addr not in self.cache and not self.account_exists(client, app_id, sig_addr):
```
[/algorand/admin.py#L482](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/admin.py#L482)

if not, construct the three transactions (seed, opt-in, and rekey):

```python
seed_txn = transaction.PaymentTxn(sender = sender.getAddress(),
                                  sp = sp,
                                  receiver = sig_addr,
                                  amt = self.seed_amt)
optin_txn = transaction.ApplicationOptInTxn(sig_addr, sp, app_id)
rekey_txn = transaction.PaymentTxn(sender=sig_addr, sp=sp, receiver=sig_addr,
                                   amt=0, rekey_to=get_application_address(app_id))
```
[/algorand/admin.py#L489-L495](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/admin.py#L489-L495)

remember that this code is not trusted, and the LogicSig will verify that they’re doing the correct thing.

Next, sign the three transactions:

```python
signed_seed = seed_txn.sign(sender.getPrivateKey())
signed_optin = transaction.LogicSigTransaction(optin_txn, lsa)
signed_rekey = transaction.LogicSigTransaction(rekey_txn, lsa)
```
[/algorand/admin.py#L499-L501](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/admin.py#L499-L501)

The first one (the payment) is signed by the user’s wallet. The second two are signed by the logic sig (so these transactions have signing authority over the associated account). Next, send the transaction

```python
client.send_transactions([signed_seed, signed_optin, signed_rekey])
self.waitForTransaction(client, signed_optin.get_txid())
```
[/algorand/admin.py#L503-L504](https://github.com/certusone/wormhole/blob/90f6187fbf9b1293ae445242c153ac07ee4d17c8/algorand/admin.py#L503-L504)

With that, an account is allocated. The client can now pass this account to wormhole, which, after validating that the address is right, will be able to use it to read and write values to.
