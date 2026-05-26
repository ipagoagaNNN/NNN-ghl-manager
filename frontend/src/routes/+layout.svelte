<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { session } from '$lib/stores/session.svelte.js';
	import { initExtensionBridge, bridge } from '$lib/stores/extension-bridge.svelte.js';

	const { children } = $props();

	const navItems = [
		{ href: '/connect', label: 'Connect', icon: '⚡' },
		{ href: '/accounts', label: 'Accounts', icon: '🏢' },
		{ href: '/custom-values', label: 'Custom Values', icon: '✏️' },
		{ href: '/funnels', label: 'Funnels', icon: '🔗' },
		{ href: '/automations', label: 'Automations', icon: '⚙️' },
		{ href: '/dashboard', label: 'Dashboard', icon: '📊' },
		{ href: '/dialers', label: 'Dialers', icon: '📞' },
	] as const;

	onMount(() => {
		initExtensionBridge();
	});
</script>

<div class="app">
	<nav class="sidebar">
		<div class="logo">GHL</div>
		{#each navItems as item}
			<a
				href={item.href}
				class="nav-btn"
				class:active={$page.url.pathname.startsWith(item.href)}
			>
				<span class="nav-icon">{item.icon}</span>
				<span class="nav-label">{item.label}</span>
			</a>
		{/each}
	</nav>

	<div class="main-wrap">
		<header class="topbar">
			<div class="status-indicator">
				<span class="status-dot" class:on={session.connected}></span>
				<span class="status-label">
					{session.connected ? `${session.locations.length} sub-accounts` : 'Not connected'}
				</span>
			</div>
			{#if bridge.ready}
				<span class="ext-badge">Extension active</span>
			{/if}
		</header>

		<main class="main">
			{@render children()}
		</main>
	</div>
</div>

<style>
	:global(*) { box-sizing: border-box; margin: 0; padding: 0; }
	:global(body) {
		font-family: 'Urbanist', sans-serif;
		background: #f5f5f7;
		color: #1a1d2e;
		--accent: #ff1d8d;
		--success: #00c97a;
		--error: #ff3b5c;
		--border: rgba(0,0,0,0.1);
		--surface: #fff;
		--text2: #6b7280;
	}

	.app { display: flex; height: 100vh; overflow: hidden; }

	.sidebar {
		width: 116px;
		background: #fff;
		border-right: 1.5px solid var(--border);
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 16px 0;
		gap: 4px;
		flex-shrink: 0;
	}

	.logo {
		font-weight: 700;
		font-size: 18px;
		color: var(--accent);
		margin-bottom: 16px;
	}

	.nav-btn {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 4px;
		width: 100px;
		padding: 10px 8px;
		border-radius: 12px;
		text-decoration: none;
		color: var(--text2);
		font-size: 11px;
		font-weight: 500;
		transition: background 0.15s, color 0.15s;
	}
	.nav-btn:hover, .nav-btn.active {
		background: rgba(255,29,141,0.07);
		color: var(--accent);
	}
	.nav-icon { font-size: 18px; }

	.main-wrap { flex: 1; display: flex; flex-direction: column; overflow: hidden; }

	.topbar {
		height: 52px;
		background: #fff;
		border-bottom: 1.5px solid var(--border);
		display: flex;
		align-items: center;
		padding: 0 24px;
		gap: 16px;
		flex-shrink: 0;
	}

	.status-indicator { display: flex; align-items: center; gap: 8px; }
	.status-dot {
		width: 8px; height: 8px;
		border-radius: 50%;
		background: #d1d5db;
		transition: background 0.2s;
	}
	.status-dot.on { background: var(--success); }
	.status-label { font-size: 13px; color: var(--text2); }

	.ext-badge {
		font-size: 11px;
		padding: 2px 8px;
		border-radius: 20px;
		background: rgba(0,201,122,0.1);
		color: var(--success);
		font-weight: 600;
	}

	.main { flex: 1; overflow-y: auto; padding: 24px; }
</style>
