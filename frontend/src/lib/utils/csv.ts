// CSV utilities — used by Accounts (library import/export), Dialers (flagged numbers),
// and any future module that touches tabular data.
// Handles quoted fields, embedded commas, and CRLF/LF line endings.

/**
 * Parse a CSV string into an array of rows keyed by header.
 * The first non-empty line is treated as the header row.
 * Quoted fields ("...") and embedded commas are handled per RFC 4180.
 */
export function parseCSV(text: string): { headers: string[]; rows: Record<string, string>[] } {
	const lines = splitCSVLines(text);
	if (lines.length === 0) return { headers: [], rows: [] };
	const headers = parseCSVLine(lines[0]);
	const rows: Record<string, string>[] = [];
	for (let i = 1; i < lines.length; i++) {
		const cells = parseCSVLine(lines[i]);
		if (cells.length === 0 || (cells.length === 1 && cells[0] === '')) continue;
		const row: Record<string, string> = {};
		for (let j = 0; j < headers.length; j++) {
			row[headers[j]] = cells[j] ?? '';
		}
		rows.push(row);
	}
	return { headers, rows };
}

/**
 * Serialize an array of objects to a CSV string. Header order is given explicitly.
 * Values with commas, quotes, or newlines are quoted; embedded quotes are doubled.
 */
export function toCSV(headers: string[], rows: Record<string, unknown>[]): string {
	const headerLine = headers.map(escapeCSVCell).join(',');
	const dataLines = rows.map((r) =>
		headers.map((h) => escapeCSVCell(String(r[h] ?? ''))).join(',')
	);
	return [headerLine, ...dataLines].join('\r\n');
}

/**
 * Trigger a browser download of CSV content with the given filename.
 */
export function downloadCSV(filename: string, content: string): void {
	const blob = new Blob([content], { type: 'text/csv;charset=utf-8' });
	const url = URL.createObjectURL(blob);
	const a = document.createElement('a');
	a.href = url;
	a.download = filename;
	document.body.appendChild(a);
	a.click();
	document.body.removeChild(a);
	URL.revokeObjectURL(url);
}

// --- internals ---

function splitCSVLines(text: string): string[] {
	// Split on CRLF or LF, BUT respect quoted fields (newlines inside "..." are kept).
	const lines: string[] = [];
	let current = '';
	let inQuotes = false;
	for (let i = 0; i < text.length; i++) {
		const c = text[i];
		if (c === '"') {
			inQuotes = !inQuotes;
			current += c;
		} else if ((c === '\n' || c === '\r') && !inQuotes) {
			if (current !== '' || lines.length > 0) lines.push(current);
			current = '';
			if (c === '\r' && text[i + 1] === '\n') i++;
		} else {
			current += c;
		}
	}
	if (current !== '') lines.push(current);
	return lines;
}

function parseCSVLine(line: string): string[] {
	const cells: string[] = [];
	let cell = '';
	let inQuotes = false;
	for (let i = 0; i < line.length; i++) {
		const c = line[i];
		if (inQuotes) {
			if (c === '"' && line[i + 1] === '"') {
				cell += '"';
				i++;
			} else if (c === '"') {
				inQuotes = false;
			} else {
				cell += c;
			}
		} else {
			if (c === ',') {
				cells.push(cell);
				cell = '';
			} else if (c === '"' && cell === '') {
				inQuotes = true;
			} else {
				cell += c;
			}
		}
	}
	cells.push(cell);
	return cells;
}

function escapeCSVCell(v: string): string {
	if (v.includes(',') || v.includes('"') || v.includes('\n') || v.includes('\r')) {
		return '"' + v.replace(/"/g, '""') + '"';
	}
	return v;
}
