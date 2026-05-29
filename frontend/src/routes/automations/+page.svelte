<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { session } from '$lib/stores/session.svelte.js';
	import { apiGet } from '$lib/api/client.js';
	import { accounts, loadLibrary } from '$lib/stores/accounts.svelte.js';

	interface Workflow {
		id: string;
		name: string;
		status: string;
		version: number;
		createdAt: string;
		updatedAt: string;
	}
	interface WorkflowsResponse {
		locationId: string;
		workflows: Workflow[];
		count: number;
	}
	interface FetchState {
		loading: boolean;
		error: string;
		workflows: Workflow[];
	}

	// Per-location fetch state, keyed by locationId.
	let results = $state<Record<string, FetchState>>({});
	let search = $state('');
	let onlySelected = $state(false);
	let refreshing = $state(false);

	onMount(async () => {
		if (!session.connected) {
			goto('/connect');
			return;
		}
		await loadLibrary();
		await loadAll();
	});

	// Locations that have a token + optional "selected only" filter.
	// (Workflows require a sub-account token — accounts without one are skipped.)
	function currentTargets() {
		let list = Object.values(accounts.entries).filter((e) => e.hasToken);
		if (onlySelected && session.selectedIds.size > 0) {
			list = list.filter((e) => session.selectedIds.has(e.locationId));
		}
		return list.sort((a, b) => (a.name || a.locationId).localeCompare(b.name || b.locationId));
	}

	async function loadAll() {
		refreshing = true;
		try {
			await Promise.all(currentTargets().map((t) => loadOne(t.locationId)));
		} finally {
			refreshing = false;
		}
	}

	async function loadOne(id: string) {
		results[id] = { loading: true, error: '', workflows: [] };
		try {
			const data = await apiGet<WorkflowsResponse>(`/api/workflows/${encodeURIComponent(id)}`);
			results[id] = { loading: false, error: '', workflows: data.workflows ?? [] };
		} catch (e) {
			results[id] = {
				loading: false,
				error: e instanceof Error ? e.message : 'Failed to load workflows',
				workflows: [],
			};
		}
	}

	function filteredWorkflows(list: Workflow[]): Workflow[] {
		const q = search.trim().toLowerCase();
		if (!q) return list;
		return list.filter(
			(wf) => wf.name.toLowerCase().includes(q) || wf.status.toLowerCase().includes(q)
		);
	}

	function fmtDate(iso: string): string {
		if (!iso) return '—';
		const d = new Date(iso);
		if (isNaN(d.getTime())) return iso;
		return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
	}

	const totalWorkflows = $derived(
		Object.values(results).reduce((sum, r) => sum + r.workflows.length, 0)
	);
</script>

