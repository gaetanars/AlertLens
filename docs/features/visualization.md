# Alert Visualization

AlertLens provides a rich interface to visualize alerts from one or multiple Alertmanager instances.

---

![AlertLens – Alerts overview](../assets/screenshots/alerts-overview.png)

## Display Modes

### Kanban View

Alerts are organized into columns by severity (`critical`, `warning`, `info`, and any other severity value present in your alerts). This gives an immediate visual sense of the current operational state.

### Dense List View

A compact table format optimized for high-alert-volume environments. Supports sorting by any column.

Switch between modes with the toggle in the top-right of the alerts page.

---

## Filtering

AlertLens uses **Alertmanager's native matcher syntax** for filtering. This means the same expressions you use in `alertmanager.yml` routes work here too.

### Syntax

| Matcher | Meaning |
|---|---|
| `severity="critical"` | Exact match |
| `env=~"prod.*"` | Regex match |
| `team!="platform"` | Not equal |
| `team!~"platform.*"` | Not regex match |

### Combining matchers

Multiple matchers are AND-ed together:

```
severity="critical" env=~"prod.*" team!="platform"
```

### Examples

```
# All critical alerts in production
severity="critical" environment="production"

# All firing alerts not acknowledged
alertlens_ack_type!="visual"

# Alerts from a specific cluster
cluster=~"eu-west-.*"
```

---

## Grouping

Group alerts by any label to reduce visual noise and focus on patterns. Grouping options typically include:

- `team`
- `environment`
- `cluster`
- `alertname`

The grouping selector is available in the filter toolbar.

---

## Multi-Instance Aggregation

When multiple Alertmanager instances are configured, AlertLens aggregates all alerts into a single view by default.

- Each alert displays a **source badge** indicating its origin instance.
- An **instance filter** in the toolbar lets you narrow the view to a specific cluster.
- Silences and acks are scoped to their source instance (a silence created in AlertLens is applied to the instance the alert came from).

---

## Routing Tree Visualizer

The routing tree page renders your `alertmanager.yml` route hierarchy as an interactive graph.

- Click any node to see which active alerts match that route.
- Each node shows its matchers, target receiver, and grouping parameters.
- Nodes with time interval restrictions display inline badges: **orange** for `mute_time_intervals` (notifications suppressed during the interval) and **green** for `active_time_intervals` (notifications restricted to the interval).
- Helps understand why an alert is (or isn't) reaching a given receiver.

!!! info "Read-only"
    The routing tree visualizer is available in read-only mode. Editing routes requires admin mode and is done in the [Configuration Builder](config-builder.md).
