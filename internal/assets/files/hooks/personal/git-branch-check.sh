#!/bin/bash
# Primera edición/subagente de la sesión en repo Git: bloquea para preguntar sobre rama.
# Variante PERSONAL: trunk simplificado — feature/fix desde develop, hotfix desde main.
INPUT=$(cat)

SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // "default"')
FLAG="/tmp/.claude-branch-asked-${SESSION_ID}"

if [ -f "$FLAG" ]; then
  exit 0
fi

CWD=$(echo "$INPUT" | jq -r '.cwd // ""')

# Detectar repo Git buscando .git/ (sin depender de git.exe)
IS_GIT=false
CHECK_DIR="$CWD"
while [ "$CHECK_DIR" != "/" ] && [ -n "$CHECK_DIR" ]; do
  if [ -d "$CHECK_DIR/.git" ]; then
    IS_GIT=true
    break
  fi
  CHECK_DIR=$(dirname "$CHECK_DIR")
done

if [ "$IS_GIT" = false ]; then
  touch "$FLAG"
  exit 0
fi

touch "$FLAG"
cat >&2 <<'MSG'
PRIMERA EDICION DE LA SESION en repo Git.
OBLIGATORIO: Usa la herramienta AskUserQuestion (NO preguntes en texto plano).

Paso 1 - Pregunta:
  "Es necesario crear una nueva rama para esta sesion?"
  Opciones:
    - "No, continuar en la rama actual"
    - "Si, crear rama nueva"

Si el usuario responde SI, Paso 2 - Pregunta:
  "Que tipo de rama necesitas?"
  Opciones (trunk simplificado):
    - "feature (desde develop)"  -> funcionalidad nueva
    - "fix (desde develop)"      -> correccion no urgente (incluye docs, chores, refactors)
    - "hotfix (desde main)"      -> correccion urgente a produccion

Paso 3 - Pide el nombre de la rama (texto plano o AskUserQuestion).

Paso 4 - Antes de ejecutar, VERIFICA que la rama base existe:
  - Para feature/fix: git rev-parse --verify develop || avisa al usuario que no existe develop y pide alternativa
  - Para hotfix: git rev-parse --verify main (o master) || avisa al usuario

Paso 5 - Ejecuta:
  git checkout <base>
  git pull origin <base>
  git checkout -b <tipo>/<nombre>

Confirma la creacion antes de continuar editando.
MSG
exit 2
