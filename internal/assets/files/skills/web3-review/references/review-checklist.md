# Checklist de Code Review — Smart Contracts Solidity

Referencia detallada para revisiones de código en contratos Solidity. Aplica todas las categorías relevantes al código bajo revisión.

## Índice

- [1 - Style Guide Compliance](#1---style-guide-compliance)
- [2 - Security Patterns](#2---security-patterns)
- [3 - Gas Optimization](#3---gas-optimization)
- [4 - Test Quality](#4---test-quality)

---

## 1 - Style Guide Compliance

### Naming Conventions

| Elemento | Convención | Ejemplo |
|----------|-----------|---------|
| Contratos, structs, enums, events | CapWords | `StakingVault`, `UserInfo` |
| Funciones, variables, modifiers | mixedCase | `getBalance`, `totalSupply` |
| Constantes | UPPER_CASE | `MAX_SUPPLY`, `FEE_DENOMINATOR` |
| Variables privadas/internas | _prefix | `_owner`, `_balances` |
| Parámetros de función | sin prefix | `amount`, `recipient` |
| Interfaces | I prefix + CapWords | `IStaking`, `IERC20` |

### Orden de Elementos dentro del Contrato

Seguir este orden estricto:

1. Pragmas y imports
2. Type declarations (using ... for ...)
3. State variables
4. Events
5. Custom errors
6. Modifiers
7. Constructor
8. Funciones por visibilidad:
   - receive() / fallback()
   - external
   - public
   - internal
   - private

### Orden de Declaración de Funciones

```solidity
function name(params)
    external          // 1. visibility
    payable           // 2. mutability (payable/view/pure)
    virtual           // 3. virtual
    override          // 4. override
    onlyOwner         // 5. custom modifiers
    returns (uint256) // 6. returns
{
```

### Formatting

- **Indentación**: 4 espacios (no tabs)
- **Línea máxima**: 120 caracteres
- **Strings**: comillas dobles `"error message"`
- **Imports**: named imports, no wildcard (`import {IERC20} from "..."` no `import "..."`)
- **Blank lines**: una línea entre funciones, dos entre secciones principales

### NatSpec

Obligatorio en todas las funciones `public` y `external`:

```solidity
/// @notice Breve descripción de lo que hace la función
/// @param amount La cantidad de tokens a depositar
/// @return shares Las shares recibidas a cambio
function deposit(uint256 amount) external returns (uint256 shares) {
```

Para contratos:

```solidity
/// @title StakingVault
/// @author [Autor]
/// @notice Descripción breve del contrato
/// @dev Detalles de implementación relevantes
contract StakingVault {
```

## 2 - Security Patterns

### Checks-Effects-Interactions (CEI)

Cada función que modifica estado y hace llamadas externas debe seguir:

```solidity
function withdraw(uint256 amount) external {
    // CHECKS - validaciones
    require(balances[msg.sender] >= amount, "Insufficient");

    // EFFECTS - actualizar estado
    balances[msg.sender] -= amount;

    // INTERACTIONS - llamadas externas
    (bool ok, ) = msg.sender.call{value: amount}("");
    require(ok);
}
```

### Access Control

- Toda función que modifique estado sensible DEBE tener access control explícito
- Preferir OpenZeppelin `Ownable2Step` o `AccessControl` sobre `Ownable`
- Verificar que los modifiers cubren todos los paths

### Custom Errors vs Require Strings

```solidity
// ❌ Gasta gas en el string
require(amount > 0, "Amount must be greater than zero");

// ✅ Custom error: más barato y descriptivo
error InvalidAmount();
if (amount == 0) revert InvalidAmount();
```

### Anti-patrones a Detectar

- `tx.origin` para autorización → usar `msg.sender`
- Variable shadowing → renombrar variables locales
- Floating pragma `^0.8.0` → anclar a versión específica `0.8.28`
- `selfdestruct` sin protección → agregar access control o eliminar
- `abi.encodePacked` con tipos dinámicos → usar `abi.encode`

## 3 - Gas Optimization

### Visibility

```solidity
// ❌ public si nunca se llama internamente
function getBalance() public view returns (uint256) { ... }

// ✅ external es más barato para datos en calldata
function getBalance() external view returns (uint256) { ... }
```

Regla: si una función no se llama desde dentro del contrato, debe ser `external`.

### Immutable y Constant

```solidity
// ❌ storage read cada vez
address public owner;
uint256 public fee = 100;

// ✅ inmutable: set en constructor, leído como constante
address public immutable owner;

// ✅ constante: conocido en compilación
uint256 public constant FEE = 100;
```

### Cache de Storage Reads

```solidity
// ❌ múltiples lecturas de storage (cada una ~2100 gas)
function process() external {
    for (uint i; i < items.length; i++) {  // items.length leído N veces
        doSomething(items[i]);
    }
}

// ✅ cache en variable local (memory read ~3 gas)
function process() external {
    uint256 len = items.length;  // una sola lectura de storage
    for (uint i; i < len; i++) {
        doSomething(items[i]);
    }
}
```

### Named Returns

```solidity
// Puede simplificar y ahorrar gas cuando el valor se calcula al principio
function getReward(address user) external view returns (uint256 reward) {
    reward = _calculateReward(user);
    // no necesita `return reward;` explícito
}
```

### Otros Patrones de Gas

- `++i` en lugar de `i++` en loops (marginal en Solidity >=0.8.0 con optimizer)
- `unchecked { ++i; }` en loops donde el overflow es imposible
- Pack structs para minimizar slots de storage (variables <32 bytes juntas)
- Usar `bytes32` en lugar de `string` cuando el tamaño es fijo y conocido
- Evitar arrays dinámicos en storage cuando un mapping funciona

## 4 - Test Quality

### Cobertura Esperada

Cada función pública/externa debe tener tests para:

| Escenario | Nomenclatura | Ejemplo |
|-----------|-------------|---------|
| Happy path | `test_function_scenario` | `test_deposit_successfulDeposit` |
| Edge cases | `test_function_edgeCase` | `test_deposit_maxAmount` |
| Reverts | `testRevert_function_scenario` | `testRevert_deposit_zeroAmount` |
| Events | `test_function_emitsEvent` | `test_deposit_emitsDepositEvent` |
| Access control | `testRevert_function_unauthorized` | `testRevert_withdraw_nonOwner` |

### Fuzz Testing

Funciones con inputs numéricos amplios deberían tener fuzz tests:

```solidity
function testFuzz_deposit(uint256 amount) public {
    amount = bound(amount, 1, type(uint128).max);
    // ...
}
```

### Tests de Invariantes

Para contratos con fondos, definir y testear invariantes:

- `totalSupply == sum(balances)`
- `contract.balance >= totalDeposits - totalWithdrawals`
- `userShares[user] <= totalShares`

### Anti-patrones en Tests

- Tests sin assertions → cada test DEBE tener al menos un `assert`
- Tests que dependen del orden de ejecución → cada test debe ser independiente
- Tests sin setup adecuado → usar `setUp()` para estado compartido
- Mock excesivo → preferir tests de integración con contratos reales cuando sea posible
