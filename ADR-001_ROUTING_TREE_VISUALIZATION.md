# ADR-001: Routing Tree Visualization Library Selection

**Status:** Approved  
**Date:** 2026-03-09  
**Decision Maker:** Architect  
**Implementation Feature:** #28 (Routing Tree Visualizer)

---

## Context

Feature #3 requires visualizing the Alertmanager routing configuration as an interactive tree diagram. The routing tree is a hierarchical structure where:

- Root node is the main route
- Each route can have child routes (matchers-based branching)
- Each node has a receiver (notification destination)
- Matchers define which alerts match each node

**Requirements:**
- Display tree with 10-100 nodes (typical Alertmanager configs)
- Interactive: click node → fetch matching alerts
- Zoom & pan for large trees
- Responsive design (desktop + tablet)
- Good performance (< 1s render for 100 nodes)

**Constraints:**
- Single-page app (SvelteKit frontend)
- TypeScript/JavaScript (no server-side rendering needed)
- Lightweight dependency (prefer <100KB gzipped)
- Active community & good documentation

---

## Options Considered

### Option A: D3.js (Recommended)

**Description:**  
D3 is a low-level, data-driven visualization library. Provides full control over layout algorithms and rendering.

**Pros:**
- ✅ Most flexible (custom layout control)
- ✅ Lightweight (~100KB gzipped)
- ✅ Largest community, most examples
- ✅ Excellent for hierarchical data (tree layout built-in)
- ✅ Proven in production for complex visualizations
- ✅ Svelte integration examples available

**Cons:**
- ❌ Steep learning curve (2-3 days ramp-up)
- ❌ Manual event handling (zoom, drag, etc.)
- ❌ More boilerplate code than alternatives

**Estimated Effort:**
- Learning: 2-3 days
- Implementation: 1.5 days
- Total: 4-4.5 days

**Cost:** Free (open-source, MIT license)

---

### Option B: Cytoscape.js

**Description:**  
Graph visualization library focused on network/biological networks. Includes built-in layout engines.

**Pros:**
- ✅ Built-in layout algorithms (breadthFirstLayout, concentric, etc.)
- ✅ Shorter learning curve (more high-level)
- ✅ Good for network/tree visualization
- ✅ Handles large graphs well

**Cons:**
- ❌ Heavier (~200KB gzipped)
- ❌ Smaller ecosystem for Svelte
- ❌ Less flexible for custom styling
- ❌ Over-engineered for tree-only use case

**Estimated Effort:**
- Learning: 1-2 days
- Implementation: 1.5 days
- Total: 2.5-3.5 days

**Cost:** Free (open-source, MIT license)

---

### Option C: ELK (Eclipse Layout Kernel)

**Description:**  
Enterprise-grade layout engine for complex graph visualization. Supports multiple layout algorithms.

**Pros:**
- ✅ Professional-grade layouts
- ✅ Many layout options
- ✅ Handles complex routing scenarios

**Cons:**
- ❌ Very heavy (~500KB+ with dependencies)
- ❌ Overkill for Alertmanager routing
- ❌ Longer learning curve
- ❌ Slower initialization

**Estimated Effort:**
- Learning: 3-4 days
- Implementation: 2 days
- Total: 5-6 days

**Cost:** Free (open-source, EPL license)

---

### Option D: Custom Canvas/SVG

**Description:**  
Implement tree layout from scratch using Canvas or SVG rendering.

**Cons:**
- ❌ High implementation effort (5+ days)
- ❌ No reusable ecosystem
- ❌ Performance management (1000+ nodes problematic)
- ❌ Accessibility challenges

**Not Recommended** (only if strict zero-dependency requirement)

---

## Decision

**✅ APPROVED: D3.js (Option A)**

**Rationale:**

1. **Best fit for requirements:**
   - Tree layout is D3's bread-and-butter use case
   - Full control over aesthetics and interaction
   - Proven for hierarchical data visualization

2. **Community & learning:**
   - Largest visualization community
   - Abundance of tree examples available
   - Multiple Svelte + D3 integration patterns documented
   - Can leverage existing examples, reducing learning time

3. **Performance:**
   - Lightweight (100KB gzipped) won't bloat bundle
   - Handles 100-node trees efficiently
   - Zoom/pan implemented natively with good performance

4. **Long-term flexibility:**
   - If future features need more complex graphs (e.g., alert dependency graphs), D3 is already in the stack
   - Not pigeonholed into network graphs only

5. **Familiarity:**
   - D3 is industry-standard for visualization
   - Skills transfer to other projects
   - Easy to hire/onboard developers with D3 experience

