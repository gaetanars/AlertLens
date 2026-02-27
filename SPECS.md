# Spécifications Produit : AlertLens

## 1. Vision et Objectifs

**AlertLens** est un outil léger et moderne conçu pour combler le fossé entre la visualisation des alertes et l'édition de la configuration d'Alertmanager. Contrairement aux solutions existantes qui se concentrent uniquement sur la lecture, AlertLens permet de comprendre, de visualiser et d'agir sur le cycle de vie complet des alertes.

**Objectifs principaux :**

- Fournir une interface utilisateur ultra-claire et sans friction.
- Rendre la création de silences intuitive et rapide.
- Démystifier et simplifier la configuration d'Alertmanager (Routing tree, Mute times, Receivers).
- Garantir une compatibilité native avec **Prometheus Alertmanager** et **Grafana Mimir**.
- Rester **stateless** : aucune base de données, aucun état local persistant.

---

## 2. Architecture et Stack Technique

Pour garantir la légèreté et la facilité de déploiement (un seul binaire ou conteneur), AlertLens adopte l'architecture suivante :

| Composant | Technologie | Justification |
| --- | --- | --- |
| **Backend** | Go (Golang) | Librairies natives Prometheus/Alertmanager, hautes performances. |
| **Frontend** | SvelteKit | Framework léger, sans Virtual DOM, idéal pour des interfaces réactives. |
| **Styling** | Tailwind CSS + shadcn/svelte | Design moderne, clair, Dark/Light mode natif. |
| **Packaging** | `go:embed` | Frontend compilé embarqué dans le binaire Go. Déploiement zéro dépendance. |
| **État** | Stateless | Aucune base de données. L'état métier repose sur l'API Alertmanager. |

**Contraintes d'API :** Alertmanager API **v2 uniquement**.

---

## 3. Configuration

### 3.1 Fichier de configuration

AlertLens est configuré via un fichier YAML. Chaque clé dispose d'une valeur par défaut.
Les variables d'environnement ont la **priorité la plus haute** et surchargent le fichier de config.

**Priorité (de la plus faible à la plus haute) :** valeurs par défaut → fichier de config → variables d'environnement.

Les variables d'environnement suivent le pattern `ALERTLENS_<SECTION>_<KEY>` (ex: `ALERTLENS_SERVER_PORT`).

```yaml
server:
  host: "0.0.0.0"          # ALERTLENS_SERVER_HOST
  port: 9000                # ALERTLENS_SERVER_PORT

auth:
  admin_password: ""        # ALERTLENS_AUTH_ADMIN_PASSWORD — si vide, mode admin désactivé
  # oidc: (prévu, non implémenté en V1)

alertmanagers:
  - name: "default"
    url: "http://localhost:9093"
    basic_auth:
      username: ""
      password: ""
    # Pour Grafana Mimir multi-tenant :
    tenant_id: ""           # Header X-Scope-OrgID
    tls_skip_verify: false

gitops:
  github:
    token: ""               # ALERTLENS_GITOPS_GITHUB_TOKEN
  gitlab:
    token: ""               # ALERTLENS_GITOPS_GITLAB_TOKEN
    url: "https://gitlab.com"
```

### 3.2 Multi-Alertmanager

AlertLens supporte plusieurs instances Alertmanager simultanément. L'interface propose :

- Une **vue agrégée** de toutes les alertes, tous clusters confondus.
- Un **filtre par instance** pour se concentrer sur un cluster spécifique.
- Un indicateur visuel de l'instance d'origine sur chaque alerte.

---

## 4. Authentification et Modes d'Accès

| Mode | Accès | Protection |
| --- | --- | --- |
| **Lecture seule** | Visualisation des alertes, silences, routing tree | Aucune (public) |
| **Admin** | Tout + création de silences, acks, édition de config | Mot de passe (config) |

- Le mode admin est activé uniquement si `auth.admin_password` est défini dans la config.
- L'authentification admin utilise une session côté frontend (JWT signé par le backend, stocké en mémoire).
- **Prévu (non V1) :** OIDC pour remplacer ou compléter le mot de passe simple.

---

## 5. Fonctionnalités

### Module A : Visualisation

#### A.1 Liste des alertes

- Affichage de toutes les alertes actives, agrégées depuis toutes les instances configurées.
- Deux modes d'affichage :
  - **Kanban** : colonnes par sévérité (`critical`, `warning`, `info`...).
  - **Liste dense** : tableau compact avec tri par colonne.
- **Moteur de filtrage** : syntaxe native Alertmanager matchers (ex: `severity="critical"`, `env=~"prod.*"`, `team!="platform"`).
- Groupement par n'importe quel label (ex: `team`, `environment`, `cluster`).
- Indicateur visuel distinctif pour les alertes en état **Ack visuel** (voir Module B.2).

#### A.2 Routing Tree Visualizer

