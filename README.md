# 🛡️ repaco — Testeur Universel pour Projets 42

**repaco** est un testeur automatisé, universel et exhaustif pour l'ensemble des projets du cursus 42, codé en **Go** avec une interface **100% TUI dynamique**.

---

## 🎯 Philosophie

1. **Zéro configuration, une seule commande** : Tapez simplement `repaco` dans le terminal. L'interface TUI s'ouvre immédiatement, détecte le projet 42 dans le dossier courant, prépare l'environnement et déroule la suite de tests en direct.
2. **Interface Full TUI immersive** : Aucune sortie texte brute ou cryptique. Tout se passe dans une interface plein écran moderne développée avec Charm (*Bubble Tea* & *Lipgloss*) :
   - Écran scindé avec liste des tests à gauche et inspecteur chirurgical à droite.
   - Badges en temps réel : `✓ PASS`, `✗ FAIL`, `💥 CRASH`, `⏱ TIME`, `💧 LEAK`.
   - Filtrage instantané des échecs avec la touche `f`.
   - Écran d'assistance guidé si aucun projet 42 n'est détecté.
3. **Exhaustivité & Précision chirurgicale** : Chaque cas de test évalue les vraies exigences du sujet (cas limites, valeurs extrêmes, `NULL`, `INT_MIN`/`INT_MAX`, `BUFFER_SIZE` extrêmes, conformité du Makefile sans relink).
4. **Compréhension immédiate des erreurs** : Lorsqu'un test échoue, repaco explique clairement :
   - Ce qui a été testé (arguments et entrées).
   - Ce qui est attendu vs ce qui a été obtenu (avec visualisation des caractères invisibles `\0`, `\n`).
   - La cause exacte (déréférencement nul, dépassement de mémoire tampon, fuite mémoire).
   - Un **conseil de correction concret** (`💡 Comment corriger : ...`).

---

## 🚀 Installation & Utilisation

### 1. Installation globale
À la racine du repo `repaco` :
```bash
make install
```
Le binaire est installé dans `~/.local/bin/repaco` (inclus dans votre `$PATH`).

### 2. Lancement
Dans n'importe quel dossier de projet 42 :
```bash
repaco
```
*(Ou en spécifiant un dossier : `repaco /chemin/vers/mon-projet`)*

L'interface TUI s'ouvre instantanément.

---

## ⌨️ Raccourcis clavier (TUI)

| Touche | Action |
|---|---|
| `q` / `Ctrl+C` | Quitter |
| `↑` / `↓` ou `k` / `j` | Naviguer dans les tests / défiler les détails |
| `Tab` | Basculer entre le panneau liste et le panneau détails |
| `f` | Filtrer : afficher uniquement les tests en échec |
| `g` / `G` | Aller au début / à la fin de la liste |

---

## 🧪 Tests du framework repaco
```bash
make test  # Exécute la suite de tests Go
make lint  # Exécute go vet
```
