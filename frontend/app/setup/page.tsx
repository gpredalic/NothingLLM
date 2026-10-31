"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { CheckCircle, XCircle, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";

type SetupStatus = {
  complete: boolean;
  has_admin: boolean;
  config: { openai_configured: boolean; qdrant_configured: boolean };
};

export default function SetupPage() {
  const router = useRouter();
  const [status, setStatus] = useState<SetupStatus | null>(null);
  const [step, setStep] = useState(1);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [testResults, setTestResults] = useState<Record<string, any>>({});

  // Step 1: Admin
  const [adminEmail, setAdminEmail] = useState("");
  const [adminPassword, setAdminPassword] = useState("");
  // Step 2: OpenAI
  const [openaiKey, setOpenaiKey] = useState("");

  useEffect(() => {
    fetchStatus();
  }, []);

  const fetchStatus = async () => {
    try {
      const res = await fetch("/api/setup/status");
      const data = await res.json();
      setStatus(data);
      if (data.complete) {
        router.push("/login");
      }
    } catch {}
  };

  const handleCreateAdmin = async () => {
    setError("");
    setLoading(true);
    try {
      const res = await fetch("/api/setup/admin", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: adminEmail, password: adminPassword }),
      });
      if (!res.ok) throw new Error((await res.json()).error || "Failed");
      setStep(2);
      fetchStatus();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed");
    } finally {
      setLoading(false);
    }
  };

  const handleSaveOpenAI = async () => {
    setError("");
    setLoading(true);
    try {
      const res = await fetch("/api/setup/config", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ openai_api_key: openaiKey }),
      });
      if (!res.ok) throw new Error((await res.json()).error || "Failed");
      setStep(3);
      fetchStatus();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed");
    } finally {
      setLoading(false);
    }
  };

  const handleTest = async () => {
    setError("");
    setLoading(true);
    setTestResults({});
    try {
      const res = await fetch("/api/setup/test", { method: "POST" });
      const data = await res.json();
      setTestResults(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Test failed");
    } finally {
      setLoading(false);
    }
  };

  const handleComplete = async () => {
    setError("");
    setLoading(true);
    try {
      const res = await fetch("/api/setup/complete", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
      if (!res.ok) throw new Error((await res.json()).error || "Failed");
      router.push("/login");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <div className="w-full max-w-md space-y-6">
        <div className="text-center">
          <h1 className="text-3xl font-bold">NothingLLM Setup</h1>
          <p className="text-muted-foreground mt-2">Configure your workspace in 3 steps</p>
        </div>

        {/* Progress */}
        <div className="flex items-center justify-center gap-2">
          {["Admin", "OpenAI", "Verify"].map((label, i) => (
            <div key={label} className="flex items-center gap-2">
              <div
                className={cn(
                  "flex h-8 w-8 items-center justify-center rounded-full text-sm font-medium",
                  step > i + 1 ? "bg-green-500 text-white" : step === i + 1 ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"
                )}
              >
                {step > i + 1 ? "✓" : i + 1}
              </div>
              <span className={cn("text-sm", step === i + 1 ? "font-medium" : "text-muted-foreground")}>{label}</span>
              {i < 2 && <div className="h-px w-8 bg-border" />}
            </div>
          ))}
        </div>

        <Card>
          <CardHeader>
            <CardTitle>
              {step === 1 ? "Create Admin Account" : step === 2 ? "OpenAI Configuration" : "Verify Connectivity"}
            </CardTitle>
            <CardDescription>
              {step === 1 ? "Set up the first administrator" : step === 2 ? "Enter your OpenAI API key" : "Test all connections before completing"}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {step === 1 && (
              <div className="space-y-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium">Email</label>
                  <Input type="email" placeholder="admin@example.com" value={adminEmail} onChange={(e) => setAdminEmail(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium">Password</label>
                  <Input type="password" placeholder="min 8 chars" value={adminPassword} onChange={(e) => setAdminPassword(e.target.value)} minLength={8} />
                </div>
                <Button className="w-full" onClick={handleCreateAdmin} disabled={loading || !adminEmail || !adminPassword}>
                  {loading ? "..." : "Create Admin"}
                </Button>
              </div>
            )}
            {step === 2 && (
              <div className="space-y-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium">API Key</label>
                  <Input type="password" placeholder="sk-..." value={openaiKey} onChange={(e) => setOpenaiKey(e.target.value)} />
                </div>
                <p className="text-xs text-muted-foreground">
                  Qdrant is auto-configured for Docker deployments.
                </p>
                <Button className="w-full" onClick={handleSaveOpenAI} disabled={loading || !openaiKey}>
                  {loading ? "..." : "Save & Continue"}
                </Button>
              </div>
            )}
            {step === 3 && (
              <div className="space-y-4">
                <Button className="w-full" onClick={handleTest} disabled={loading}>
                  {loading ? "Testing..." : "Run Tests"}
                </Button>
                {Object.keys(testResults).length > 0 && (
                  <div className="space-y-2">
                    {Object.entries(testResults).map(([svc, r]) => (
                      <div key={svc} className="flex items-center justify-between text-sm">
                        <span className="capitalize">{svc}</span>
                        <span className={cn("flex items-center gap-1 font-medium", (r as any).status === "ok" ? "text-green-600" : "text-destructive")}>
                          {(r as any).status === "ok" ? <CheckCircle className="h-3 w-3" /> : <XCircle className="h-3 w-3" />}
                          {(r as any).status}
                          {(r as any).error ? `: ${(r as any).error}` : ""}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
                <Button className="w-full" onClick={handleComplete} disabled={loading || Object.keys(testResults).length === 0}>
                  {loading ? "..." : "Complete Setup"}
                </Button>
              </div>
            )}
          </CardContent>
        </Card>
        {error && <p className="text-sm text-destructive text-center">{error}</p>}
      </div>
    </div>
  );
}
