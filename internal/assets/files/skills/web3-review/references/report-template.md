# Plantilla de Reporte de Code Review — Smart Contracts

## Formato Completo (invocación manual)

Usa esta estructura exacta para el reporte:

```markdown
# 📝 Code Review — Smart Contracts

## Metadatos de la Revisión
- **Alcance**: [Contratos cambiados / Proyecto completo]
- **Framework**: [Foundry / Hardhat / Ape]
- **Versión Solidity**: [ej. 0.8.28]
- **Contratos revisados**: [Lista de contratos]
- **Verificaciones ejecutadas**: [forge fmt --check, forge build --sizes, forge test]
- **Fecha**: [Fecha actual]

## Resumen Ejecutivo

[2-3 oraciones: total de hallazgos por categoría, calidad general del código, áreas de mejora principales]

## Hallazgos

### [EMOJI] [CATEGORÍA] — [Título descriptivo]
- **Severidad**: Must Fix / Should Fix / Nice to Have
- **Ubicación**: `Contrato.sol:42`
- **Problema**: [Descripción concisa del problema]
- **Sugerencia**:
  ```solidity
  // Código o patrón correcto
  ```

[Repetir para cada hallazgo, agrupados por categoría]

## Tabla Resumen

| # | Categoría | Severidad | Hallazgo | Contrato:Línea |
|---|-----------|-----------|----------|----------------|
| 1 | 🎨 Style | Should Fix | [Título] | Token.sol:12 |
| 2 | 🛡️ Security | Must Fix | [Título] | Staking.sol:87 |
| 3 | ⛽ Gas | Nice to Have | [Título] | Vault.sol:34 |
| 4 | 🧪 Testing | Should Fix | [Título] | — |

## Verificación Automática

| Comando | Resultado |
|---------|-----------|
| `forge fmt --check` | ✅ / ❌ (detalles) |
| `forge build --sizes` | ✅ / ❌ (contratos que exceden 24kb) |
| `forge test` | ✅ / ❌ (N passed, N failed) |

## Recomendaciones

[Lista priorizada de acciones para mejorar la calidad del código]
```

---

## Formato Condensado (invocación desde commit-and-push)

```markdown
## 📝 Code Review — Resumen

**Contratos revisados**: [N contratos]
**Verificación**: fmt ✅/❌ | build ✅/❌ | test ✅/❌
**Hallazgos**: [N must fix, N should fix, N nice to have]

### Hallazgos Bloqueantes (Must Fix)

| # | Categoría | Hallazgo | Contrato:Línea | Sugerencia breve |
|---|-----------|----------|----------------|------------------|
| 1 | 🛡️ Security | [Título] | Staking.sol:87 | [1 línea] |

### Hallazgos No Bloqueantes

| # | Categoría | Severidad | Hallazgo | Contrato:Línea |
|---|-----------|-----------|----------|----------------|
| 2 | 🎨 Style | Should Fix | [Título] | Token.sol:12 |

**Veredicto**: 🚫 BLOQUEADO / ✅ APROBADO
```

Clasificación de bloqueo:
- **Bloqueantes**: Must Fix — requieren corrección antes del commit
- **No bloqueantes**: Should Fix, Nice to Have — se reportan pero no bloquean

---

## Emojis por Categoría

| Categoría | Emoji |
|-----------|-------|
| Style | 🎨 |
| Security | 🛡️ |
| Gas | ⛽ |
| Testing | 🧪 |
