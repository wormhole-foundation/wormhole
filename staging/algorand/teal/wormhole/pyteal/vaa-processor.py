#!/usr/bin/python3
"""
================================================================================================

The VAA Processor Program

(c) 2022 Wormhole Project Contributors

Changelog.

v1.0                - Initial design
v1.1    211214      - Group must end with either a dummy or app-call such as Store price onchain.

------------------------------------------------------------------------------------------------

This program is the core client to signed VAAs from Wormhole, working in tandem with the
verify-vaa.teal stateless programs.

The following application calls are available.

setvphash: Set verify program hash.
setauthid: Set the authorized app-id of the last transaction call in the group, used as consumer
              of the verified VAA.

verify: Verify guardian signature subset i..j, works in tandem with stateless program.
        Arguments:  #0 guardian public keys subset i..j  (must match stored in global state)
                    #1 guardian signatures subset i..j
                    TX Note: payload to verify
        Last verification step triggers the VAA commiting stage,
        where we decide what to do based on the payload. A last work transaction must be issued,
        with a call to an authorized app-id (authid).  This serves for example to call a Pricekeeper
        contract to store price data on-chain.
        If nothing is to be done, any dummy app-call must be called for the group to be approved.

------------------------------------------------------------------------------------------------

Global state:

"vphash"   :  Hash of verification program logic
"gsexp"    :  Guardian set expiration time
"gscount"  :  Guardian set size
"vssize"   :  Verification step size.
"authid"   :  The authorized app-id of the last transaction call in the group, used as consumer
              of the verified VAA.
key N      :  address of guardian N

------------------------------------------------------------------------------------------------
Stores in scratch: 

SLOT 255:  number of guardians in set 
================================================================================================

"""
from pyteal.ast import *
from pyteal.types import *
from pyteal.compiler import *
from pyteal.ir import *
from globals import *
import sys

GUARDIAN_ADDRESS_SIZE = 20
METHOD = Txn.application_args[0]
VERIFY_ARG_GUARDIAN_KEY_SUBSET = Txn.application_args[1]
VERIFY_ARG_GUARDIAN_SET_SIZE = Txn.application_args[2]
VERIFY_ARG_PAYLOAD = Txn.note()
SLOTID_TEMP_0 = 251
SLOTID_VERIFIED_BIT = 254
STATELESS_LOGIC_HASH = App.globalGet(Bytes("vphash"))
NUM_GUARDIANS = App.globalGet(Bytes("gscount"))
AUTHORIZED_APP_ID = App.globalGet(Bytes("authid"))
SLOT_VERIFIED_BITFIELD = ScratchVar(TealType.uint64, SLOTID_VERIFIED_BIT)
SLOT_TEMP = ScratchVar(TealType.uint64, SLOTID_TEMP_0)

# defined chainId/contracts

GOVERNANCE_CHAIN_ID = 1
GOVERNANCE_EMITTER_ID = '00000000000000000000000000000000000000000000'
PYTH2WORMHOLE_CHAIN_ID = 1
# PYTH2WORMHOLE_EMITTER_ID = '0x71f8dcb863d176e2c420ad6610cf687359612b6fb392e0642b0ca6b1f186aa3b'

# Testnet emitter. 
PYTH2WORMHOLE_EMITTER_ID =   '0x3afda841c1f43dd7d546c8a581ba1f92a139f4133f9f6ab095558f6a359df5d4'

# VAA fields

VAA_RECORD_EMITTER_CHAIN_POS = 8
VAA_RECORD_EMITTER_CHAIN_LEN = 2
VAA_RECORD_EMITTER_ADDR_POS = 10
VAA_RECORD_EMITTER_ADDR_LEN = 32

# -------------------------------------------------------------------------------------------------


