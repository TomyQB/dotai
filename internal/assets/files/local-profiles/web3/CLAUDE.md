# Solidity & Foundry Project — Claude Code Rules

## Project
- **Framework**: Foundry (forge, cast, anvil)
- **Language**: Solidity ^0.8.28
- **Structure**: `src/` contracts, `test/` tests (*.t.sol), `script/` deploy scripts (*.s.sol), `lib/` dependencies

## Build & Test Commands
- `forge build` — compile
- `forge test` — run tests
- `forge test -vvv` — verbose tests with traces
- `forge fmt` — format code
- `forge fmt --check` — verify formatting

---

## Solidity Style Guide (Official)

### Naming Conventions
| Element | Convention | Example |
|---|---|---|
| Contracts, Structs, Events, Enums | `CapWords` | `SimpleToken`, `Transfer` |
| Functions, Modifiers, Arguments | `mixedCase` | `getBalance`, `onlyOwner` |
| Constants, Immutables | `UPPER_CASE` | `MAX_SUPPLY`, `ADMIN_ROLE` |
| Internal/Private functions | `_prefixUnderscore` | `_validateInput` |
| Internal/Private state vars | `_prefixUnderscore` | `_totalSupply` |
| Function return named vars | `trailingUnderscore_` | `result_`, `balance_` |
| Local variables | `trailingUnderscore_` | `name_`, `remainingAmount_` |

### Contract Element Ordering
1. Type declarations (using, struct, enum)
2. State variables
3. Events
4. Errors
5. Modifiers
6. Constructor
7. Receive/Fallback (if any)
8. External functions
9. Public functions
10. Internal functions
11. Private functions
- Within each visibility: non-mutating (`view`, `pure`) go LAST

### Function Declaration Order
`visibility` > `mutability` > `virtual` > `override` > `custom modifiers`

### Formatting
- 4 spaces indentation (never tabs)
- Max 120 characters per line
- Double quotes for strings
- 2 blank lines between top-level declarations
- 1 blank line between functions

### NatSpec
- Required on all `external` and `public` functions
- Use `/// @notice`, `/// @param`, `/// @return`, `/// @dev`
- Document the "why", not the "what"

---

## Security Rules (MANDATORY)

### Checks-Effects-Interactions (CEI)
ALWAYS follow this order in every function:
1. **Checks**: validate inputs, require/revert conditions
2. **Effects**: modify contract state
3. **Interactions**: external calls, transfers

### Access Control
- NEVER use `tx.origin` for authorization — always `msg.sender`
- Every state-modifying function MUST have explicit access control or be intentionally public
- Use custom errors (`error Unauthorized()`) over `require` with strings

### Reentrancy
- Update state BEFORE external calls
- Use ReentrancyGuard from OpenZeppelin for functions with ETH transfers or external calls

### Error Handling
- Custom errors > require with string (gas efficient)
- NEVER swallow errors silently
- Always check return values from external calls (`.call`, `.transfer`, `.send`)
- Use `SafeERC20` for token interactions

### Data & Privacy
- NEVER store sensitive data on-chain (private vars are readable)
- Lock pragma to specific minor version (`^0.8.28`, not `>=0.8.0`)

### Avoid
- Floating pragma (`>=0.8.0`)
- Variable shadowing
- Unbounded loops (always have a fixed upper bound)
- `delegatecall` to untrusted contracts
- Block properties for randomness (`block.timestamp`, `blockhash`)
- `selfdestruct` (deprecated)

---

## Gas Optimization

- `external` over `public` for functions not called internally
- Custom errors over `require("string")`
- Cache `array.length` in loop variables: `uint256 len = arr.length;`
- Named return variables when it simplifies the function
- Use `immutable` for constructor-set variables that never change
- Use `constant` for compile-time known values
- Avoid storage reads in loops — cache in memory
- Short-circuit conditions: cheapest checks first

---

## Testing Standards (Foundry)

### Minimum Coverage Per Function
- Happy path (expected inputs and outputs)
- Edge cases (zero, max values, boundary conditions)
- Revert cases (invalid inputs, unauthorized access)
- Event emission verification (`vm.expectEmit`)

### Test Actors
- NEVER use `address(this)` to simulate users — use explicit addresses with `vm.addr()`
- Create named actors as state variables: `address user = vm.addr(1);`, `address attacker = vm.addr(2);`
- Use `vm.prank(actor)` before each call to simulate the correct caller
- `address(this)` is only acceptable when the test contract itself must be the receiver (e.g., callbacks)

### Test Scope — Test YOUR Code, Not Dependencies
- NEVER test behavior that belongs to inherited libraries (OpenZeppelin, forge-std)
- If `ERC20._mint` works, don't test that `balanceOf` accumulates across multiple mints — OZ already covers that
- Don't test that `Ownable` sets the owner correctly — that's OZ's responsibility
- Don't test that ERC20 emits `Transfer` events — test only YOUR custom events
- Merge event verification (`vm.expectEmit`) into the happy path test instead of separate `_emitsEvent` tests
- One happy path test per function is enough — don't create variants (multipleMints, toMultipleAddresses, etc.)
- Rule of thumb: if removing the test wouldn't miss a bug in YOUR code, the test is redundant

### Additional Requirements
- Fuzz testing (`testFuzz_`) for functions with broad input ranges
- Access control tests for every restricted function
- Overflow/underflow tests for arithmetic operations
- Test file naming: `ContractName.t.sol`

### Test Naming Convention
`test_functionName_scenario()` for unit tests
`testFuzz_functionName()` for fuzz tests
`testFork_functionName()` for fork tests

---

## Pre-Commit Checklist
Before ANY commit, verify:
1. `forge fmt --check` passes (formatting)
2. `forge build --sizes` succeeds (compilation + contract sizes)
3. `forge test` all green (no failures)
4. No `TODO` or `FIXME` in committed code
5. No hardcoded addresses, keys, or secrets
6. NatSpec on all public/external functions

---

## Reference Sources
- [Solidity Style Guide](https://docs.soliditylang.org/en/latest/style-guide.html)
- [Solidity Security](https://docs.soliditylang.org/en/latest/security-considerations.html)
- [Consensys Best Practices](https://consensys.github.io/smart-contract-best-practices/)
- [OpenZeppelin Contracts](https://docs.openzeppelin.com/contracts)
- [SWC Registry](https://swcregistry.io/)
- [Foundry Book](https://book.getfoundry.sh/)
