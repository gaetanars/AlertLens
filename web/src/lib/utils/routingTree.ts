import * as d3 from "d3";

export function transformConfigToHierarchy(config: any): d3.HierarchyNode<any> {
  const rootNode = { name: "Root", children: [] };

  function processNode(configNode: any, parentChildren: any[]) {
    const node: any = {
      name: configNode.path || "Catch-all",
      data: configNode,
      children: [],
      _children: []
    };

    if (configNode.routes && configNode.routes.length > 0) {
      configNode.routes.forEach((childConfig: any) => {
        processNode(childConfig, node.children);
      });
    }

    parentChildren.push(node);
  }

  if (config && config.routes && config.routes.length > 0) {
    config.routes.forEach((configNode: any) => {
      processNode(configNode, rootNode.children);
    });
  } else if (config && config.path) { // Handle single route case
    processNode(config, rootNode.children);
  }

  const hierarchy = d3.hierarchy(rootNode, d => d.children);
  hierarchy.descendants().forEach((d, index) => {
      d.id = index; // Assign unique ID to each node
      d.parent = d.parent; // Retain parent reference
      d.data._expanded = true; // Initially expanded
  });
  return hierarchy;
}
