export const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export const API_ENDPOINTS = {
  AGENTS: `${API_BASE_URL}/api/v1/agents`,
  AGENT: (name: string) => `${API_BASE_URL}/api/v1/agents/${name}`,
  ALERTS: `${API_BASE_URL}/api/v1/alerts`,
  EVENTS: `${API_BASE_URL}/api/v1/events`,
  HEALTH: `${API_BASE_URL}/api/v1/health`,
} as const;
