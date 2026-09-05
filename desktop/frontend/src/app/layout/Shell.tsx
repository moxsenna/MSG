import * as React from "react";
import { NavLink, useLocation } from "react-router-dom";
import { Button, Text, Badge, Divider } from "@fluentui/react-components";
import {
  Home24Regular, Search24Regular, People24Regular, Board24Regular,
  CalendarClock24Regular, History24Regular, Settings24Regular, PresenceAvailable24Regular,
} from "@fluentui/react-icons";
import Logo from "../../components/Logo";

type Props = { children: React.ReactNode; themeMode: string; onThemeModeChange: (m: "system"|"light"|"dark")=>void };

const items = [
  { to: "/", label: "Home", icon: Home24Regular },
  { to: "/discover", label: "Discover", icon: Search24Regular },
  { to: "/leads", label: "Leads", icon: People24Regular },
  { to: "/pipeline", label: "Pipeline", icon: Board24Regular },
  { to: "/followups", label: "Follow-ups", icon: CalendarClock24Regular },
  { to: "/activity", label: "Activity", icon: History24Regular },
  { to: "/settings", label: "Settings", icon: Settings24Regular },
];

export default function Shell({ children, themeMode, onThemeModeChange }: Props) {
  const loc = useLocation();
  const [collapsed, setCollapsed] = React.useState(false);
  React.useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if (e.ctrlKey && e.key === ",") { e.preventDefault(); window.location.hash = "#/settings"; }
      if (e.key === "Escape") { (document.activeElement as HTMLElement)?.blur(); }
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, []);
  return (
    <div style={{ display: "grid", gridTemplateColumns: collapsed ? "64px 1fr" : "240px 1fr", minHeight: "100vh" }}>
      <nav aria-label="Primary" style={{ borderRight: "1px solid #e0e0e0", padding: 8, display: "flex", flexDirection: "column", gap: 4 }}>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", padding: "8px 4px" }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <Logo size={collapsed ? 32 : 28} />
            {!collapsed && <Text weight="semibold" size={400}>MSG</Text>}
          </div>
          <Button size="small" appearance="transparent" onClick={() => setCollapsed(v=>!v)} aria-label="Toggle navigation">{collapsed ? "»" : "«"}</Button>
        </div>
        <Text size={200} style={{ display: collapsed ? "none" : "block", padding: "0 4px 8px", opacity: 0.7 }}>Local Prospecting</Text>
        {items.map(({ to, label, icon: Icon }) => {
          const active = loc.pathname === to;
          return (
            <NavLink key={to} to={to} style={{ textDecoration: "none" }}>
              <Button appearance={active ? "primary" : "transparent"} icon={<Icon />} style={{ width: "100%", justifyContent: "flex-start" }}>
                {!collapsed && label}
              </Button>
            </NavLink>
          );
        })}
        <Divider style={{ margin: "8px 0" }} />
        <div style={{ display: "flex", alignItems: "center", gap: 8, padding: 8 }}>
          <PresenceAvailable24Regular /> {!collapsed && <Badge appearance="outline" color="success">Engine Ready</Badge>}
        </div>
        {!collapsed && (
          <div style={{ marginTop: "auto", display: "flex", gap: 4, padding: 4 }}>
            <Button size="small" appearance={themeMode==="light"?"primary":"outline"} onClick={()=>onThemeModeChange("light")}>Light</Button>
            <Button size="small" appearance={themeMode==="dark"?"primary":"outline"} onClick={()=>onThemeModeChange("dark")}>Dark</Button>
            <Button size="small" appearance={themeMode==="system"?"primary":"outline"} onClick={()=>onThemeModeChange("system")}>System</Button>
          </div>
        )}
      </nav>
      <main style={{ padding: 16, overflow: "auto" }}>{children}</main>
    </div>
  );
}
