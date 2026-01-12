#!/usr/bin/env python3

from dataclasses import dataclass
import matplotlib.pyplot as plt

@dataclass
class GasVsImplementationChart:
    title:  str
    labels: list[str]
    costs:  list[int]

# Data: v1 (multisig)
v1_multisig = GasVsImplementationChart(
    title="Gas Cost per VAA (v1 VAA Multisig) — Relative Reductions from Mainnet Core",
    labels=[
        "mainnet core",            "CoreBridgeLib",    "mod core (calldata)",
        "mod core (optimized)",    "VerifV2 100B",     "VerifV2 5000B", 
        "VerifV2 header+digest",   "VerifV2 batch x4"
    ],
    costs=[
        134689,                    108341,             88686,
         69570,                     51061,             52886,
         50836,                     47080
    ]
)

# Data: v2 (threshold schnorr)
v2_schnorr = GasVsImplementationChart(
    title="Gas Cost per VAA (v2 VAA Threshold) — Relative Reductions from early proxied implementation",
    labels=[
        "early VerifV2\nproxied",    "early VerifV2\nno proxy",     "VerifV2\n100B",
        "VerifV2\n5000B",            "VerifV2\nheader+digest",      "VerifV2\nbatch x4"
    ],
    costs=[
        13874,                        8962,                          8544,
        10430,                        6177,                          4347
    ]
)

# Data: mainnet core and header + digest verifications for v1 and v2 VAAs
current_vs_new = GasVsImplementationChart(
    title="Current vs VerificationV2 — Relative Reductions from Mainnet Core",
    labels=[
        "mainnet core",      "VerifV2\nv1 Multisig",  "VerifV2\nv2 Schnorr"
    ],
    costs=[
         134689,             50836,                   6177
    ]
)

def create_chart(chartData):
    baseline = chartData.costs[0]

    fig, ax = plt.subplots(figsize=(12, 6))
    bars = ax.bar(chartData.labels, chartData.costs, color='royalblue')

    max_cost = max(chartData.costs)

    # Annotate % reduction vs baseline
    for i, cost in enumerate(chartData.costs):
        if i == 0:
            continue  # skip baseline itself
        reduction = (baseline - cost) / baseline * 100
        x = bars[i].get_x() + bars[i].get_width() / 2
        y = cost + (0.01 * max_cost)  # place a bit above the bar
        if reduction < 0:
            arrow = "↑"
            color = "darkred"
        else:
            arrow = "↓"
            color = "darkgreen"

        ax.annotate(f"{arrow}{abs(reduction):.1f}%", (x, y), ha='center', color=color, fontsize=9)

    # Decorations
    ax.set_ylabel("Gas cost / VAA")
    ax.set_title(chartData.title)
    ax.set_xticks(range(len(chartData.labels)))
    ax.set_xticklabels(chartData.labels, rotation=45, ha='right')

    plt.tight_layout()
    plt.show()


create_chart(v1_multisig)
create_chart(v2_schnorr)
create_chart(current_vs_new)