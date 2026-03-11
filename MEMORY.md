# MEMORY.md — Long-term Memory

_Curated knowledge. Updated when something significant happens._

## Identité
- Je suis **Clawde** ⚙️, assistant personnel de Gaëtan
- Superviseur autonome d'équipe multi-agents
- **Paradigme** : Exécuter avant de demander, synthétiser avant de répondre
- Valeurs : excellence, autonomie, sobriété tokens, concision

## Gaëtan
- Senior tech, DevOps expert
- Objectif : produits innovants, techniquement exigeants
- Contact : Telegram @zenman33370 (id: 8719274407)
- GitHub : gaetanars
- Attente : Clawde = superviseur autonome, pas interface passive

## Infrastructure
- OS : Ubuntu arm64, Oracle Cloud
- GitHub : ClawdeGenai33 (PAT, SSH+GPG configurés)
- Claude Code Pro disponible (abonnement Anthropic)
- Accès free tier Google Generative AI ✓

## Équipe Agents (7 agents)
| Agent | Modèle Primary | Fallback |
|-------|---|---|
| **main** (moi) | gemini-3-flash-preview | gemini-2.5-flash → haiku |
| **planner** 📋 | gemini-2.5-flash | haiku |
| **architect** ⚡ | gemini-2.5-flash | haiku |
| **developer** 🔨 | sonnet-4-6 | gemini-2.5-flash |
| **code-review** 🔍 | gemini-2.5-flash-lite | haiku |
| **test-engineer** 🧪 | gemini-2.5-flash-lite | haiku |
| **documenter** 📝 | gemini-2.5-flash-lite | haiku |
| **researcher** 🔬 | gemini-2.5-flash | haiku |

**Stratégie** : Google free tier + Anthropic, agents légers sur gemini-lite, developer sur sonnet pour complexité.

## Projets Actifs

### AlertLens (AlertLens/AlertLens)
**Vision:** Modern UI for Prometheus Alertmanager — visualize, silence, manage configs.

**URL:** https://github.com/AlertLens/AlertLens

**Stack:** Go 1.22.2 + SvelteKit + Tailwind CSS | Stateless (no DB) | go:embed packaging

**Phase 1 (Current):** Visualization + Alertmanager Configuration
- Alert Kanban/list views (filtering, grouping)
- Multi-instance aggregation
- Routing tree visualizer
- Silences + bulk actions
- Configuration builder (YAML preview)
- GitOps integration (GitHub/GitLab)
- JWT admin auth

**Phase 2 (Future):** IRM (Incident Response Management)
- Workflow management, escalations, incident tracking
- Performance-first, accessible UX

**Principles:**
- Excellence & state-of-the-art UX
- Security: 0 flaws (auth, YAML injection, CSRF, XSS)
- Quality: Comprehensive functional testing (local setup, automated)
- Code quality: clean, well-reviewed
- Token efficiency: careful phasing
- Stateless architecture

**Roadmap Status:** ✅ Defined & tracked in GitHub issues
- **Phase 1** (#20-#24): Visualization + Config (5 features)
- **Phase 2** (#25-#29): IRM Epic (5 modules)  
- **Security** (#30-#33): 4 critical vectors

**Next:** Architect designs Phase 1 security architecture

## GitHub Access
- **ClawdeGenai33:** Flagged as spam (2026-03-07), Ticket #4138748 (monitoring)
- **AlertLens PAT:** New dedicated token (expires 2026-04-07) ✓
  - Scopes: Limited to `AlertLens/AlertLens` repository only
  - Account: gaetanars (via `gh auth login`)
  - Status: **ACTIVE** — clone synced, push/pull working ✓
- **Local Dev:** `/tmp/AlertLens` ready for development
- **Tools:** Chromium ✓, Go 1.22.2 ✓, Node/npm ✓, gh CLI ✓

## Conventions
- Commits signés GPG
- Issues avant code
- Toujours vérifier les données avant proposer
- PRs complètes (desc, tests, docs)
