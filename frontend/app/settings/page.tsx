"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useTheme } from "@/components/theme-provider";
import { getAdminSettings, updateAdminSettings, testConnection, AdminSettings } from "@/lib/api";
import { Moon, Sun, Monitor, Key, Database, Cpu, CheckCircle, XCircle, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";

type TestResult = { status: string; error?: string };

export default function SettingsPage() {
  const { theme, setTheme } = useTheme();
  const [settings, setSettings] = useState<AdminSettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testResults, setTestResults] = useState<Record<string, TestResult>>({});
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  // Form state
  const [openaiKey, setOpenaiKey] = useState("");
  const [qdrantUrl, setQdrantUrl] = useState("");
  const [qdrantKey, setQdrantKey] = useState("");
  const [embeddingModel, setEmbeddingModel] = useState("");

  const load = useCallback(() => {
    setLoading(true);
    getAdminSettings()
      .then((s) => {
        setSettings(s);
        setQdrantUrl(s.qdrant_url || "");
        setEmbeddingModel(s.embedding_model || "");
      })
      .catch(() => setMessage({ type: "error", text: "Failed to load settings" }))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleSave = async () => {
    setSaving(true);
    setMessage(null);
    try {
      const updates: Partial<AdminSettings> = {};
      if (openaiKey) updates.openai_api_key = openaiKey;
      if (qdrantUrl && qdrantUrl !== settings?.qdrant_url) updates.qdrant_url = qdrantUrl;
      if (qdrantKey) updates.qdrant_api_key = qdrantKey;
      if (embeddingModel && embeddingModel !== settings?.embedding_model) updates.embedding_model = embeddingModel;
      if (Object.keys(updates).length > 0) {
        await updateAdminSettings(updates);
      }
      setMessage({ type: "success", text: "Settings saved successfully" });
      setOpenaiKey("");
      setQdrantKey("");
      load();
    } catch (e) {
      setMessage({ type: "error", text: e instanceof Error ? e.message : "Failed to save settings" });
    } finally {
      setSaving(false);
    }
  };

  const handleTest = async () => {
    setTesting(true);
    setTestResults({});
    try {
      const results = await testConnection();
      setTestResults(results);
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
        <div className="space-y-4 max-w-lg">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-32 animate-pulse rounded-lg bg-muted" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <h1 className="text-2xl font-semibold mb-6">Settings</h1>

      {message && (
        <div className={cn(
          "mb-4 rounded-md border p-3 text-sm",
          message.type === "success"
            ? "border-green-500/50 bg-green-500/10 text-green-700 dark:text-green-400"
            : "border-destructive/50 bg-destructive/10 text-destructive"
        )}>
          {message.text}
        </div>
      )}

      <div className="max-w-lg space-y-6">
        {/* Appearance */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Appearance</CardTitle>
            <CardDescription>Choose your preferred theme.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex gap-2">
              <button
                className={cn("flex items-center gap-2 rounded-md border p-3 text-sm transition-colors",
                  theme === "light" ? "bg-accent" : "hover:bg-accent")}
                onClick={() => setTheme("light")}
              >
                <Sun className="h-4 w-4" /> Light
              </button>
              <button
                className={cn("flex items-center gap-2 rounded-md border p-3 text-sm transition-colors",
                  theme === "dark" ? "bg-accent" : "hover:bg-accent")}
                onClick={() => setTheme("dark")}
              >
                <Moon className="h-4 w-4" /> Dark
              </button>
              <button
                className={cn("flex items-center gap-2 rounded-md border p-3 text-sm transition-colors",
                  theme === "system" ? "bg-accent" : "hover:bg-accent")}
                onClick={() => setTheme("system")}
              >
                <Monitor className="h-4 w-4" /> System
              </button>
            </div>
          </CardContent>
        </Card>

        {/* OpenAI Configuration */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <Key className="h-4 w-4" /> OpenAI
            </CardTitle>
            <CardDescription>
              Current key: {settings?.openai_api_key ? <span className="font-mono">{settings.openai_api_key}</span> : <span className="text-destructive">Not configured</span>}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="space-y-2">
              <label className="text-sm font-medium">New API Key</label>
              <Input
                type="password"
                placeholder="sk-..."
                value={openaiKey}
                onChange={(e) => setOpenaiKey(e.target.value)}
              />
            </div>
          </CardContent>
        </Card>

        {/* Qdrant Configuration */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <Database className="h-4 w-4" /> Qdrant
            </CardTitle>
            <CardDescription>
              Current URL: {settings?.qdrant_url ? <span className="font-mono">{settings.qdrant_url}</span> : <span className="text-destructive">Not configured</span>}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="space-y-2">
              <label className="text-sm font-medium">Qdrant URL</label>
              <Input
                type="text"
                placeholder="http://localhost:6333"
                value={qdrantUrl}
                onChange={(e) => setQdrantUrl(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">API Key (optional)</label>
              <Input
                type="password"
                placeholder="Leave blank if no auth"
                value={qdrantKey}
                onChange={(e) => setQdrantKey(e.target.value)}
              />
            </div>
          </CardContent>
        </Card>

        {/* Embedding Model */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <Cpu className="h-4 w-4" /> Embedding Model
            </CardTitle>
            <CardDescription>
              Default: {settings?.embedding_model ? <span className="font-mono">{settings.embedding_model}</span> : "text-embedding-3-small"}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="space-y-2">
              <label className="text-sm font-medium">Model</label>
              <Input
                type="text"
                placeholder="text-embedding-3-small"
                value={embeddingModel}
                onChange={(e) => setEmbeddingModel(e.target.value)}
              />
            </div>
          </CardContent>
        </Card>

        {/* Actions */}
        <div className="flex gap-3">
          <Button onClick={handleSave} disabled={saving} className="flex-1">
            {saving ? "Saving..." : "Save Settings"}
          </Button>
          <Button variant="outline" onClick={handleTest} disabled={testing} className="flex-1">
            {testing ? <Loader2 className="h-4 w-4 animate-spin" /> : "Test Connections"}
          </Button>
        </div>

        {/* Test Results */}
        {Object.keys(testResults).length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Connection Test Results</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                {Object.entries(testResults).map(([svc, result]) => (
                  <div key={svc} className="flex items-center justify-between rounded-md border p-3">
                    <span className="text-sm font-medium capitalize">{svc}</span>
                    <div className="flex items-center gap-2">
                      {result.status === "ok" ? (
                        <CheckCircle className="h-4 w-4 text-green-500" />
                      ) : (
                        <XCircle className="h-4 w-4 text-destructive" />
                      )}
                      <span className={cn("text-sm", result.status === "ok" ? "text-green-600" : "text-destructive")}>
                        {result.status}
                        {result.error ? `: ${result.error}` : ""}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        )}

        {/* About */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">About</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 text-sm text-muted-foreground">
              <p>NothingLLM v0.1.0</p>
              <p>A minimal AI workspace manager.</p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