@Subroutine(TealType.uint64)
# Arg0: Bootstrap with the initial list of guardians packed.
# Arg1: Expiration time in second argument.
# Arg2: Guardian set Index.
#
# Guardian public keys are 20-bytes wide, so
# using arguments a maximum 1000/20 ~ 200 public keys can be specified in this version.
def bootstrap():
    guardian_count = ScratchVar(TealType.uint64)
    i = SLOT_TEMP
    return Seq([
        Assert(Txn.application_args.length() == Int(3)),
        Assert(Len(Txn.application_args[0]) %
               Int(GUARDIAN_ADDRESS_SIZE) == Int(0)),
        guardian_count.store(
            Len(Txn.application_args[0]) / Int(GUARDIAN_ADDRESS_SIZE)),
        Assert(guardian_count.load() > Int(0)),
        For(i.store(Int(0)), i.load() < guardian_count.load(), i.store(i.load() + Int(1))).Do(
            App.globalPut(Itob(i.load()), Extract(
                Txn.application_args[0], i.load() * Int(GUARDIAN_ADDRESS_SIZE), Int(GUARDIAN_ADDRESS_SIZE)))
        ),
        App.globalPut(Bytes("gscount"), guardian_count.load()),
        App.globalPut(Bytes("gsexp"), Btoi(Txn.application_args[1])),
        App.globalPut(Bytes("gsindex"), Btoi(Txn.application_args[2])),
        App.globalPut(Bytes("vssize"), Int(MAX_SIGNATURES_PER_VERIFICATION_STEP)),
        Approve()
    ])


@Subroutine(TealType.uint64)
def is_creator():
    return Txn.sender() == Global.creator_address()


@Subroutine(TealType.uint64)
def check_guardian_key_subset():
    # Verify that the passed argument for guardian keys [i..j] match the
    # global state for the same keys.
    #
    i = SLOT_TEMP
    sig_count = ScratchVar(TealType.uint64)
    idx_base = ScratchVar(TealType.uint64)
    return Seq([
        idx_base.store(Int(MAX_SIGNATURES_PER_VERIFICATION_STEP) * Txn.group_index()),
        sig_count.store(get_sig_count_in_step(Txn.group_index(), NUM_GUARDIANS)),
        For(i.store(Int(0)),
            i.load() < sig_count.load(),
            i.store(i.load() + Int(1))).Do(
            If(
                App.globalGet(Itob(i.load() + idx_base.load())) != Extract(VERIFY_ARG_GUARDIAN_KEY_SUBSET,
                                                         i.load() * Int(GUARDIAN_ADDRESS_SIZE),
                                                         Int(GUARDIAN_ADDRESS_SIZE))).Then(Return(Int(0)))  # get and compare stored global key
        ),
        Return(Int(1))
    ])


@Subroutine(TealType.uint64)
def check_guardian_set_size():
    #
    # Verify that the passed argument for guardian set size matches the global state.
    #
    return NUM_GUARDIANS == Btoi(VERIFY_ARG_GUARDIAN_SET_SIZE)


@Subroutine(TealType.uint64)
def handle_governance():
    return Int(1)


@Subroutine(TealType.uint64)
def handle_pyth_price_ticker():
    return Int(1)


@Subroutine(TealType.uint64)
#
# Unpack the verified VAA payload and process it according to
# the source based by emitterChainId, emitterAddress.
#
# NOTE: This will work when contract-to-contract call is available in the AVM.
#       Now, the transaction group must end with a call to process the VAA or
#       do-nothing, if you want only to do verification chores without processing anything.
#
def commit_vaa():
    chainId = Btoi(Extract(VERIFY_ARG_PAYLOAD, Int(
        VAA_RECORD_EMITTER_CHAIN_POS), Int(VAA_RECORD_EMITTER_CHAIN_LEN)))
    emitterId = Extract(VERIFY_ARG_PAYLOAD, Int(
        VAA_RECORD_EMITTER_ADDR_POS), Int(VAA_RECORD_EMITTER_ADDR_LEN))
    return Seq([
        If(And(
            chainId == Int(GOVERNANCE_CHAIN_ID),
            emitterId == Bytes(GOVERNANCE_EMITTER_ID))).Then(
            Return(handle_governance()))
        .ElseIf(And(
            chainId == Int(PYTH2WORMHOLE_CHAIN_ID),
            emitterId == Bytes('base16', PYTH2WORMHOLE_EMITTER_ID)
        )).Then(
            Return(handle_pyth_price_ticker())
        ).Else(
            Reject()
        )
    ])


@Subroutine(TealType.uint64)
def check_final_verification_state():
    #
    # Verifies that previous steps had set their verification bits.
    #
    i = SLOT_TEMP
    return Seq([
        For(i.store(Int(1)),
            i.load() < Global.group_size() - Int(1),
            i.store(i.load() + Int(1))).Do(Seq([
                Assert(Gtxn[i.load()].type_enum() == TxnType.ApplicationCall),
                Assert(Gtxn[i.load()].application_id() == Txn.application_id()),
                Assert(GetBit(ImportScratchValue(i.load() - Int(1), SLOTID_VERIFIED_BIT), i.load() - Int(1)) == Int(1))
            ])
        ),
        Return(Int(1))
    ])


