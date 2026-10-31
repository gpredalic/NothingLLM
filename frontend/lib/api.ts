const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    credentials: "include",
    ...options,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error || `Request failed: ${res.status}`);
  }
  return res.json();
}

// Auth
export async function register(email: string, password: string, name?: string) {
  return request<{ id: string; email: string }>(`/api/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password, name }),
  });
}

export async function login(email: string, password: string) {
  return request<{ ok: boolean }>(`/api/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
}

export async function logout() {
  return request<{ ok: boolean }>(`/api/auth/logout`, { method: "POST" });
}

// Users
export interface User {
  id: string;
  email: string;
  role: string;
  created_at: string;
}

export async function listUsers() {
  return request<User[]>(`/api/users`);
}

export async function getUser(id: string) {
  return request<User>(`/api/users/${id}`);
}

export async function createUser(email: string, password: string, name?: string) {
  return request<{ id: string; email: string }>(`/api/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password, name }),
  });
}

export async function updateUser(id: string, updates: { role?: string; password?: string }) {
  return request<{ status: string }>(`/api/users/${id}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(updates),
  });
}

export async function deleteUser(id: string) {
  return request<{ status: string }>(`/api/users/${id}`, { method: "DELETE" });
}

// Workspaces
export interface Workspace {
  id: string;
  name: string;
  description: string;
  created_at: string;
}

export interface WorkspaceMember {
  id: string;
  email: string;
  role: string;
}

export async function listWorkspaces() {
  return request<Workspace[]>(`/api/workspaces`);
}

export async function getWorkspace(id: string) {
  return request<Workspace>(`/api/workspaces/${id}`);
}

export async function listWorkspaceMembers(wsId: string) {
  return request<WorkspaceMember[]>(`/api/workspaces/${wsId}/members`);
}

export async function createWorkspace(name: string, description?: string) {
  return request<{ id: string; name: string }>(`/api/workspaces`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, description }),
  });
}

export async function deleteWorkspace(id: string) {
  return request<{ status: string }>(`/api/workspaces/${id}`, { method: "DELETE" });
}

export async function addMember(wsId: string, userId: string, role: string) {
  return request<{ status: string }>(`/api/workspaces/${wsId}/members`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ user_id: userId, role }),
  });
}

export async function removeMember(wsId: string, userId: string) {
  return request<{ status: string }>(`/api/workspaces/${wsId}/members/${userId}`, { method: "DELETE" });
}

// Collections
export interface Collection {
  name: string;
  vector_count: number;
  dimensions: number;
  status: string;
  discovered_at: string;
  updated_at: string;
}

export async function listCollections() {
  return request<Collection[]>(`/api/collections`);
}

export async function getCollection(name: string) {
  return request<Collection>(`/api/collections/${name}`);
}

export async function listWorkspaceCollections(wsId: string) {
  return request<string[]>(`/api/workspaces/${wsId}/collections`);
}

export async function assignCollection(wsId: string, name: string) {
  return request<{ status: string }>(`/api/workspaces/${wsId}/collections/${name}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({}),
  });
}

export async function unassignCollection(wsId: string, name: string) {
  return request<{ status: string }>(`/api/workspaces/${wsId}/collections/${name}`, {
    method: "DELETE",
  });
}

// RAG
export interface ChatMessage {
  role: string;
  content: string;
}

export async function search(wsId: string, query: string, topK?: number) {
  return request<{ id: string; score: number; payload: Record<string, unknown> }[]>(
    `/api/workspaces/${wsId}/search`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ query, top_k: topK || 5 }),
    }
  );
}

export async function chat(
  wsId: string,
  messages: ChatMessage[],
  modelId?: string
): Promise<{ content: string; model: string }> {
  return request<{ content: string; model: string }>(`/api/workspaces/${wsId}/chat`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ messages, model_id: modelId }),
  });
}

// Models
export interface Model {
  id: string;
  name: string;
  context_window: number;
  enabled: boolean;
  synced_at: string;
  created_at?: string;
  is_default: boolean;
}

export async function listModels() {
  return request<Model[]>(`/api/models`);
}

export async function updateModel(id: string, enabled: boolean) {
  return request<{ status: string }>(`/api/models/${id}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ enabled }),
  });
}

export async function setDefaultModel(id: string) {
  return request<{ status: string }>(`/api/models/default`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id }),
  });
}

// Admin Settings
export interface AdminSettings {
  openai_api_key: string;
  qdrant_url: string;
  qdrant_api_key: string;
  embedding_model: string;
}

export async function getAdminSettings() {
  return request<AdminSettings>(`/api/admin/settings`);
}

export async function updateAdminSettings(settings: Partial<AdminSettings>) {
  return request<{ status: string }>(`/api/admin/settings`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(settings),
  });
}

export async function testConnection() {
  return request<Record<string, { status: string; error?: string }>>(`/api/admin/test-connection`, {
    method: "POST",
  });
}

// Admin Stats
export interface AdminStats {
  user_count: number;
  workspace_count: number;
  collection_count: number;
  vector_count: number;
  active_sessions: number;
}

export async function getAdminStats() {
  return request<AdminStats>(`/api/admin/stats`);
}

// Health
export interface HealthStatus {
  database: { status: string; error?: string };
  qdrant: { status: string; error?: string };
  openai: { status: string; error?: string };
}

export async function getHealth() {
  return request<HealthStatus>(`/api/health`);
}
