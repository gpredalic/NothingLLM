"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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
  // Step 3: Qdrant
  const [qdrantUrl, setQdrantUrl] = useState("http://localhost:6333");
  const [qdrantKey, setQdrantKey] = useState("");

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

  const handleSaveQdrant = async () => {
    setError("");
    setLoading(true);
    try {
      const res = await fetch("/api/setup/config", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ qdrant_url: qdrantUrl, qdrant_api_key: qdrantKey }),
      });
      if (!res.ok) throw new Error((await res.json()).error || "Failed");
      setStep(4);
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
      const res = await fetch("/api/setup/complete", { method: "POST" });
      if (!res.ok) throw new Error((await res.json()).error || "Failed");
      router.push("/login");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed");
      setLoading(false);
    }
  };

  if (!status) return <div className="flex h-screen items-center justify-center">Loading...</div>;

  return (
    <div className="flex h-screen items-center justify-center bg-background">
      <div className="w-full max-w-md space-y-6">
        <div className="text-center">
          <h1 className="text-2xl font-semibold">NothingLLM Setup</h1>
          <p className="text-sm text-muted-foreground mt-1">First-run configuration wizard</p>
        </div>

        <div className="space-y-4">
          {step === 1 && (
            <div className="space-y-4">
              <h2 className="text-lg font-medium">Create Administrator</h2>
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
              <h2 className="text-lg font-medium">OpenAI Configuration</h2>
              <div className="space-y-2">
                <label className="text-sm font-medium">API Key</label>
                <Input type="password" placeholder="sk-..." value={openaiKey} onChange={(e) => setOpenaiKey(e.target.value)} />
              </div>
              <Button className="w-full" onClick={handleSaveOpenAI} disabled={loading || !openaiKey}>
                {loading ? "..." : "Save & Continue"}
              </Button>
            </div>
          )}
          {step === 3 && (
            <div className="space-y-4">
              <h2 className="text-lg font-medium">Qdrant Configuration</h2>
              <div className="space-y-2">
                <label className="text-sm font-medium">URL</label>
                <Input type="text" value={qdrantUrl} onChange={(e) => setQdrantUrl(e.target.value)} />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">API Key (optional)</label>
                <Input type="password" value={qdrantKey} onChange={(e) => setQdrantKey(e.target.value)} />
              </div>
              <Button className="w-full" onClick={handleSaveQdrant} disabled={loading || !qdrantUrl}>
                {loading ? "..." : "Save & Continue"}
              </Button>
            </div>
          )}
          {step === 4 && (
            <div className="space-y-4">
              <h2 className="text-lg font-medium">Test Connectivity</h2>
              <Button className="w-full" onClick={handleTest} disabled={loading}>
                {loading ? "Testing..." : "Run Tests"}
              </Button>
              {Object.keys(testResults).length > 0 && (
                <div className="space-y-2">
                  {Object.entries(testResults).map(([svc, r]) => (
                    <div key={svc} className="flex items-center justify-between text-sm">
                      <span className="capitalize">{svc}</span>
                      <span className={cn("font-medium", (r as any).status === "ok" ? "text-green-600" : "text-destructive")}>
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
        </div>
        {error && <p className="text-sm text-destructive text-center">{error}</p>}
      </div>
    </div>
  );
}