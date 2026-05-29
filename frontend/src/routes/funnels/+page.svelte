<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { session } from '$lib/stores/session.svelte.js';
	import { apiGet } from '$lib/api/client.js';
	import { accounts, loadLibrary } from '$lib/stores/accounts.svelte.js';
	import { brandPixels, pixelSnippet, brandForDomain, ghlFunnelsURL } from '$lib/data/pixel-snippets.js';

	interface FunnelStep {
		name: string;
		slug: string;
		url: string;
	}
	interface Funnel {
		id: string;
		name: string;
		domain: string;
		steps: FunnelStep[];
		configuredPixel?: string;
		hasTrackingPixel: boolean;
	}
	interface FunnelsResponse {
		locationId: string;
		funnels: Funnel[];
		count: number;
	}
	interface PageAudit {
		funnel: string;
		step: string;
		url: string;
		fetchOk: boolean;
		hasPixel: boolean;
		pixelId: string;
		expectedPixel?: string;
		pixelStatus: 'ok' | 'missing' | 'wrong-pixel' | 'unknown-domain' | 'no-url';
		hasUtm: boolean;
		hasAcuity: boolean;
		error?: string;
	}
	interface AuditSummary {
		funnels: number;
		pages: number;
		ok: number;
		missing: number;
		wrong: number;
		errors: number;
	}
	interface AuditResponse {
		locationId: string;
		pages: PageAudit[];
		summary: AuditSummary;
	}

	interface ListState {
		loading: boolean;
		error: string;
		funnels: Funnel[];
	}
	interface AuditState {
		loading: boolean;
		error: string;
		byUrl: Record<string, PageAudit>;
		summary: AuditSummary | null;
	}

	let lists = $state<Record<string, ListState>>({});
	let audits = $state<Record<string, AuditState>>({});
	let onlySelected = $state(false);
	let refreshing = $state(false);
	let superScanning = $state(false);

	onMount(async () => {
		if (!session.connected) {
			goto('/connect');
			return;
		}
		await loadLibrary();
		await loadAllLists();
	});

	function currentTargets() {
		let list = Object.values(accounts.entries).filter((e) => e.hasToken);
		if (onlySelected && session.selectedIds.size > 0) {
			list = list.filter((e) => session.selectedIds.has(e.locationId));
		}
		return list.sort((a, b) => (a.name || a.locationId).localeCompare(b.name || b.locationId));
	}

	async function loadAllLists() {
		refreshing = true;
		try {
			await Promise.all(currentTargets().map((t) => loadList(t.locationId)));
		} finally {
			refreshing = false;
		}
	}

	async function loadList(id: string) {
		lists[id] = { loading: true, error: '', funnels: [] };
		try {
			const data = await apiGet<FunnelsResponse>(`/api/funnels/${encodeURIComponent(id)}`);
			lists[id] = { loading: false, error: '', funnels: data.funnels ?? [] };
		} catch (e) {
			lists[id] = {
				loading: false,
				error: e instanceof Error ? e.message : 'Failed to load funnels',
				funnels: []
			};
		}
	}

	async function scanAccount(id: string) {
		audits[id] = { loading: true, error: '', byUrl: {}, summary: null };
		try {
			const data = await apiGet<AuditResponse>(`/api/funnels/${encodeURIComponent(id)}/audit`);
			const byUrl: Record<string, PageAudit> = {};
			for (const p of data.pages) byUrl[p.url] = p;
			audits[id] = { loading: false, error: '', byUrl, summary: data.summary };
		} catch (e) {
			audits[id] = {
				loading: false,
				error: e instanceof Error ? e.message : 'Scan failed',
				byUrl: {},
				summary: null
			};
		}
	}

	async function superScan() {
		superScanning = true;
		try {
			await Promise.all(currentTargets().map((t) => scanAccount(t.locationId)));
		} finally {
			superScanning = false;
		}
	}

	function badgeLabel(p: PageAudit | undefined): string {
		if (!p) return '';
		switch (p.pixelStatus) {
			case 'ok':
				return `✓ pixel ${p.pixelId}`;
			case 'wrong-pixel':
				return `⚠ wrong: ${p.pixelId} (expected ${p.expectedPixel})`;
			case 'missing':
				return p.error ? `✗ ${p.error}` : '✗ no pixel';
			case 'unknown-domain':
				return p.hasPixel ? `pixel ${p.pixelId} (no expected set)` : 'no pixel (domain not configured)';
			case 'no-url':
				return 'no URL';
			default:
				return p.pixelStatus;
		}
	}

	const totalFunnels = $derived(
		Object.values(lists).reduce((sum, s) => sum + s.funnels.length, 0)
	);

	// --- Assisted-manual pixel fix (Phase 2e-2): hand the operator the snippet ---
	let fixOpen = $state<Record<string, boolean>>({});
	let fixBrand = $state<Record<string, string>>({}); // funnelId -> selected pixelId
	let copiedId = $state('');

	function toggleFix(funnel: Funnel) {
		const open = !fixOpen[funnel.id];
		fixOpen = { ...fixOpen, [funnel.id]: open };
		if (open && !fixBrand[funnel.id]) {
			const auto = brandForDomain(funnel.domain);
			if (auto) fixBrand = { ...fixBrand, [funnel.id]: auto.pixelId };
		}
	}

	async function copySnippet(funnelId: string, pixelId: string) {
		try {
			await navigator.clipboard.writeText(pixelSnippet(pixelId));
			copiedId = funnelId;
			setTimeout(() => {
				if (copiedId === funnelId) copiedId = '';
			}, 1600);
		} catch {
			copiedId = '';
		}
	}
