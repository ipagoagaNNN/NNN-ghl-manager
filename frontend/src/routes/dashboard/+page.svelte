<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { session } from '$lib/stores/session.svelte.js';
	import { apiGet } from '$lib/api/client.js';
	import { accounts, loadLibrary } from '$lib/stores/accounts.svelte.js';

	interface DayCount {
		date: string;
		count: number;
	}
	interface SourceCount {
		source: string;
		count: number;
	}
	interface DashboardData {
		locationId: string;
		total: number;
		leadsByDay: DayCount[];
		topSources: SourceCount[];
		fetchedAt: string;
		truncated: boolean;
	}

	let startDate = $state('');
	let endDate = $state('');
	let loading = $state(false);
	let error = $state('');
	let perAccount = $state<{ locationId: string; name: string; total: number; error?: string }[]>([]);
	// Merged aggregate across all fetched accounts.
	let mergedByDay = $state<DayCount[]>([]);
	let mergedSources = $state<SourceCount[]>([]);
	let grandTotal = $state(0);
	let anyTruncated = $state(false);

	onMount(async () => {
		if (!session.connected) {
			goto('/connect');
			return;
		}
		await loadLibrary();
		await loadDashboard();
	});

	function targets() {
		let list = Object.values(accounts.entries).filter((e) => e.hasToken);
		if (session.selectedIds.size > 0) {
			list = list.filter((e) => session.selectedIds.has(e.locationId));
		}
		return list;
	}

	async function loadDashboard() {
		const ts = targets();
		if (ts.length === 0) {
			perAccount = [];
			mergedByDay = [];
			mergedSources = [];
			grandTotal = 0;
			return;
		}
		loading = true;
		error = '';
		try {
			const qs = new URLSearchParams();
			if (startDate) qs.set('startDate', new Date(startDate).toISOString());
			if (endDate) qs.set('endDate', new Date(endDate).toISOString());
			const suffix = qs.toString() ? `?${qs.toString()}` : '';

			const settled = await Promise.allSettled(
				ts.map((t) =>
					apiGet<DashboardData>(`/api/dashboard/${encodeURIComponent(t.locationId)}/contacts${suffix}`)
				)
			);

			const dayMap = new Map<string, number>();
			const srcMap = new Map<string, number>();
			let gt = 0;
			let trunc = false;
			const pa: typeof perAccount = [];

			settled.forEach((res, i) => {
				const t = ts[i];
				if (res.status === 'fulfilled') {
					const d = res.value;
					pa.push({ locationId: t.locationId, name: t.name || t.locationId, total: d.total });
					gt += d.total;
					if (d.truncated) trunc = true;
					for (const dc of d.leadsByDay) dayMap.set(dc.date, (dayMap.get(dc.date) ?? 0) + dc.count);
					for (const sc of d.topSources) srcMap.set(sc.source, (srcMap.get(sc.source) ?? 0) + sc.count);
				} else {
					const msg = res.reason instanceof Error ? res.reason.message : 'fetch failed';
					pa.push({ locationId: t.locationId, name: t.name || t.locationId, total: 0, error: msg });
				}
			});

			perAccount = pa;
			grandTotal = gt;
			anyTruncated = trunc;
			mergedByDay = [...dayMap.entries()]
				.map(([date, count]) => ({ date, count }))
				.sort((a, b) => a.date.localeCompare(b.date));
			mergedSources = [...srcMap.entries()]
				.map(([source, count]) => ({ source, count }))
				.sort((a, b) => b.count - a.count)
				.slice(0, 10);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load dashboard';
		} finally {
			loading = false;
		}
	}

	// SVG bar-chart geometry (no charting dependency).
	const chartW = 880;
	const chartH = 220;
	const padL = 36;
	const padB = 28;
	const maxCount = $derived(Math.max(1, ...mergedByDay.map((d) => d.count)));
	const barGap = 2;
	const barW = $derived(
		mergedByDay.length > 0
			? Math.max(1, (chartW - padL - 10 - barGap * mergedByDay.length) / mergedByDay.length)
			: 0
	);

	function barX(i: number): number {
		return padL + i * (barW + barGap);
	}
	function barH(count: number): number {
		return ((chartH - padB - 10) * count) / maxCount;
	}
	function shortDate(d: string): string {
		return d.length >= 10 ? d.slice(5) : d; // MM-DD
	}
</script>

