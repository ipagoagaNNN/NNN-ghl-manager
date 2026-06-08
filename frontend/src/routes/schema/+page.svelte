<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { session } from '$lib/stores/session.svelte.js';
	import { apiGet } from '$lib/api/client.js';
	import { accounts, loadLibrary } from '$lib/stores/accounts.svelte.js';

	interface GhlObject {
		id: string;
		key: string;
		labels?: Record<string, string>;
		description?: string;
		primaryDisplayProperty?: string;
		type?: string;
	}
	interface CustomFieldItem {
		id: string;
		name: string;
		dataType?: string;
		fieldKey?: string;
		position?: number;
	}
	interface ObjectsResponse {
		locationId: string;
		objects: GhlObject[];
		count: number;
	}
	interface CustomFieldsResponse {
		locationId: string;
		customFields: CustomFieldItem[];
		count: number;
	}

	let selectedLoc = $state('');
	let objects = $state<GhlObject[]>([]);
	let fields = $state<CustomFieldItem[]>([]);
	let loading = $state(false);
	let error = $state('');
	let loaded = $state(false);

	const tokenAccounts = $derived(Object.values(accounts.entries).filter((e) => e.hasToken));

	onMount(async () => {
		if (!session.connected) {
			goto('/connect');
			return;
		}
		await loadLibrary();
		const first = Object.values(accounts.entries).find((e) => e.hasToken);
		if (first) {
			selectedLoc = first.locationId;
			await load();
		}
	});

	async function load() {
		if (!selectedLoc) return;
		loading = true;
		error = '';
		objects = [];
		fields = [];
		try {
			// Objects use Version 2023-02-21, custom fields use 2021-04-15 — both
			// resolved server-side (store.GHLVersionFor); the client just calls the API.
			const [o, f] = await Promise.all([
				apiGet<ObjectsResponse>(`/api/objects/${encodeURIComponent(selectedLoc)}`),
				apiGet<CustomFieldsResponse>(`/api/customfields/${encodeURIComponent(selectedLoc)}`)
			]);
			objects = o.objects ?? [];
			fields = f.customFields ?? [];
			loaded = true;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load schema';
		} finally {
			loading = false;
		}
	}

	function objLabel(o: GhlObject): string {
		return o.labels?.singular || o.labels?.plural || o.key;
	}
</script>

<div class="page">
	<header class="head">
		<div>
			<h1>Schema</h1>
			<p class="sub">
				Object schema (business / opportunity / contact + any custom objects) and custom-field schema
				for a sub-account. Read-only.
			</p>
		</div>
		<div class="toolbar">
			<select class="search" bind:value={selectedLoc} onchange={load}>
				{#each tokenAccounts as a (a.locationId)}
					<option value={a.locationId}>{a.name || a.locationId}</option>
				{/each}
			</select>
			<button class="btn-secondary" onclick={load} disabled={loading || !selectedLoc}>
				{loading ? 'Loading…' : 'Reload'}
			</button>
		</div>
	</header>

	{#if error}<div class="error">{error}</div>{/if}

	{#if tokenAccounts.length === 0}
		<div class="empty">No accounts with tokens. Go to <a href="/accounts">Accounts</a> to save one.</div>
	{:else}
		<section class="card">
			<h2>Objects <span class="count">{objects.length}</span></h2>
			{#if loaded && objects.length === 0}
				<div class="info">No objects returned for this account.</div>
			{:else}
				<div class="grid">
					{#each objects as o (o.id)}
						<div class="row">
							<span class="key">{o.key}</span>
							<span class="label">{objLabel(o)}</span>
							<span class="muted">{o.type || ''}</span>
						</div>
					{/each}
				</div>
			{/if}
		</section>

		<section class="card">
			<h2>Custom Fields <span class="count">{fields.length}</span></h2>
			{#if loaded && fields.length === 0}
				<div class="info">No custom fields returned for this account.</div>
			{:else}
				<div class="grid">
					{#each fields as f (f.id)}
						<div class="row">
							<span class="label">{f.name}</span>
							<span class="muted">{f.dataType || ''}</span>
							<span class="key">{f.fieldKey || ''}</span>
						</div>
					{/each}
				</div>
			{/if}
		</section>
	{/if}
</div>

<style>
	.page { display: flex; flex-direction: column; gap: 16px; }
	.head { display: flex; justify-content: space-between; align-items: flex-end; gap: 16px; flex-wrap: wrap; }
	h1 { font-size: 24px; font-weight: 700; }
	.sub { font-size: 13px; color: var(--text2); margin-top: 4px; max-width: 560px; }
	.toolbar { display: flex; gap: 8px; flex-wrap: wrap; }
	.search {
		padding: 8px 12px; border: 1.5px solid var(--border); border-radius: 8px;
		font-family: inherit; font-size: 13px; min-width: 200px; background: #fff;
	}
	.search:focus { outline: none; border-color: var(--accent); }
	.btn-secondary {
		padding: 8px 14px; border-radius: 8px; font-family: inherit; font-size: 13px; font-weight: 600;
		cursor: pointer; background: rgba(0,0,0,0.04); color: var(--text2); border: 1.5px solid var(--border);
	}
	.btn-secondary:disabled { opacity: 0.6; cursor: not-allowed; }
	.card {
		background: var(--surface); border: 1.5px solid var(--border);
		border-radius: 14px; padding: 18px; display: flex; flex-direction: column; gap: 10px;
	}
	.card h2 { font-size: 16px; font-weight: 700; display: flex; align-items: center; gap: 8px; }
	.count {
		font-size: 11px; font-weight: 700; color: var(--text2);
		background: rgba(0,0,0,0.05); padding: 2px 8px; border-radius: 999px;
	}
	.grid { display: flex; flex-direction: column; gap: 4px; }
	.row {
		display: grid; grid-template-columns: 200px 1fr 160px; gap: 12px; align-items: center;
		padding: 6px 8px; border-radius: 8px;
	}
	.row:nth-child(odd) { background: rgba(0,0,0,0.02); }
	.key { font-family: ui-monospace, monospace; font-size: 12px; color: var(--accent); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.label { font-size: 13px; font-weight: 600; }
	.muted { font-size: 12px; color: var(--text2); }
	.info, .error, .empty { padding: 12px 16px; border-radius: 10px; font-size: 13px; }
	.info { background: rgba(0,0,0,0.04); color: var(--text2); }
	.error { background: rgba(255,59,92,0.08); color: var(--error); font-weight: 600; }
	.empty { color: var(--text2); padding: 32px; text-align: center; }
	.empty a { color: var(--accent); font-weight: 600; }
</style>