<div class="page">
	<header class="head">
		<div>
			<h1>Automations</h1>
			<p class="sub">
				{currentTargets().length} account(s) · {totalWorkflows} workflow(s) · read-only
			</p>
		</div>
		<div class="toolbar">
			<input class="search" type="text" placeholder="Search workflow name or status…" bind:value={search} />
			<label class="toggle">
				<input type="checkbox" bind:checked={onlySelected} onchange={loadAll} />
				<span>Selected only</span>
			</label>
			<button class="btn-secondary" onclick={loadAll} disabled={refreshing}>
				{refreshing ? 'Refreshing…' : 'Refresh'}
			</button>
		</div>
	</header>

	{#if currentTargets().length === 0}
		<div class="empty">
			No accounts with tokens yet. Go to <a href="/accounts">Accounts</a> and save a sub-account token
			to load its workflows.
		</div>
	{/if}

	{#each currentTargets() as target (target.locationId)}
		{@const state = results[target.locationId]}
		<section class="account">
			<div class="account-head">
				<h2>{target.name || target.locationId}</h2>
				<span class="loc-id">{target.locationId}</span>
				{#if state && !state.loading && !state.error}
					<span class="count-badge">{filteredWorkflows(state.workflows).length} shown</span>
				{/if}
			</div>

			{#if !state || state.loading}
				<div class="info">Loading workflows…</div>
			{:else if state.error}
				<div class="error">{state.error}</div>
			{:else if state.workflows.length === 0}
				<div class="info">No workflows for this account.</div>
			{:else}
				{@const rows = filteredWorkflows(state.workflows)}
				{#if rows.length === 0}
					<div class="info">No workflows match "{search}".</div>
				{:else}
					<table class="wf-table">
						<thead>
							<tr>
								<th>Name</th>
								<th>Status</th>
								<th>Version</th>
								<th>Updated</th>
							</tr>
						</thead>
						<tbody>
							{#each rows as wf (wf.id)}
								<tr>
									<td class="wf-name">{wf.name}</td>
									<td>
										<span class="status-badge" class:published={wf.status === 'published'}>
											{wf.status || 'unknown'}
										</span>
									</td>
									<td>v{wf.version}</td>
									<td class="wf-date">{fmtDate(wf.updatedAt)}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{/if}
			{/if}
		</section>
	{/each}
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

	.toolbar { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
	.search {
		padding: 8px 12px;
		border: 1.5px solid var(--border);
		border-radius: 8px;
		font-family: inherit;
		font-size: 13px;
		min-width: 260px;
	}
	.search:focus { outline: none; border-color: var(--accent); }
	.toggle { display: flex; align-items: center; gap: 6px; font-size: 13px; color: var(--text2); cursor: pointer; }
	.toggle input { width: 15px; height: 15px; }

	.btn-secondary {
		padding: 8px 14px;
		border-radius: 8px;
		font-family: inherit;
		font-size: 13px;
		font-weight: 600;
		cursor: pointer;
		background: rgba(0,0,0,0.04);
		color: var(--text2);
		border: 1.5px solid var(--border);
	}
	.btn-secondary:hover { background: rgba(0,0,0,0.08); }
	.btn-secondary:disabled { opacity: 0.6; cursor: not-allowed; }

	.info, .error, .empty {
		padding: 12px 16px;
		border-radius: 10px;
		font-size: 13px;
	}
	.info { background: rgba(0,0,0,0.04); color: var(--text2); }
	.error { background: rgba(255,59,92,0.08); color: var(--error); font-weight: 600; }
	.empty { color: var(--text2); padding: 32px; text-align: center; }
	.empty a { color: var(--accent); font-weight: 600; }

	.account {
		background: var(--surface);
		border: 1.5px solid var(--border);
		border-radius: 14px;
		padding: 18px;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.account-head { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
	.account-head h2 { font-size: 16px; font-weight: 700; }
	.loc-id {
		font-family: ui-monospace, monospace;
		font-size: 11px;
		color: var(--text2);
		padding: 3px 8px;
		background: rgba(0,0,0,0.04);
		border-radius: 6px;
	}
	.count-badge {
		font-size: 11px;
		font-weight: 600;
		color: var(--text2);
		margin-left: auto;
	}

	.wf-table { width: 100%; border-collapse: collapse; font-size: 13px; }
	.wf-table th {
		text-align: left;
		padding: 8px 10px;
		font-size: 11px;
		text-transform: uppercase;
		letter-spacing: 0.4px;
		color: var(--text2);
		border-bottom: 1.5px solid var(--border);
	}
	.wf-table td { padding: 10px; border-bottom: 1px solid var(--border); }
	.wf-table tr:last-child td { border-bottom: none; }
	.wf-name { font-weight: 600; }
	.wf-date { color: var(--text2); }

	.status-badge {
		padding: 2px 10px;
		border-radius: 20px;
		font-size: 11px;
		font-weight: 600;
		background: rgba(0,0,0,0.06);
		color: var(--text2);
		text-transform: capitalize;
	}
	.status-badge.published { background: rgba(0,201,122,0.12); color: var(--success); }
</style>