- Représentation graphique (arbre/nodale) de l'arbre de routage Alertmanager.
- Cliquer sur un nœud affiche les alertes actives correspondant à cette route.
- Affichage des matchers, du receiver cible, et des paramètres de grouping pour chaque nœud.
- Bibliothèque de visualisation : robuste, légère, rendu SVG ou Canvas (à trancher lors de l'implémentation — candidats : D3.js, ELK.js).

---

### Module B : Opérations Live

#### B.1 Silence natif (Alertmanager)

- Création d'un silence AM standard depuis une alerte active (1 clic).
- **Pré-remplissage intelligent** : les matchers sont pré-remplis à partir des labels de l'alerte, éditables avant confirmation.
- Sélecteur de durée humain : "1 heure", "4 heures", "Jusqu'à la fin de la journée", "Weekend", durée personnalisée.
- Gestion des silences existants : liste, édition, expiration anticipée.

#### B.2 Ack visuel (AlertLens)

Mécanisme de "prise en charge" d'une alerte pour indiquer qui travaille sur quoi.

**Fonctionnement :**

- Crée un silence AM en arrière-plan avec des labels réservés :
  - `alertlens_ack_type: "visual"`
  - `alertlens_ack_by: "<identifiant saisi par l'utilisateur>"`
  - `alertlens_ack_comment: "<commentaire optionnel>"`
- L'alerte reste visible dans l'interface (non masquée), mais affiche une indication visuelle distincte (badge, couleur, icône).
- La liste des "Acks actifs" est reconstituée en lisant les silences AM filtrés par `alertlens_ack_type="visual"`.
- **Stateless** : toute l'information est portée par le silence AM. Aucun stockage côté AlertLens.

#### B.3 Bulk Actions

- Sélection multiple d'alertes par checkbox.
- Actions disponibles sur la sélection : **Silence natif** ou **Ack visuel**.
- Les matchers communs sont pré-calculés pour le silence groupé.

---

### Module C : Configuration Builder (Admin uniquement)

*Différenciateur majeur d'AlertLens. Accès restreint au mode admin.*

#### C.1 Éditeur de Routing visuel

- Interface drag & drop ou formulaires imbriqués pour construire l'arbre de routage sans éditer le YAML.
- Gestion de la totalité des champs de route AM : `match`, `match_re`, `matchers`, `continue`, `group_by`, `group_wait`, `group_interval`, `repeat_interval`, `receiver`, routes enfants.
- Support des champs de planification temporelle sur les routes enfants : `mute_time_intervals` (supprime les notifications pendant l'intervalle) et `active_time_intervals` (ne notifie que pendant l'intervalle). Ces champs référencent des time intervals définis dans la section C.2. La route racine ne peut pas porter ces champs (contrainte Alertmanager).
- Aperçu YAML en temps réel du résultat généré.

#### C.2 Time Intervals Manager

- Interface pour définir les `time_intervals` (section racine de la config Alertmanager).
- Chaque time interval est nommé et contient une ou plusieurs spécifications temporelles : plages horaires (`times`), jours de la semaine (`weekdays`), jours du mois (`days_of_month`), mois (`months`), années (`years`), fuseau horaire (`location`).
- Ces intervalles sont ensuite référencés dans les routes (C.1) comme `mute_time_intervals` ou `active_time_intervals`.
- Support des récurrences complexes (jours de la semaine, plages horaires, mois).

#### C.3 Gestion des Receivers

- Formulaires guidés pour chaque type de receiver supporté par AM : Slack, PagerDuty, email, webhook générique, OpsGenie, etc.
- Validation des champs obligatoires avant sauvegarde.

#### C.4 Stratégies de sauvegarde

Deux modes, configurables par instance :

**Mode Écriture disque :**

- AlertLens écrit le fichier `alertmanager.yml` à un chemin configuré (AlertLens doit avoir accès au filesystem de l'AM).
- Après écriture, appel optionnel d'un webhook HTTP configurable (ex: pour déclencher un reload ou une notification).

**Mode Git (GitHub / GitLab) :**

- AlertLens commit et push le fichier `alertmanager.yml` sur un dépôt distant via l'API GitHub ou GitLab (HTTPS + access token).
- Paramètres configurables : repo, branche cible, chemin du fichier, message de commit.
- Après le push, déclenchement optionnel d'un webhook HTTP configurable (ex: pour ouvrir une PR, déclencher un pipeline CI).

**Dans tous les cas :**

- Le backend Go **valide** la configuration avec les packages officiels Prometheus avant toute écriture ou push.
- Affichage d'un diff YAML avant confirmation.

---

## 6. Compatibilité et Sécurité

- **Multi-Tenancy Mimir :** Header `X-Scope-OrgID` configurable par instance.
- **Basic Auth :** Configurable par instance Alertmanager.
- **TLS :** Option `tls_skip_verify` par instance (pour les environnements internes).
- **Validation stricte :** Toute configuration générée est validée avec les packages officiels Prometheus avant soumission ou export.

---

## 7. Hors périmètre (décisions explicites)

| Sujet | Décision |
| --- | --- |
| Base de données / persistance locale | Hors scope — stateless uniquement |
| Alertmanager API v1 | Hors scope — v2 uniquement |
| Forge Git autre que GitHub/GitLab | Hors scope pour l'instant (Gitea envisageable plus tard) |
| OIDC | Prévu, non implémenté en V1 |
| Déploiement multi-utilisateurs avec gestion de droits fins | Hors scope — deux niveaux uniquement (lecture / admin) |
