# Recuperacion: mover commits a otra rama

Si el usuario hizo push en la rama equivocada y necesita mover commits a una rama nueva.

## Mapping de rama base segun perfil

El perfil instalado determina las ramas base validas. Detectalo mirando el script
`git-branch-check.sh` instalado o el marcador `_dotai.profile` en `settings.json`.

### Perfil Personal (trunk simplificado)

| Tipo de rama | Rama base |
|---|---|
| `feature/*` | `develop` |
| `fix/*` | `develop` |
| `hotfix/*` | `main` |

### Perfil Work (gitflow tradicional)

| Tipo de rama | Rama base |
|---|---|
| `feature/*` | `develop` |
| `fix-release/*` | `release` |
| `hotfix/*` | `hotfixes` |

## Flujo

1. **Identificar los commits** a mover: `git log --oneline -N`
2. **Guardar cambios locales**: `git stash --include-untracked`
3. **Ir a la rama base** segun el mapping de arriba: `git checkout <rama-base>`
4. **Actualizar**: `git pull`
5. **Crear nueva rama**: `git checkout -b <tipo>/<nombre>`
6. **Cherry-pick** de los commits: `git cherry-pick <commit-hash>`
7. **Limpiar stash**: `git stash drop`
8. **Push**: `git push -u origin <tipo>/<nombre>`

## Notas

- Usar el `git` nativo del sistema (no `git.exe`) — respeta el setup multi-cuenta `includeIf`.
- Si hay cambios locales que bloquean el cherry-pick, hacer `git stash --include-untracked` primero.
- Si hay conflictos en el cherry-pick, avisar al usuario para resolucion manual.