**Timeline Impact:**
- Learning: 2-3 days (allocated in Sprint 2)
- Feature #3 total: 6 days (includes learning)

---

## Implementation Details

### Dependencies to Add

```json
{
  "dependencies": {
    "d3": "^7.8.5"
  },
  "devDependencies": {
    "@types/d3": "^7.4.0"
  }
}
```

### Integration Pattern

**File:** `web/src/routes/routing-tree/+page.svelte`

```svelte
<script>
  import * as d3 from 'd3';
  import RoutingNodeDetail from '../../components/RoutingNodeDetail.svelte';

  let svgElement;
  let selectedNodeId = null;
  let routingData = {}; // From API: GET /api/routing-tree

  onMount(async () => {
    const response = await fetch('/api/routing-tree');
    routingData = await response.json();
    
    // Build D3 hierarchy
    const hierarchy = d3.hierarchy(routingData);
    
    // Layout
    const tree = d3.tree().size([width, height]);
    const nodes = tree(hierarchy).descendants();
    const links = hierarchy.links();
    
    // Render SVG
    renderTree(nodes, links);
  });

  function renderTree(nodes, links) {
    // D3 implementation (standard pattern)
  }

  function onNodeClick(d) {
    selectedNodeId = d.data.id;
    // Fetch alerts for this node via API
  }
</script>

<div class="routing-tree-container">
  <svg bind:this={svgElement}></svg>
  {#if selectedNodeId}
    <RoutingNodeDetail nodeId={selectedNodeId} />
  {/if}
</div>
```

### Learning Resources

**Recommended:**
1. Official D3 documentation: https://d3js.org/
2. "Observable" notebooks with interactive examples
3. Tree layout examples: search "d3 tree layout" on Observable
4. Svelte + D3 guide: https://github.com/sveltedev/svelte/...

**Time allocation:**
- Days 1-2 (Sprint 2): Tutorial + small example
- Days 3-4 (Sprint 2): Full implementation

---

## Alternatives & Fallback Plan

**If D3 performance degrades (100+ nodes):**
1. Implement WebGL rendering using Three.js (much heavier but handles 1000+ nodes)
2. Or: Switch to canvas-based rendering with d3-force for physics

**If team prefers higher-level API:**
- Consider Cytoscape.js as fallback (effort: rework 1 day)

---

## Testing Strategy

### Unit Tests
- Tree building: correct hierarchy structure
- Node coordinates: within SVG bounds
- Click handler: correct node selected

### Visual Regression Tests
- Node rendering (size, position, styling)
- Link rendering (curves, colors)
- Zoom/pan transforms applied correctly

### Performance Tests
- 100-node tree: render < 1s
- Pan/zoom: 60 FPS maintained
- Memory: no leaks on repeated renders

---

## Security Considerations

- **XSS Prevention:** All node labels HTML-encoded before SVG insertion
- **DoS:** Limit tree depth to 50, node count to 500 (enforced in backend)

---

## Dependencies & Coordination

- **Depends on:** Feature #1 (#25, #26) — alert system must be in place
- **Enables:** Integration with alert filtering (Feature #2, #4)
- **Blocks:** Feature #5 (Config Builder) depends on tree structure understanding

---

## Success Criteria

- [ ] D3 dependency added to package.json
- [ ] Tree layout renders correctly for sample data
- [ ] Click node → fetches alerts (integration with Feature #1)
- [ ] Zoom & pan functional
- [ ] Responsive on mobile (tree fits viewport)
- [ ] Unit & visual regression tests pass
- [ ] Performance: 100-node tree < 1s
- [ ] Documentation: inline comments + learning resources in README

---

## Timeline

- **Duration:** 6 days total (Feature #3)
  - Days 1-2: Learning D3
  - Days 3-4: Tree rendering
  - Days 5-6: Detail panel, zoom, testing

- **Sprint:** Sprint 2 (Weeks 2-3), parallel with Feature #2

---

## Related ADRs

- ADR-002: Form Framework Selection
- ADR-003: Config Storage & Rollback Strategy
- ADR-004: Real-time Update Strategy

---

## Approval Sign-off

- **Architect:** ✅ Approved 2026-03-09
- **Developer:** ⬜ To confirm on implementation
- **Security:** ✅ No security concerns
- **Performance:** ✅ Acceptable for target use case

---

## Notes

1. **D3 community is thriving:** New versions, plugins, and integrations regularly released
2. **Svelte integration:** Svelte's reactivity pairs well with D3 if used correctly (separate concerns: D3 owns DOM, Svelte owns component state)
3. **Learning investment:** Worth it even as a side benefit — D3 skills widely applicable

---

**End of ADR-001**
