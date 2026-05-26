// Session store — replaces localStorage globals from the HTML prototype.
// Tokens are NEVER stored here; they live server-side in the Go vault.

export interface LocationSummary {
	id: string;
	name: string;
	businessName?: string;
	hasToken: boolean;
}

export const session = $state({
	companyId: '',
	connected: false,
	locations: [] as LocationSummary[],
	selectedIds: new Set<string>(),
	loading: false,
	error: '',
});

export function selectLocation(id: string) {
	session.selectedIds.add(id);
}

export function deselectLocation(id: string) {
	session.selectedIds.delete(id);
}

export function toggleLocation(id: string) {
	if (session.selectedIds.has(id)) {
		session.selectedIds.delete(id);
	} else {
		session.selectedIds.add(id);
	}
}

export function selectAll() {
	session.locations.forEach((l) => session.selectedIds.add(l.id));
}

export function clearSelection() {
	session.selectedIds.clear();
}

export function resetSession() {
	session.companyId = '';
	session.connected = false;
	session.locations = [];
	session.selectedIds.clear();
	session.error = '';
}
