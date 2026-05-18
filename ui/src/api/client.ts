import axios, { type AxiosError } from 'axios';

const api = axios.create({
  // In dev: Vite proxy forwards /api/* to localhost:8080, so we use the relative path.
  // In production: VITE_API_URL should point to the backend (e.g. https://api.example.com/api/v1).
  baseURL: import.meta.env.VITE_API_URL || '/api/v1',
  timeout: 30_000, // 30 s — long enough for LLM calls but won't hang forever
  headers: {
    'Content-Type': 'application/json',
  },
});

// Attach API key if configured at build-time (optional — for self-hosted deployments).
const apiKey = import.meta.env.VITE_API_KEY;
if (apiKey) {
  api.defaults.headers.common['X-API-Key'] = apiKey;
}

// Response interceptor: normalise error messages so every catch block
// receives a string instead of needing to dig into AxiosError internals.
api.interceptors.response.use(
  (res) => res,
  (error: AxiosError) => {
    const data = error.response?.data as Record<string, unknown> | undefined;
    // Backend always returns { "error": "..." } — extract that first.
    const serverMsg = typeof data?.error === 'string' ? data.error : null;
    const message = serverMsg ?? error.message ?? 'An unknown error occurred';
    // Attach a clean message so callers can do `err.message` reliably.
    error.message = message;
    return Promise.reject(error);
  }
);

export default api;
