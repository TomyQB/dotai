# Plantilla de Reporte de Auditoría Web3 — Smart Contracts

## Formato Completo (invocación manual)

Usa esta estructura exacta para el reporte:

```markdown
# 🛡️ Informe de Auditoría de Seguridad — Smart Contracts

## Metadatos de la Auditoría
- **Alcance**: [Contratos cambiados / Proyecto completo]
- **Framework**: [Foundry / Hardhat / Ape]
- **Versión Solidity**: [ej. 0.8.28]
- **Contratos analizados**: [Lista de contratos revisados]
- **Tests ejecutados**: [Resultado de `forge test -vvv` o equivalente]
- **Fecha**: [Fecha actual]

## Resumen Ejecutivo

[2-3 oraciones: total de hallazgos por severidad, evaluación general de riesgo, áreas de mayor preocupación]

## Hallazgos

### [EMOJI] [SEVERIDAD] — [Título descriptivo del hallazgo]
- **ID**: AUD-XXX
- **Categoría**: [A-M del checklist]
- **Referencia**: [SWC-XXX o fuente externa]
- **Ubicación**: `Contrato.sol:42`
- **Descripción**: [Explicación clara de la vulnerabilidad y por qué es un riesgo en el contexto del contrato]
- **Impacto**: [Qué podría lograr un atacante: robo de fondos, DoS, manipulación de estado, etc.]
- **Prueba de Concepto**:
  ```solidity
  // Escenario de ataque paso a paso
  ```
- **Remediación**:
  ```solidity
  // Código corregido con la solución
  ```

[Repetir para cada hallazgo, ordenados de CRÍTICO a GAS]

## Tabla Resumen

| ID | Severidad | Hallazgo | Categoría | Contrato:Línea | Estado |
|----|-----------|----------|-----------|----------------|--------|
| AUD-001 | 🔴 Critical | [Título] | A | Staking.sol:42 | Abierto |
| AUD-002 | 🟠 High | [Título] | B | Token.sol:88 | Abierto |
| AUD-003 | 🟡 Medium | [Título] | I | Router.sol:15 | Abierto |
| AUD-004 | 🔵 Low | [Título] | E | Utils.sol:23 | Abierto |
| AUD-005 | ⚪ Info | [Título] | F | — | Abierto |
| AUD-006 | ⛽ Gas | [Título] | M | Vault.sol:67 | Abierto |

Esta tabla SIEMPRE debe incluirse completa. Es la referencia principal del usuario para localizar y priorizar hallazgos.

## Puntuación General

| Métrica | Resultado |
|---------|-----------|
| **Score de Seguridad** | Secure / Mostly Secure / Needs Attention / Critical Issues |
| **Hallazgos Críticos** | N |
| **Hallazgos Altos** | N |
| **Hallazgos Medios** | N |
| **Hallazgos Bajos** | N |
| **Informativo** | N |
| **Gas** | N |

## Tests de Seguridad Faltantes

[Lista de tests que deberían existir pero no se encontraron en `test/`]

## Recomendaciones

[Lista priorizada de acciones para mejorar la postura de seguridad del protocolo]
```

---

## Formato Condensado (invocación desde commit-and-push)

Cuando el skill se invoca como parte del flujo de commit, usar este formato reducido:

```markdown
## 🛡️ Auditoría Web3 — Resumen

**Contratos auditados**: [N contratos]
**Tests**: [forge test resultado]
**Hallazgos**: [N critical, N high, N medium, N low, N info, N gas]

### Hallazgos Bloqueantes (requieren acción antes del commit)

| ID | Severidad | Hallazgo | Contrato:Línea | Remediación breve |
|----|-----------|----------|----------------|-------------------|
| AUD-001 | 🔴 Critical | [Título] | Staking.sol:42 | [1 línea de acción] |

### Hallazgos No Bloqueantes (informativos)

| ID | Severidad | Hallazgo | Contrato:Línea |
|----|-----------|----------|----------------|
| AUD-003 | 🟡 Medium | [Título] | Router.sol:15 |

**Veredicto**: 🚫 BLOQUEADO / ✅ APROBADO
```

Clasificación de bloqueo:
- **Bloqueantes**: Critical y High — requieren corrección antes del commit
- **No bloqueantes**: Medium, Low, Informational, Gas — se reportan pero no bloquean

---

## Ejemplo de Hallazgo Completo

```markdown
### 🔴 CRITICAL — Reentrancy en función de withdraw permite drenar fondos

- **ID**: AUD-001
- **Categoría**: A (Reentrancy e Interacciones Externas)
- **Referencia**: SWC-107
- **Ubicación**: `Staking.sol:87`
- **Descripción**: La función `withdraw()` envía ETH al usuario antes de actualizar su balance en storage. Un contrato atacante puede re-entrar en `withdraw()` durante la llamada externa y drenar repetidamente su balance antes de que se actualice a cero.
- **Impacto**: Pérdida total de fondos del contrato. Cualquier usuario con un contrato atacante puede drenar todo el ETH depositado por otros usuarios.
- **Prueba de Concepto**:
  ```solidity
  contract Attacker {
      Staking target;
      constructor(address _target) { target = Staking(_target); }

      function attack() external payable {
          target.deposit{value: msg.value}();
          target.withdraw();
      }

      receive() external payable {
          if (address(target).balance >= 1 ether) {
              target.withdraw();
          }
      }
  }
  ```
- **Remediación**:
  ```solidity
  function withdraw() external {
      uint256 amount = balances[msg.sender];
      require(amount > 0, "No balance");

      // Effects ANTES de interactions
      balances[msg.sender] = 0;

      // Interaction DESPUÉS de effects
      (bool success, ) = msg.sender.call{value: amount}("");
      require(success, "Transfer failed");
  }
  ```
```
