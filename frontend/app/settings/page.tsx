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
      <div className="p-4 sm:p-6">
        <div className="h-8 w-32 animate-pulse rounded bg-muted mb-6" />
        <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-36 animate-pulse rounded-lg bg-muted" />
          ))}
        </div>
      </div>
    );
  }

  const openaiStatus = health?.openai?.status || "unknown";
  const qdrantStatus = health?.qdrant?.status || "unknown";
  const bgeM3Status = (health as any)?.bge_m3?.status || "unknown";

  return (
    <div className="p-4 sm:p-6">
      <h1 className="text-xl sm:text-2xl font-semibold mb-5">Settings</h1>
      {message && (
        <div className={cn("mb-4 rounded-md border p-3 text-sm", message.type === "success" ? "border-green-300 bg-green-50 text-green-700" : "border-destructive/50 bg-destructive/10 text-destructive")}>
          {message.text}
        </div>
      )}
      <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
        {/* Appearance */}
        <Card className="h-full">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm flex items-center gap-2">
              {theme === "dark" ? <Moon className="h-3.5 w-3.5" /> : theme === "light" ? <Sun className="h-3.5 w-3.5" /> : <Monitor className="h-3.5 w-3.5" />}
              Appearance
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex gap-1.5">
              {(["light", "dark", "system"] as const).map((t) => (
                <Button
                  key={t}
                  variant={theme === t ? "default" : "outline"}
                  size="sm"
                  className="flex-1 h-8 text-xs"
                  onClick={() => setTheme(t)}
                >
                  {t.charAt(0).toUpperCase() + t.slice(1)}
                </Button>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* OpenAI-Compatible API */}
        <Card className="h-full">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm flex items-center gap-2">
              <Key className="h-3.5 w-3.5" /> LLM Provider
            </CardTitle>
            <CardDescription className="text-xs">
              {openaiStatus === "ok" ? (
                <span className="flex items-center gap-1 text-green-600"><CheckCircle className="h-3 w-3" /> Connected</span>
              ) : openaiStatus === "down" ? (
                <span className="flex items-center gap-1 text-destructive"><XCircle className="h-3 w-3" /> Disconnected</span>
              ) : (
                <span className="text-muted-foreground">Not configured</span>
              )}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            <Input
              type="text"
              placeholder="http://host:port/v1"
              value={openaiUrl}
              onChange={(e) => setOpenaiUrl(e.target.value)}
              className="h-8 text-xs"
            />
            <Input
              type="password"
              placeholder={settings?.openai_api_key ? "**** " + (settings.openai_api_key.slice(-4)) : "API Key"}
              value={openaiKey}
              onChange={(e) => setOpenaiKey(e.target.value)}
              className="h-8 text-xs"
            />
            <Button variant="outline" size="sm" onClick={handleTestConnection} disabled={testing} className="w-full h-8 text-xs">
              {testing ? <Loader2 className="h-3 w-3 animate-spin mr-1.5" /> : <Plug className="h-3 w-3 mr-1.5" />}
              {testing ? "Testing..." : "Test Connection"}
            </Button>
          </CardContent>
        </Card>

        {/* Embeddings */}
        <Card className="h-full">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm flex items-center gap-2">
              <Cpu className="h-3.5 w-3.5" /> Embeddings
            </CardTitle>
            <CardDescription className="text-xs">
              {bgeM3Status === "ok" ? (
                <span className="flex items-center gap-1 text-green-600"><CheckCircle className="h-3 w-3" /> Connected</span>
              ) : bgeM3Status === "down" || bgeM3Status === "failed" ? (
                <span className="flex items-center gap-1 text-destructive"><XCircle className="h-3 w-3" /> Disconnected</span>
              ) : (
                <span className="text-muted-foreground">Not configured</span>
              )}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2.5 rounded-md border p-2.5">
              <Cpu className="h-4 w-4 text-muted-foreground shrink-0" />
              <div className="min-w-0">
                <div className="text-xs font-medium">BGE-M3</div>
                <div className="text-[10px] text-muted-foreground truncate">BAAI/bge-m3</div>
              </div>
              {bgeM3Status === "ok" && <CheckCircle className="h-3.5 w-3.5 text-green-500 ml-auto shrink-0" />}
              {(bgeM3Status === "down" || bgeM3Status === "failed") && <XCircle className="h-3.5 w-3.5 text-destructive ml-auto shrink-0" />}
            </div>
          </CardContent>
        </Card>

        {/* Services */}
        <Card className="h-full sm:col-span-2 lg:col-span-2">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm flex items-center gap-2">
              <Database className="h-3.5 w-3.5" /> Services
            </CardTitle>
            <CardDescription className="text-xs">Internal service status</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-2 grid-cols-1 sm:grid-cols-3">
              <div className="flex items-center justify-between rounded-md border px-3 py-2">
                <span className="text-xs font-medium">Qdrant</span>
                {qdrantStatus === "ok" ? (
                  <span className="flex items-center gap-1 text-[10px] text-green-600"><CheckCircle className="h-3 w-3" /> Connected</span>
                ) : (
                  <span className="flex items-center gap-1 text-[10px] text-destructive"><XCircle className="h-3 w-3" /> Down</span>
                )}
              </div>
              <div className="flex items-center justify-between rounded-md border px-3 py-2">
                <span className="text-xs font-medium">PostgreSQL</span>
                {health?.database?.status === "ok" ? (
                  <span className="flex items-center gap-1 text-[10px] text-green-600"><CheckCircle className="h-3 w-3" /> Connected</span>
                ) : (
                  <span className="flex items-center gap-1 text-[10px] text-destructive"><XCircle className="h-3 w-3" /> Down</span>
                )}
              </div>
              <div className="flex items-center justify-between rounded-md border px-3 py-2">
                <span className="text-xs font-medium">BGE-M3</span>
                {bgeM3Status === "ok" ? (
                  <span className="flex items-center gap-1 text-[10px] text-green-600"><CheckCircle className="h-3 w-3" /> Connected</span>
                ) : (
                  <span className="flex items-center gap-1 text-[10px] text-destructive"><XCircle className="h-3 w-3" /> Down</span>
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Save */}
        <Card className="h-full">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">Save Changes</CardTitle>
          </CardHeader>
          <CardContent>
            <Button onClick={handleSave} disabled={saving} className="w-full h-8 text-xs">
              {saving ? <Loader2 className="h-3 w-3 animate-spin mr-1.5" /> : null}
              {saving ? "Saving..." : "Save Settings"}
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
