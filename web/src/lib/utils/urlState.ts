/**
 * urlState.ts — URL search-param ↔ store synchronisation helpers.
 *
 * ADR-006: All alert view state is persisted in URL search params so that
 * hard-refresh and link-sharing reproduce the exact same view.
 *
 * Strategy:
 *   - `replaceState` for incremental filter changes (no history pollution).
 *   - `pushState` for view-mode switches (kanban ↔ list) so Back/Forward work.
 */

/** Read every relevant param from a URLSearchParams instance. */
export interface AlertURLState {
	view: 'kanban' | 'list';
	q: string;
	instance: string;
	severity: string[];
	status: string[];
	groupBy: string;
	sort: 'alertname' | 'severity' | 'startsAt' | 'alertmanager';
	sortDir: 'asc' | 'desc';
}

const VALID_VIEWS = new Set(['kanban', 'list']);
const VALID_SORTS = new Set(['alertname', 'severity', 'startsAt', 'alertmanager']);
const VALID_DIRS  = new Set(['asc', 'desc']);

export function parseAlertURLState(params: URLSearchParams): AlertURLState {
	const view      = params.get('view') ?? 'kanban';
	const sort      = params.get('sort') ?? 'startsAt';
	const sortDir   = params.get('sortDir') ?? 'desc';
	const severityRaw = params.get('severity') ?? '';
	const statusRaw   = params.get('status')   ?? '';

	return {
		view:     VALID_VIEWS.has(view)  ? (view as AlertURLState['view'])     : 'kanban',
		q:        params.get('q')        ?? '',
		instance: params.get('instance') ?? '',
		severity: severityRaw ? severityRaw.split(',').filter(Boolean) : [],
		status:   statusRaw   ? statusRaw.split(',').filter(Boolean)   : [],
		groupBy:  params.get('groupBy')  ?? 'severity',
		sort:     VALID_SORTS.has(sort)  ? (sort as AlertURLState['sort'])     : 'startsAt',
		sortDir:  VALID_DIRS.has(sortDir)? (sortDir as AlertURLState['sortDir']): 'desc',
	};
}

export function buildAlertURLParams(state: AlertURLState): URLSearchParams {
	const p = new URLSearchParams();
	if (state.view    !== 'kanban')   p.set('view',     state.view);
	if (state.q)                      p.set('q',        state.q);
	if (state.instance)               p.set('instance', state.instance);
	if (state.severity.length)        p.set('severity', state.severity.join(','));
	if (state.status.length)          p.set('status',   state.status.join(','));
	if (state.groupBy !== 'severity') p.set('groupBy',  state.groupBy);
	if (state.sort    !== 'startsAt') p.set('sort',     state.sort);
	if (state.sortDir !== 'desc')     p.set('sortDir',  state.sortDir);
	return p;
}

/**
 * Push a new URL state into browser history.
 * @param state   New alert URL state.
 * @param push    If true, uses pushState (creates history entry). Default: replaceState.
 */
export function syncURLState(state: AlertURLState, push = false): void {
	if (typeof window === 'undefined') return;
	const params = buildAlertURLParams(state);
	const search = params.toString();
	const url    = search ? `${window.location.pathname}?${search}` : window.location.pathname;
	if (push) {
		window.history.pushState({}, '', url);
	} else {
		window.history.replaceState({}, '', url);
	}
}
