declare const process: {
	env: Record<string, string | undefined>;
};
if (process.env.NODE_ENV === 'production') {
	console.log('Running in production mode');
} else {
	console.log('Running in development mode, BASE_URL set to localhost');
}
// const BASE_URL = 'http://localhost:3000/api';
const BASE_URL = process.env.NODE_ENV === 'production' ? '/api' : 'http://localhost:3000/api';

export interface ApiOptions<T> {
	method?: 'GET' | 'POST' | 'PUT' | 'DELETE';
	body?: T;
	headers?: Record<string, string>;
}

export async function api<Req, Res>(path: string, options: ApiOptions<Req> = {}): Promise<Res> {
	const res = await fetch(`${BASE_URL}${path}`, {
		method: options.method ?? 'GET',
		headers: {
			'Content-Type': 'application/json',
			...(options.headers ?? {})
		},
		body: options.body ? JSON.stringify(options.body) : undefined
	});

	if (!res.ok) {
		const message = await res.text();
		throw new Error(message);
	}

	return res.json() as Promise<Res>;
}
