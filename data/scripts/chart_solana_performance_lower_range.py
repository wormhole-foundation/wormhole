#!/usr/bin/env python3

import matplotlib.pyplot as plt
from matplotlib.ticker import FuncFormatter
import numpy as np

from decimal import Decimal as D

from solana_data import solana_comparison_shim
from utils import format_lamports, format_sol

x_formatter = FuncFormatter(format_lamports)
y_formatter = FuncFormatter(format_sol)

points = 200
priority_prices_lower_limit = 0
priority_prices_upper_limit = 30_000
# X-axis: Priority prices in µLamports
priority_prices = [D(n) for n in np.linspace(priority_prices_lower_limit, priority_prices_upper_limit, num=points)]

fig, ax = plt.subplots(figsize=(10, 6))

max_priority_price = max(priority_prices)
# estimate for 30 mLamport/CU
# We could calculate this in time exactly if we weren't calculating the lines and charting at the same time
max_sol_cost = D("0.00003")

line_colors = ["blue", "gold", "red"]

# Plot one line per implementation
for i, name in enumerate(solana_comparison_shim.labels):
    compute = solana_comparison_shim.compute[i]
    rent = solana_comparison_shim.rent[i]
    # Convert µLamports to lamports, then compute SOL cost
    # compute cost in Lamports
    lamport_costs = [compute * priority_price / D("1e6") for priority_price in priority_prices]
    # Y-axis: Total SOL cost
    sol_costs = [rent + (lamport_cost / D("1e9")) for lamport_cost in lamport_costs]
    ax.plot(priority_prices, sol_costs, color=line_colors[i], label=name.replace(" - ", "\n"), linewidth=4)

    if i == 0:
        # Define baseline
        baseline_implementation_sol_costs = sol_costs
    else:
        reduction = [ (D(1) - (sol_costs[i] / baseline_implementation_sol_costs[i])) * 100 for i in range(len(priority_prices))]
        segment_size = points // 5
        for j in range(segment_size // 2, len(priority_prices), segment_size):
            # Annotate reduction
            if reduction[j] < 0:
                arrow = "↑"
                color = "darkred"
            else:
                arrow = "↓"
                color = "darkgreen"

            x = priority_prices[j]
            y = sol_costs[j]
            ax.annotate(
                f"{arrow}{abs(reduction[j]):.1f}%",
                xy=(x, y),
                xytext=(x - 100, y + (-1 if i == 2 else 1) * max_sol_cost * D("0.03")),
                fontsize=10,
                color=color,
                arrowprops=dict(arrowstyle='->', lw=0.8),
                bbox=dict(boxstyle="round,pad=0.2", fc="white", ec="gray", lw=0.5)
            )

    # Add label at the end of each line
    x_label = priority_prices[-1]
    y_label = sol_costs[-1]
    ax.text(
        x_label + (D("0.003") * max_priority_price),
        y_label + ((-1 if i == 2 else i) * max_sol_cost * D("0.015")),
        name.replace(" - ", "\n"),
        fontsize=11, va='center'
    )


# --- Vertical line at 10_000 µlamports
threshold_price = D(10_000)
ax.axvline(threshold_price, color='red', linestyle='--', linewidth=1)
ax.annotate(
    "We expect priority\nprices to be around\nhere typically",
    xy=(threshold_price, 0),
    xytext=(threshold_price + 2000, ax.get_ylim()[1] * 0.5),
    textcoords='data',
    arrowprops=dict(arrowstyle='->', lw=1),
    fontsize=10,
    bbox=dict(boxstyle="round,pad=0.3", fc="white", ec="gray", lw=0.5)
)

ax.set_xlim(priority_prices_lower_limit, max_priority_price * D("1.12"))

# Labels and legend
ax.xaxis.set_major_formatter(x_formatter)
ax.yaxis.set_major_formatter(y_formatter)
plt.xlabel("Priority Price")
plt.ylabel("Total Cost")
plt.title(solana_comparison_shim.title)
plt.legend()
plt.grid(True)
plt.tight_layout()
plt.show()