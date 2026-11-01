"use client";

import { usePathname } from "next/navigation";
import { Sidebar } from "@/components/layout/sidebar";
import { SetupGuard } from "@/components/setup-guard";

/**
 * AppShell wraps all pages.
 * - On /setup and /login: renders full-screen (no sidebar)
 * - On all other routes: renders sidebar + SetupGuard
 */
export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();

  // Public pages render without sidebar or guard
  if (pathname === "/setup" || pathname === "/login") {
    return <>{children}</>;
  }

  return (
    <div className="flex h-screen">
      <Sidebar />
      <main className="flex-1 overflow-auto">
        <SetupGuard>{children}</SetupGuard>
      </main>
    </div>
  );
}