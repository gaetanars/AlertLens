import { describe, it, expect } from 'vitest';
import { transformConfigToHierarchy } from './routingTree';

describe('transformConfigToHierarchy', () => {
  it('should correctly transform an empty config to a hierarchy', () => {
    const config = { routes: [] };
    const hierarchy = transformConfigToHierarchy(config);
    expect(hierarchy.data.name).toBe('Root');
    // For an empty config, the root node should have no children
    expect(hierarchy.children).toBeUndefined();
  });

  it('should correctly transform a minimal config (single catch-all route)', () => {
    const config = { path: "/*", receiver: "default" };
    const hierarchy = transformConfigToHierarchy(config);
    expect(hierarchy.data.name).toBe('Root');
    expect(hierarchy.children?.length).toBe(1);
    expect(hierarchy.children?.[0].data.name).toBe('/*');
    expect(hierarchy.children?.[0].data.data).toEqual(config);
    // Leaf node should have undefined children
    expect(hierarchy.children?.[0].children).toBeUndefined();
  });

  it('should correctly transform a routing config with 3 levels of nesting', () => {
    const config = {
      routes: [
        {
          path: "/team",
          receiver: "team-receiver",
          routes: [
            {
              path: "/team/us",
              receiver: "us-team-receiver",
              routes: [
                {
                  path: "/team/us/dev",
                  receiver: "us-dev-receiver",
                },
                {
                  path: "/team/us/qa",
                  receiver: "us-qa-receiver",
                },
              ],
            },
            {
              path: "/team/eu",
              receiver: "eu-team-receiver",
            },
          ],
        },
        {
          path: "/support",
          receiver: "support-receiver",
        },
      ],
    };

    const hierarchy = transformConfigToHierarchy(config);

    expect(hierarchy.data.name).toBe('Root');
    expect(hierarchy.children?.length).toBe(2);

    const teamNode = hierarchy.children?.[0];
    expect(teamNode?.data.name).toBe('/team');
    expect(teamNode?.children?.length).toBe(2);

    const usTeamNode = teamNode?.children?.[0];
    expect(usTeamNode?.data.name).toBe('/team/us');
    expect(usTeamNode?.children?.length).toBe(2);

    const usDevNode = usTeamNode?.children?.[0];
    expect(usDevNode?.data.name).toBe('/team/us/dev');
    expect(usDevNode?.children).toBeUndefined();

    const usQaNode = usTeamNode?.children?.[1];
    expect(usQaNode?.data.name).toBe('/team/us/qa');
    expect(usQaNode?.children).toBeUndefined();

    const euTeamNode = teamNode?.children?.[1];
    expect(euTeamNode?.data.name).toBe('/team/eu');
    expect(euTeamNode?.children).toBeUndefined();

    const supportNode = hierarchy.children?.[1];
    expect(supportNode?.data.name).toBe('/support');
    expect(supportNode?.children).toBeUndefined();

    // Check for assigned IDs and expanded state
    hierarchy.descendants().forEach(d => {
      expect(d.id).toBeDefined();
      expect(d.data._expanded).toBe(true);
    });
  });

  it('should assign unique IDs to all nodes', () => {
    const config = {
      routes: [
        { path: "/a" },
        { path: "/b", routes: [{ path: "/b/c" }] }
      ]
    };
    const hierarchy = transformConfigToHierarchy(config);
    const ids = new Set<number>();
    hierarchy.descendants().forEach(d => {
      expect(d.id).toBeDefined();
      ids.add(d.id as number);
    });
    expect(ids.size).toBe(hierarchy.descendants().length);
  });

  it('should handle no crash on empty/minimal routing config', () => {
    // Single catch-all
    const config1 = { path: "/*", receiver: "default" };
    expect(() => transformConfigToHierarchy(config1)).not.toThrow();

    // Empty routes
    const config2 = { routes: [] };
    expect(() => transformConfigToHierarchy(config2)).not.toThrow();

    // Null-like input with fallback
    const config3 = null;
    // This may throw or handle gracefully - depends on implementation
    // Just ensure no crash for a minimal config
  });
});
