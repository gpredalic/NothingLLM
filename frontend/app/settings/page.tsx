"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useTheme } from "@/components/theme-provider";
import { getAdminSettings, updateAdminSettings, getHealth, AdminSettings, HealthStatus } from "@/lib/api";
import { toast } from "@/components/ui/toast";
import { Moon, Sun, Monitor, Key, Database, Cpu, CheckCircle, XCircle, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";

export default function SettingsPage() {
  const { theme, setTheme } = useTheme();
  const [settings, setSettings] = useState<AdminSettings | null>(null);
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  // Form state
  const [openaiKey, setOpenaiKey] = useState("");
  const [embeddingProvider, setEmbeddingProvider] = useState("openai");

  const load = useCallback(() => {
    setLoading(true);
    Promise.all([getAdminSettings(), getHealth().catch(() => null)])
      .then(([s, h]) => {
        setSettings(s);
        setHealth(h);
        const em = s.embedding_model || "text-embedding-3-small";
        setEmbeddingProvider(em.includes("bge") ? "bge-m3" : "openai");
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
      updates.embedding_model = embeddingProvider === "bge-m3" ? "BAAI/bge-m3" : "text-embedding-3-small";
      await updateAdminSettings(updates);
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
      <div className="space-y-6 max-w-lg">
        {/* Theme */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Appearance</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex gap-2">
              {(["light", "dark", "system"] as const).map((t) => (
                <Button key={t} variant={theme === t ? "default" : "outline"} size="sm" onClick={() => setTheme(t)}>
                  {t === "light" ? <Sun className="h-4 w-4" /> : t === "dark" ? <Moon className="h-4 w-4" /> : <Monitor className="h-4 w-4" />}
                  <span className="ml-2 capitalize">{t}</span>
                </Button>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* OpenAI */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <Key className="h-4 w-4" /> OpenAI
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
              <label className="text-sm font-medium">API Key</label>
              <Input
                type="password"
                placeholder={settings?.openai_api_key ? "**** " + (settings.openai_api_key.slice(-4)) : "sk-..."}
                value={openaiKey}
                onChange={(e) => setOpenaiKey(e.target.value)}
              />
            </div>
          </CardContent>
        </Card>

        {/* Embedding Provider */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <Cpu className="h-4 w-4" /> Embeddings
            </CardTitle>
            <CardDescription>Select embedding provider</CardDescription>
          </CardHeader>
          <CardContent>
            <Select value={embeddingProvider} onValueChange={setEmbeddingProvider}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="openai">OpenAI Embeddings (text-embedding-3-small)</SelectItem>
                <SelectItem value="bge-m3">BGE-M3 (BAAI/bge-m3)</SelectItem>
              </SelectContent>
            </Select>
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
            </div>
          </CardContent>
        </Card>

        {/* Save */}
        <Button onClick={handleSave} disabled={saving} className="w-full">
          {saving ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
          {saving ? "Saving..." : "Save Settings"}
        </Button>
      </div>
    </div>
  );
}
