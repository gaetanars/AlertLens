# Visual Ack

Visual Ack is AlertLens's mechanism for acknowledging ownership of an alert — "I'm on it" — without silencing it.

---

## The Problem

Alertmanager silences mute alerts entirely: they disappear from the UI. This makes it hard to track which alerts are being actively investigated.

Visual Ack solves this by keeping the alert **visible** while marking it as acknowledged, giving your team situational awareness.

---

## How It Works

When you ack an alert, AlertLens creates a standard Alertmanager silence with three reserved labels:

| Label | Value |
|---|---|
| `alertlens_ack_type` | `"visual"` |
| `alertlens_ack_by` | The identifier you entered (name, username, etc.) |
| `alertlens_ack_comment` | Your optional comment |

The silence matches the alert, but AlertLens **does not hide it**. Instead, the alert is displayed with a distinct visual indicator (badge, color, or icon) showing who acknowledged it.

### Stateless by design

There is no database. All ack information is stored inside the silence in Alertmanager. If AlertLens is restarted or replaced, the ack state is fully preserved.

The list of active acks is reconstructed at runtime by reading silences filtered on `alertlens_ack_type="visual"`.

---

## Creating a Visual Ack

!!! note "Admin mode required"
    Creating acks requires admin mode.

1. Click on an active alert.
2. Click **Ack** (not **Silence**).
3. Enter your name or identifier.
4. Add an optional comment (e.g., "Investigating disk pressure on node-03").
5. Pick a duration — how long you expect to be working on it.
6. Confirm.

The alert immediately shows the ack badge in the alert list.

---

## Bulk Ack

Select multiple alerts and choose **Ack selected** from the bulk action toolbar. All selected alerts will be acked with the same identifier and comment.

---

## Removing an Ack

Acks are silences and expire automatically at the duration you set. To remove an ack early:

1. Go to the **Silences** page.
2. Find the ack (it shows the `alertlens_ack_type: visual` label).
3. Click **Expire**.

Or, remove it directly from the alert detail panel if the alert is still active.

---

## Visual Indicators

Acked alerts in the alert list display:

- A badge showing the acknowledger's name
- An optional comment tooltip
- A distinct color or icon to differentiate from non-acked alerts

This makes it immediately clear in a high-alert situation who is handling what.
