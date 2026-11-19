# Context packing algorithm

Standard retrieval systems split source code at arbitrary token offsets, severing functions in half and dropping parameter signatures. `codekernel` uses a greedy knapsack algorithm constrained by syntactical AST boundaries and graph proximity.

## Problem definition

Let $G = (V, E)$ be a directed graph representing symbols and call invocations:
- $V$: set of symbols (functions, methods, classes, interfaces).
- $E$: set of directed invocation edges $(v_i, v_j)$ where $v_i$ calls or implements $v_j$.

Given a query entrypoint $v_0$ and a maximum token budget $B$ (default 4,096 tokens):

$$\max \sum_{v_i \in V} w(v_i) \cdot x_i, \quad \text{subject to } \sum_{v_i \in V} T(v_i) \cdot x_i \le B, \quad x_i \in \{0, 1\}$$

where:
- $T(v_i)$: estimated token size of symbol $v_i$.
- $\text{Dist}(v_0, v_i)$: shortest path distance in $G$ from $v_0$ to $v_i$.
- $\text{deg}^-(v_i)$: in-degree centrality (number of components calling $v_i$).
- $w(v_i)$: weight assigned to symbol $v_i$, defined as:

$$w(v_i) = \frac{1}{\text{Dist}(v_0, v_i) + 1} \times \left(1.0 + \min(0.5, \text{deg}^-(v_i) \times 0.05)\right)$$

## Packing order and compression fallbacks

1. **Hop 0 (Target node)**:
   The entry symbol $v_0$ receives highest priority ($w = 1.0$). If its full body fits within budget $B$, its full implementation and docstring are included. If $T(v_0) > B$, its body is compressed to its typed signature stub with an omission tag.

2. **Hop 1 (Direct callers and callees)**:
   Candidates at distance 1 ($w \approx 0.5$) are sorted by weight descending, followed by export status. For each node:
   - If the full body fits in remaining budget, it is added.
   - If the full body exceeds remaining budget, the engine evaluates whether its compressed signature fits. If so, the signature is included.

3. **Hop 2+ (Extended dependencies)**:
   Nodes at distance $\ge 2$ ($w \le 0.33$) are compressed to signatures by default. This saves tokens for interface contracts without spending attention on remote internal implementation details.
