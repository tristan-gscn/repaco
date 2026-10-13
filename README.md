# 🛡️ repaco — Testeur Universel pour Projets 42

**repaco** est un testeur automatisé, universel et exhaustif pour l'ensemble des projets du cursus 42, codé en **Go**.

---

## 🎯 Philosophie

1. **Zéro configuration, une seule commande** : Lancez simplement `repaco` dans le dossier de votre projet. Il détecte automatiquement le projet 42 concerné, vérifie le Makefile / les normes de compilation, prépare l'environnement et déroule la suite de tests.
2. **Exhaustivité & Précision chirurgicale** : Pas de simples tests "OK / KO" cryptiques. Chaque cas de test teste des cas limites réels (valeurs extrêmes, `NULL`, `INT_MIN`/`INT_MAX`, buffers minimaux, relink, etc.).
3. **Clarté totale des erreurs** : Lorsqu'un test échoue, repaco explique **pourquoi** (différence précise caractère par caractère avec affichage des caractères invisibles `\0`, `\n`, trace de crash SIGSEGV/SIGBUS, fuite mémoire AddressSanitizer/leaks) et vous donne un **conseil d'action concret** pour corriger le bug.
4. **Interface TUI dynamique & ergonomique** : Développée avec la suite moderne Charm (*Bubble Tea* & *Lipgloss*), avec vue en écran partagé, navigation clavier fluide, filtrage instantané des échecs (`f`), et mode sans TUI (`--no-tui`) pour la CI.
5. **Esprit critique sur le code existant** : Les projets validés par les pairs peuvent contenir des cas limites oubliés par les correcteurs ; `repaco` est conçu pour viser la perfection absolue.

---

## 🏗️ Architecture du Projet

```
repaco/
├── Makefile                       # Build, test, lint du binaire repaco
├── go.mod / go.sum                # Dépendances Go (Charmbracelet Bubbletea, Lipgloss, Bubbles)
├── .gitignore
├── cmd/
│   └── repaco/
│       └── main.go                # Point d'entrée CLI, flags, loop TUI/CLI
├── internal/
│   ├── config/                    # Configuration globale et options d'exécution
│   ├── model/                     # Modèles de données (TestCase, Status, TestResult, Stats, Event)
│   ├── detector/                  # Moteur heuristique d'identification des projets 42
│   ├── builder/                   # Vérification Makefile (all, clean, fclean, re, relink) et build C
│   ├── runner/                    # Isolation de sous-processus, timeout, signaux, AddressSanitizer
│   ├── engine/                    # Orchestrateur d'exécution et émetteur d'événements temps réel
│   ├── ui/
│   │   ├── tui/                   # Interface TUI dynamique interactive (Bubble Tea)
│   │   └── cli/                   # Afficheur terminal non-interactif (CI / headless)
│   └── projects/                  # Suites de tests modulaires par projet
│       ├── registry.go            # Registre d'enregistrement des suites
│       ├── libft/                 # Suite Libft
│       ├── ftprintf/              # Suite ft_printf
│       ├── getnextline/           # Suite get_next_line
│       └── pushswap/              # Suite push_swap (nouvelle version 1.1)
└── pkg/
    └── diff/                      # Moteur de diff avec échappement des caractères invisibles
```

---

## 🚀 Utilisation

### Compilation
```bash
make build
```
Le binaire est produit dans `bin/repaco`.

### Exécution
```bash
# Tester le projet dans le dossier courant en mode interactif (TUI) :
repaco

# Tester un projet situé dans un autre dossier :
repaco /chemin/vers/mon-projet

# Mode headless (console pure, idéal pour scripts ou CI) :
repaco --no-tui /chemin/vers/mon-projet

# Forcer un projet particulier si besoin :
repaco --project push_swap /chemin/vers/mon-projet

# Arrêter au premier échec :
repaco --fail-fast
```

### Raccourcis de l'interface TUI
| Touche | Action |
|---|---|
| `q` / `Ctrl+C` | Quitter l'application |
| `↑` / `↓` ou `k` / `j` | Naviguer dans la liste des tests / faire défiler les détails |
| `Tab` | Basculer entre le panneau liste et le panneau détails |
| `f` | Basculer le filtre (afficher uniquement les échecs) |
| `Home` / `End` | Aller au premier / dernier test |

---

## 🧪 Tests du framework
```bash
make test  # Exécute tous les tests unitaires Go
make lint  # Exécute go vet
```
