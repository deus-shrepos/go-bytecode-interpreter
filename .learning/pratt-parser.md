####  Vaughan Pratt's "top-down operator precedence parsing"


##### Single-Pass Compilation

a compiler essentially has two jobs:

1. parse's user code to understand what it means
2. conver that to similar semantics (that is, user's intented output to a low-level instruction)


Some compiler split this into 2 phases: AST and code generation/traversal

we can also combine the two phases into one (conserving memory). So parsing and compiling means the same thing here.



### Evaluating Expressions (Parsing)
