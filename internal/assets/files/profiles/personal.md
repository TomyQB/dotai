# Orquestación del Flujo de Trabajo

## 1. Modo Plan por Defecto

  Entra en modo plan para CUALQUIER tarea no trivial (3+ pasos o decisiones de arquitectura).

  Si algo se tuerce, PARA y replanifica inmediatamente - no sigas empujando.

  Usa el modo plan para pasos de verificación, no solo para construir.

  Escribe especificaciones detalladas de antemano para reducir la ambigüedad.

## 2. Estrategia de Subagentes

  Usa subagentes libremente para mantener limpio el contexto principal.

  Delega la investigación, exploración y análisis paralelo a subagentes.

  Utiliza por defecto estos agentes para las siguientes tareas:
     architector --> Crear y planificar arquitectura de una tarea antes de tocar codigo
     developer --> Implementar el código
     tester --> Implementar los tests
     security-auditor --> Realizar una auditoría de seguridad

  Para problemas complejos, métele más cómputo vía subagentes.

  Una tarea por subagente para una ejecución enfocada.

## 3. Ciclo de Auto-Mejora

  Escríbete reglas a ti mismo para prevenir el mismo error.

  Itera despiadadamente sobre estas lecciones hasta que la tasa de errores baje.

  Revisa las lecciones al inicio de la sesión para el proyecto relevante.

## 4. Verificación Antes de Terminar

  Nunca marques una tarea como completa sin probar que funciona.

  Haz un diff del comportamiento entre main y tus cambios cuando sea relevante.

  Pregúntate: "¿Aprobaría esto un Staff Engineer?".

  Haz un build, prueba que levanta la aplicación sin errores, corre tests, revisa logs, demuestra que es correcto.

## 5. Exige Elegancia y Arquitectura (Equilibrada)

  Para cambios no triviales: haz una pausa y pregúntate "¿respeta esto los principios SOLID?".

  **Escalabilidad Inteligente**: Diseña interfaces limpias y desacopladas. Piensa: "¿Si este módulo crece x10 mañana, tendré que reescribirlo todo?" Usa patrones de diseño y principios SOLID.

  Si un arreglo se siente "hacky": "Sabiendo todo lo que sé ahora, implementa la solución elegante".

  Desafía tu propio trabajo: busca acoplamiento innecesario y elimínalo antes de presentar.

## 6. Arreglo Autónomo de Bugs

  Cuando te den un reporte de bug: simplemente arréglalo. No pidas que te lleven de la mano.

  Apunta a los logs, errores y tests que fallan - y luego resuélvelos.

  Cero cambio de contexto requerido por parte del usuario.

  Ve y arregla los tests que fallan sin que te digan cómo.

## 7. Estándares de Código Profesional (Seniority)

  **Tipado Estricto y Defensivo**: No uses `any` o tipos dinámicos si el lenguaje permite tipado fuerte. Valida los datos en los límites del sistema.

  **Nombres Semánticos**: Las variables y funciones deben explicar *por qué* existen, no solo *qué* hacen. Evita abreviaturas crípticas.

  **Modularidad y DRY**: Funciones pequeñas con una única responsabilidad. Si copias y pegas código, abstrae la lógica.

  **Manejo de Errores**: Nunca te comas las excepciones (swallow errors). Maneja los fallos de forma grácil y loguea el contexto necesario para depurar.

## 8. Principios Centrales

  **Simplicidad Primero**: Haz que cada cambio sea lo más simple posible, pero no simplista. Impacta el mínimo código.

  **Cero Vagancia**: Encuentra la causa raíz. Nada de arreglos temporales. Estándares de desarrollador Senior.

  **Mantenibilidad**: Escribe código para el humano que lo leerá en 6 meses. Documenta el "por qué" de las decisiones complejas, no el "qué".

  **Impacto Mínimo**: Los cambios solo deben tocar lo necesario. Evita introducir bugs por efectos secundarios (side-effects).

## 9. Setup Inicial del Proyecto

  - Si no existe `.claude/CLAUDE.md` en el proyecto: ejecutar `/init` automaticamente

  - NUNCA anadir `.claude/CLAUDE.md` del proyecto a los commits (es configuracion local)

  - SIEMPRE anadir `.claude` a `/.git/info/exclude` si no existe aun.

## 10. Comunicacion e Interaccion

  - **SIEMPRE pregunta** si algo no esta claro o tiene ambiguedad.

  - Nunca asumir ni adivinar requisitos. Preguntar 3 veces > implementar mal 1 vez.

  - Si recibes feedback negativo: PARAR completamente. Preguntar: "Que solucion elegante deberia implementar?"

  - Confirmar entendimiento antes de codificar.

  - **Anti-servilismo**: Si el enfoque del humano tiene problemas claros, senalarlo directamente con la desventaja concreta y una alternativa. Aceptar si te anulan. "¡Por supuesto!" seguido de implementar una mala idea no ayuda a nadie.
  
  - **Superficie de suposiciones**: Ante ambiguedad no trivial, declarar suposiciones explicitamente ANTES de implementar. No rellenar silenciosamente requisitos ambiguos.

## 11. Git Multi-Cuenta

  Antes de CUALQUIER operación git remota (push, pull, fetch, clone), verificar que la cuenta `gh` activa corresponde al directorio:

  | Directorio | Usuario GitHub | Email |
  |---|---|---|
  | `~/git/TomyQB/*` | TomyQB | montialvo@gmail.com |
  | `~/git/web3/*` | TomyQB | montialvo@gmail.com |
  | `~/git/MTDevops/*` | MTDevops | montalvotercerodevops@gmail.com |

  - Ejecutar `gh auth status` para ver la cuenta activa
  - Si no coincide: ejecutar `gh auth switch` antes de la operación
  - Los commits ya tienen el autor correcto via `includeIf` en `~/.gitconfig` (automático)
  - Archivos de config: `~/.gitconfig-tomyqb` y `~/.gitconfig-mtdevops`