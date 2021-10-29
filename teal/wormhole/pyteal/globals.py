#!/usr/bin/python3

# The number of signatures verified by each transaction in the group.
# Since the last transaction of the group is the VAA processing one,
# the total count of required transactions to verify all guardian signatures is
#
# floor(guardian_count  / SIGNATURES_PER_TRANSACTION)
#
SIGNATURES_PER_VERIFICATION_STEP = 6

from pyteal.ast import App, Seq, subroutine
from pyteal.ast.bytes import Bytes
from pyteal.ast.global_ import Global
from pyteal.ast.if_ import If
from pyteal.ast.int import Int
from pyteal.ast.return_ import Return
from pyteal.types import TealType

@subroutine.Subroutine(TealType.uint64)
def is_proper_group_size(gssize):
    # Let G be the guardian count, N number of signatures per verification step, group must have CEIL(G/N) transactions.
    
    q = gssize / Int(SIGNATURES_PER_VERIFICATION_STEP)
    r = gssize % Int(SIGNATURES_PER_VERIFICATION_STEP)
    return Seq([
        If(r != Int(0)).Then(
            Return(Global.group_size() == q + Int(1))
        ).Else(Return(Global.group_size() == q))
    ])

