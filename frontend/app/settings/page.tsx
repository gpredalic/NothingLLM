"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useTheme } from "@/components/theme-provider";
import { getAdminSettings, updateAdminSettings, getHealth, testConnection, AdminSettings, HealthStatus } from "@/lib/api";
import { toast } from "@/components/ui/toast";
import { Moon, Sun, Monitor, Key, Database, Cpu, CheckCircle, XCircle, Loader2, Plug } from "lucide-react";
import { cn } from "@/lib/utils";

export default function SettingsPage() {
  const { theme, setTheme } = useTheme();
  const [settings, setSettings] = useState<AdminSettings | null>(null);
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  // Form state
  const [openaiUrl, setOpenaiUrl] = useState("");
  const [openaiKey, setOpenaiKey] = useState("");

  const load = useCallback(() => {
    setLoading(true);
    Promise.all([getAdminSettings(), getHealth().catch(() => null)])
      .then(([s, h]) => {
        setSettings(s);
        setHealth(h);
        setOpenaiUrl((s as any).openai_base_url || "https://api.openai.com/v1");
      })
      .catch(() => setMessage({ type: "error", text: "Failed to load settings" }))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleSave = async () => {
    setSaving(true);
    setMessage(null);
    try {
      const updates: Record<string, string> = { openai_base_url: openaiUrl };
      if (openaiKey) updates.openai_api_key = openaiKey;
      updates.embedding_model = "BAAI/bge-m3";
      await updateAdminSettings(updates as Partial<AdminSettings>);
      setMessage({ type: "success", text: "Settings saved successfully" });
      toast("success", "Settings saved");
      setOpenaiKey("");
      load();
    } catch (e) {
      setMessage({ type: "error", text: e instanceof Error ? e.message : "Failed to save settings" });
    } finally {
      setSaving(false);
    }
  };

  const handleTestConnection = async () => {
    setTesting(true);
    setMessage(null);
    try {
      const result = await testConnection();
      const openai = result.openai;
      if (openai?.status === "ok") {
        setMessage({ type: "success", text: "Connection successful" });
        toast("success", "Connection successful");
      } else {
        setMessage({ type: "error", text: `Connection failed: ${openai?.error || "unknown error"}` });
        toast("error", "Connection failed");
      }
    } catch (e) {
      setMessage({ type: "error", text: e instanceof Error ? e.message : "Connection test failed" });
    } finally {
      setTesting(false);
    }
  };

  if (loading) {
    return (
      <div className="p-6">
        <div className="h-8 w-32 animate-pulse rounded bg-muted mb-6" />
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-40 animate-pulse rounded-lg bg-muted" />
          ))}
        </div>
      </div>
    );
  }

  const openaiStatus = health?.openai?.status || "unknown";
  const qdrantStatus = health?.qdrant?.status || "unknown";

  return (
    <div className="p-6">
      <h1 className="text-2xl font-semibold mb-6">Settings</h1>
      {message && (
        <div className={cn("mb-4 rounded-md border p-3 text-sm", message.type === "success" ? "border-green-300 bg-green-50 text-green-700" : "border-destructive/50 bg-destructive/10 text-destructive")}>
          {message.text}
        </div>
      )}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {/* Theme */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              {theme === "dark" ? <Moon className="h-4 w-4" /> : theme === "light" ? <Sun className="h-4 w-4" /> : <Monitor className="h-4 w-4" />}
              Appearance
            </CardTitle>
            <CardDescription>Select theme mode</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex gap-2">
              {(["light", "dark", "system"] as const).map((t) => (
                <Button
                  key={t}
                  variant={theme === t ? "default" : "outline"}
                  size="sm"
                  className="flex-1"
                  onClick={() => setTheme(t)}
                >
                  {t === "light" ? <Sun className="h-3 w-3 mr-1" /> : t === "dark" ? <Moon className="h-3 w-3 mr-1" /> : <Monitor className="h-3 w-3 mr-1" />}
                  {t.charAt(0).toUpperCase() + t.slice(1)}
                </Button>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* OpenAI-Compatible API Provider */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <Key className="h-4 w-4" /> OpenAI-Compatible API
            </CardTitle>
            <CardDescription>
              {openaiStatus === "ok" ? (
                <span className="flex items-center gap-1 text-green-600"><CheckCircle className="h-3 w-3" /> Connected</span>
              ) : openaiStatus === "down" ? (
                <span className="flex items-center gap-1 text-destructive"><XCircle className="h-3 w-3" /> Not Connected</span>
              ) : (
                <span className="text-muted-foreground">Not configured</span>
              )}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="space-y-2">
              <label className="text-sm font-medium">Base URL / Endpoint</label>
              <Input
                type="text"
                placeholder="https://api.openai.com/v1"
                value={openaiUrl}
                onChange={(e) => setOpenaiUrl(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                e.g. http://ollama:11434/v1, https://api.openai.com/v1
              </p>
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">API Key</label>
              <Input
                type="password"
                placeholder={settings?.openai_api_key ? "**** " + (settings.openai_api_key.slice(-4)) : "sk-..."}
                value={openaiKey}
                onChange={(e) => setOpenaiKey(e.target.value)}
              />
            </div>
            <Button variant="outline" size="sm" onClick={handleTestConnection} disabled={testing} className="w-full">
              {testing ? <Loader2 className="h-3 w-3 animate-spin mr-2" /> : <Plug className="h-3 w-3 mr-2" />}
              {testing ? "Testing..." : "Test Connection"}
            </Button>
          </CardContent>
        </Card>

        {/* Embeddings */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <Cpu className="h-4 w-4" /> Embeddings
            </CardTitle>
            <CardDescription>Embedding model provider</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-3 rounded-md border p-3">
              <Cpu className="h-5 w-5 text-muted-foreground" />
              <div>
                <div className="text-sm font-medium">BGE-M3</div>
                <div className="text-xs text-muted-foreground">BAAI/bge-m3 (local)</div>
              </div>
              <CheckCircle className="h-4 w-4 text-green-500 ml-auto" />
            </div>
          </CardContent>
        </Card>

        {/* Service Status */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <Database className="h-4 w-4" /> Services
            </CardTitle>
            <CardDescription>Internal service status</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span>Qdrant (Vector DB)</span>
                {qdrantStatus === "ok" ? (
                  <span className="flex items-center gap-1 text-green-600"><CheckCircle className="h-3 w-3" /> Connected</span>
                ) : (
                  <span className="flex items-center gap-1 text-destructive"><XCircle className="h-3 w-3" /> Not Available</span>
                )}
              </div>
              <div className="flex items-center justify-between text-sm">
                <span>PostgreSQL</span>
                {(health?.database?.status === "ok") ? (
                  <span className="flex items-center gap-1 text-green-600"><CheckCircle className="h-3 w-3" /> Connected</span>
                ) : (
                  <span className="flex items-center gap-1 text-destructive"><XCircle className="h-3 w-3" /> Not Available</span>
                )}
              </div>
              <div className="flex items-center justify-between text-sm">
                <span>BGE-M3 (Embeddings)</span>
                <span className="flex items-center gap-1 text-green-600"><CheckCircle className="h-3 w-3" /> Local</span>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Save */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Save Changes</CardTitle>
            <CardDescription>Apply your configuration</CardDescription>
          </CardHeader>
          <CardContent>
            <Button onClick={handleSave} disabled={saving} className="w-full">
              {saving ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
              {saving ? "Saving..." : "Save Settings"}
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