<div class="page">
	<header class="head">
		<div>
			<h1>Dashboard</h1>
			<p class="sub">
				{perAccount.length} account(s) · {grandTotal} lead(s){anyTruncated ? ' · capped at 5k/acct' : ''}
			</p>
		</div>
		<div class="toolbar">
			<label class="date-field">
				<span>From</span>
				<input type="date" bind:value={startDate} />
			</label>
			<label class="date-field">
				<span>To</span>
				<input type="date" bind:value={endDate} />
			</label>
			<button class="btn" onclick={loadDashboard} disabled={loading}>
				{loading ? 'Loading…' : 'Load'}
			</button>
		</div>
	</header>

	{#if error}<div class="error">{error}</div>{/if}

	{#if perAccount.length === 0 && !loading}
		<div class="empty">
			No accounts with tokens (or none selected). Go to <a href="/accounts">Accounts</a> to save tokens.
		</div>
	{/if}

	{#if grandTotal > 0}
		<section class="card">
			<h2>Leads by day</h2>
			<svg class="chart" viewBox={`0 0 ${chartW} ${chartH}`} preserveAspectRatio="xMidYMid meet" role="img" aria-label="Leads by day bar chart">
				<!-- baseline -->
				<line x1={padL} y1={chartH - padB} x2={chartW - 6} y2={chartH - padB} stroke="var(--border)" stroke-width="1" />
				<text x="4" y={chartH - padB} font-size="10" fill="var(--text2)">0</text>
				<text x="4" y="14" font-size="10" fill="var(--text2)">{maxCount}</text>
				{#each mergedByDay as d, i (d.date)}
					<rect
						x={barX(i)}
						y={chartH - padB - barH(d.count)}
						width={barW}
						height={barH(d.count)}
						rx={barW > 4 ? 2 : 0}
						fill="var(--accent)"
					>
						<title>{d.date}: {d.count}</title>
					</rect>
					{#if mergedByDay.length <= 31}
						<text
							x={barX(i) + barW / 2}
							y={chartH - padB + 12}
							font-size="9"
							fill="var(--text2)"
							text-anchor="middle"
						>{shortDate(d.date)}</text>
					{/if}
				{/each}
			</svg>
		</section>

		<div class="two-col">
			<section class="card">
				<h2>Top sources</h2>
				<table class="tbl">
					<thead><tr><th>Source</th><th>Leads</th></tr></thead>
					<tbody>
						{#each mergedSources as s (s.source)}
							<tr>
								<td>{s.source}</td>
								<td class="num">{s.count}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</section>

			<section class="card">
				<h2>By account</h2>
				<table class="tbl">
					<thead><tr><th>Account</th><th>Leads</th></tr></thead>
					<tbody>
						{#each perAccount as a (a.locationId)}
							<tr>
								<td>{a.name}{#if a.error}<span class="row-err"> · {a.error}</span>{/if}</td>
								<td class="num">{a.total}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</section>
		</div>
	{/if}
</div>

<style>
	.page { display: flex; flex-direction: column; gap: 16px; }

	.head { display: flex; justify-content: space-between; align-items: flex-end; gap: 16px; flex-wrap: wrap; }
	h1 { font-size: 24px; font-weight: 700; }
	.sub { font-size: 13px; color: var(--text2); margin-top: 4px; }

	.toolbar { display: flex; gap: 10px; align-items: flex-end; flex-wrap: wrap; }
	.date-field { display: flex; flex-direction: column; gap: 4px; }
	.date-field > span { font-size: 11px; font-weight: 600; color: var(--text2); text-transform: uppercase; letter-spacing: 0.4px; }
	.date-field input {
		padding: 7px 10px; border: 1.5px solid var(--border); border-radius: 8px;
		font-family: inherit; font-size: 13px;
	}
	.date-field input:focus { outline: none; border-color: var(--accent); }

	.btn {
		padding: 8px 16px; border-radius: 8px; font-family: inherit; font-size: 13px;
		font-weight: 600; cursor: pointer; border: none; background: var(--accent); color: #fff;
	}
	.btn:disabled { opacity: 0.5; cursor: not-allowed; }

	.error, .empty { padding: 12px 16px; border-radius: 10px; font-size: 13px; }
	.error { background: rgba(255,59,92,0.08); color: var(--error); font-weight: 600; }
	.empty { color: var(--text2); padding: 32px; text-align: center; }
	.empty a { color: var(--accent); font-weight: 600; }

	.card {
		background: var(--surface); border: 1.5px solid var(--border);
		border-radius: 14px; padding: 18px; display: flex; flex-direction: column; gap: 12px;
	}
	.card h2 { font-size: 15px; font-weight: 700; }
	.chart { width: 100%; height: auto; }

	.two-col { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }

	.tbl { width: 100%; border-collapse: collapse; font-size: 13px; }
	.tbl th {
		text-align: left; padding: 8px 10px; font-size: 11px; text-transform: uppercase;
		letter-spacing: 0.4px; color: var(--text2); border-bottom: 1.5px solid var(--border);
	}
	.tbl td { padding: 9px 10px; border-bottom: 1px solid var(--border); }
	.tbl tr:last-child td { border-bottom: none; }
	.num { text-align: right; font-variant-numeric: tabular-nums; font-weight: 600; }
	.row-err { color: var(--error); font-size: 11px; }

	@media (max-width: 720px) { .two-col { grid-template-columns: 1fr; } }
</style>
