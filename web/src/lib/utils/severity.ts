export type Severity = 'critical' | 'warning' | 'info' | 'none';

export function getSeverity(labels: Record<string, string>): Severity {
	const s = labels['severity']?.toLowerCase();
	if (s === 'critical') return 'critical';
	if (s === 'warning') return 'warning';
	if (s === 'info') return 'info';
	return 'none';
}

export const SEVERITY_ORDER: Severity[] = ['critical', 'warning', 'info', 'none'];

export const SEVERITY_CLASSES: Record<Severity, string> = {
	critical: 'border-l-4 border-red-500 bg-red-50 dark:bg-red-950/30',
	warning:  'border-l-4 border-yellow-500 bg-yellow-50 dark:bg-yellow-950/30',
	info:     'border-l-4 border-blue-500 bg-blue-50 dark:bg-blue-950/30',
	none:     'border-l-4 border-gray-300 bg-gray-50 dark:bg-gray-900/30'
};

export const SEVERITY_BADGE: Record<Severity, string> = {
	critical: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
	warning:  'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
	info:     'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
	none:     'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200'
};
