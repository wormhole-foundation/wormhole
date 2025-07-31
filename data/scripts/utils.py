#!/usr/bin/env python3

from decimal import Decimal as D, ROUND_HALF_UP

# Try to keep numbers to within 3 digits
def format_lamports(x, pos):
    if x < 1_000:
        return f"{int(x)} µLamport"
    if x < 1_000_000:
        return f"{int(x / 1_000)} mLamport"
    return f"{int(x / 1_000_000)} Lamport"

def format_sol(x, pos):
    if x < 0.001:
        return f"{(D(x) * D("1e6")).quantize(D("0"), rounding=ROUND_HALF_UP)} µSOL"
    if x < 1:
        return f"{(D(x) * D("1e3")).quantize(D("0"), rounding=ROUND_HALF_UP)} mSOL"
    return f"{D(x).quantize(D("0"), rounding=ROUND_HALF_UP)} SOL"