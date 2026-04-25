#!/bin/bash
# Primera edición/subagente de la sesión en repo Git: bloquea para preguntar sobre rama.
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
OBLIGATORIO: Usa la herramienta AskUserQuestion (NO preguntes en texto plano) con estas opciones:
- Pregunta: "Es necesario crear una nueva rama para esta sesion?"
- Opciones: "No, continuar en la rama actual" / "Si, crear rama nueva"

Si el usuario responde SI, usa AskUserQuestion DE NUEVO para preguntar:
- Pregunta: "Que tipo de rama necesitas?"
- Opciones: "feature (desde develop)" / "fix-release (desde release)" / "hotfix (desde hotfixes)"
Luego pide el nombre de la rama (puede ser en texto plano o AskUserQuestion).
Ejecutar: git.exe checkout <base> && git.exe pull && git.exe checkout -b <tipo>/<nombre>
Confirmar creacion antes de continuar editando.
MSG
exit 2
