"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useTheme } from "@/components/theme-provider";
import { getAdminSettings, updateAdminSettings, getHealth, testConnection, AdminSettings, HealthStatus } from "@/lib/api";
import { toast } from "@/components/ui/toast";
import { Moon, Sun, Monitor, Key, CheckCircle, XCircle, Loader2, Plug } from "lucide-react";
import { cn } from "@/lib/utils";

export default function SettingsPage() {
  const { theme, setTheme } = useTheme();
  const [settings, setSettings] = useState<AdminSettings | null>(null);
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  const [openaiUrl, setOpenaiUrl] = useState("");
  const [openaiKey, setOpenaiKey] = useState("");

  const load = useCallback(() => {
    setLoading(true);
    Promise.all([getAdminSettings(), getHealth().catch(() => null)])
      .then(([s, h]) => {
        setSettings(s);
        setHealth(h);
        setOpenaiUrl(s.openai_base_url || "https://api.openai.com/v1");
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
      getHealth().then(setHealth).catch(() => {});
    } catch (e) {
      setMessage({ type: "error", text: e instanceof Error ? e.message : "Connection test failed" });
    } finally {
      setTesting(false);
    }
  };

  if (loading) {
    return (
      <div className="flex h-full flex-col p-4 sm:p-6">
        <div className="h-8 w-32 animate-pulse rounded bg-muted mb-6" />
        <div className="grid flex-1 gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-32 animate-pulse rounded-lg bg-muted" />
          ))}
        </div>
      </div>
    );
  }

  const openaiStatus = health?.openai?.status || "unknown";
  const bgeM3Status = health?.bge_m3?.status || "unknown";

  return (
    <div className="flex h-full flex-col p-4 sm:p-6">
      <div className="flex items-center justify-between mb-5">
        <h1 className="text-xl font-semibold">Settings</h1>
        <div className="flex items-center gap-2">
          <div className="flex rounded-md border p-0.5">
            {([["light", Sun], ["dark", Moon], ["system", Monitor]] as const).map(([t, Icon]) => (
              <button
                key={t}
                onClick={() => setTheme(t)}
                className={cn(
                  "rounded p-1.5 transition-colors",
                  theme === t ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground"
                )}
                title={t}
              >
                <Icon className="h-3.5 w-3.5" />
              </button>
            ))}
          </div>
          <Button onClick={handleSave} disabled={saving} size="sm" className="h-8">
            {saving && <Loader2 className="h-3.5 w-3.5 animate-spin mr-1.5" />}
            {saving ? "Saving..." : "Save"}
          </Button>
        </div>
      </div>

      {message && (
        <div className={cn(
          "mb-4 rounded-md border px-3 py-2 text-xs",
          message.type === "success"
            ? "border-green-200 bg-green-50 text-green-700"
            : "border-red-200 bg-red-50 text-red-700"
        )}>
          {message.text}
        </div>
      )}

      <div className="grid flex-1 gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 content-start">
        {/* LLM Provider */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm flex items-center justify-between">
              <span className="flex items-center gap-2">
                <Key className="h-3.5 w-3.5" /> LLM Provider
              </span>
              {openaiStatus === "ok" ? (
                <span className="flex items-center gap-1 text-[10px] font-medium text-green-600">
                  <CheckCircle className="h-3 w-3" /> Connected
                </span>
              ) : openaiStatus === "not_configured" ? (
                <span className="text-[10px] text-muted-foreground">Not configured</span>
              ) : (
                <span className="flex items-center gap-1 text-[10px] font-medium text-destructive">
                  <XCircle className="h-3 w-3" /> Down
                </span>
              )}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">Base URL</label>
              <Input
                value={openaiUrl}
                onChange={(e) => setOpenaiUrl(e.target.value)}
                placeholder="https://api.openai.com/v1"
                className="h-8 text-xs"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">API Key</label>
              <Input
                type="password"
                value={openaiKey}
                onChange={(e) => setOpenaiKey(e.target.value)}
                placeholder={settings?.openai_api_key ? "••••••••" : "Enter API key"}
                className="h-8 text-xs"
              />
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={handleTestConnection}
              disabled={testing}
              className="w-full h-7 text-xs"
            >
              {testing ? <Loader2 className="h-3 w-3 animate-spin mr-1.5" /> : <Plug className="h-3 w-3 mr-1.5" />}
              {testing ? "Testing..." : "Test Connection"}
            </Button>
          </CardContent>
        </Card>

        {/* Embeddings */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm flex items-center justify-between">
              <span className="flex items-center gap-2">
                <Plug className="h-3.5 w-3.5" /> Embeddings
              </span>
              {bgeM3Status === "ok" ? (
                <span className="flex items-center gap-1 text-[10px] font-medium text-green-600">
                  <CheckCircle className="h-3 w-3" /> Connected
                </span>
              ) : (
                <span className="flex items-center gap-1 text-[10px] font-medium text-destructive">
                  <XCircle className="h-3 w-3" /> Down
                </span>
              )}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-1">
              <div className="text-xs font-medium">BGE-M3</div>
              <div className="text-[10px] text-muted-foreground">Provider: Local (Docker)</div>
              <div className={cn("text-[10px]", bgeM3Status === "ok" ? "text-green-600" : "text-destructive")}>
                Status: {bgeM3Status === "ok" ? "Connected" : "Disconnected"}
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Services */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">Services</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <div className="flex items-center justify-between text-xs">
                <span>Qdrant</span>
                {health?.qdrant?.status === "ok" ? (
                  <span className="flex items-center gap-1 text-[10px] text-green-600"><CheckCircle className="h-3 w-3" /> OK</span>
                ) : (
                  <span className="flex items-center gap-1 text-[10px] text-destructive"><XCircle className="h-3 w-3" /> Down</span>
                )}
              </div>
              <div className="flex items-center justify-between text-xs">
                <span>PostgreSQL</span>
                {health?.database?.status === "ok" ? (
                  <span className="flex items-center gap-1 text-[10px] text-green-600"><CheckCircle className="h-3 w-3" /> OK</span>
                ) : (
                  <span className="flex items-center gap-1 text-[10px] text-destructive"><XCircle className="h-3 w-3" /> Down</span>
                )}
              </div>
              <div className="flex items-center justify-between text-xs">
                <span>BGE-M3</span>
                {bgeM3Status === "ok" ? (
                  <span className="flex items-center gap-1 text-[10px] text-green-600"><CheckCircle className="h-3 w-3" /> OK</span>
                ) : (
                  <span className="flex items-center gap-1 text-[10px] text-destructive"><XCircle className="h-3 w-3" /> Down</span>
                )}
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
