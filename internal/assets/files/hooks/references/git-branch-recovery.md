# Recuperacion: mover commits a otra rama

Si el usuario hizo push en la rama equivocada y necesita mover commits a una rama nueva:

## Flujo

1. **Identificar los commits** a mover: `git log --oneline -N`
2. **Guardar cambios locales**: `git stash --include-untracked`
3. **Ir a la rama base**: `git checkout <rama-base>` (develop, release, hotfixes segun el tipo)
4. **Actualizar**: `git pull`
5. **Crear nueva rama**: `git checkout -b <tipo>/<nombre>`
6. **Cherry-pick** de los commits: `git cherry-pick <commit-hash>`
7. **Limpiar stash**: `git stash drop`
8. **Push**: `git push -u origin <tipo>/<nombre>`

## Notas

- Usar `git` (Windows) para cherry-pick y push porque requieren identidad/credenciales configuradas en Windows.
- Si hay cambios locales que bloquean el cherry-pick, hacer `git stash --include-untracked` primero.
- Si hay conflictos en el cherry-pick, avisar al usuario para resolucion manual.
