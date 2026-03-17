<script lang="ts">
  import * as d3 from "d3";
  import { onMount } from "svelte";
  import { transformConfigToHierarchy } from "../../utils/routingTree";

  export let routingConfig: any; // Raw routing config from the API
  export let matchedRoutePath: string | null = null; // Path of the matched route for highlighting

  let svgWidth = 800;
  let svgHeight = 600;
  let margin = { top: 20, right: 120, bottom: 20, left: 120 };
  let i = 0; // counter for node IDs
  let tree: d3.TreeLayout<any>;
  let svg: d3.Selection<any, any, any, any>;
  let g: d3.Selection<any, any, any, any>;
  let root: d3.HierarchyNode<any>;
  let selectedNode: d3.HierarchyNode<any> | null = null; // Track selected node for detail panel

  onMount(() => {
    // Set up the D3 tree layout
    tree = d3.tree().size([svgHeight, svgWidth]);

    svg = d3
      .select("#routing-tree-svg")
      .attr("width", svgWidth + margin.right + margin.left)
      .attr("height", svgHeight + margin.top + margin.bottom);

    g = svg
      .append("g")
      .attr("transform", "translate(" + margin.left + "," + margin.top + ")");

    // Transform the routing config and create the root node
    root = transformConfigToHierarchy(routingConfig);
    root.x0 = svgHeight / 2;
    root.y0 = 0;

    update(root);
  });

  function click(event: MouseEvent, d: d3.HierarchyNode<any>) {
    // Toggle expand/collapse
    if (d.children) {
      d._children = d.children;
      d.children = null;
    } else {
      d.children = d._children;
      d._children = null;
    }
    d.data._expanded = !d.data._expanded;
    update(d);
  }

  function selectNode(d: d3.HierarchyNode<any>) {
    selectedNode = d;
  }

  function isNodeMatched(node: d3.HierarchyNode<any>): boolean {
    if (!matchedRoutePath) return false;
    return node.data.name === matchedRoutePath;
  }

  function update(source: d3.HierarchyNode<any>) {
    // Assigns the x and y position for the nodes
    const treeData = tree(root);

    // Compute the new tree layout.
    const nodes = treeData.descendants();
    const links = treeData.links();

    // Normalize for fixed-depth.
    nodes.forEach((d) => (d.y = d.depth * 180));

    // ****************** Nodes section ******************

    // Update the nodes...
    const node = g.selectAll("g.node").data(nodes, (d: any) => d.id || (d.id = ++i));

    // Enter any new nodes at the parent's previous position.
    const nodeEnter = node
      .enter()
      .append("g")
      .attr("class", "node")
      .attr("transform", (d) => `translate(${source.y0},${source.x0})`)
      .on("click", click)
      .on("mouseenter", function(event, d) {
        selectNode(d);
      });

    // Add Circle for the nodes
    nodeEnter
      .append("circle")
      .attr("class", "node")
      .attr("r", 1e-6)
      .style("fill", (d) => (d._children ? "lightsteelblue" : "#fff"));

    // Add labels for the nodes
    nodeEnter
      .append("text")
      .attr("dy", ".35em")
      .attr("x", (d) => (d.children || d._children ? -13 : 13))
      .attr("text-anchor", (d) => (d.children || d._children ? "end" : "start"))
      .text((d) => d.data.name);

    // UPDATE
    const nodeUpdate = nodeEnter.merge(node);

    // Transition to the proper position for the node
    nodeUpdate
      .transition()
      .duration(750)
      .attr("transform", (d) => `translate(${d.y},${d.x})`);

    // Update the node attributes and style
    nodeUpdate
      .select("circle.node")
      .attr("r", 10)
      .style("fill", (d) => {
        if (isNodeMatched(d)) return "#ff6b6b"; // Red for matched
        return d._children ? "lightsteelblue" : "#fff";
      })
      .style("stroke", (d) => {
        if (isNodeMatched(d)) return "#cc0000"; // Darker red stroke for matched
        return "steelblue";
      })
      .attr("cursor", "pointer");

    // Add rings for matched nodes
    nodeUpdate.selectAll("circle.matched-ring").remove();
    nodeUpdate
      .filter((d) => isNodeMatched(d))
      .append("circle")
      .attr("class", "matched-ring")
      .attr("r", 15)
      .style("fill", "none")
      .style("stroke", "#ff6b6b")
      .style("stroke-width", "2px")
      .style("stroke-dasharray", "5,5");

    // Remove any exiting nodes
    const nodeExit = node
      .exit()
      .transition()
      .duration(750)
      .attr("transform", (d) => `translate(${source.y},${source.x})`)
      .remove();

    // On exit reduce the node circles size to 0
    nodeExit.select("circle").attr("r", 1e-6);

    // On exit reduce the opacity of the text labels
    nodeExit.select("text").style("fill-opacity", 1e-6);

    // ****************** Links section ******************

    // Update the links...
    const link = g.selectAll("path.link").data(links, (d: any) => d.id);

    // Enter any new links at the parent's previous position.
    const linkEnter = link
      .enter()
      .insert("path", "g")
      .attr("class", "link")
      .attr("d", (d) => {
        const o = { x: source.x0, y: source.y0 };
        return diagonal(o, o);
      });

    // UPDATE
    const linkUpdate = linkEnter.merge(link);

    // Transition back to the parent element position
    linkUpdate
      .transition()
      .duration(750)
      .attr("d", (d) => diagonal(d.source, d.target));

    // Remove any exiting links
    link
      .exit()
      .transition()
      .duration(750)
      .attr("d", (d) => {
        const o = { x: source.x, y: source.y };
        return diagonal(o, o);
      })
      .remove();

    // Store the old positions for transition.
    nodes.forEach((d) => {
      d.x0 = d.x;
      d.y0 = d.y;
    });

    // Function to draw the links
    function diagonal(s: any, d: any) {
      const path = `M ${s.y} ${s.x}
                    C ${(s.y + d.y) / 2} ${s.x},
                      ${(s.y + d.y) / 2} ${d.x},
                      ${d.y} ${d.x}`;

      return path;
    }
  }

  $: if (routingConfig && root) {
    // Re-render when routingConfig changes
    root = transformConfigToHierarchy(routingConfig);
    root.x0 = svgHeight / 2;
    root.y0 = 0;
    update(root);
  }

  $: if (root) {
    // Highlight matched node when matchedRoutePath changes
    update(root);
  }
