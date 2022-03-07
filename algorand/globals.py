#!/usr/bin/python3

# The maximum number of signatures verified by each transaction in the group.
# the total count of required transactions to verify all guardian signatures is
#
# floor(guardian_count  / SIGNATURES_PER_TRANSACTION)
#
from pyteal.types import *
from pyteal.ast import *
MAX_SIGNATURES_PER_VERIFICATION_STEP = 8

"""
Math ceil function.
"""


@Subroutine(TealType.uint64)
def ceil(n, d):
    q = n / d
    r = n % d
    return Seq([
        If(r != Int(0)).Then(Return(q + Int(1))).Else(Return(q))
    ])


"""
Return the minimum Uint64 of A,B
"""


@Subroutine(TealType.uint64)
def min(a, b):
    If(Int(a) < Int(b), Return(a), Return(b))


"""
Let G be the guardian count, N number of signatures per verification step, group must have CEIL(G/N) transactions.
"""


@Subroutine(TealType.uint64)
def get_group_size(num_guardians):
    return ceil(num_guardians, Int(MAX_SIGNATURES_PER_VERIFICATION_STEP))


"""
Get the number of signatures to verify in current step
"""


@Subroutine(TealType.uint64)
def get_sig_count_in_step(step, num_guardians):
    r = num_guardians % Int(MAX_SIGNATURES_PER_VERIFICATION_STEP)
    return Seq(
        If(r == Int(0)).Then(Return(Int(MAX_SIGNATURES_PER_VERIFICATION_STEP)))
        .ElseIf(step < get_group_size(num_guardians) - Int(1))
        .Then(
            Return(Int(MAX_SIGNATURES_PER_VERIFICATION_STEP)))
        .Else(
            Return(num_guardians % Int(MAX_SIGNATURES_PER_VERIFICATION_STEP))))
