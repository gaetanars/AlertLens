package alertmanager

// ExportedGroupAlerts exposes groupAlerts for unit tests in external packages.
func ExportedGroupAlerts(alerts []EnrichedAlert, groupBy []string) []AlertGroup {
	return groupAlerts(alerts, groupBy)
}

// ExportedApplyViewFilters exposes applyViewFilters for unit tests.
func ExportedApplyViewFilters(alerts []EnrichedAlert, params AlertsViewParams) []EnrichedAlert {
	return applyViewFilters(alerts, params)
}
