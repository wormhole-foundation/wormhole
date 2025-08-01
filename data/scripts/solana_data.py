#!/usr/bin/env python3

from dataclasses import dataclass
import numpy as np

from decimal import Decimal as D

@dataclass
class ComputeUnitsVsImplementationChart:
    title:   str
    labels:  list[str]
    compute: list[D]
    rent:    list[D]

solana_comparison_all = ComputeUnitsVsImplementationChart(
    title="Current vs VerificationV2",
    labels=[
        "old core - 4 txs",        "shim verify - 2 txs",              "early VerificationV2 - 1 tx",
        "VerificationV2 - 1 tx",   "VerificationV2 + decode - 1 tx",   "VerificationV2 + digest - 1 tx"
    ],
    compute=[
                     D("146709"),                        D("337883"),                       D("53417"),
                      D("33902"),                         D("34057"),                       D("33378")],

    rent=[      D("0.003874272"),                   D("0.000015040"),                           D("0"),
                D("0"),                             D("0"),                                     D("0")]
)

solana_comparison_core_shim = ComputeUnitsVsImplementationChart(
    title="Current vs VerificationV2 - Relative reductions from Old Core",
    labels=[
        "old core - 4 txs",        "shim verify - 2 txs",              "early VerificationV2 - 1 tx",
        "VerificationV2 only digest - 1 tx"
    ],
    compute=[
                     D("146709"),                        D("337883"),                       D("53417"),
                      D("33378")],

    rent=[      D("0.003874272"),                   D("0.000015040"),                           D("0"),
                D("0")]
)

solana_comparison_shim = ComputeUnitsVsImplementationChart(
    title="Shim core vs VerificationV2 - Relative reductions from Shim core",
    labels=[
        "shim verify - 2 txs",              "VerificationV2 only digest - 1 tx"
    ],
    compute=[     D("337883"),                       D("33378")],

    rent=[   D("0.000015040"),                           D("0")]
)
