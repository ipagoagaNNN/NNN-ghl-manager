<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { session, toggleLocation, selectAll, clearSelection } from '$lib/stores/session.svelte.js';
	import {
		accounts,
		loadLibrary,
		seedFromLocations,
		saveToken,
		saveMeta,
		type LibraryEntry,
	} from '$lib/stores/accounts.svelte.js';
	import { toCSV, downloadCSV, parseCSV } from '$lib/utils/csv.js';

	// Per-row pending edits (never echoed back to server until save).
	let pendingTokens = $state<Record<string, string>>({});
	let pendingMeta = $state<Record<string, Omit<LibraryEntry, 'locationId' | 'hasToken'>>>({});
	let search = $state('');
	let statusMsg = $state('');
	let importing = $state(false);

	onMount(async () => {
		if (!session.connected) {
			goto('/connect');
			return;
		}
		seedFromLocations(session.locations);
		await loadLibrary();
		seedFromLocations(session.locations); // re-seed in case library was empty
		// Initialize per-row pending state from current entries
		for (const e of Object.values(accounts.entries)) {
			pendingMeta[e.locationId] = {
				name: e.name,
				domain: e.domain,
				acuityField: e.acuityField,
				calendarIds: e.calendarIds,
				active: e.active,
			};
		}
	});

	const visibleEntries = $derived.by(() => {
		const list = Object.values(accounts.entries);
		const q = search.trim().toLowerCase();
		if (!q) return list;
		return list.filter(
			(e) =>
				e.name.toLowerCase().includes(q) ||
				e.locationId.toLowerCase().includes(q) ||
				e.domain.toLowerCase().includes(q)
		);
	});

	function ensurePending(id: string, current: LibraryEntry) {
		if (!pendingMeta[id]) {
			pendingMeta[id] = {
				name: current.name,
				domain: current.domain,
				acuityField: current.acuityField,
				calendarIds: current.calendarIds,
				active: current.active,
			};
		}
	}

	async function onSaveToken(id: string) {
		const tok = (pendingTokens[id] ?? '').trim();
		if (!tok) {
			statusMsg = `Token for ${id} is empty — nothing saved.`;
			return;
		}
		try {
			await saveToken(id, tok);
			pendingTokens[id] = ''; // clear immediately — never re-echo
			statusMsg = `Token saved server-side for ${id}.`;
		} catch (e) {
			statusMsg = e instanceof Error ? e.message : 'Token save failed';
		}
	}

	async function onSaveMeta(id: string) {
		const pending = pendingMeta[id];
		if (!pending) return;
		try {
			await saveMeta(id, pending);
			statusMsg = `Metadata saved for ${pending.name || id}.`;
		} catch (e) {
			statusMsg = e instanceof Error ? e.message : 'Metadata save failed';
		}
	}

	function exportCSV() {
		const headers = ['locationId', 'name', 'domain', 'acuityField', 'calendarIds', 'active', 'hasToken'];
		const rows = Object.values(accounts.entries).map((e) => ({
			locationId: e.locationId,
			name: e.name,
			domain: e.domain,
			acuityField: e.acuityField,
			calendarIds: e.calendarIds,
			active: e.active ? 'true' : 'false',
			hasToken: e.hasToken ? 'true' : 'false',
		}));
		const csv = toCSV(headers, rows);
		const ts = new Date().toISOString().slice(0, 10);
		downloadCSV(`ghl-accounts-library-${ts}.csv`, csv);
		statusMsg = `Exported ${rows.length} account(s).`;
	}

	async function importCSV(ev: Event) {
		const input = ev.target as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		importing = true;
		try {
			const text = await file.text();
			const parsed = parseCSV(text);
			let imported = 0;
			for (const row of parsed.rows) {
				const id = row['locationId'];
				if (!id) continue;
				const meta = {
					name: row['name'] || '',
					domain: row['domain'] || '',
					acuityField: row['acuityField'] || '',
					calendarIds: row['calendarIds'] || '',
					active: (row['active'] || '').toLowerCase() === 'true',
				};
				await saveMeta(id, meta);
				pendingMeta[id] = meta;
				imported++;
			}
			statusMsg = `Imported metadata for ${imported} account(s). (Tokens are not imported via CSV — set per row.)`;
		} catch (e) {
			statusMsg = e instanceof Error ? e.message : 'CSV import failed';
		} finally {
			importing = false;
			input.value = '';
		}
	}
