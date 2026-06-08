<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { session } from '$lib/stores/session.svelte.js';
	import { apiGet, apiPost, apiDelete } from '$lib/api/client.js';
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
		const updates: { locationId: string; customValueId: string; name: string; value: string }[] = [];
		for (const loc of locations) {
			const map = pending[loc.locationId];
			if (!map) continue;
			for (const cv of loc.cvs) {
				if (map[cv.id] !== undefined && map[cv.id] !== cv.value) {
					// Send name alongside value — GHL PUT needs it (matches prototype).
					updates.push({ locationId: loc.locationId, customValueId: cv.id, name: cv.name, value: map[cv.id] });
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

	let deletingId = $state<string | null>(null);

	async function deleteCV(locId: string, cv: CVItem) {
		if (!confirm(`Delete custom value "${cv.name}"? This cannot be undone.`)) return;
		deletingId = cv.id;
		error = '';
		try {
			await apiDelete(`/api/cv/${encodeURIComponent(locId)}/${encodeURIComponent(cv.id)}`);
			const loc = locations.find((l) => l.locationId === locId);
			if (loc) loc.cvs = loc.cvs.filter((c) => c.id !== cv.id);
			statusMsg = `Deleted "${cv.name}".`;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Delete failed';
		} finally {
			deletingId = null;
		}
	}

	function filteredCVs(cvs: CVItem[]): CVItem[] {
		const q = search.trim().toLowerCase();
		if (!q) return cvs;
		return cvs.filter((cv) => cv.name.toLowerCase().includes(q) || cv.value.toLowerCase().includes(q));
	}

	// ── New Patient Link scanner (M3 decision D-3.1) ──────────────────────────
	// Read-only audit of each account's booking-link CV against its registered
	// domain. Fixes are applied through the same POST /api/cv/bulk path.
	interface NPLResult {
		locationId: string;
		name: string;
		domain: string;
		cvId?: string;
		cvName?: string;
		value: string;
		verdict: 'valid' | 'valid_unverified' | 'missing' | 'malformed' | 'domain_mismatch' | 'error';
		detail?: string;
		error?: string;
	}
	interface NPLSummary {
		total: number;
		valid: number;
		missing: number;
		malformed: number;
		domainMismatch: number;
		errors: number;
	}
	interface NPLScanResponse {
		key: string;
		results: NPLResult[];
		summary: NPLSummary;
	}

	const NPL_BADGE: Record<string, { label: string; cls: string }> = {
		valid: { label: 'Valid', cls: 'ok' },
		valid_unverified: { label: 'Valid (no domain)', cls: 'warn' },
		missing: { label: 'Missing', cls: 'bad' },
		malformed: { label: 'Malformed', cls: 'bad' },
		domain_mismatch: { label: 'Domain mismatch', cls: 'warn' },
		error: { label: 'Error', cls: 'bad' }
	};

	let nplKey = $state('New Patient Link');
	let nplScan = $state<NPLScanResponse | null>(null);
	let nplLoading = $state(false);
	let nplApplying = $state(false);
	let nplError = $state('');
	let nplProblemsOnly = $state(true);
	// nplFixes[locationId] = corrected link (only meaningful for fixable rows)
	let nplFixes = $state<Record<string, string>>({});

	function nplIsFixable(r: NPLResult): boolean {
		return !!r.cvId && (r.verdict === 'malformed' || r.verdict === 'domain_mismatch');
	}

	async function runNplScan() {
		const ids = targetIds();
		if (ids.length === 0) {
			nplError = 'No accounts with tokens (or none selected).';
			return;
		}
		nplLoading = true;
		nplError = '';
		nplScan = null;
		nplFixes = {};
		try {
			const q = `/api/cv/npl-scan?locationIds=${encodeURIComponent(ids.join(','))}&key=${encodeURIComponent(nplKey)}`;
			const data = await apiGet<NPLScanResponse>(q);
			nplScan = data;
			// Prefill fixable rows with the current (wrong) value so the operator can correct it.
			for (const r of data.results) {
				if (nplIsFixable(r)) nplFixes[r.locationId] = r.value;
			}
		} catch (e) {
			nplError = e instanceof Error ? e.message : 'NPL scan failed';
		} finally {
			nplLoading = false;
		}
	}

	function nplVisibleResults(): NPLResult[] {
		if (!nplScan) return [];
		if (!nplProblemsOnly) return nplScan.results;
		return nplScan.results.filter((r) => r.verdict !== 'valid' && r.verdict !== 'valid_unverified');
	}

	const nplFixCount = $derived.by(() => {
		if (!nplScan) return 0;
		let n = 0;
		for (const r of nplScan.results) {
			const v = (nplFixes[r.locationId] ?? '').trim();
			if (nplIsFixable(r) && v !== '' && v !== r.value) n++;
		}
		return n;
	});

	async function applyNplFixes() {
		if (!nplScan) return;
		const updates: { locationId: string; customValueId: string; name: string; value: string }[] = [];
		for (const r of nplScan.results) {
			if (!nplIsFixable(r) || !r.cvId) continue;
			const v = (nplFixes[r.locationId] ?? '').trim();
			if (v === '' || v === r.value) continue;
			updates.push({ locationId: r.locationId, customValueId: r.cvId, name: r.cvName || nplScan.key, value: v });
		}
		if (updates.length === 0) {
			nplError = 'No corrected links to apply.';
			return;
		}
		nplApplying = true;
		nplError = '';
		try {
			await apiPost<BulkResult>('/api/cv/bulk', { updates });
			await runNplScan(); // re-scan to reflect server truth
		} catch (e) {
			nplError = e instanceof Error ? e.message : 'Applying fixes failed';
		} finally {
			nplApplying = false;
		}
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

	<section class="npl">
		<div class="npl-head">
			<div>
				<h2>New Patient Link scanner</h2>
				<p class="npl-sub">
					Audits each account's booking link against its registered domain. Read-only — corrections
					apply through the same bulk update.
				</p>
			</div>
			<div class="npl-controls">
				<input
					class="search"
					type="text"
					placeholder="CV key…"
					bind:value={nplKey}
					title="The custom-value name that holds the booking link"
				/>
				<button class="btn-secondary" onclick={runNplScan} disabled={nplLoading}>
					{nplLoading ? 'Scanning…' : 'Scan links'}
				</button>
			</div>
		</div>

		{#if nplError}<div class="error">{nplError}</div>{/if}

		{#if nplScan}
			<div class="npl-summary">
				<span class="pill ok">{nplScan.summary.valid} valid</span>
				<span class="pill warn">{nplScan.summary.domainMismatch} domain mismatch</span>
				<span class="pill bad">{nplScan.summary.malformed} malformed</span>
				<span class="pill bad">{nplScan.summary.missing} missing</span>
				{#if nplScan.summary.errors > 0}<span class="pill bad">{nplScan.summary.errors} error</span>{/if}
				<div class="npl-actions">
					<label class="npl-toggle">
						<input type="checkbox" bind:checked={nplProblemsOnly} />
						Problems only
					</label>
					<button class="btn" onclick={applyNplFixes} disabled={nplApplying || nplFixCount === 0}>
						{nplApplying ? 'Applying…' : `Apply ${nplFixCount} fix(es)`}
					</button>
				</div>
			</div>

			{#if nplVisibleResults().length === 0}
				<div class="info">
					No accounts to show{nplProblemsOnly ? ' — every scanned link is valid 🎉' : ''}.
				</div>
			{:else}
				<div class="npl-list">
					{#each nplVisibleResults() as r (r.locationId)}
						<div class="npl-row">
							<div class="npl-acct">
								<span class="npl-name">{r.name || r.locationId}</span>
								<span class="npl-domain">{r.domain || 'no domain set'}</span>
							</div>
							<span class="pill {NPL_BADGE[r.verdict]?.cls ?? 'warn'}">
								{NPL_BADGE[r.verdict]?.label ?? r.verdict}
							</span>
							{#if nplIsFixable(r)}
								<input
									class="cv-input npl-fix"
									type="text"
									value={nplFixes[r.locationId] ?? ''}
									oninput={(e) => (nplFixes[r.locationId] = e.currentTarget.value)}
									placeholder="corrected https://…"
								/>
							{:else if r.verdict === 'missing'}
								<span class="npl-note">no "{nplScan.key}" value — create it in GHL first</span>
							{:else if r.verdict === 'error'}
								<span class="npl-note">{r.error}</span>
							{:else}
								<span class="npl-value" title={r.value}>{r.value}</span>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
		{/if}
	</section>

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
							<span
								class="changed-dot"
								class:visible={isChanged(loc.locationId, cv)}
								title="Edited — not yet applied"
							></span>
							<button
								class="cv-del"
								title="Delete this custom value"
								aria-label={`Delete ${cv.name}`}
								onclick={() => deleteCV(loc.locationId, cv)}
								disabled={deletingId === cv.id}
							>{deletingId === cv.id ? '…' : '🗑'}</button>
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
		display: grid; grid-template-columns: 260px 1fr 16px 30px; gap: 12px; align-items: center;
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
	.changed-dot { width: 8px; height: 8px; border-radius: 50%; background: transparent; }
	.changed-dot.visible { background: var(--accent); }
	.cv-del {
		width: 28px; height: 28px; border: 1.5px solid var(--border); border-radius: 8px;
		background: transparent; cursor: pointer; font-size: 13px; line-height: 1; padding: 0;
		display: flex; align-items: center; justify-content: center;
	}
	.cv-del:hover { border-color: var(--error); background: rgba(255,59,92,0.06); }
	.cv-del:disabled { opacity: 0.5; cursor: not-allowed; }

	/* New Patient Link scanner */
	.npl {
		background: var(--surface); border: 1.5px solid var(--border);
		border-radius: 14px; padding: 18px; display: flex; flex-direction: column; gap: 12px;
	}
	.npl-head { display: flex; justify-content: space-between; align-items: flex-start; gap: 16px; flex-wrap: wrap; }
	.npl-head h2 { font-size: 16px; font-weight: 700; }
	.npl-sub { font-size: 12px; color: var(--text2); margin-top: 2px; max-width: 520px; }
	.npl-controls { display: flex; gap: 8px; flex-wrap: wrap; }
	.npl-summary { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
	.npl-actions { margin-left: auto; display: flex; align-items: center; gap: 14px; }
	.npl-toggle { font-size: 12px; color: var(--text2); display: flex; align-items: center; gap: 5px; cursor: pointer; }
	.pill { font-size: 11px; font-weight: 700; padding: 4px 10px; border-radius: 999px; white-space: nowrap; }
	.pill.ok { background: rgba(0,201,122,0.12); color: var(--success); }
	.pill.warn { background: rgba(255,170,0,0.16); color: #b76e00; }
	.pill.bad { background: rgba(255,59,92,0.12); color: var(--error); }
	.npl-list { display: flex; flex-direction: column; gap: 4px; }
	.npl-row {
		display: grid; grid-template-columns: 220px 140px 1fr; gap: 12px; align-items: center;
		padding: 7px 8px; border-radius: 8px;
	}
	.npl-row:nth-child(odd) { background: rgba(0,0,0,0.02); }
	.npl-acct { display: flex; flex-direction: column; min-width: 0; }
	.npl-name { font-size: 13px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.npl-domain { font-size: 11px; color: var(--text2); font-family: ui-monospace, monospace; }
	.npl-value {
		font-size: 12px; color: var(--text2); font-family: ui-monospace, monospace;
		overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
	}
	.npl-note { font-size: 12px; color: var(--text2); font-style: italic; }
	.npl-fix { width: 100%; }
</style>
