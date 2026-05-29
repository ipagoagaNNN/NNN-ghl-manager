<script lang="ts">
	import { goto } from '$app/navigation';
	import { apiPost } from '$lib/api/client.js';
	import { session } from '$lib/stores/session.svelte.js';

	// --- Agency mode (agency-level PIT → discover all sub-accounts) ---
	let agencyToken = $state('');
	let companyId = $state('');
	let agencyError = $state('');
	let agencyLoading = $state(false);

	async function connectAgency() {
		if (!agencyToken.trim() || !companyId.trim()) {
			agencyError = 'Please fill both fields.';
			return;
		}
		agencyError = '';
		agencyLoading = true;
		try {
			const data = await apiPost<{ locationCount: number; locations: Array<{ id: string; name: string; businessName?: string }> }>(
				'/api/connect',
				{ agencyToken: agencyToken.trim(), companyId: companyId.trim() }
			);
			session.companyId = companyId.trim();
			session.connected = true;
			session.locations = data.locations.map((l) => ({
				id: l.id,
				name: l.name || l.businessName || l.id,
				businessName: l.businessName,
				hasToken: false,
			}));
			agencyToken = ''; // token stored server-side — clear input
			goto('/accounts');
		} catch (e: unknown) {
			agencyError =
				e instanceof Error
					? `${e.message} — if this is a 403, your token is a sub-account PIT, not an agency PIT. Use "Connect a single sub-account" instead.`
					: 'Connection failed';
		} finally {
			agencyLoading = false;
		}
	}

	// --- Location mode (sub-account PIT → single location, no agency needed) ---
	let locId = $state('');
	let locToken = $state('');
	let locError = $state('');
	let locLoading = $state(false);

	async function connectLocation() {
		if (!locId.trim() || !locToken.trim()) {
			locError = 'Please fill both fields.';
			return;
		}
		locError = '';
		locLoading = true;
		try {
			const data = await apiPost<{ locationId: string; name: string; valid: boolean }>(
				'/api/connect/location',
				{ locationId: locId.trim(), token: locToken.trim() }
			);
			session.connected = true;
			// Add (or refresh) this location in the session with its token saved.
			const existing = session.locations.find((l) => l.id === data.locationId);
			if (existing) {
				existing.hasToken = true;
			} else {
				session.locations = [
					...session.locations,
					{ id: data.locationId, name: data.name || data.locationId, hasToken: true },
				];
			}
			session.selectedIds.add(data.locationId);
			locToken = ''; // token stored server-side — clear input
			goto('/accounts');
		} catch (e: unknown) {
			locError = e instanceof Error ? e.message : 'Connection failed';
		} finally {
			locLoading = false;
		}
	}
</script>

<div class="connect-page">
	<p class="lead">Tokens are stored server-side in the vault and never sent back to the browser.</p>

	<!-- Location mode: works with a sub-account PIT -->
	<div class="card">
		<h2>Connect a single sub-account</h2>
		<p class="subtitle">
			Use a <strong>sub-account (location) Private Integration Token</strong> — created inside that
			sub-account's <code>Settings → Private Integrations</code>. No agency token needed.
		</p>

		<div class="field">
			<label for="loc-id">Location ID</label>
			<input id="loc-id" type="text" bind:value={locId} placeholder="e.g. 6KtnVX1w8kxKgXeMzNGd" autocomplete="off" />
		</div>
		<div class="field">
			<label for="loc-token">Sub-account Token (pit-…)</label>
			<input id="loc-token" type="password" bind:value={locToken} placeholder="pit-xxxx-xxxx-xxxx" autocomplete="off" />
		</div>

		{#if locError}<p class="error">{locError}</p>{/if}

		<button class="btn" onclick={connectLocation} disabled={locLoading}>
			{locLoading ? 'Validating…' : 'Connect this sub-account'}
		</button>
	</div>

	<div class="divider"><span>or</span></div>

	<!-- Agency mode: needs an agency-level PIT -->
	<div class="card">
		<h2>Connect agency (load all sub-accounts)</h2>
		<p class="subtitle">
			Use an <strong>agency-level PIT</strong> — created in the <em>agency</em> view's
			<code>Settings → Private Integrations</code>. The <strong>Company ID</strong> is the ID in that
			page's URL (<code>…/private-integrations/COMPANY_ID</code>).
		</p>

		<div class="field">
			<label for="agency-token">Agency Token (pit-…)</label>
			<input id="agency-token" type="password" bind:value={agencyToken} placeholder="pit-xxxx-xxxx-xxxx" autocomplete="off" />
		</div>
		<div class="field">
			<label for="company-id">Company ID</label>
			<input id="company-id" type="text" bind:value={companyId} placeholder="24-character company ID" autocomplete="off" />
		</div>

		{#if agencyError}<p class="error">{agencyError}</p>{/if}

		<button class="btn btn-secondary" onclick={connectAgency} disabled={agencyLoading}>
			{agencyLoading ? 'Connecting…' : 'Connect & Load Sub-Accounts'}
		</button>
	</div>
</div>

<style>
	.connect-page { max-width: 520px; margin: 32px auto; display: flex; flex-direction: column; gap: 16px; }
	.lead { font-size: 13px; color: var(--text2); text-align: center; }
	.card {
		background: var(--surface); border: 1.5px solid var(--border); border-radius: 16px;
		padding: 28px; display: flex; flex-direction: column; gap: 14px;
	}
	h2 { font-size: 19px; font-weight: 700; }
	.subtitle { font-size: 12.5px; color: var(--text2); line-height: 1.5; }
	.subtitle code { font-family: ui-monospace, monospace; background: rgba(0,0,0,0.05); padding: 1px 5px; border-radius: 4px; }
	.field { display: flex; flex-direction: column; gap: 6px; }
	label { font-size: 13px; font-weight: 600; }
	input {
		padding: 10px 14px; border: 1.5px solid var(--border); border-radius: 10px;
		font-family: inherit; font-size: 14px; outline: none; transition: border-color 0.15s;
	}
	input:focus { border-color: var(--accent); }
	.error { font-size: 13px; color: var(--error); line-height: 1.4; }
	.btn {
		padding: 12px 20px; background: var(--accent); color: #fff; border: none; border-radius: 10px;
		font-family: inherit; font-size: 14px; font-weight: 600; cursor: pointer; transition: opacity 0.15s;
	}
	.btn:disabled { opacity: 0.6; cursor: not-allowed; }
	.btn-secondary { background: rgba(0,0,0,0.05); color: var(--text2); border: 1.5px solid var(--border); }
	.divider { display: flex; align-items: center; text-align: center; color: var(--text2); font-size: 12px; }
	.divider::before, .divider::after { content: ''; flex: 1; border-bottom: 1.5px solid var(--border); }
	.divider span { padding: 0 12px; }
</style>
