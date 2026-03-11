<script lang="ts">
	/**
	 * InstanceTopology — D3 hub-and-spoke diagram for AlertLens multi-instance view.
	 *
	 * Renders AlertLens (hub) in the centre with one spoke per Alertmanager instance.
	 * Each spoke endpoint is a circle coloured by health status and annotated with
	 * the live alert count and per-severity breakdown.
	 *
	 * Props:
	 *   topology  — HubTopology object from GET /api/hub/topology
	 *   onSpokeClick — optional callback when a spoke node is clicked
	 */

	import type { HubTopology, SpokeStats } from '$lib/api/types';
	import { onMount } from 'svelte';
	import * as d3 from 'd3';

	let {
		topology,
		onSpokeClick
	}: {
		topology: HubTopology | null;
		onSpokeClick?: (spoke: SpokeStats) => void;
	} = $props();

	let container: HTMLDivElement;

	// ─── Layout constants ─────────────────────────────────────────────────────

	const HUB_R = 44;          // hub circle radius
	const SPOKE_R = 36;        // spoke circle radius
	const ORBIT_R = 180;       // distance from hub centre to spoke centre
	const BADGE_R = 14;        // alert-count badge radius
	const SVG_W = 520;
	const SVG_H = 440;
	const CX = SVG_W / 2;
	const CY = SVG_H / 2;

	// ─── Colour helpers ───────────────────────────────────────────────────────

	function spokeColor(spoke: SpokeStats): string {
		if (!spoke.healthy) return 'hsl(var(--destructive))';
		if ((spoke.severity_counts['critical'] ?? 0) > 0) return '#ef4444';   // red-500
		if ((spoke.severity_counts['warning'] ?? 0) > 0)  return '#f59e0b';   // amber-500
		return '#22c55e'; // green-500
	}

	function badgeColor(spoke: SpokeStats): string {
		if (!spoke.healthy) return '#6b7280'; // gray
		if ((spoke.severity_counts['critical'] ?? 0) > 0) return '#dc2626';
		if ((spoke.severity_counts['warning'] ?? 0) > 0)  return '#d97706';
		return '#16a34a';
	}

	// ─── Render ───────────────────────────────────────────────────────────────

	$effect(() => {
		if (topology && container) render(topology);
	});

	function render(topo: HubTopology) {
		container.innerHTML = '';

		const spokes = topo.spokes;
		const n = spokes.length;

		const svg = d3.select(container)
			.append('svg')
			.attr('width', '100%')
			.attr('height', SVG_H)
			.attr('viewBox', `0 0 ${SVG_W} ${SVG_H}`)
			.attr('aria-label', 'Hub-and-spoke topology diagram');

		// ── Defs: subtle radial gradient for hub circle ──────────────────────
		const defs = svg.append('defs');
		const grad = defs.append('radialGradient')
			.attr('id', 'hub-grad')
			.attr('cx', '40%').attr('cy', '35%');
		grad.append('stop').attr('offset', '0%').attr('stop-color', 'hsl(var(--primary))').attr('stop-opacity', 0.9);
		grad.append('stop').attr('offset', '100%').attr('stop-color', 'hsl(var(--primary))').attr('stop-opacity', 0.6);

		// ── Spoke positions (evenly spaced around a circle) ──────────────────
		const angleStep = n > 0 ? (2 * Math.PI) / n : 0;
		const spokePositions = spokes.map((_, i) => ({
			x: CX + ORBIT_R * Math.cos(i * angleStep - Math.PI / 2),
			y: CY + ORBIT_R * Math.sin(i * angleStep - Math.PI / 2)
		}));

		// ── Connector lines (hub → spoke) ─────────────────────────────────────
		svg.selectAll('line.spoke-line')
			.data(spokes)
			.enter()
			.append('line')
			.attr('class', 'spoke-line')
			.attr('x1', CX)
			.attr('y1', CY)
			.attr('x2', (_d, i) => spokePositions[i].x)
			.attr('y2', (_d, i) => spokePositions[i].y)
			.attr('stroke', 'hsl(var(--border))')
			.attr('stroke-width', 1.5)
			.attr('stroke-dasharray', '4 3');

		// ── Hub circle ────────────────────────────────────────────────────────
		const hubG = svg.append('g').attr('class', 'hub');

		hubG.append('circle')
			.attr('cx', CX).attr('cy', CY)
			.attr('r', HUB_R)
			.attr('fill', 'url(#hub-grad)')
			.attr('stroke', 'hsl(var(--primary))')
			.attr('stroke-width', 2);

		// Hub label
		hubG.append('text')
			.attr('x', CX).attr('y', CY - 6)
			.attr('text-anchor', 'middle')
			.attr('font-size', 11)
			.attr('font-weight', 700)
			.attr('fill', 'hsl(var(--primary-foreground))')
			.text('AlertLens');

		// Hub sub-label: total alerts
		hubG.append('text')
			.attr('x', CX).attr('y', CY + 9)
			.attr('text-anchor', 'middle')
			.attr('font-size', 9)
			.attr('fill', 'hsl(var(--primary-foreground))')
			.attr('opacity', 0.85)
			.text(`${topo.hub.total_alerts} alert${topo.hub.total_alerts !== 1 ? 's' : ''}`);

		// Hub sub-label: instances healthy
		hubG.append('text')
			.attr('x', CX).attr('y', CY + 21)
			.attr('text-anchor', 'middle')
			.attr('font-size', 8)
			.attr('fill', 'hsl(var(--primary-foreground))')
			.attr('opacity', 0.7)
			.text(`${topo.hub.healthy_instances}/${topo.hub.total_instances} healthy`);

		// ── Spoke nodes ───────────────────────────────────────────────────────
		const spokeG = svg.selectAll('g.spoke')
			.data(spokes)
			.enter()
			.append('g')
			.attr('class', 'spoke')
			.attr('transform', (_d, i) => `translate(${spokePositions[i].x},${spokePositions[i].y})`)
			.style('cursor', 'pointer')
			.on('click', (_event, d) => onSpokeClick?.(d));

		// Spoke circle
		spokeG.append('circle')
			.attr('r', SPOKE_R)
			.attr('fill', 'hsl(var(--card))')
			.attr('stroke', (d) => spokeColor(d))
			.attr('stroke-width', 2.5);

		// Spoke name
		spokeG.append('text')
			.attr('text-anchor', 'middle')
			.attr('y', -6)
			.attr('font-size', 10)
			.attr('font-weight', 600)
			.attr('fill', 'hsl(var(--foreground))')
			.text((d) => truncate(d.name, 12));

		// Spoke version (small)
		spokeG.filter((d) => !!d.version)
			.append('text')
			.attr('text-anchor', 'middle')
			.attr('y', 6)
			.attr('font-size', 8)
			.attr('fill', 'hsl(var(--muted-foreground))')
			.text((d) => d.version ? `v${d.version}` : '');

		// Error badge for unhealthy spokes
		spokeG.filter((d) => !d.healthy)
			.append('text')
			.attr('text-anchor', 'middle')
			.attr('y', 18)
			.attr('font-size', 8)
			.attr('fill', 'hsl(var(--destructive))')
			.text('offline');

		// ── Alert-count badge (top-right of spoke circle) ─────────────────────
		const badgeG = spokeG.filter((d) => d.healthy && d.alert_count > 0)
			.append('g')
			.attr('transform', `translate(${SPOKE_R * 0.7},${-SPOKE_R * 0.7})`);

		badgeG.append('circle')
			.attr('r', BADGE_R)
			.attr('fill', (d) => badgeColor(d))
			.attr('stroke', 'hsl(var(--card))')
			.attr('stroke-width', 1.5);

		badgeG.append('text')
			.attr('text-anchor', 'middle')
			.attr('dy', '0.35em')
			.attr('font-size', 9)
			.attr('font-weight', 700)
			.attr('fill', '#fff')
			.text((d) => d.alert_count > 99 ? '99+' : String(d.alert_count));

		// ── Severity mini-bars (bottom of spoke card) ─────────────────────────
		// Show up to 3 severity dots: critical (red), warning (amber), info (blue)
		const severities = [
			{ key: 'critical', color: '#ef4444' },
			{ key: 'warning',  color: '#f59e0b' },
			{ key: 'info',     color: '#3b82f6' }
		];

		spokeG.filter((d) => d.healthy).each(function(d) {
			const g = d3.select(this);
			const hasSev = severities.some(s => (d.severity_counts[s.key] ?? 0) > 0);
			if (!hasSev) return;

			let xOffset = -(severities.length * 10) / 2 + 5;
			for (const sev of severities) {
				const cnt = d.severity_counts[sev.key] ?? 0;
				if (cnt === 0) { xOffset += 10; continue; }
				g.append('circle')
					.attr('cx', xOffset)
					.attr('cy', SPOKE_R - 10)
					.attr('r', 4)
					.attr('fill', sev.color)
					.attr('opacity', 0.85);
				g.append('text')
					.attr('x', xOffset)
					.attr('y', SPOKE_R - 10)
					.attr('dy', '0.35em')
					.attr('text-anchor', 'middle')
					.attr('font-size', 6)
					.attr('fill', '#fff')
					.attr('font-weight', 700)
					.text(cnt > 9 ? '9+' : cnt);
				xOffset += 11;
			}
		});
	}

	function truncate(s: string, maxLen: number): string {
		return s.length > maxLen ? s.slice(0, maxLen - 1) + '…' : s;
	}

	onMount(() => {
		if (topology) render(topology);
	});
</script>

<div bind:this={container} class="w-full overflow-hidden rounded-lg border bg-card"></div>
