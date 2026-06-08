<script lang="ts">
	// The canonical manual lives at docs/MANUAL.md (repo root). We import it raw and
	// render it, so this Help page is always in sync with the manual — single source.
	import manualRaw from '../../../../docs/MANUAL.md?raw';
	import { mdToHtml } from '$lib/utils/markdown.js';

	const html = mdToHtml(manualRaw);
</script>

<div class="help">
	<article class="doc">
		{@html html}
	</article>
</div>

<style>
	.help { max-width: 920px; margin: 0 auto; }
	.doc {
		background: var(--surface); border: 1.5px solid var(--border);
		border-radius: 14px; padding: 28px 34px;
	}
	.doc :global(h1) { font-size: 26px; font-weight: 800; margin: 6px 0 14px; }
	.doc :global(h2) {
		font-size: 20px; font-weight: 700; margin: 28px 0 10px;
		padding-top: 16px; border-top: 1.5px solid var(--border);
	}
	.doc :global(h3) { font-size: 16px; font-weight: 700; margin: 18px 0 8px; }
	.doc :global(p) { font-size: 13.5px; line-height: 1.65; margin: 8px 0; color: #2a2d3a; }
	.doc :global(ul) { margin: 8px 0 10px 22px; display: flex; flex-direction: column; gap: 4px; }
	.doc :global(li) { font-size: 13.5px; line-height: 1.55; color: #2a2d3a; }
	.doc :global(a) { color: var(--accent); font-weight: 600; }
	.doc :global(code) {
		font-family: ui-monospace, monospace; font-size: 12px;
		background: rgba(0,0,0,0.05); padding: 1px 5px; border-radius: 4px;
	}
	.doc :global(pre.code) {
		background: #1a1d2e; color: #e8e8ee; padding: 14px 16px; border-radius: 10px;
		overflow-x: auto; margin: 12px 0;
	}
	.doc :global(pre.code code) { background: none; color: inherit; padding: 0; font-size: 12px; line-height: 1.5; }
	.doc :global(blockquote) {
		border-left: 3px solid var(--accent); background: rgba(255,29,141,0.04);
		padding: 10px 16px; border-radius: 0 8px 8px 0; margin: 12px 0;
		font-size: 13px; color: var(--text2); line-height: 1.6;
	}
	.doc :global(hr) { border: none; border-top: 1.5px solid var(--border); margin: 22px 0; }
	.doc :global(table) { border-collapse: collapse; width: 100%; margin: 12px 0; font-size: 12.5px; }
	.doc :global(th), .doc :global(td) {
		border: 1px solid var(--border); padding: 7px 10px; text-align: left; vertical-align: top;
	}
	.doc :global(th) { background: rgba(0,0,0,0.04); font-weight: 700; }
	.doc :global(strong) { font-weight: 700; }
</style>
