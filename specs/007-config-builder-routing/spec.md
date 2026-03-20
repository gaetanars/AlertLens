# Spec: Config Builder — Routing Tree Editor

**Status**: Approved
**Feature ID**: 007
**Depends on**: 006 (security-foundation)
**GitHub issues**: #50

## Context

Alertmanager's routing tree is the most critical part of its configuration: it
determines which receiver handles which alert. Editing it today requires writing
YAML directly, which is error-prone and opaque.

AlertLens exposes the config builder API (`/api/builder/route`, `/api/builder/export`)
and a visual routing tree (`/routing`) from feature 005. Feature 007 connects
these two pieces: a form-driven editor that lets `config-editor` users build and
modify the routing tree without ever touching YAML.

The backend is **fully implemented** — all builder endpoints are in place.
The frontend has a skeleton (`web/src/routes/config/routing/+page.svelte` and
`RouteNodeEditor.svelte`) but several acceptance criteria remain unmet (see below).

**Target users**: SRE / on-call engineers with the `config-editor` role who
configure alert routing across one or more Alertmanager instances.

## User stories

- As a config-editor, I want to add a child route to any node so that I can
  route a new alert group without editing raw YAML.
- As a config-editor, I want to edit matchers, receiver, and timing fields on
  any route node so that I can fine-tune routing rules through a form.
- As a config-editor, I want to delete a route node (with a confirmation step)
  so that I don't accidentally remove a live rule.
- As a config-editor, I want to reorder sibling routes using up/down controls
  so that I can control Alertmanager's first-match priority.
- As a config-editor, I want to pick a receiver from a dropdown so that I can't
  misspell a receiver name.
- As a config-editor, I want to see the assembled YAML update live as I edit
  the form so that I always know what will be pushed.
- As a viewer or silencer, I want to open the routing config page and see the
  current routing tree in read-only mode so that I understand the routing
  without being able to make changes.

## Acceptance criteria

- [ ] AC-1: A `config-editor` user can add a child route to any node in the
  visual form editor. The new node appears inline, ready to fill in.
- [ ] AC-2: A `config-editor` user can edit all fields on any node: matchers
  (label, operator, value), receiver, `group_by`, `group_wait`,
  `group_interval`, `repeat_interval`, `mute_time_intervals`,
  `active_time_intervals`, and `continue`.
- [ ] AC-3: A `config-editor` user can delete a non-root child route. A
  confirmation dialog is shown before the node is removed.
- [ ] AC-4: A `config-editor` user can reorder sibling routes using up/down
  arrow controls. The first route in the list has the highest priority in
  Alertmanager.
- [ ] AC-5: The receiver field on each route node is a `<select>` populated
  from `GET /api/builder/receivers?instance=<name>`. The dropdown updates when
  the selected instance changes.
- [ ] AC-6: The YAML preview panel updates automatically every time the form
  state changes (no manual "sync" action needed from the user).
- [ ] AC-7: A user with a role lower than `config-editor` (viewer, silencer)
  sees the page in read-only mode: the routing tree is displayed but all edit
  controls (add, delete, reorder, save) are hidden or disabled.
- [ ] AC-8: The form editor uses `GET /api/builder/route` to load the current
  routing tree and `PUT /api/builder/route` (via `POST /api/builder/export` +
  `POST /api/config/save`) to persist changes, not raw YAML string manipulation.
- [ ] AC-9: There is a typed `builder.ts` API client in `web/src/lib/api/`
  covering `getRoute`, `setRoute`, `exportConfig`, and `listReceivers` (the
  last call is reused from feature 008 but the type must be defined here).
- [ ] AC-10: The "Preview diff" → "Confirm and save" two-step flow continues to
  work as it does today.

## Out of scope

- Drag-and-drop reordering (up/down arrows are sufficient for MVP; D&D can be
  added in a later iteration).
- Editing receiver configuration (that is feature 008 — config-builder-receivers).
- Editing time interval definitions (that is feature 008 as well).
- Config save history / diff view between versions (that is feature 009).
- Any Alertmanager instance management (add/remove instances from the pool).
- GitOps push options UI changes — the existing save mode selector (disk /
  github / gitlab) is kept as-is.

## Decisions

- **Q1 — Receiver dropdown**: call `GET /api/builder/receivers` now (backend ready). The receiver field is a `<select>` populated from the live list. Type stub `{ name: string }[]` to be extended by feature 008.
- **Q2 — Live YAML preview**: local rendering via `js-yaml` from form state. No extra network round-trip on each keystroke.
- **Q3 — Read-only mode**: reuse `RoutingTree.svelte` for viewers and silencers. The form editor is only mounted for `config-editor` and above.
