import { writable } from 'svelte/store';

/**
 * Holds the most-recently assembled proposed Alertmanager config YAML from any
 * of the Config Builder tabs (Routing, Receivers, Time Intervals).
 *
 * The Save & Deploy page reads this to pre-populate its diff view so the user
 * can review and publish the assembled changes without re-entering them.
 *
 * Set to null when no changes have been assembled since the page was loaded.
 */
export const configDraftStore = writable<{ instance: string; rawYaml: string } | null>(null);
