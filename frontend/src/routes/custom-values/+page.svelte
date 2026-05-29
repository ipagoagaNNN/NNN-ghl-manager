<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { session } from '$lib/stores/session.svelte.js';
	import { apiGet, apiPost } from '$lib/api/client.js';
	import { accounts, loadLibrary } from '$lib/stores/accounts.svelte.js';
	import { BRAND_PRESETS } from '$lib/data/brand-presets.js';

	interface CVItem {
		id: string;
		name: string;
		value: string;
		fieldType?: string;
	}
	interface LocationCV {
		locationId: string;
		name: string;
		cvs: CVItem[];
		error?: string;
	}
	interface CVListResponse {
		locations: LocationCV[];
		count: number;
	}
	interface BulkResult {
		results: { locationId: string; customValueId: string; ok: boolean; error?: string }[];
		okCount: number;
		errorCount: number;
	}

	let locations = $state<LocationCV[]>([]);
	// pending[locationId][cvId] = staged value (only present if edited)
	let pending = $state<Record<string, Record<string, string>>>({});
	let loading = $state(false);
	let applying = $state(false);
	let error = $state('');
	let statusMsg = $state('');
	let search = $state('');
	let lastResult = $state<BulkResult | null>(null);

	onMount(async () => {
		if (!session.connected) {
			goto('/connect');
			return;
		}
		await loadLibrary();
		await loadCVs();
	});

	function targetIds(): string[] {
		let list = Object.values(accounts.entries).filter((e) => e.hasToken);
		if (session.selectedIds.size > 0) {
			list = list.filter((e) => session.selectedIds.has(e.locationId));
		}
		return list.map((e) => e.locationId);
	}

	async function loadCVs() {
		const ids = targetIds();
		if (ids.length === 0) {
			locations = [];
			return;
		}
		loading = true;
		error = '';
		lastResult = null;
		try {
			const data = await apiGet<CVListResponse>(`/api/cv?locationIds=${encodeURIComponent(ids.join(','))}`);
			locations = data.locations ?? [];
			pending = {}; // reset staged edits on fresh load
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load custom values';
		} finally {
			loading = false;
		}
	}

	function currentValue(locId: string, cv: CVItem): string {
		const staged = pending[locId]?.[cv.id];
		return staged !== undefined ? staged : cv.value;
	}

	function isChanged(locId: string, cv: CVItem): boolean {
		const staged = pending[locId]?.[cv.id];
		return staged !== undefined && staged !== cv.value;
	}

	function setPending(locId: string, cvId: string, value: string) {
		if (!pending[locId]) pending[locId] = {};
		pending[locId][cvId] = value;
	}

	function clearPending() {
		pending = {};
		statusMsg = 'Cleared all staged edits.';
	}

	const pendingCount = $derived.by(() => {
		let n = 0;
		for (const loc of locations) {
			const map = pending[loc.locationId];
			if (!map) continue;
			for (const cv of loc.cvs) {
				if (map[cv.id] !== undefined && map[cv.id] !== cv.value) n++;
			}
		}
		return n;
	});

	const totalCVs = $derived(locations.reduce((sum, l) => sum + l.cvs.length, 0));

	// Apply a brand preset: stage values for every CV whose NAME matches a preset key,
	// across all loaded accounts.
	function applyPreset(presetKey: string) {
		const preset = BRAND_PRESETS.find((p) => p.key === presetKey);
		if (!preset) return;
		let staged = 0;
		for (const loc of locations) {
			for (const cv of loc.cvs) {
				if (Object.prototype.hasOwnProperty.call(preset.values, cv.name)) {
					const v = preset.values[cv.name];
					if (v !== cv.value) {
						setPending(loc.locationId, cv.id, v);
						staged++;
					}
				}
			}
		}
		statusMsg =
			staged > 0
				? `Staged ${staged} value(s) from "${preset.label}". Review, then Apply.`
				: `No matching CV names found for "${preset.label}" in the loaded accounts.`;
	}

	async function applyChanges() {
		const updates: { locationId: string; customValueId: string; value: string }[] = [];
		for (const loc of locations) {
			const map = pending[loc.locationId];
			if (!map) continue;
			for (const cv of loc.cvs) {
				if (map[cv.id] !== undefined && map[cv.id] !== cv.value) {
					updates.push({ locationId: loc.locationId, customValueId: cv.id, value: map[cv.id] });
				}
			}
		}
		if (updates.length === 0) {
			statusMsg = 'Nothing to apply.';
			return;
		}
		applying = true;
		error = '';
		try {
			const res = await apiPost<BulkResult>('/api/cv/bulk', { updates });
			lastResult = res;
			statusMsg = `Applied: ${res.okCount} ok, ${res.errorCount} failed.`;
			await loadCVs(); // refresh to reflect server truth
		} catch (e) {
			error = e instanceof Error ? e.message : 'Bulk apply failed';
		} finally {
			applying = false;
		}
	}

	function filteredCVs(cvs: CVItem[]): CVItem[] {
		const q = search.trim().toLowerCase();
		if (!q) return cvs;
		return cvs.filter((cv) => cv.name.toLowerCase().includes(q) || cv.value.toLowerCase().includes(q));
	}