def setvphash():
    #
    # Sets the hash of the verification stateless program.
    #

    return Seq([
        Assert(is_creator()),
        Assert(Global.group_size() == Int(1)),
        Assert(Txn.application_args.length() == Int(2)),
        Assert(Len(Txn.application_args[1]) == Int(32)),
        App.globalPut(Bytes("vphash"), Txn.application_args[1]),
        Approve()
    ])

def setauthid():
    #
    # Sets the app-id of an authorized program to be executed as a 
    # last call in the group to optionally consume the verified VAA.
    #

    return Seq([
        Assert(is_creator()),
        Assert(Global.group_size() == Int(1)),
        Assert(Txn.application_args.length() == Int(2)),
        App.globalPut(Bytes("authid"), Btoi(Txn.application_args[1])),
        Approve()
    ])

def verify():
    # * Sender must be stateless logic.
    # * Let N be the number of signatures per verification step, for the TX(i) in group, we verify signatures [j..k] where j = i*N, k = j+(N-1)
    # * Argument 0 must contain guardian public keys for guardians [i..j] (read by stateless logic).
    #   Public keys are 32 bytes long so expected argument length is 32 * (j - i + 1)
    # * Argument 1 must contain current guardian set size (read by stateless logic)
    # * Passed guardian public keys [i..j] must match the current global state.
    # * Note must contain VAA message-in-digest (header+payload) (up to 1KB)  (read by stateless logic)
    #
    # Last verify TX in group  will trigger VAA handling depending on payload. It is required that
    # all previous transactions are app-calls for this AppId and all bitfields are set.
    # Last TX in group must be call to authorized applications.

    return Seq([
        SLOT_VERIFIED_BITFIELD.store(Int(0)),
        Assert(Global.group_size() == get_group_size(NUM_GUARDIANS) + Int(1)),
        Assert(Gtxn[Global.group_size() - Int(1)].type_enum() == TxnType.ApplicationCall),
        Assert(Gtxn[Global.group_size() - Int(1)].application_id() == AUTHORIZED_APP_ID),
        Assert(Txn.application_args.length() == Int(3)),
        Assert(Txn.sender() == STATELESS_LOGIC_HASH),
        Assert(check_guardian_set_size()),
        Assert(check_guardian_key_subset()),
        SLOT_VERIFIED_BITFIELD.store(
            SetBit(SLOT_VERIFIED_BITFIELD.load(), Txn.group_index(), Int(1))),
        If(Txn.group_index() == Global.group_size() -
           Int(2)).Then(
            Return(Seq([
                Assert(check_final_verification_state()),
                commit_vaa()
            ]))),
        Approve()])


def vaa_processor_program():
    handle_create = Return(bootstrap())
    handle_update = Return(is_creator())
    handle_delete = Return(is_creator())
    handle_noop = Cond(
        [METHOD == Bytes("setvphash"), setvphash()],
        [METHOD == Bytes("setauthid"), setauthid()],
        [METHOD == Bytes("verify"), verify()],
        [METHOD == Bytes("nop"), Return(Int(1))],
    )
    return Cond(
        [Txn.application_id() == Int(0), handle_create],
        [Txn.on_completion() == OnComplete.UpdateApplication, handle_update],
        [Txn.on_completion() == OnComplete.DeleteApplication, handle_delete],
        [Txn.on_completion() == OnComplete.NoOp, handle_noop]
    )


def clear_state_program():
    return Int(1)


if __name__ == "__main__":

    approval_outfile = "teal/wormhole/build/vaa-processor-approval.teal"
    clear_state_outfile = "teal/wormhole/build/vaa-processor-clear.teal"
    
    if len(sys.argv) >= 2:
        approval_outfile = sys.argv[1]

    if len(sys.argv) >= 3:
        clear_state_outfile = sys.argv[2]

    print("VAA Processor Program, (c) 2022 Wormhole Project Contributors ")
    print("Compiling approval program...")

    with open(approval_outfile, "w") as f:
        compiled = compileTeal(vaa_processor_program(),
                               mode=Mode.Application, version=5)
        f.write(compiled)

    print("Written to " + approval_outfile)
    print("Compiling clear state program...")

    with open(clear_state_outfile, "w") as f:
        compiled = compileTeal(clear_state_program(),
                               mode=Mode.Application, version=5)
        f.write(compiled)

    print("Written to " + clear_state_outfile)
