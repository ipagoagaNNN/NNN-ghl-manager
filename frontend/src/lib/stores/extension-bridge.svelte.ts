// Chrome extension bridge — wraps the postMessage/CustomEvent protocol.
// Protocol is identical to the HTML prototype; this just makes it reactive.
// See docs/extension-bridge.md for full protocol specification.

import type { NumbersMatchData } from './dialers.svelte.js';

export const bridge = $state({
	ready: false,
	lastNumbersSync: null as NumbersMatchData | null,
	lastFlaggedImport: null as { headers: string[]; rows: Record<string, string>[]; fileName: string; importedAt: string } | null,
});

// Call once at app mount (e.g. in +layout.svelte)
export function initExtensionBridge() {
	// Inbound: postMessage from extension (source: 'cnam-extension')
	window.addEventListener('message', (ev) => {
		if (ev.source !== window) return;
		const data = ev.data ?? {};
		if (data?.source !== 'cnam-extension') return;

		if (data.type === 'ghl-extension-ready') {
			bridge.ready = true;
		}
	});

	// Inbound: CustomEvents from extension
	window.addEventListener('ghl-flagged-import', (ev: Event) => {
		const detail = (ev as CustomEvent).detail;
		if (detail) bridge.lastFlaggedImport = detail;
	});

	window.addEventListener('ghl-numbers-sync', (ev: Event) => {
		const detail = (ev as CustomEvent).detail;
		if (detail) bridge.lastNumbersSync = detail;
	});
}

// Outbound: request numbers sync from extension
export function requestNumbersSync(forceSync = false) {
	const payload = { source: 'ghl-manager', type: 'ghl-request-numbers-sync', forceSync };
	window.postMessage(payload, '*');
	window.dispatchEvent(new CustomEvent('ghl-request-numbers-sync', { detail: { forceSync } }));
}

// Outbound: register a number (NV / Hiya / Both)
export function registerNumber(
	mode: 'nv' | 'hiya' | 'both',
	items: Array<{ number: string; hiyaNumber: string; departmentName: string }>
) {
	const payload = { source: 'ghl-manager', type: 'ghl-register-number', mode, items };
	window.postMessage(payload, '*');
	window.dispatchEvent(new CustomEvent('ghl-register-number', { detail: { mode, items } }));
}
