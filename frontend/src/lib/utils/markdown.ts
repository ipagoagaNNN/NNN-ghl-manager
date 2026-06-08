// Minimal, dependency-free Markdown → HTML for our own trusted docs (MANUAL.md).
// Not a full CommonMark implementation — it handles the constructs our manual
// uses: headings, bold, inline code, fenced code blocks, tables, unordered lists,
// blockquotes, horizontal rules, links, and paragraphs. All text is HTML-escaped
// first, so it is safe to feed the result to {@html}. Source is build-time content,
// never user input.

function esc(s: string): string {
	return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function inline(s: string): string {
	let t = esc(s);
	t = t.replace(/`([^`]+)`/g, '<code>$1</code>');
	t = t.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
	t = t.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>');
	return t;
}

function splitRow(s: string): string[] {
	return s
		.replace(/^\s*\|/, '')
		.replace(/\|\s*$/, '')
		.split('|')
		.map((c) => c.trim());
}

export function mdToHtml(md: string): string {
	const lines = md.split(/\r?\n/);
	const out: string[] = [];
	let i = 0;
	let inList = false;
	const closeList = () => {
		if (inList) {
			out.push('</ul>');
			inList = false;
		}
	};

	while (i < lines.length) {
		const line = lines[i];
		const trimmed = line.trim();

		// fenced code block
		if (trimmed.startsWith('```')) {
			closeList();
			const buf: string[] = [];
			i++;
			while (i < lines.length && !lines[i].trim().startsWith('```')) {
				buf.push(lines[i]);
				i++;
			}
			i++; // skip closing fence
			out.push('<pre class="code"><code>' + esc(buf.join('\n')) + '</code></pre>');
			continue;
		}

		// table: header row immediately followed by a separator row (---|---)
		if (
			line.includes('|') &&
			i + 1 < lines.length &&
			/^[\s|:-]+$/.test(lines[i + 1]) &&
			lines[i + 1].includes('-')
		) {
			closeList();
			const header = splitRow(line);
			i += 2;
			const rows: string[][] = [];
			while (i < lines.length && lines[i].includes('|') && lines[i].trim() !== '') {
				rows.push(splitRow(lines[i]));
				i++;
			}
			let tbl =
				'<table><thead><tr>' +
				header.map((h) => `<th>${inline(h)}</th>`).join('') +
				'</tr></thead><tbody>';
			for (const r of rows) tbl += '<tr>' + r.map((c) => `<td>${inline(c)}</td>`).join('') + '</tr>';
			tbl += '</tbody></table>';
			out.push(tbl);
			continue;
		}

		// heading
		const h = /^(#{1,6})\s+(.*)$/.exec(line);
		if (h) {
			closeList();
			const lvl = h[1].length;
			out.push(`<h${lvl}>${inline(h[2])}</h${lvl}>`);
			i++;
			continue;
		}

		// horizontal rule
		if (/^-{3,}$/.test(trimmed)) {
			closeList();
			out.push('<hr />');
			i++;
			continue;
		}

		// blockquote (one or more consecutive > lines)
		if (trimmed.startsWith('>')) {
			closeList();
			const buf: string[] = [];
			while (i < lines.length && lines[i].trim().startsWith('>')) {
				buf.push(lines[i].trim().replace(/^>\s?/, ''));
				i++;
			}
			out.push('<blockquote>' + buf.map((b) => (b ? inline(b) : '')).join('<br />') + '</blockquote>');
			continue;
		}

		// unordered list
		const li = /^[-*]\s+(.*)$/.exec(trimmed);
		if (li) {
			if (!inList) {
				out.push('<ul>');
				inList = true;
			}
			out.push(`<li>${inline(li[1])}</li>`);
			i++;
			continue;
		}

		// blank line
		if (trimmed === '') {
			closeList();
			i++;
			continue;
		}

		// paragraph
		closeList();
		out.push(`<p>${inline(line)}</p>`);
		i++;
	}
	closeList();
	return out.join('\n');
}
