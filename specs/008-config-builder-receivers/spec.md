# Spec: Config Builder — Receivers & Time Intervals

**Status**: Approved
**Feature ID**: 008
**Depends on**: 007 (config-builder-routing)
**GitHub issues**: #51

## Context

The backend CRUD endpoints for receivers (`/api/builder/receivers/*`) and time intervals
(`/api/builder/time-intervals/*`) are fully implemented and deployed. The `configbuilder`
package supports list, get, upsert, delete, and standalone validation for both resources.

The frontend `/config/receivers` and `/config/time-intervals` pages exist, but both
currently manipulate the full Alertmanager config YAML directly (parse → mutate → rebuild),
bypassing the builder API entirely. This means:

- No per-field inline validation (only full-config save errors surface)
- The receivers page is missing OpsGenie support and a raw YAML fallback for
  unknown integrations
- The time-intervals page is missing months, days-of-month, and years selectors
- Deleting a receiver has no guard against routes that still reference it
- The frontend API client (`builder.ts`) stubs receiver types as `{ name: string }`
  with no CRUD methods for time intervals

This feature migrates both pages to use the builder API, completes the field coverage,
adds inline validation, and adds a referential guard on receiver deletion.

## User stories

- As a config-editor, I want to add a Slack webhook receiver through a form so that I
  don't have to hand-edit YAML.
- As a config-editor, I want to add a PagerDuty, email, OpsGenie, or generic webhook
  receiver through the same interface so that common integrations have first-class support.
- As a config-editor, I want to edit an unknown/custom receiver type as raw YAML so that
  I don't lose configuration that AlertLens doesn't model natively.
- As a config-editor, I want to see validation errors inline as I fill in a receiver form
  so that I catch mistakes before saving.
- As a config-editor, I want a warning when deleting a receiver that is still referenced
  by a route so that I don't accidentally break routing.
- As a config-editor, I want to create a time interval that restricts a receiver to
  business hours so that on-call noise is reduced outside working hours.
- As a config-editor, I want to specify months, days-of-month, and years in a time
  interval so that I can model complex maintenance windows.
- As a config-editor, I want inline validation on the time interval form so that I catch
  malformed ranges immediately.
- As a viewer, I want to see the receiver list and time interval list in read-only mode
  so that I can understand the current configuration without edit access.

## Acceptance criteria

- [ ] AC-1: The receivers page fetches data via `GET /api/builder/receivers` and all
  mutations go through `PUT /api/builder/receivers/{name}` and
  `DELETE /api/builder/receivers/{name}`.
- [ ] AC-2: The receiver editor supports all five integration types: Slack webhook,
  PagerDuty, email, OpsGenie, and generic webhook. Each type renders only the fields
  relevant to that integration (matching `SlackConfigDef`, `PagerdutyConfigDef`,
  `EmailConfigDef`, `OpsgenieConfigDef`, `WebhookConfigDef`).
- [ ] AC-3: A receiver whose type is not one of the five known types falls back to a
  raw YAML textarea for editing, preserving the original content.
- [ ] AC-4: Changing any receiver field triggers `POST /api/builder/receivers/validate`
  within 500 ms (debounced); validation errors are displayed inline next to the
  relevant field or as a summary below the form.
- [ ] AC-5: Attempting to delete a receiver whose name appears in any route's `receiver`
  field (at any nesting depth) shows a confirmation dialog naming the affected routes
  before proceeding.
- [ ] AC-6: Can successfully save a new Slack webhook receiver end-to-end (add → fill
  form → validate → diff preview → save to Alertmanager).
- [ ] AC-7: The time-intervals page fetches data via `GET /api/builder/time-intervals`
  and all mutations go through `PUT /api/builder/time-intervals/{name}` and
  `DELETE /api/builder/time-intervals/{name}`.
- [ ] AC-8: The time interval editor includes all six field groups: time ranges (HH:MM),
  weekdays, days-of-month, months, years, and timezone (IANA). Each group is
  individually optional.
- [ ] AC-9: Changing any time interval field triggers
  `POST /api/builder/time-intervals/validate` within 500 ms (debounced); validation
  errors are displayed inline.
- [ ] AC-10: Can successfully save a new time interval restricting to business hours
  (Mon–Fri, 09:00–17:00) end-to-end.
- [ ] AC-11: Both pages enforce the `config-editor` role: viewers see the list in
  read-only mode (no edit/add/delete controls), non-authenticated users are redirected
  to `/login`.
- [ ] AC-12: The frontend API client (`builder.ts`) exports typed CRUD functions for
  both resources: `listReceivers`, `getReceiver`, `upsertReceiver`, `deleteReceiver`,
  `validateReceiver`, `listTimeIntervals`, `getTimeInterval`, `upsertTimeInterval`,
  `deleteTimeInterval`, `validateTimeInterval`.
- [ ] AC-13: All TypeScript types for `ReceiverDef`, `SlackConfigDef`, `EmailConfigDef`,
  `PagerdutyConfigDef`, `OpsgenieConfigDef`, `WebhookConfigDef`, `TimeIntervalEntry`,
  `TimeIntervalDef`, and `TimeRangeDef` are defined in `types.ts` and match the Go
  backend structs.

## Out of scope

- Saving directly to disk or GitOps (Git push). Save behaviour follows the existing
  pattern from the routing editor: the builder endpoints update the in-memory builder
  state; the save step (disk / GitHub / GitLab) is handled by the config-save flow
  which is part of feature 009 (config-builder-save-history).
- Testing or sending a notification through a receiver (feature 023,
  receiver-test).
- Support for receiver types beyond the five listed in AC-2 other than the raw YAML
  fallback (e.g., VictorOps, Telegram, MSTeams — future additions).
- Bulk import or export of receivers or time intervals.
- Reordering receivers or time intervals (order is cosmetic for Alertmanager).

## Decisions

- **Q1 — Save flow**: Wire up the same diff-and-save mechanism as the routing editor.
  If the save step requires work from feature 009 that is not yet done, advance 009
  in parallel rather than blocking this feature.

- **Q2 — Delete guard**: Add a backend endpoint that returns the route paths referencing
  a given receiver name. Frontend calls this endpoint on delete intent and shows the
  result; no tree-walking logic in the frontend.

- **Q3 — Unknown receiver fallback**: The raw YAML fallback applies at the whole
  receiver level. If any integration config in the receiver is of an unknown type, the
  entire receiver is represented as a single editable YAML textarea.

- **Q4 — Days-of-month / months UX**: Plain text inputs using Alertmanager's native
  range syntax (e.g. `1:15`, `-1`, `january:march`), consistent with the timezone field
  pattern already present on the time-intervals page.
