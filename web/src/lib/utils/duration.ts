export interface DurationPreset {
	label: string;
	getValue: () => [Date, Date]; // [startsAt, endsAt]
}

export const DURATION_PRESETS: DurationPreset[] = [
	{
		label: '1 heure',
		getValue: () => {
			const now = new Date();
			return [now, new Date(now.getTime() + 60 * 60 * 1000)];
		}
	},
	{
		label: '2 heures',
		getValue: () => {
			const now = new Date();
			return [now, new Date(now.getTime() + 2 * 60 * 60 * 1000)];
		}
	},
	{
		label: '4 heures',
		getValue: () => {
			const now = new Date();
			return [now, new Date(now.getTime() + 4 * 60 * 60 * 1000)];
		}
	},
	{
		label: "Fin de journée",
		getValue: () => {
			const now = new Date();
			const endOfDay = new Date(now);
			endOfDay.setHours(23, 59, 59, 0);
			return [now, endOfDay];
		}
	},
	{
		label: 'Weekend',
		getValue: () => {
			const now = new Date();
			const monday = new Date(now);
			const day = monday.getDay(); // 0=Sun, 6=Sat
			const daysUntilMonday = day === 0 ? 1 : 8 - day;
			monday.setDate(monday.getDate() + daysUntilMonday);
			monday.setHours(8, 0, 0, 0);
			return [now, monday];
		}
	}
];

export function formatDuration(ms: number): string {
	const seconds = Math.floor(ms / 1000);
	if (seconds < 60) return `${seconds}s`;
	const minutes = Math.floor(seconds / 60);
	if (minutes < 60) return `${minutes}m`;
	const hours = Math.floor(minutes / 60);
	if (hours < 24) return `${hours}h`;
	return `${Math.floor(hours / 24)}j`;
}

export function formatRelative(dateStr: string): string {
	const date = new Date(dateStr);
	const now = new Date();
	const diff = now.getTime() - date.getTime();
	if (diff < 0) return 'à venir';
	const seconds = Math.floor(diff / 1000);
	if (seconds < 60) return 'à l\'instant';
	const minutes = Math.floor(seconds / 60);
	if (minutes < 60) return `il y a ${minutes}m`;
	const hours = Math.floor(minutes / 60);
	if (hours < 24) return `il y a ${hours}h`;
	return `il y a ${Math.floor(hours / 24)}j`;
}
