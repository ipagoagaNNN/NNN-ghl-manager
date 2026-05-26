// All GHL API calls go through the backend proxy.
// The frontend NEVER calls services.leadconnectorhq.com directly.

type FetchOptions = RequestInit & { json?: unknown };

async function apiFetch(path: string, opts: FetchOptions = {}): Promise<Response> {
	const { json, ...rest } = opts;
	const init: RequestInit = {
		...rest,
		credentials: 'include', // sends session cookie (Phase 2 auth)
		headers: {
			...(json ? { 'Content-Type': 'application/json' } : {}),
			...rest.headers,
		},
	};
	if (json !== undefined) {
		init.body = JSON.stringify(json);
	}
	return fetch(path, init);
}

export async function apiGet<T>(path: string): Promise<T> {
	const res = await apiFetch(path);
	if (!res.ok) throw new Error(`GET ${path} → ${res.status}`);
	return res.json();
}

export async function apiPost<T>(path: string, body: unknown): Promise<T> {
	const res = await apiFetch(path, { method: 'POST', json: body });
	if (!res.ok) {
		const text = await res.text();
		throw new Error(`POST ${path} → ${res.status}: ${text}`);
	}
	return res.json();
}

export async function apiPut<T>(path: string, body: unknown): Promise<T> {
	const res = await apiFetch(path, { method: 'PUT', json: body });
	if (!res.ok) {
		const text = await res.text();
		throw new Error(`PUT ${path} → ${res.status}: ${text}`);
	}
	return res.json();
}

// Typed GHL proxy helpers — route through /api/ghl/*
export async function ghlGet<T>(ghlPath: string): Promise<T> {
	return apiGet(`/api/ghl${ghlPath}`);
}

export async function ghlPut<T>(ghlPath: string, body: unknown): Promise<T> {
	return apiPut(`/api/ghl${ghlPath}`, body);
}
