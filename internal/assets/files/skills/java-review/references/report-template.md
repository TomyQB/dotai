# Plantilla de Reporte de Code Review — Java / Spring Boot

## Formato Completo (invocación manual)

Usa esta estructura exacta para el reporte:

```markdown
# 📝 Code Review — Java / Spring Boot

## Metadatos de la Revisión
- **Alcance**: [Archivos cambiados / Proyecto completo]
- **Java version**: [ej. 21]
- **Spring Boot version**: [ej. 3.3.x]
- **Build tool**: [Maven / Gradle]
- **Archivos revisados**: [Lista de archivos]
- **Verificaciones ejecutadas**: [compilación, tests, checkstyle]
- **Fecha**: [Fecha actual]

## Resumen Ejecutivo

[2-3 oraciones: total de hallazgos por categoría, calidad general del código, áreas de mejora principales]

## Hallazgos

### [EMOJI] [CATEGORÍA] — [Título descriptivo]
- **Severidad**: Must Fix / Should Fix / Nice to Have
- **Ubicación**: `UserService.java:42`
- **Problema**: [Descripción concisa del problema]
- **Sugerencia**:
  ```java
  // Código o patrón correcto
  ```

[Repetir para cada hallazgo, agrupados por categoría]

## Tabla Resumen

| # | Categoría | Severidad | Hallazgo | Archivo:Línea |
|---|-----------|-----------|----------|---------------|
| 1 | 🎨 Style | Should Fix | [Título] | UserDto.java:12 |
| 2 | 🛡️ Security | Must Fix | [Título] | AuthController.java:87 |
| 3 | ⚡ Performance | Nice to Have | [Título] | OrderService.java:34 |
| 4 | 🧪 Testing | Should Fix | [Título] | — |
| 5 | 🍃 Spring | Must Fix | [Título] | AppConfig.java:22 |

## Verificación Automática

| Comando | Resultado |
|---------|-----------|
| Compilación (`mvn compile` / `gradle build`) | ✅ / ❌ (detalles) |
| Tests (`mvn test` / `gradle test`) | ✅ / ❌ (N passed, N failed) |
| Checkstyle (si configurado) | ✅ / ❌ (detalles) |

## Recomendaciones

[Lista priorizada de acciones para mejorar la calidad del código]
```

---

## Formato Condensado (invocación desde commit-and-push)

```markdown
## 📝 Code Review Java — Resumen

**Archivos revisados**: [N archivos]
**Verificación**: compile ✅/❌ | test ✅/❌
**Hallazgos**: [N must fix, N should fix, N nice to have]

### Hallazgos Bloqueantes (Must Fix)

| # | Categoría | Hallazgo | Archivo:Línea | Sugerencia breve |
|---|-----------|----------|---------------|------------------|
| 1 | 🛡️ Security | [Título] | AuthController.java:87 | [1 línea] |

### Hallazgos No Bloqueantes

| # | Categoría | Severidad | Hallazgo | Archivo:Línea |
|---|-----------|-----------|----------|---------------|
| 2 | 🎨 Style | Should Fix | [Título] | UserDto.java:12 |

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
| Security Patterns | 🛡️ |
| Performance | ⚡ |
| Testing | 🧪 |
| Spring Boot | 🍃 |