</script>

<div class="page">
	<header class="head">
		<div>
			<h1>Accounts</h1>
			<p class="sub">
				{Object.keys(accounts.entries).length} sub-accounts · {session.selectedIds.size} selected
			</p>
		</div>
		<div class="toolbar">
			<input
				class="search"
				type="text"
				placeholder="Search name, ID, domain…"
				bind:value={search}
			/>
			<button class="btn-secondary" onclick={selectAll}>Select all</button>
			<button class="btn-secondary" onclick={clearSelection}>Clear</button>
			<button class="btn-secondary" onclick={exportCSV}>Export CSV</button>
			<label class="btn-secondary file-label">
				{importing ? 'Importing…' : 'Import CSV'}
				<input type="file" accept=".csv" onchange={importCSV} disabled={importing} />
			</label>
		</div>
	</header>

	{#if accounts.loading}
		<div class="info">Loading library…</div>
	{/if}
	{#if accounts.error}
		<div class="error">{accounts.error}</div>
	{/if}
	{#if statusMsg}
		<div class="status">{statusMsg}</div>
	{/if}

	<div class="grid">
		{#each visibleEntries as e (e.locationId)}
			{@const _ = ensurePending(e.locationId, e)}
			<div class="card" class:selected={session.selectedIds.has(e.locationId)}>
				<div class="card-head">
					<label class="select">
						<input
							type="checkbox"
							checked={session.selectedIds.has(e.locationId)}
							onchange={() => toggleLocation(e.locationId)}
						/>
						<span class="name">{e.name || e.locationId}</span>
					</label>
					<span class="token-badge" class:on={e.hasToken}>
						{e.hasToken ? '✓ token saved' : '⚠ no token'}
					</span>
				</div>

				<div class="loc-id">{e.locationId}</div>

				<div class="row">
					<label class="field">
						<span>Sub-account token</span>
						<input
							type="password"
							placeholder={e.hasToken ? '••••••••• (saved — replace to update)' : 'pit-...'}
							bind:value={pendingTokens[e.locationId]}
							autocomplete="off"
						/>
					</label>
					<button
						class="btn"
						disabled={accounts.savingId === e.locationId}
						onclick={() => onSaveToken(e.locationId)}
					>
						Save token
					</button>
				</div>

				<div class="meta-grid">
					<label class="field">
						<span>Display name</span>
						<input type="text" bind:value={pendingMeta[e.locationId].name} />
					</label>
					<label class="field">
						<span>Domain</span>
						<input
							type="text"
							placeholder="example.com"
							bind:value={pendingMeta[e.locationId].domain}
						/>
					</label>
					<label class="field">
						<span>Acuity field</span>
						<input
							type="text"
							placeholder="field:12345678"
							bind:value={pendingMeta[e.locationId].acuityField}
						/>
					</label>
					<label class="field">
						<span>Calendar IDs (comma-separated)</span>
						<input
							type="text"
							placeholder="cal_abc, cal_xyz"
							bind:value={pendingMeta[e.locationId].calendarIds}
						/>
					</label>
					<label class="checkbox">
						<input type="checkbox" bind:checked={pendingMeta[e.locationId].active} />
						<span>Active (include in bulk operations)</span>
					</label>
				</div>

				<div class="actions">
					<button
						class="btn"
						disabled={accounts.savingId === e.locationId}
						onclick={() => onSaveMeta(e.locationId)}
					>
						{accounts.savingId === e.locationId ? 'Saving…' : 'Save metadata'}
					</button>
				</div>
			</div>
		{/each}

		{#if visibleEntries.length === 0 && !accounts.loading}
			<div class="empty">
				{search ? `No accounts match "${search}"` : 'No accounts yet — go to /connect to load sub-accounts.'}
			</div>
		{/if}
	</div>
</div>

<style>
	.page { display: flex; flex-direction: column; gap: 16px; }

	.head {
		display: flex;
		justify-content: space-between;
		align-items: flex-end;
		gap: 16px;
		flex-wrap: wrap;
	}
	h1 { font-size: 24px; font-weight: 700; }
	.sub { font-size: 13px; color: var(--text2); margin-top: 4px; }

	.toolbar { display: flex; gap: 8px; flex-wrap: wrap; }
	.search {
		padding: 8px 12px;
		border: 1.5px solid var(--border);
		border-radius: 8px;
		font-family: inherit;
		font-size: 13px;
		min-width: 240px;
	}
	.search:focus { outline: none; border-color: var(--accent); }

	.btn, .btn-secondary {
		padding: 8px 14px;
		border-radius: 8px;
		font-family: inherit;
		font-size: 13px;
		font-weight: 600;
		cursor: pointer;
		transition: opacity 0.15s, background 0.15s;
		border: none;
	}
	.btn { background: var(--accent); color: #fff; }
	.btn:disabled { opacity: 0.6; cursor: not-allowed; }
	.btn-secondary {
		background: rgba(0,0,0,0.04);
		color: var(--text2);
		border: 1.5px solid var(--border);
	}
	.btn-secondary:hover { background: rgba(0,0,0,0.08); }
	.file-label { display: inline-flex; align-items: center; cursor: pointer; }
	.file-label input[type='file'] { display: none; }

	.info, .error, .status, .empty {
		padding: 12px 16px;
		border-radius: 10px;
		font-size: 13px;
	}
	.info { background: rgba(0,0,0,0.04); color: var(--text2); }
	.error { background: rgba(255,59,92,0.08); color: var(--error); font-weight: 600; }
	.status { background: rgba(0,201,122,0.08); color: var(--success); font-weight: 600; }
	.empty { color: var(--text2); padding: 40px; text-align: center; }

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(440px, 1fr));
		gap: 16px;
	}

	.card {
		background: var(--surface);
		border: 1.5px solid var(--border);
		border-radius: 14px;
		padding: 18px;
		display: flex;
		flex-direction: column;
		gap: 12px;
		transition: border-color 0.15s;
	}
	.card.selected { border-color: var(--accent); }

	.card-head {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 8px;
	}
	.select { display: flex; align-items: center; gap: 8px; cursor: pointer; }
	.select input { width: 16px; height: 16px; cursor: pointer; }
	.name { font-weight: 700; font-size: 15px; }

	.token-badge {
		padding: 2px 8px;
		border-radius: 20px;
		font-size: 11px;
		font-weight: 600;
		background: rgba(255,59,92,0.1);
		color: var(--error);
	}
	.token-badge.on { background: rgba(0,201,122,0.1); color: var(--success); }

	.loc-id {
		font-family: ui-monospace, monospace;
		font-size: 11px;
		color: var(--text2);
		padding: 4px 8px;
		background: rgba(0,0,0,0.04);
		border-radius: 6px;
		align-self: flex-start;
	}

	.row { display: flex; gap: 8px; align-items: flex-end; }
	.row .field { flex: 1; }

	.field { display: flex; flex-direction: column; gap: 4px; }
	.field > span { font-size: 11px; font-weight: 600; color: var(--text2); text-transform: uppercase; letter-spacing: 0.4px; }
	.field input[type='text'], .field input[type='password'] {
		padding: 8px 12px;
		border: 1.5px solid var(--border);
		border-radius: 8px;
		font-family: inherit;
		font-size: 13px;
	}
	.field input:focus { outline: none; border-color: var(--accent); }

	.meta-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 10px;
	}
	.checkbox {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 12px;
		grid-column: 1 / -1;
	}
	.checkbox input { width: 16px; height: 16px; }

	.actions { display: flex; justify-content: flex-end; }
</style>
