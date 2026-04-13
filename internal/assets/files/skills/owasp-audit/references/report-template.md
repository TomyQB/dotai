# Plantilla de Reporte de Auditoría OWASP ASVS

## Formato Completo (invocación manual)

Usa esta estructura exacta para el reporte:

```markdown
# 🔒 Informe de Auditoría de Seguridad — OWASP ASVS

## Metadatos de la Auditoría
- **Alcance**: [Archivos cambiados / Proyecto completo]
- **Stack detectado**: [ej. Next.js 14 + Prisma + PostgreSQL]
- **Archivos analizados**: [Lista de archivos revisados]
- **Versión ASVS**: 4.0
- **Fecha**: [Fecha actual]

## Resumen Ejecutivo

[2-3 oraciones: total de hallazgos por severidad, evaluación general de riesgo, elementos de máxima prioridad]

## Hallazgos

### [EMOJI] [SEVERIDAD] — [Título descriptivo del hallazgo]
- **Requisito ASVS**: [ej. V5.3.4 — Codificación de Salida]
- **Ubicación**: `archivo.ts:42`
- **Descripción**: [Explicación clara de la vulnerabilidad y por qué es un problema]
- **Impacto**: [Qué podría lograr un atacante explotando esto]
- **Evidencia**:
  ```[lang]
  // Fragmento del código vulnerable
  ```
- **Remediación**:
  ```[lang]
  // Código corregido con la solución
  ```
- **Referencias**: [CWE-XXX, enlace OWASP relevante]

[Repetir para cada hallazgo, ordenados de CRÍTICO a INFORMATIVO]

## Tabla Resumen

| ID | Severidad | Hallazgo | Req. ASVS | Archivo:Línea | Estado |
|----|-----------|----------|-----------|---------------|--------|
| C1 | 🔴 CRÍTICO | [Título] | V2.1.1 | auth.ts:42 | Abierto |
| H1 | 🟠 ALTO | [Título] | V5.3.4 | api.ts:88 | Abierto |
| M1 | 🟡 MEDIO | [Título] | V14.2.1 | config.ts:15 | Abierto |
| L1 | 🔵 BAJO | [Título] | V7.1.1 | logger.ts:23 | Abierto |
| I1 | ⚪ INFO | [Título] | V1.2.3 | — | Abierto |

Esta tabla SIEMPRE debe incluirse completa. Es la referencia principal del usuario para localizar y priorizar hallazgos.

## Recomendaciones

[Lista priorizada de acciones para mejorar la postura de seguridad, ordenada por impacto]
```

---

## Formato Condensado (invocación desde commit-and-push)

Cuando el skill se invoca como parte del flujo de commit, usar este formato reducido:

```markdown
## 🔒 Auditoría OWASP — Resumen

**Archivos auditados**: [N archivos]
**Hallazgos**: [N críticos, N altos, N medios, N bajos, N info]

### Hallazgos Bloqueantes (requieren acción antes del commit)

| ID | Severidad | Hallazgo | Archivo:Línea | Remediación breve |
|----|-----------|----------|---------------|-------------------|
| C1 | 🔴 CRÍTICO | [Título] | auth.ts:42 | [1 línea de acción] |

### Hallazgos No Bloqueantes (informativos)

| ID | Severidad | Hallazgo | Archivo:Línea |
|----|-----------|----------|---------------|
| M1 | 🟡 MEDIO | [Título] | config.ts:15 |

**Veredicto**: 🚫 BLOQUEADO / ✅ APROBADO
```

Clasificación de bloqueo:
- **Bloqueantes**: CRÍTICO y ALTO — requieren corrección antes del commit
- **No bloqueantes**: MEDIO, BAJO, INFORMATIVO — se reportan pero no bloquean

---

## Ejemplo de Hallazgo Completo

```markdown
### 🔴 CRÍTICO — Inyección SQL en endpoint de búsqueda de usuarios

- **Requisito ASVS**: V5.3.4 — Prevención de Inyección SQL
- **Ubicación**: `src/routes/users.ts:47`
- **Descripción**: El endpoint `/api/users/search` concatena directamente el parámetro `query` del usuario en la sentencia SQL sin parametrizar, permitiendo inyección SQL arbitraria.
- **Impacto**: Un atacante puede extraer toda la base de datos, modificar registros, o escalar privilegios mediante UNION-based o blind SQL injection.
- **Evidencia**:
  ```typescript
  // src/routes/users.ts:47
  const result = await db.query(`SELECT * FROM users WHERE name LIKE '%${req.query.q}%'`);
  ```
- **Remediación**:
  ```typescript
  // Usar queries parametrizadas
  const result = await db.query(
    'SELECT * FROM users WHERE name LIKE $1',
    [`%${req.query.q}%`]
  );
  ```
- **Referencias**: CWE-89, OWASP SQL Injection Prevention Cheat Sheet
```
