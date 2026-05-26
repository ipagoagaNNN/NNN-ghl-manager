<script lang="ts">
	import { goto } from '$app/navigation';
	import { apiPost } from '$lib/api/client.js';
	import { session } from '$lib/stores/session.svelte.js';

	let agencyToken = $state('');
	let companyId = $state('');
	let error = $state('');
	let loading = $state(false);

	async function connect() {
		if (!agencyToken.trim() || !companyId.trim()) {
			error = 'Please fill both fields.';
			return;
		}
		error = '';
		loading = true;
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
			// Token is stored server-side — clear the input
			agencyToken = '';
			goto('/accounts');
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'Connection failed';
		} finally {
			loading = false;
		}
	}
</script>

<div class="connect-page">
	<div class="card">
		<h2>Connect Agency</h2>
		<p class="subtitle">Tokens are stored server-side and never sent back to the browser.</p>

		<div class="field">
			<label for="agency-token">Agency Token</label>
			<input
				id="agency-token"
				type="password"
				bind:value={agencyToken}
				placeholder="pit-..."
				autocomplete="off"
			/>
		</div>

		<div class="field">
			<label for="company-id">Company ID</label>
			<input
				id="company-id"
				type="text"
				bind:value={companyId}
				placeholder="24-character company ID"
				autocomplete="off"
			/>
		</div>

		{#if error}
			<p class="error">{error}</p>
		{/if}

		<button class="btn" onclick={connect} disabled={loading}>
			{loading ? 'Connecting...' : 'Connect & Load Sub-Accounts'}
		</button>
	</div>
</div>

<style>
	.connect-page {
		max-width: 480px;
		margin: 40px auto;
	}
	.card {
		background: var(--surface);
		border: 1.5px solid var(--border);
		border-radius: 16px;
		padding: 32px;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	h2 { font-size: 22px; font-weight: 700; }
	.subtitle { font-size: 13px; color: var(--text2); }
	.field { display: flex; flex-direction: column; gap: 6px; }
	label { font-size: 13px; font-weight: 600; }
	input {
		padding: 10px 14px;
		border: 1.5px solid var(--border);
		border-radius: 10px;
		font-family: inherit;
		font-size: 14px;
		outline: none;
		transition: border-color 0.15s;
	}
	input:focus { border-color: var(--accent); }
	.error { font-size: 13px; color: var(--error); }
	.btn {
		padding: 12px 20px;
		background: var(--accent);
		color: #fff;
		border: none;
		border-radius: 10px;
		font-family: inherit;
		font-size: 14px;
		font-weight: 600;
		cursor: pointer;
		transition: opacity 0.15s;
	}
	.btn:disabled { opacity: 0.6; cursor: not-allowed; }
</style>
