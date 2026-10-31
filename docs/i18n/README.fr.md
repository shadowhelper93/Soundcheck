# Soundcheck (Français)

> **Sonde de Vérification Préalable Déterministe et Résolveur de Rôles pour Agents de Code IA**

Langues : [English](../../README.md) · [Español](README.es.md) · [فارسی](README.fa.md)

Soundcheck teste vos outils CLI d'agents de développement installés (`claude`, `codex`, `opencode`, `agy`, `gemini`, `copilot`) avant de leur confier des tâches. Il vérifie leur disponibilité, intercepte les faux codes de sortie zéro, normalise les comportements anormaux, résout les attributions de rôles de manière déterministe sans consommer de jetons LLM, et orchestre le basculement automatique en cas de défaillance.

---

## Fonctionnalités Principales

- **Détection des Faux Succès** : De nombreux outils CLI se terminent avec un code de sortie `0` même lors d'une défaillance critique (limites de débit, jetons OAuth expirés). Soundcheck normalise ces retours en catégories d'erreur précises (`429`, `401`, `corrupted`).
- **Attribution Déterministe de Rôles à Coût Nul** : Détermine quel agent prend en charge les rôles `primary`, `subagent`, `reviewer` ou `auxiliary` en ≤ 50µs sans solliciter d'API LLM payante.
- **Exécution Headless avec Tolérance aux Pannes** : Lancez vos invites via `soundcheck exec "prompt"`. Si l'agent principal est bloqué ou non authentifié, Soundcheck bascule automatiquement vers l'agent valide suivant dans la chaîne de secours.
- **Sondage en Quatre Niveaux (Tiers)** :
  - **L0 Présence** : Vérifie l'existence du binaire et sa version dans le `$PATH` (≤ 150ms).
  - **L1 Authentification** : Contrôle les identifiants et sessions avec mise en cache TTL pour protéger les flux de rafraîchissement OAuth.
  - **L2 Test de Validation (Smoke)** : Exécution de contrôle rapide et non destructive.
  - **L3 Vérification du Modèle** : Test direct de réponse du modèle en direct.
- **Binaire Statique Autonome** : Serveur backend et interface utilisateur web React intégrés dans un exécutable statique unique (~8.3 Mo, budget ≤ 20 Mo). Aucune dépendance externe à l'exécution.
- **Protocole de Contexte de Modèle (MCP)** : Serveur stdio MCP natif (`soundcheck mcp`) utilisable immédiatement par Claude Desktop, Cursor et Antigravity.
- **Extension VS Code Officielle** : Disponible dans `editors/vscode` avec barre d'état interactive, explorateur d'arborescence et tableau de bord web intégré.

---

## Installation

### macOS et Linux
```bash
curl -fsSL https://raw.githubusercontent.com/soundcheck/soundcheck/main/scripts/install.sh | sh
```

### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/soundcheck/soundcheck/main/scripts/install.ps1 | iex
```

### Homebrew
```bash
brew install soundcheck/tap/soundcheck
```

### Scoop (Windows)
```powershell
scoop bucket add soundcheck https://github.com/soundcheck/scoop-bucket
scoop install soundcheck
```

### Mise à Jour Automatique
```bash
soundcheck upgrade
```

### Compilation depuis les Sources
```bash
git clone https://github.com/soundcheck/soundcheck.git
cd soundcheck
make build
./bin/soundcheck doctor
```

---

## Référence des Commandes CLI

| Commande | Description |
|---|---|
| `soundcheck doctor [--deep] [--json]` | Bilan de santé complet des adaptateurs installés |
| `soundcheck probe <adapter> [--deep]` | Sonde un adaptateur spécifique (ex. `soundcheck probe claude`) |
| `soundcheck resolve [role]` | Affiche l'attribution déterministe des rôles (ex. `soundcheck resolve primary`) |
| `soundcheck exec <prompt> [--role <role>] [--adapter <adapter>]` | Exécute une tâche avec basculement automatique |
| `soundcheck run <adapter> [-- <args>]` | Ouvre une session PTY interactive avec flux en direct |
| `soundcheck manifest [--json]` | Exporte le manifeste de capacités en format JSON |
| `soundcheck serve [--port 3000] [--token <token>]` | Lance le serveur HTTP et le tableau de bord web (port 3000 par défaut) |
| `soundcheck mcp` | Lance le serveur stdio conforme au Model Context Protocol (MCP) |
| `soundcheck upgrade [--check]` | Vérifie les mises à jour et met à jour l'exécutable |
| `soundcheck version` | Affiche la version et les métadonnées de compilation |

---

## Référence de l'API REST et Streaming

En lançant `soundcheck serve [--port 3000]`, les points de terminaison suivants sont accessibles :

| Point d'accès | Méthode | Description |
|---|---|---|
| `/api/health` | `GET` | Contrôle d'état et version du serveur |
| `/api/v1/doctor` | `GET` | Rapport d'état complet (flux SSE via `Accept: text/event-stream`) |
| `/api/v1/probe` | `GET` | Sondage de l'ensemble des adaptateurs CLI |
| `/api/v1/probe/{id}` | `POST` | Déclenche le sondage ciblé d'un adaptateur |
| `/api/v1/resolve` | `GET`, `POST` | Résolution déterministe des rôles et chaînes de secours |
| `/api/v1/manifest` | `GET` | Manifeste complet des capacités en JSON |
| `/api/v1/config` | `GET` | Lecture de la configuration active (en-tête `X-Config-Path`) |
| `/api/v1/config` | `PUT` | Validation atomique et sauvegarde sur fichier YAML |
| `/api/v1/config/validate` | `POST` | Vérifie les candidats face aux adaptateurs réels du système |
| `/api/v1/terminal/ws` | `WS` | Session PTY interactive en WebSocket |

---

## Découverte Dynamique des Modèles (`modelsCommand`)

Les adaptateurs peuvent déclarer une commande de découverte dans leur configuration YAML (par exemple `modelsCommand: "codex debug models"` pour Codex).

- **Avec `modelsCommand`** : Soundcheck exécute cette commande lors du sondage. Les modèles réellement disponibles sur le compte connecté remplacent ou complètent le catalogue local.
- **Sans `modelsCommand`** : Pour les outils sans commande d'inspection, Soundcheck conserve le catalogue statique déclaré dans le fichier YAML.

---

## Architecture

Soundcheck applique une **Architecture Hexagonale** stricte :

- **Noyau Fonctionnel Pur** (`internal/core/*`) :
  - Déterministe : données en entrée → données en sortie.
  - Zéro E/S, zéro horloge système directe, zéro variable d'environnement.
  - Bibliothèque standard de Go uniquement, validé par `internal/arch_test.go`.
- **Enveloppe Impérative** (`internal/edge/*`, `internal/transport/*`) :
  - Exécution de sous-processus (`internal/edge/exec`), streaming PTY, écriture atomique de fichiers.
  - Transports : CLI (`cobra`), HTTP (`net/http` avec interface React intégrée), WebSocket et serveur MCP.

---

## Documentation Contributeurs

- [Guide de Création d'Adaptateurs](../../docs/ADAPTER_AUTHORING.md) — Spécification pour intégrer de nouveaux agents.
- [Guide des Quirks et Anomalies](../../docs/QUIRK_AUTHORING.md) — Recensement et règles de normalisation des faux succès.
- [Guide de Contribution](../../CONTRIBUTING.md) — Organisation par voies (lanes), règles de code et critères de validation.
- [Manuel d'Architecture Claude/Agent](../../CLAUDE.md) — Règles techniques pour agents d'assistance et développement.

---

## Licence

Apache-2.0. Copyright (c) 2026 Contributeurs Soundcheck.