</script>

<div class="routing-tree-container">
  <div class="tree-wrapper">
    <svg id="routing-tree-svg"></svg>
  </div>
  {#if selectedNode && selectedNode.data.name !== 'Root'}
    <div class="detail-panel">
      <h3>Route Details</h3>
      <div class="detail-content">
        <div class="detail-field">
          <span class="label">Path:</span>
          <span class="value">{selectedNode.data.name}</span>
        </div>
        {#if selectedNode.data.data.receiver}
          <div class="detail-field">
            <span class="label">Receiver:</span>
            <span class="value">{selectedNode.data.data.receiver}</span>
          </div>
        {/if}
        {#if selectedNode.data.data.group_by}
          <div class="detail-field">
            <span class="label">Group By:</span>
            <span class="value">{selectedNode.data.data.group_by.join(', ')}</span>
          </div>
        {/if}
        {#if selectedNode.data.data.group_wait}
          <div class="detail-field">
            <span class="label">Group Wait:</span>
            <span class="value">{selectedNode.data.data.group_wait}</span>
          </div>
        {/if}
        {#if selectedNode.data.data.group_interval}
          <div class="detail-field">
            <span class="label">Group Interval:</span>
            <span class="value">{selectedNode.data.data.group_interval}</span>
          </div>
        {/if}
        {#if selectedNode.data.data.repeat_interval}
          <div class="detail-field">
            <span class="label">Repeat Interval:</span>
            <span class="value">{selectedNode.data.data.repeat_interval}</span>
          </div>
        {/if}
        {#if selectedNode.data.data.match}
          <div class="detail-field">
            <span class="label">Match:</span>
            <span class="value">{JSON.stringify(selectedNode.data.data.match)}</span>
          </div>
        {/if}
        {#if selectedNode.data.data.match_re}
          <div class="detail-field">
            <span class="label">Match (Regex):</span>
            <span class="value">{JSON.stringify(selectedNode.data.data.match_re)}</span>
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  .routing-tree-container {
    display: flex;
    gap: 20px;
    padding: 20px;
  }

  .tree-wrapper {
    flex: 1;
    border: 1px solid #ccc;
    border-radius: 8px;
    overflow: auto;
  }

  :global(.node circle) {
    fill: #fff;
    stroke: steelblue;
    stroke-width: 3px;
    transition: fill 0.2s, stroke 0.2s;
  }

  :global(.node text) {
    font: 12px sans-serif;
    pointer-events: none;
  }

  :global(.link) {
    fill: none;
    stroke: #ccc;
    stroke-width: 2px;
  }

  .detail-panel {
    width: 300px;
    padding: 15px;
    border: 1px solid #ccc;
    border-radius: 8px;
    background-color: #f9f9f9;
    max-height: 600px;
    overflow-y: auto;
  }

  .detail-panel h3 {
    margin: 0 0 15px 0;
    font-size: 16px;
    color: #333;
  }

  .detail-content {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .detail-field {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .detail-field .label {
    font-weight: 600;
    color: #555;
    font-size: 12px;
  }

  .detail-field .value {
    color: #333;
    font-size: 13px;
    word-break: break-word;
    padding: 4px 8px;
    background-color: #f0f0f0;
    border-radius: 4px;
  }
</style>
