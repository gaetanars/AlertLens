import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import InstanceSelector from './InstanceSelector.svelte';
import type { InstanceStatus } from '$lib/api/types';

const healthyInstances: InstanceStatus[] = [
	{ name: 'prod-eu', url: 'http://eu.example.com', healthy: true, version: '0.27.0', has_tenant: false },
	{ name: 'prod-us', url: 'http://us.example.com', healthy: true, version: '0.27.0', has_tenant: false }
];

const degradedInstances: InstanceStatus[] = [
	{ name: 'prod-eu', url: 'http://eu.example.com', healthy: true, version: '0.27.0', has_tenant: false },
	{ name: 'prod-us', url: 'http://us.example.com', healthy: false, version: '', has_tenant: false, error: 'connection refused' }
];

describe('InstanceSelector', () => {
	it('renders "All instances" option by default', () => {
		render(InstanceSelector, { instances: healthyInstances, value: '' });
		const select = screen.getByRole('combobox', { name: /filter by alertmanager instance/i });
		expect(select).toBeTruthy();
		// Default option should be "All instances"
		const options = select.querySelectorAll('option');
		expect(options[0].value).toBe('');
		expect(options[0].textContent?.trim()).toBe('All instances');
	});

	it('renders all instance options', () => {
		render(InstanceSelector, { instances: healthyInstances, value: '' });
		const select = screen.getByRole('combobox');
		const options = select.querySelectorAll('option');
		// +1 for "All instances"
		expect(options).toHaveLength(3);
		expect(options[1].value).toBe('prod-eu');
		expect(options[2].value).toBe('prod-us');
	});

	it('shows degraded badge when any instance is unhealthy', () => {
		render(InstanceSelector, { instances: degradedInstances, value: '' });
		const badge = screen.getByText(/degraded/i);
		expect(badge).toBeTruthy();
	});

	it('does not show degraded badge when all instances are healthy', () => {
		render(InstanceSelector, { instances: healthyInstances, value: '' });
		const badge = screen.queryByText(/degraded/i);
		expect(badge).toBeNull();
	});

	it('calls onChange when selection changes', async () => {
		const onChange = vi.fn();
		render(InstanceSelector, { instances: healthyInstances, value: '', onChange });
		const select = screen.getByRole('combobox');
		await fireEvent.change(select, { target: { value: 'prod-eu' } });
		expect(onChange).toHaveBeenCalledWith('prod-eu');
	});

	it('is disabled when no instances are available', () => {
		render(InstanceSelector, { instances: [], value: '' });
		const select = screen.getByRole('combobox');
		expect(select).toBeDisabled();
	});

	it('renders unhealthy instance label with warning icon', () => {
		render(InstanceSelector, { instances: degradedInstances, value: '' });
		const select = screen.getByRole('combobox');
		const options = select.querySelectorAll('option');
		// The unhealthy instance (prod-us) should have '⚠' in its label
		const usOption = Array.from(options).find((o) => o.value === 'prod-us');
		expect(usOption?.textContent).toContain('⚠');
	});

	it('shows green health dot for selected healthy instance', () => {
		const { container } = render(InstanceSelector, { instances: healthyInstances, value: 'prod-eu' });
		const dot = container.querySelector('span.bg-green-500');
		expect(dot).toBeTruthy();
	});

	it('shows red health dot for selected unhealthy instance', () => {
		const { container } = render(InstanceSelector, { instances: degradedInstances, value: 'prod-us' });
		const dot = container.querySelector('span.bg-red-500');
		expect(dot).toBeTruthy();
	});
});