</script>

<div class="page">
	<header class="head">
		<div>
			<h1>Custom Values</h1>
			<p class="sub">
				{locations.length} account(s) · {totalCVs} value(s) · {pendingCount} staged edit(s)
			</p>
		</div>
		<div class="toolbar">
			<input class="search" type="text" placeholder="Search name or value…" bind:value={search} />
			<button class="btn-secondary" onclick={loadCVs} disabled={loading}>
				{loading ? 'Loading…' : 'Reload'}
			</button>
		</div>
	</header>

	<div class="presets">
		<span class="presets-label">Brand bulk-fill:</span>
		{#each BRAND_PRESETS as preset (preset.key)}
			<button
				class="preset-btn"
				style="--brand:{preset.color}"
				onclick={() => applyPreset(preset.key)}
				title={`${Object.keys(preset.values).length} values`}
			>
				{preset.label}
			</button>
		{/each}
		<div class="apply-bar">
			<button class="btn-ghost" onclick={clearPending} disabled={pendingCount === 0}>Clear edits</button>
			<button class="btn" onclick={applyChanges} disabled={applying || pendingCount === 0}>
				{applying ? 'Applying…' : `Apply ${pendingCount} change(s)`}
			</button>
		</div>
	</div>

	{#if error}<div class="error">{error}</div>{/if}
	{#if statusMsg}<div class="status">{statusMsg}</div>{/if}

	{#if lastResult && lastResult.errorCount > 0}
		<div class="result-log">
			<strong>{lastResult.errorCount} update(s) failed:</strong>
			<ul>
				{#each lastResult.results.filter((r) => !r.ok) as r}
					<li><code>{r.locationId}</code> · <code>{r.customValueId}</code> — {r.error}</li>
				{/each}
			</ul>
		</div>
	{/if}

	{#if locations.length === 0 && !loading}
		<div class="empty">
			No accounts with tokens (or none selected). Go to <a href="/accounts">Accounts</a> to save tokens,
			or select accounts there to scope this view.
		</div>
	{/if}

	{#each locations as loc (loc.locationId)}
		{@const rows = filteredCVs(loc.cvs)}
		<section class="account">
			<div class="account-head">
				<h2>{loc.name || loc.locationId}</h2>
				<span class="loc-id">{loc.locationId}</span>
				<span class="count-badge">{rows.length} / {loc.cvs.length} shown</span>
			</div>

			{#if loc.error}
				<div class="error">{loc.error}</div>
			{:else if loc.cvs.length === 0}
				<div class="info">No custom values for this account.</div>
			{:else if rows.length === 0}
				<div class="info">No values match "{search}".</div>
			{:else}
				<div class="cv-list">
					{#each rows as cv (cv.id)}
						<div class="cv-row" class:changed={isChanged(loc.locationId, cv)}>
							<label class="cv-name" for={`cv-${loc.locationId}-${cv.id}`}>{cv.name}</label>
							<input
								id={`cv-${loc.locationId}-${cv.id}`}
								class="cv-input"
								type="text"
								value={currentValue(loc.locationId, cv)}
								oninput={(e) => setPending(loc.locationId, cv.id, e.currentTarget.value)}
							/>
							{#if isChanged(loc.locationId, cv)}
								<span class="changed-dot" title="Edited — not yet applied"></span>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
		</section>
	{/each}
</div>

<style>
	.page { display: flex; flex-direction: column; gap: 16px; }

	.head { display: flex; justify-content: space-between; align-items: flex-end; gap: 16px; flex-wrap: wrap; }
	h1 { font-size: 24px; font-weight: 700; }
	.sub { font-size: 13px; color: var(--text2); margin-top: 4px; }

	.toolbar { display: flex; gap: 8px; flex-wrap: wrap; }
	.search {
		padding: 8px 12px; border: 1.5px solid var(--border); border-radius: 8px;
		font-family: inherit; font-size: 13px; min-width: 220px;
	}
	.search:focus { outline: none; border-color: var(--accent); }

	.presets {
		display: flex; align-items: center; gap: 8px; flex-wrap: wrap;
		background: var(--surface); border: 1.5px solid var(--border);
		border-radius: 12px; padding: 12px 16px;
	}
	.presets-label { font-size: 12px; font-weight: 600; color: var(--text2); }
	.preset-btn {
		padding: 7px 14px; border-radius: 8px; font-family: inherit; font-size: 13px; font-weight: 600;
		cursor: pointer; border: 1.5px solid var(--brand); color: var(--brand); background: transparent;
		transition: background 0.15s, color 0.15s;
	}
	.preset-btn:hover { background: var(--brand); color: #fff; }
	.apply-bar { margin-left: auto; display: flex; gap: 8px; }

	.btn, .btn-secondary, .btn-ghost {
		padding: 8px 14px; border-radius: 8px; font-family: inherit; font-size: 13px;
		font-weight: 600; cursor: pointer; border: none;
	}
	.btn { background: var(--accent); color: #fff; }
	.btn:disabled { opacity: 0.5; cursor: not-allowed; }
	.btn-secondary { background: rgba(0,0,0,0.04); color: var(--text2); border: 1.5px solid var(--border); }
	.btn-secondary:hover { background: rgba(0,0,0,0.08); }
	.btn-secondary:disabled { opacity: 0.6; cursor: not-allowed; }
	.btn-ghost { background: transparent; color: var(--text2); border: 1.5px solid var(--border); }
	.btn-ghost:disabled { opacity: 0.5; cursor: not-allowed; }

	.info, .error, .status, .empty, .result-log {
		padding: 12px 16px; border-radius: 10px; font-size: 13px;
	}
	.info { background: rgba(0,0,0,0.04); color: var(--text2); }
	.error { background: rgba(255,59,92,0.08); color: var(--error); font-weight: 600; }
	.status { background: rgba(0,201,122,0.08); color: var(--success); font-weight: 600; }
	.empty { color: var(--text2); padding: 32px; text-align: center; }
	.empty a { color: var(--accent); font-weight: 600; }
	.result-log { background: rgba(255,59,92,0.06); color: var(--error); }
	.result-log ul { margin: 6px 0 0 18px; }
	.result-log code { font-family: ui-monospace, monospace; font-size: 11px; }

	.account {
		background: var(--surface); border: 1.5px solid var(--border);
		border-radius: 14px; padding: 18px; display: flex; flex-direction: column; gap: 12px;
	}
	.account-head { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
	.account-head h2 { font-size: 16px; font-weight: 700; }
	.loc-id {
		font-family: ui-monospace, monospace; font-size: 11px; color: var(--text2);
		padding: 3px 8px; background: rgba(0,0,0,0.04); border-radius: 6px;
	}
	.count-badge { font-size: 11px; font-weight: 600; color: var(--text2); margin-left: auto; }

	.cv-list { display: flex; flex-direction: column; gap: 8px; }
	.cv-row {
		display: grid; grid-template-columns: 260px 1fr 16px; gap: 12px; align-items: center;
		padding: 6px 8px; border-radius: 8px;
	}
	.cv-row.changed { background: rgba(255,29,141,0.05); }
	.cv-name { font-size: 13px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.cv-input {
		padding: 8px 12px; border: 1.5px solid var(--border); border-radius: 8px;
		font-family: inherit; font-size: 13px;
	}
	.cv-input:focus { outline: none; border-color: var(--accent); }
	.cv-row.changed .cv-input { border-color: var(--accent); }
	.changed-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--accent); }
</style>
