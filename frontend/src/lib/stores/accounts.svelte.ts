// Accounts store — per-location metadata + token presence flag.
// Tokens are NEVER stored client-side; this store only tracks whether the
// server-side vault has one (hasToken: boolean).

import { apiGet, apiPost, apiPut } from '$lib/api/client.js';

export interface LibraryEntry {
	locationId: string;
	name: string;
	domain: string;
	acuityField: string;
	calendarIds: string;
	active: boolean;
	hasToken: boolean;
}

export interface LibraryResponse {
	library: LibraryEntry[];
	count: number;
}

interface AccountsState {
	entries: Record<string, LibraryEntry>; // keyed by locationId
	loading: boolean;
	error: string;
	savingId: string | null;
}

export const accounts = $state<AccountsState>({
	entries: {},
	loading: false,
	error: '',
	savingId: null,
});

/** Fetch the library from the backend and merge into the store. */
export async function loadLibrary(): Promise<void> {
	accounts.loading = true;
	accounts.error = '';
	try {
		const data = await apiGet<LibraryResponse>('/api/accounts/library');
		const map: Record<string, LibraryEntry> = {};
		for (const e of data.library) {
			map[e.locationId] = e;
		}
		accounts.entries = map;
	} catch (e) {
		accounts.error = e instanceof Error ? e.message : 'Failed to load library';
	} finally {
		accounts.loading = false;
	}
}

/** Seed the store from session.locations after Connect, so accounts show up even before metadata is saved. */
export function seedFromLocations(
	locations: { id: string; name: string; businessName?: string }[]
): void {
	for (const loc of locations) {
		if (!accounts.entries[loc.id]) {
			accounts.entries[loc.id] = {
				locationId: loc.id,
				name: loc.name || loc.businessName || loc.id,
				domain: '',
				acuityField: '',
				calendarIds: '',
				active: true,
				hasToken: false,
			};
		}
	}
}

/** Save the sub-account token (POST /api/tokens/:locationId). Server-only — token never echoed back. */
export async function saveToken(locationId: string, token: string): Promise<void> {
	accounts.savingId = locationId;
	try {
		const res = await fetch(`/api/tokens/${encodeURIComponent(locationId)}`, {
			method: 'POST',
			credentials: 'include',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ token }),
		});
		if (!res.ok) {
			const t = await res.text();
			throw new Error(`save token failed: ${res.status} ${t}`);
		}
		// Update local flag — but never persist the token value.
		const entry = accounts.entries[locationId];
		if (entry) entry.hasToken = true;
	} finally {
		accounts.savingId = null;
	}
}

/** Save per-location metadata (PUT /api/accounts/:locationId/meta). */
export async function saveMeta(locationId: string, meta: Omit<LibraryEntry, 'locationId' | 'hasToken'>): Promise<void> {
	accounts.savingId = locationId;
	try {
		await apiPut(`/api/accounts/${encodeURIComponent(locationId)}/meta`, {
			name: meta.name,
			domain: meta.domain,
			acuityField: meta.acuityField,
			calendarIds: meta.calendarIds,
			active: meta.active,
		});
		const entry = accounts.entries[locationId];
		if (entry) {
			entry.name = meta.name;
			entry.domain = meta.domain;
			entry.acuityField = meta.acuityField;
			entry.calendarIds = meta.calendarIds;
			entry.active = meta.active;
		}
	} finally {
		accounts.savingId = null;
	}
}

// Re-export the API helpers to silence "unused import" warnings if a caller imports
// only the store — apiPost is reserved for the future CSV-library bulk endpoint.
export { apiPost };
