<script lang="ts">
	import type { RouteNode } from '$lib/api/types';
	import { onMount, onDestroy } from 'svelte';
	import * as d3 from 'd3';

	let { route, onNodeClick }: {
		route: RouteNode | null;
		onNodeClick?: (node: RouteNode) => void;
	} = $props();

	let container: HTMLDivElement;
	let svg: d3.Selection<SVGSVGElement, unknown, null, undefined>;

	const NODE_W = 220;
	const NODE_H = 96;  // slightly taller to accommodate alert count row
	const GAP_X = 60;
	const GAP_Y = 20;

	interface TreeNode extends d3.HierarchyNode<RouteNode> {}

	$effect(() => {
		if (route && container) render(route);
	});

	function render(root: RouteNode) {
		// SEC-XSS: resetting innerHTML to '' clears child nodes before D3 rebuilds
		// the SVG. No user-controlled content is inserted here — all data from the
		// API is injected via D3's .text() which sets textContent (not innerHTML),
		// so it is automatically XSS-safe.
		container.innerHTML = '';

		const hierarchy = d3.hierarchy<RouteNode>(root, (d) => d.routes ?? []);
		const treeLayout = d3.tree<RouteNode>()
			.nodeSize([NODE_H + GAP_Y, NODE_W + GAP_X]);

		const treeData = treeLayout(hierarchy);

		// Bounds
		let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
		treeData.each((d) => {
			minX = Math.min(minX, d.x - NODE_H / 2);
			maxX = Math.max(maxX, d.x + NODE_H / 2);
			minY = Math.min(minY, d.y);
			maxY = Math.max(maxY, d.y + NODE_W);
		});

		const W = maxY - minY + NODE_W + 80;
		const H = maxX - minX + NODE_H + 80;

		svg = d3.select(container)
			.append('svg')
			.attr('width', '100%')
			.attr('height', H)
			.attr('viewBox', `${minY - 40} ${minX - 40} ${W} ${H}`);

		const g = svg.append('g');

		// Zoom + pan
		const zoom = d3.zoom<SVGSVGElement, unknown>()
			.scaleExtent([0.3, 2])
			.on('zoom', (event) => g.attr('transform', event.transform));
		svg.call(zoom);

		// Links
		g.selectAll('path.link')
			.data(treeData.links())
			.enter()
			.append('path')
			.attr('class', 'link')
			.attr('fill', 'none')
			.attr('stroke', 'hsl(var(--border))')
			.attr('stroke-width', 1.5)
			.attr('d', d3.linkHorizontal<d3.HierarchyLink<RouteNode>, d3.HierarchyPointNode<RouteNode>>()
				.x((d) => d.y)
				.y((d) => d.x));

		// Nodes
		const node = g.selectAll('g.node')
			.data(treeData.descendants())
			.enter()
			.append('g')
			.attr('class', 'node')
			.attr('transform', (d) => `translate(${d.y},${d.x - NODE_H / 2})`)
			.style('cursor', 'pointer')
			.on('click', (_event, d) => onNodeClick?.(d.data));

		// Node rect
		node.append('rect')
			.attr('width', NODE_W)
			.attr('height', NODE_H)
			.attr('rx', 6)
			.attr('ry', 6)
			.attr('fill', 'hsl(var(--card))')
			.attr('stroke', 'hsl(var(--border))')
			.attr('stroke-width', 1.5);

		// Receiver name
		node.append('text')
			.attr('x', 10)
			.attr('y', 20)
			.attr('font-size', 12)
			.attr('font-weight', 600)
			.attr('fill', 'hsl(var(--foreground))')
			.text((d) => d.data.receiver ?? '(default)');

		// Matchers
		node.append('text')
			.attr('x', 10)
			.attr('y', 38)
			.attr('font-size', 10)
			.attr('fill', 'hsl(var(--muted-foreground))')
			.text((d) => {
				const m = (d.data.matchers ?? [])
					.map(m => `${m.name}="${m.value}"`)
					.join(', ');
				return m.length > 30 ? m.slice(0, 30) + '…' : m || '(catch-all)';
			});

		// Group by
		node.append('text')
			.attr('x', 10)
			.attr('y', 55)
			.attr('font-size', 9)
			.attr('fill', 'hsl(var(--muted-foreground))')
			.text((d) => {
				const gb = d.data.group_by?.join(', ');
				return gb ? `group_by: ${gb}` : '';
			});

		// Continue badge
		node.filter((d) => d.data.continue)
			.append('rect')
			.attr('x', NODE_W - 55)
			.attr('y', 4)
			.attr('width', 48)
			.attr('height', 16)
			.attr('rx', 4)
			.attr('fill', 'hsl(var(--primary))')
			.attr('opacity', 0.15);
		node.filter((d) => d.data.continue)
			.append('text')
			.attr('x', NODE_W - 31)
			.attr('y', 15)
			.attr('text-anchor', 'middle')
			.attr('font-size', 9)
			.attr('fill', 'hsl(var(--primary))')
			.text('continue');

		// ── Alert count badge (top-left corner, when annotated) ───────────────
		// Nodes with alert_count > 0 get a coloured pill showing the count.
		node.filter((d) => (d.data.alert_count ?? 0) > 0)
			.append('rect')
			.attr('x', NODE_W - 46)
			.attr('y', NODE_H - 18)
			.attr('width', 38)
			.attr('height', 14)
			.attr('rx', 7)
			.attr('fill', (d) => {
				const sc = d.data.severity_counts ?? {};
				if ((sc['critical'] ?? 0) > 0) return '#ef4444';
				if ((sc['warning'] ?? 0) > 0)  return '#f59e0b';
				return '#22c55e';
			})
			.attr('opacity', 0.9);

		node.filter((d) => (d.data.alert_count ?? 0) > 0)
			.append('text')
			.attr('x', NODE_W - 27)
			.attr('y', NODE_H - 8)
			.attr('text-anchor', 'middle')
			.attr('font-size', 8)
			.attr('font-weight', 700)
			.attr('fill', '#fff')
			.text((d) => {
				const cnt = d.data.alert_count!;
				return cnt > 99 ? '99+' : `${cnt} alert${cnt !== 1 ? 's' : ''}`;
			});

		// Nodes with alert_count = 0 (annotated, but no match) get a subtle grey pill
		node.filter((d) => d.data.alert_count === 0 && d.data.severity_counts !== undefined)
			.append('text')
			.attr('x', NODE_W - 27)
			.attr('y', NODE_H - 8)
			.attr('text-anchor', 'middle')
			.attr('font-size', 8)
			.attr('fill', 'hsl(var(--muted-foreground))')
			.attr('opacity', 0.6)
			.text('0 alerts');

		// ── Time interval badges ──────────────────────────────────────────────
		// Separator line — only on nodes that have at least one interval
		node.filter((d) =>
			(d.data.mute_time_intervals?.length ?? 0) +
			(d.data.active_time_intervals?.length ?? 0) > 0
		)
			.append('line')
			.attr('x1', 8).attr('x2', NODE_W - 8)
			.attr('y1', 63).attr('y2', 63)
			.attr('stroke', 'hsl(var(--border))')
			.attr('stroke-width', 0.5);

		// Mute time intervals (orange) — row 1 of badge area
		node.filter((d) => (d.data.mute_time_intervals?.length ?? 0) > 0)
			.append('text')
			.attr('x', 10)
			.attr('y', 75)
			.attr('font-size', 8)
			.attr('fill', '#f97316')
			.text((d) => {
				const names = d.data.mute_time_intervals!.join(', ');
				const label = 'mute: ' + names;
				return label.length > 34 ? label.slice(0, 34) + '…' : label;
			});

		// Active time intervals (green) — row 2 if mute present, row 1 otherwise
		node.filter((d) => (d.data.active_time_intervals?.length ?? 0) > 0)
			.append('text')
			.attr('x', 10)
			.attr('y', (d) => (d.data.mute_time_intervals?.length ?? 0) > 0 ? 86 : 75)
			.attr('font-size', 8)
			.attr('fill', '#16a34a')
			.text((d) => {
				const names = d.data.active_time_intervals!.join(', ');
				const label = 'active: ' + names;
				return label.length > 34 ? label.slice(0, 34) + '…' : label;
			});
	}
</script>

<div bind:this={container} class="w-full overflow-hidden rounded-lg border bg-card min-h-[300px]"></div>