</script>

<div class="page">
	<header class="head">
		<div>
			<h1>Sites &amp; Funnels</h1>
			<p class="sub">
				{currentTargets().length} account(s) · {totalFunnels} funnel(s) · pixel audit is read-only
			</p>
		</div>
		<div class="toolbar">
			<label class="toggle">
				<input type="checkbox" bind:checked={onlySelected} onchange={loadAllLists} />
				<span>Selected only</span>
			</label>
			<button class="btn-secondary" onclick={loadAllLists} disabled={refreshing}>
				{refreshing ? 'Refreshing…' : 'Refresh funnels'}
			</button>
			<button class="btn-primary" onclick={superScan} disabled={superScanning}>
				{superScanning ? 'Scanning…' : 'Super Scan all'}
			</button>
		</div>
	</header>

	<p class="note">
		Pixel injection (writing tracking code back to GHL) is not available via the public API — this
		page audits which pages are <strong>missing</strong> or have the <strong>wrong</strong> pixel.
		Fix flagged pages in GHL’s funnel settings.
	</p>

	{#if currentTargets().length === 0}
		<div class="empty">
			No accounts with tokens yet. Go to <a href="/accounts">Accounts</a> and save a sub-account token
			to load its funnels.
		</div>
	{/if}

	{#each currentTargets() as target (target.locationId)}
		{@const ls = lists[target.locationId]}
		{@const as = audits[target.locationId]}
		<section class="account">
			<div class="account-head">
				<h2>{target.name || target.locationId}</h2>
				<span class="loc-id">{target.locationId}</span>
				{#if as?.summary}
					<span class="sum-badges">
						<span class="sb ok">{as.summary.ok} ok</span>
						<span class="sb miss">{as.summary.missing} missing</span>
						<span class="sb wrong">{as.summary.wrong} wrong</span>
						{#if as.summary.errors > 0}<span class="sb err">{as.summary.errors} err</span>{/if}
					</span>
				{/if}
				<button
					class="btn-secondary scan-btn"
					onclick={() => scanAccount(target.locationId)}
					disabled={as?.loading}
				>
					{as?.loading ? 'Scanning…' : 'Scan pixels'}
				</button>
			</div>

			{#if !ls || ls.loading}
				<div class="info">Loading funnels…</div>
			{:else if ls.error}
				<div class="error">{ls.error}</div>
			{:else if ls.funnels.length === 0}
				<div class="info">No funnels for this account.</div>
			{:else}
				{#if as?.error}<div class="error">{as.error}</div>{/if}
				{#each ls.funnels as funnel (funnel.id)}
					<div class="funnel">
						<div class="funnel-head">
							<span class="funnel-name">{funnel.name}</span>
							<span class="funnel-domain">{funnel.domain || 'no domain'}</span>
							{#if funnel.hasTrackingPixel}
								<span class="track-badge ok">tracking code: pixel {funnel.configuredPixel || '—'}</span>
							{:else}
								<span class="track-badge none">no pixel in tracking code</span>
							{/if}
							<span class="step-count">{funnel.steps.length} page(s)</span>
						</div>
						{#if funnel.steps.length > 0}
							<ul class="steps">
								{#each funnel.steps as step (step.url || step.name)}
									{@const audit = as?.byUrl?.[step.url]}
									<li class="step">
										<span class="step-name">{step.name}</span>
										{#if step.url}
											<a class="step-url" href={step.url} target="_blank" rel="noopener noreferrer">
												{step.url}
											</a>
										{:else}
											<span class="step-url muted">no URL (no domain/slug)</span>
										{/if}
										{#if audit}
											<span class="pixel-badge {audit.pixelStatus}">{badgeLabel(audit)}</span>
											{#if audit.fetchOk}
												<span class="mini" class:on={audit.hasUtm}>UTM</span>
												<span class="mini" class:on={audit.hasAcuity}>Acuity</span>
											{/if}
										{/if}
									</li>
								{/each}
							</ul>
						{/if}

						{#if !funnel.hasTrackingPixel}
							<div class="fix">
								<button class="fix-toggle" onclick={() => toggleFix(funnel)}>
									{fixOpen[funnel.id] ? '▾' : '▸'} Add Meta pixel to this funnel
								</button>
								{#if fixOpen[funnel.id]}
									<div class="fix-body">
										<div class="brand-row">
											<span class="fix-label">Brand:</span>
											{#each brandPixels as bp (bp.pixelId)}
												<button
													class="brand-btn"
													class:sel={fixBrand[funnel.id] === bp.pixelId}
													onclick={() => (fixBrand = { ...fixBrand, [funnel.id]: bp.pixelId })}
												>
													{bp.brand}
												</button>
											{/each}
										</div>
										{#if fixBrand[funnel.id]}
											<textarea class="snippet" readonly rows="6"
												>{pixelSnippet(fixBrand[funnel.id])}</textarea
											>
											<div class="fix-actions">
												<button class="btn-primary sm" onclick={() => copySnippet(funnel.id, fixBrand[funnel.id])}>
													{copiedId === funnel.id ? 'Copied!' : 'Copy snippet'}
												</button>
												<a class="ghl-link" href={ghlFunnelsURL(target.locationId)} target="_blank" rel="noopener noreferrer">
													Open funnels in GHL ↗
												</a>
											</div>
											<p class="fix-hint">
												Paste into GHL → Funnels → {funnel.name} → Settings → Tracking Code (Head), then Save.
											</p>
										{/if}
									</div>
								{/if}
							</div>
						{/if}
					</div>
				{/each}
			{/if}
		</section>
	{/each}
</div>

<style>
	.page { display: flex; flex-direction: column; gap: 16px; }

	.head { display: flex; justify-content: space-between; align-items: flex-end; gap: 16px; flex-wrap: wrap; }
	h1 { font-size: 24px; font-weight: 700; }
	.sub { font-size: 13px; color: var(--text2); margin-top: 4px; }

	.toolbar { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
	.toggle { display: flex; align-items: center; gap: 6px; font-size: 13px; color: var(--text2); cursor: pointer; }
	.toggle input { width: 15px; height: 15px; }

	.btn-secondary, .btn-primary {
		padding: 8px 14px; border-radius: 8px; font-family: inherit; font-size: 13px;
		font-weight: 600; cursor: pointer;
	}
	.btn-secondary { background: rgba(0,0,0,0.04); color: var(--text2); border: 1.5px solid var(--border); }
	.btn-secondary:hover { background: rgba(0,0,0,0.08); }
	.btn-primary { background: var(--accent); color: #fff; border: none; }
	.btn-secondary:disabled, .btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }
	.scan-btn { margin-left: auto; }

	.note {
		font-size: 12.5px; color: var(--text2); line-height: 1.5;
		background: rgba(255,29,141,0.05); border: 1.5px solid rgba(255,29,141,0.12);
		border-radius: 10px; padding: 10px 14px;
	}

	.info, .error, .empty { padding: 12px 16px; border-radius: 10px; font-size: 13px; }
	.info { background: rgba(0,0,0,0.04); color: var(--text2); }
	.error { background: rgba(255,59,92,0.08); color: var(--error); font-weight: 600; }
	.empty { color: var(--text2); padding: 32px; text-align: center; }
	.empty a { color: var(--accent); font-weight: 600; }

	.account {
		background: var(--surface); border: 1.5px solid var(--border); border-radius: 14px;
		padding: 18px; display: flex; flex-direction: column; gap: 12px;
	}
	.account-head { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
	.account-head h2 { font-size: 16px; font-weight: 700; }
	.loc-id {
		font-family: ui-monospace, monospace; font-size: 11px; color: var(--text2);
		padding: 3px 8px; background: rgba(0,0,0,0.04); border-radius: 6px;
	}
	.sum-badges { display: flex; gap: 6px; flex-wrap: wrap; }
	.sb { font-size: 11px; font-weight: 700; padding: 2px 8px; border-radius: 20px; }
	.sb.ok { background: rgba(0,201,122,0.12); color: var(--success); }
	.sb.miss { background: rgba(255,59,92,0.1); color: var(--error); }
	.sb.wrong { background: rgba(255,149,0,0.14); color: #b25e00; }
	.sb.err { background: rgba(0,0,0,0.06); color: var(--text2); }

	.funnel { border: 1px solid var(--border); border-radius: 10px; padding: 12px 14px; }
	.funnel-head { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
	.funnel-name { font-weight: 700; font-size: 14px; }
	.funnel-domain { font-size: 12px; color: var(--text2); font-family: ui-monospace, monospace; }
	.step-count { font-size: 11px; color: var(--text2); margin-left: auto; }

	.track-badge { font-size: 11px; font-weight: 700; padding: 2px 8px; border-radius: 20px; }
	.track-badge.ok { background: rgba(0,201,122,0.12); color: var(--success); }
	.track-badge.none { background: rgba(255,149,0,0.14); color: #b25e00; }

	.steps { list-style: none; margin-top: 8px; display: flex; flex-direction: column; gap: 4px; }
	.step {
		display: flex; align-items: center; gap: 8px; flex-wrap: wrap;
		padding: 6px 8px; border-radius: 8px; font-size: 12.5px;
	}
	.step:nth-child(odd) { background: rgba(0,0,0,0.02); }
	.step-name { font-weight: 600; min-width: 120px; }
	.step-url { color: var(--accent); text-decoration: none; font-size: 11.5px; word-break: break-all; }
	.step-url:hover { text-decoration: underline; }
	.step-url.muted { color: var(--text2); }

	.pixel-badge { margin-left: auto; font-size: 11px; font-weight: 700; padding: 2px 8px; border-radius: 20px; }
	.pixel-badge.ok { background: rgba(0,201,122,0.12); color: var(--success); }
	.pixel-badge.missing { background: rgba(255,59,92,0.1); color: var(--error); }
	.pixel-badge.wrong-pixel { background: rgba(255,149,0,0.14); color: #b25e00; }
	.pixel-badge.unknown-domain, .pixel-badge.no-url { background: rgba(0,0,0,0.06); color: var(--text2); }

	.mini {
		font-size: 10px; font-weight: 700; padding: 1px 6px; border-radius: 6px;
		background: rgba(0,0,0,0.06); color: var(--text2); opacity: 0.5;
	}
	.mini.on { background: rgba(0,201,122,0.12); color: var(--success); opacity: 1; }

	.fix { margin-top: 10px; border-top: 1px dashed var(--border); padding-top: 8px; }
	.fix-toggle {
		background: none; border: none; cursor: pointer; font-family: inherit;
		font-size: 12px; font-weight: 700; color: var(--accent); padding: 2px 0;
	}
	.fix-body { display: flex; flex-direction: column; gap: 8px; margin-top: 8px; }
	.brand-row { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
	.fix-label { font-size: 12px; color: var(--text2); font-weight: 600; }
	.brand-btn {
		font-family: inherit; font-size: 12px; padding: 4px 10px; border-radius: 20px;
		border: 1.5px solid var(--border); background: var(--surface); color: var(--text2); cursor: pointer;
	}
	.brand-btn.sel { background: var(--accent); color: #fff; border-color: var(--accent); }
	.snippet {
		width: 100%; font-family: ui-monospace, monospace; font-size: 11px; line-height: 1.4;
		border: 1.5px solid var(--border); border-radius: 8px; padding: 8px 10px; resize: vertical;
		background: rgba(0,0,0,0.02); color: #1a1d2e;
	}
	.fix-actions { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
	.btn-primary.sm { padding: 6px 14px; font-size: 12px; }
	.ghl-link { font-size: 12px; font-weight: 600; color: var(--accent); text-decoration: none; }
	.ghl-link:hover { text-decoration: underline; }
	.fix-hint { font-size: 11.5px; color: var(--text2); line-height: 1.4; }
</style>
