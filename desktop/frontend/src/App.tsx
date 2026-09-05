import * as React from "react";
import { FluentProvider, webLightTheme, webDarkTheme, BrandVariants, createLightTheme, createDarkTheme } from "@fluentui/react-components";
import { HashRouter, Routes, Route, Navigate } from "react-router-dom";
import Shell from "./app/layout/Shell";
import Home from "./features/home";
import Discover from "./features/discover";
import Leads from "./features/leads";
import Pipeline from "./features/pipeline";
import FollowUps from "./features/followups";
import Activity from "./features/activity";
import Settings from "./features/settings";
import ErrorBoundary from "./components/ErrorBoundary";
import { Button, Text } from "@fluentui/react-components";
import { useNavigate } from "react-router-dom";

const brand: BrandVariants = {
  10: "#0e2a4d", 20: "#12365f", 30: "#165187", 40: "#1a5fa0", 50: "#1e6cb9",
  60: "#2a7cd0", 70: "#4a8fd6", 80: "#6aa2dc", 90: "#8ab5e2", 100: "#a9c8e8",
  110: "#c9dbe8", 120: "#e8eef2", 130: "#f2f5f7", 140: "#f8fafb", 150: "#ffffff", 160: "#ffffff",
};

const light = createLightTheme(brand);
const dark = createDarkTheme(brand);
dark.colorNeutralBackground1 = "#1a1a1a";

type ThemeMode = "system" | "light" | "dark";

function useThemeMode(): [ThemeMode, React.Dispatch<React.SetStateAction<ThemeMode>>, typeof light] {
  const [mode, setMode] = React.useState<ThemeMode>(() => (localStorage.getItem("msg.theme") as ThemeMode) || "system");
  React.useEffect(() => { localStorage.setItem("msg.theme", mode); }, [mode]);
  const prefersDark = window.matchMedia?.("(prefers-color-scheme: dark)").matches ?? false;
  const resolved = mode === "system" ? (prefersDark ? dark : light) : mode === "dark" ? dark : light;
  // also set webLightTheme/webDarkTheme base if needed
  void webLightTheme; void webDarkTheme;
  return [mode, setMode, resolved];
}

function UpdateBanner() {
  const [latest, setLatest] = React.useState<string | null>(null);
  const [dismissed, setDismissed] = React.useState(false);
  const nav = useNavigate();
  React.useEffect(() => {
    const app = (window as unknown as { go?: { main?: { App?: {
      GetSetting?: (k: string) => Promise<string>;
      CheckForUpdates?: () => Promise<{ available: boolean; latest: string }>;
    } } } }).go?.main?.App;
    if (!app?.GetSetting) return;
    let cancelled = false;
    app.GetSetting("auto_update_check").then(v => {
      if (cancelled || v !== "1") return;
      return app.CheckForUpdates?.().then(r => {
        if (!cancelled && r?.available) setLatest(r.latest);
      }).catch(() => {});
    }).catch(() => {});
    return () => { cancelled = true; };
  }, []);
  if (!latest || dismissed) return null;
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 8, padding: "8px 16px", background: "#e6f4ea", borderBottom: "1px solid #0f7b0f" }}>
      <Text size={200} weight="semibold">Update {latest} tersedia.</Text>
      <Button size="small" appearance="primary" onClick={() => nav("/settings")}>Buka Settings</Button>
      <Button size="small" appearance="subtle" onClick={() => setDismissed(true)}>Nanti</Button>
    </div>
  );
}

export default function App() {
  const [mode, setMode, theme] = useThemeMode();
  return (
    <FluentProvider theme={theme} style={{ minHeight: "100vh" }}>
      <ErrorBoundary>
        <HashRouter>
          <UpdateBanner />
          <Shell themeMode={mode} onThemeModeChange={setMode}>
            <Routes>
              <Route path="/" element={<Home />} />
              <Route path="/discover" element={<Discover />} />
              <Route path="/leads" element={<Leads />} />
              <Route path="/pipeline" element={<Pipeline />} />
              <Route path="/followups" element={<FollowUps />} />
              <Route path="/activity" element={<Activity />} />
              <Route path="/settings" element={<Settings />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </Shell>
        </HashRouter>
      </ErrorBoundary>
    </FluentProvider>
  );
}
