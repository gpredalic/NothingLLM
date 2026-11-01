"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";

/**
 * SetupGuard checks if the backend is in setup mode.
 * If setup is not complete, redirects to /register.
 * Renders children only after confirming setup is complete.
 */
export function SetupGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [state, setState] = useState<"checking" | "ready" | "setup">("checking");

  useEffect(() => {
    let cancelled = false;

    async function check() {
      try {
        const res = await fetch("/api/setup/status", { cache: "no-store" });
        if (!res.ok) return; // Can't reach API, let the page handle it
        const data = await res.json();
        if (cancelled) return;

        if (!data.has_admin) {
          setState("setup");
          router.replace("/setup");
        } else {
          setState("ready");
        }
      } catch {
        // API unreachable — show children (they'll show their own error)
        if (!cancelled) setState("ready");
      }
    }

    check();
    return () => { cancelled = true; };
  }, [router]);

  if (state === "setup") {
    return null; // Redirecting
  }

  if (state === "checking") {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
          <span className="text-sm text-muted-foreground">Loading...</span>
        </div>
      </div>
    );
  }

  return <>{children}</>;
}