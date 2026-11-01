"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import {
  LayoutDashboard,
  FolderKanban,
  Database,
  MessageSquare,
  Users,
  Settings,
  Cpu,
  Sun,
  Moon,
  Monitor,
  LogOut,
  PanelLeftClose,
  PanelLeftOpen,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { useTheme } from "@/components/theme-provider";
import { useState } from "react";
import { logout } from "@/lib/api";

const navItems = [
  { href: "/", label: "Dashboard", icon: LayoutDashboard },
  { href: "/workspaces", label: "Workspaces", icon: FolderKanban },
  { href: "/collections", label: "Collections", icon: Database },
  { href: "/chat", label: "Chat", icon: MessageSquare },
  { href: "/users", label: "Users", icon: Users },
  { href: "/models", label: "Models", icon: Cpu },
  { href: "/settings", label: "Settings", icon: Settings },
];

const themeIcons: Record<string, React.ComponentType<{ className?: string }>> = {
  light: Sun,
  dark: Moon,
  system: Monitor,
};

export function Sidebar() {
  const pathname = usePathname();
  const { theme, setTheme } = useTheme();
  const [open, setOpen] = useState(true);

  // Cycle through themes: light -> dark -> system
  const cycleTheme = () => {
    const next = theme === "light" ? "dark" : theme === "dark" ? "system" : "light";
    setTheme(next as "light" | "dark" | "system");
  };

  const ThemeIcon = themeIcons[theme] || Monitor;

  return (
    <aside className={cn("flex h-full flex-col border-r bg-card transition-all", open ? "w-56" : "w-14")}>
      <div className="flex h-14 items-center border-b px-4">
        <span className={cn("font-semibold text-sm", !open && "hidden")}>NothingLLM</span>
      </div>

      <nav className="flex-1 space-y-1 p-2">
        {navItems.map((item) => {
          const active = pathname === item.href || (item.href !== "/" && pathname.startsWith(item.href));
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                active
                  ? "bg-primary text-primary-foreground"
                  : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              )}
              title={!open ? item.label : undefined}
            >
              <item.icon className="h-4 w-4 shrink-0" />
              {open && <span>{item.label}</span>}
            </Link>
          );
        })}
      </nav>

      <div className="border-t p-3 space-y-2">
        <Button
          variant="ghost"
          size="sm"
          className="w-full justify-start text-muted-foreground"
          onClick={() => setOpen(!open)}
        >
          {open ? <PanelLeftClose className="h-4 w-4" /> : <PanelLeftOpen className="h-4 w-4" />}
          {open && <span className="ml-2">Collapse</span>}
        </Button>
        <div className="flex items-center justify-between gap-2">
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={cycleTheme}
            title={`Theme: ${theme} (click to cycle)`}
          >
            <ThemeIcon className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={async () => {
              await logout();
              window.location.href = "/login";
            }}
            title="Logout"
          >
            <LogOut className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </aside>
  );
}
