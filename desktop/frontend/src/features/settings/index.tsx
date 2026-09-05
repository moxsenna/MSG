import * as React from "react";
import { Card, Text, Input, Button, Switch, Divider, Dropdown, Option } from "@fluentui/react-components";

interface AIKeySlot {
  id: string;
  provider: string;
  label: string;
  key_hint: string;
  base_url?: string;
  model?: string;
  enabled: boolean;
  priority: number;
}

interface AIApp {
  ListAIKeys?: () => Promise<AIKeySlot[]>;
  AddAIKey?: (req: { provider: string; label: string; key: string; base_url: string; model: string }) => Promise<AIKeySlot>;
  UpdateAIKey?: (id: string, label: string, enabled: boolean, baseUrl: string, model: string) => Promise<void>;
  DeleteAIKey?: (id: string) => Promise<void>;
  MoveAIKey?: (id: string, direction: number) => Promise<void>;
  TestAIKey?: (id: string) => Promise<string>;
}

function aiApp(): AIApp | null {
  const w = window as unknown as { go?: { main?: { App?: AIApp } } };
  return w.go?.main?.App || null;
}

interface UpdateInfo {
  available: boolean;
  current: string;
  latest: string;
  notes: string;
}

function UpdateSection() {
  const [version, setVersion] = React.useState("");
  const [autoCheck, setAutoCheck] = React.useState(false);
  const [info, setInfo] = React.useState<UpdateInfo | null>(null);
  const [checking, setChecking] = React.useState(false);
  const [installing, setInstalling] = React.useState(false);
  const [progress, setProgress] = React.useState("");
  const [message, setMessage] = React.useState<string | null>(null);

  const app = () => (window as unknown as { go?: { main?: { App?: {
    GetVersion?: () => Promise<string>;
    GetSetting?: (k: string) => Promise<string>;
    SetSetting?: (k: string, v: string) => Promise<void>;
    CheckForUpdates?: () => Promise<UpdateInfo>;
    DownloadAndInstallUpdate?: () => Promise<string>;
    DownloadInstallerAndMigrate?: () => Promise<string>;
    GetUpdateSource?: () => Promise<string>;
    SetUpdateSource?: (s: string) => Promise<void>;
  } } } }).go?.main?.App;

  React.useEffect(() => {
    const a = app();
    a?.GetVersion?.().then(setVersion).catch(() => {});
    a?.GetSetting?.("auto_update_check").then(v => setAutoCheck(v === "1")).catch(() => {});
    const rt = (window as unknown as { runtime?: { EventsOn?: (e: string, cb: (d: unknown) => void) => void; EventsOff?: (e: string) => void } }).runtime;
    if (!rt?.EventsOn) return;
    const onProg = (d: unknown) => {
      const p = d as { stage?: string; done?: number; total?: number };
      if (p?.stage === "downloading" && typeof p.done === "number") {
        const pct = p.total && p.total > 0 ? ` (${Math.round((p.done / p.total) * 100)}%)` : "";
        setProgress(`Mengunduh...${pct}`);
      } else if (p?.stage) {
        setProgress(p.stage === "verifying" ? "Memverifikasi checksum..." : p.stage === "applying" ? "Memasang update..." : p.stage === "done" ? "Selesai." : p.stage);
      }
    };
    rt.EventsOn("update:progress", onProg);
    return () => { try { rt.EventsOff?.("update:progress"); } catch {} };
  }, []);

  const toggleAuto = (v: boolean) => {
    setAutoCheck(v);
    app()?.SetSetting?.("auto_update_check", v ? "1" : "0").catch(() => {});
  };

  const [source, setSource] = React.useState("github");
  const [folderPath, setFolderPath] = React.useState("D:\\Scrape Data\\Rilis-MSG");
  const [sourceMsg, setSourceMsg] = React.useState<string | null>(null);

  React.useEffect(() => {
    const a = app();
    a?.GetUpdateSource?.().then(s => {
      if (s.startsWith("folder:")) {
        setSource("folder");
        setFolderPath(s.slice("folder:".length));
      } else {
        setSource("github");
      }
    }).catch(() => {});
  }, []);

  const saveSource = () => {
    setSourceMsg(null);
    const a = app();
    if (!a?.SetUpdateSource) {
      setSourceMsg("Backend belum tersedia.");
      return;
    }
    const value = source === "folder" ? `folder:${folderPath.trim()}` : "github";
    a.SetUpdateSource(value)
      .then(() => {
        setSourceMsg(source === "folder" ? `Sumber update: folder lokal ${folderPath.trim()}. Gua taruh rilis baru di situ tiap ada fix.` : "Sumber update: GitHub Releases.");
        setInfo(null);
        setMessage(null);
      })
      .catch((e: unknown) => setSourceMsg(`Gagal: ${e instanceof Error ? e.message : String(e)}`));
  };

  const check = () => {
    setChecking(true);
    setMessage(null);
    setInfo(null);
    app()?.CheckForUpdates?.()
      .then(r => {
        setInfo(r);
        if (!r.available) setMessage(`Sudah versi terbaru (${r.current || version}).`);
      })
      .catch((e: unknown) => setMessage(`Cek gagal: ${e instanceof Error ? e.message : String(e)}`))
      .finally(() => setChecking(false));
  };

  const install = () => {
    setInstalling(true);
    setProgress("Memulai...");
    setMessage(null);
    setMigrateMsg(null);
    app()?.DownloadAndInstallUpdate?.()
      .then(r => { setMessage(r); setProgress(""); check(); })
      .catch((e: unknown) => {
        const msg = e instanceof Error ? e.message : String(e);
        setMessage(`Update gagal: ${msg}`);
        setProgress("");
        if (/access is denied|permission denied|admin/i.test(msg)) {
          setMigrateMsg("Kelihatannya aplikasi terinstall di folder yang butuh Admin (installasi lama). Pakai tombol Migrasi di bawah — sekali saja, data aman.");
        }
      })
      .finally(() => setInstalling(false));
  };

  const [migrateMsg, setMigrateMsg] = React.useState<string | null>(null);
  const [migrating, setMigrating] = React.useState(false);

  const migrate = () => {
    if (!window.confirm("Download installer terbaru & pindah ke installasi per-user (tanpa Admin)? Aplikasi akan tertutup dan installer terbuka. Data SQLite tidak hilang.")) return;
    setMigrating(true);
    setMigrateMsg(null);
    const a = app();
    if (!a?.DownloadInstallerAndMigrate) {
      setMigrateMsg("Backend belum tersedia.");
      setMigrating(false);
      return;
    }
    a.DownloadInstallerAndMigrate()
      .then(r => setMigrateMsg(r))
      .catch((e: unknown) => {
        setMigrateMsg(`Migrasi gagal: ${e instanceof Error ? e.message : String(e)}`);
        setMigrating(false);
      });
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
      <Text size={200}>Versi terpasang: <b>{version || "..."}</b></Text>
      <Text weight="semibold" size={200}>Sumber update</Text>
      <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
        <Button size="small" appearance={source === "github" ? "primary" : "outline"} onClick={() => setSource("github")}>GitHub</Button>
        <Button size="small" appearance={source === "folder" ? "primary" : "outline"} onClick={() => setSource("folder")}>Folder lokal</Button>
      </div>
      {source === "folder" && (
        <Input value={folderPath} onChange={(_, d) => setFolderPath(d.value)} placeholder="D:\Scrape Data\Rilis-MSG" />
      )}
      <div style={{ display: "flex", gap: 8 }}>
        <Button size="small" appearance="secondary" onClick={saveSource}>Pakai sumber ini</Button>
      </div>
      {sourceMsg && <Text size={200}>{sourceMsg}</Text>}
      <Switch checked={autoCheck} onChange={(_, d) => toggleAuto(d.checked)} label="Cek update otomatis tiap buka aplikasi" />
      <div style={{ display: "flex", gap: 8 }}>
        <Button size="small" appearance="secondary" disabled={checking} onClick={check}>{checking ? "Mengecek..." : "Check for updates"}</Button>
        {info?.available && (
          <Button size="small" appearance="primary" disabled={installing} onClick={install}>
            {installing ? "Mengupdate..." : `Download & install ${info.latest}`}
          </Button>
        )}
      </div>
      {progress && <Text size={200} style={{ color: "#0078d4" }}>{progress}</Text>}
      {(migrateMsg || /access is denied|permission denied|admin/i.test(message || "")) && (
        <Card style={{ padding: 12, background: "#fff4ce", border: "1px solid #c19c00" }}>
          <Text weight="semibold" size={200}>Installasi lama butuh Admin — pindah sekali saja</Text>
          <Text size={200} style={{ display: "block", marginTop: 4 }}>
            {migrateMsg || "Update langsung gagal karena folder installasi butuh Admin. Migrasi download installer terbaru & pindah ke per-user (tanpa Admin). Data SQLite tidak hilang."}
          </Text>
          <div style={{ marginTop: 8 }}>
            <Button size="small" appearance="primary" disabled={migrating} onClick={migrate}>
              {migrating ? "Menyiapkan migrasi..." : "Download installer & migrasi ke per-user"}
            </Button>
          </div>
        </Card>
      )}
      {info?.available && (
        <Card style={{ padding: 12, background: "#e6f4ea" }}>
          <Text weight="semibold" size={200}>Versi baru tersedia: {info.latest}</Text>
          {info.notes && <Text size={200} style={{ display: "block", marginTop: 4, whiteSpace: "pre-wrap" }}>{info.notes.slice(0, 500)}</Text>}
        </Card>
      )}
      {message && <Text size={200}>{message}</Text>}
    </div>
  );
}

export default function SettingsPage() {
  const [telemetry, setTelemetry] = React.useState(false);
  const [saved, setSaved] = React.useState(false);
  const [backupMsg, setBackupMsg] = React.useState<string | null>(null);
  const [backupBusy, setBackupBusy] = React.useState(false);

  React.useEffect(() => {
    const app = (window as unknown as { go?: { main?: { App?: { GetSetting?: (k: string) => Promise<string> } } } }).go?.main?.App;
    app?.GetSetting?.("telemetry_opt_in").then(v => setTelemetry(v === "1")).catch(() => {});
  }, []);

  const toggleTelemetry = (v: boolean) => {
    setTelemetry(v);
    const app = (window as unknown as { go?: { main?: { App?: { SetSetting?: (k: string, v: string) => Promise<void> } } } }).go?.main?.App;
    app?.SetSetting?.("telemetry_opt_in", v ? "1" : "0").then(() => {
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    }).catch(() => {});
  };

  const handleBackup = () => {
    setBackupBusy(true);
    setBackupMsg(null);
    const app = (window as unknown as { go?: { main?: { App?: { BackupNow?: () => Promise<string> } } } }).go?.main?.App;
    if (!app?.BackupNow) {
      setBackupMsg("Backend belum tersedia.");
      setBackupBusy(false);
      return;
    }
    app.BackupNow()
      .then(p => setBackupMsg(`Backup tersimpan: ${p}`))
      .catch((e: unknown) => setBackupMsg(`Backup gagal: ${e instanceof Error ? e.message : String(e)}`))
      .finally(() => setBackupBusy(false));
  };
  const [slots, setSlots] = React.useState<AIKeySlot[]>([]);
  const [slotsError, setSlotsError] = React.useState<string | null>(null);
  const [provider, setProvider] = React.useState("gemini");
  const [label, setLabel] = React.useState("");
  const [keyInput, setKeyInput] = React.useState("");
  const [baseUrl, setBaseUrl] = React.useState("");
  const [model, setModel] = React.useState("");
  const [formError, setFormError] = React.useState<string | null>(null);
  const [testingId, setTestingId] = React.useState<string | null>(null);
  const [testResult, setTestResult] = React.useState<string | null>(null);

  const reloadSlots = React.useCallback(() => {
    const app = aiApp();
    if (!app?.ListAIKeys) {
      setSlotsError("Backend belum tersedia (jalankan via aplikasi desktop).");
      return;
    }
    app.ListAIKeys()
      .then(rows => { setSlots(Array.isArray(rows) ? rows : []); setSlotsError(null); })
      .catch((e: unknown) => setSlotsError(e instanceof Error ? e.message : String(e)));
  }, []);

  React.useEffect(() => { reloadSlots(); }, [reloadSlots]);

  const handleAddKey = () => {
    setFormError(null);
    const app = aiApp();
    if (!app?.AddAIKey) { setFormError("Backend belum tersedia."); return; }
    if (keyInput.trim().length < 8) { setFormError("Key minimal 8 karakter."); return; }
    if (!label.trim()) { setFormError("Isi label dulu, mis. akun 1."); return; }
    if (provider === "custom" && !baseUrl.trim()) { setFormError("Custom provider wajib isi Base URL."); return; }
    app.AddAIKey({ provider, label: label.trim(), key: keyInput.trim(), base_url: baseUrl.trim(), model: model.trim() })
      .then(() => {
        setKeyInput(""); setLabel(""); setBaseUrl(""); setModel("");
        setSaved(true);
        setTimeout(() => setSaved(false), 2000);
        reloadSlots();
      })
      .catch((e: unknown) => setFormError(e instanceof Error ? e.message : String(e)));
  };

  const handleToggle = (s: AIKeySlot) => {
    aiApp()?.UpdateAIKey?.(s.id, s.label, !s.enabled, s.base_url || "", s.model || "")
      .then(reloadSlots)
      .catch((e: unknown) => setSlotsError(e instanceof Error ? e.message : String(e)));
  };

  const handleMove = (s: AIKeySlot, dir: number) => {
    aiApp()?.MoveAIKey?.(s.id, dir)
      .then(reloadSlots)
      .catch((e: unknown) => setSlotsError(e instanceof Error ? e.message : String(e)));
  };

  const handleDelete = (s: AIKeySlot) => {
    if (!window.confirm(`Hapus key "${s.label}" (${s.provider})? Secret ikut terhapus dari SecureStore.`)) return;
    aiApp()?.DeleteAIKey?.(s.id)
      .then(reloadSlots)
      .catch((e: unknown) => setSlotsError(e instanceof Error ? e.message : String(e)));
  };

  const handleTest = (s: AIKeySlot) => {
    setTestingId(s.id);
    setTestResult(null);
    aiApp()?.TestAIKey?.(s.id)
      .then(r => setTestResult(r))
      .catch((e: unknown) => setTestResult(`Gagal: ${e instanceof Error ? e.message : String(e)}`))
      .finally(() => setTestingId(null));
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16, maxWidth: 640 }}>
      <div>
        <Text size={600} weight="semibold">Pengaturan MSG Desktop</Text>
        <Text size={300} style={{ display: "block", color: "gray" }}>Konfigurasi akuisisi, penyimpanan, dan keamanan aplikasi lokal</Text>
      </div>

      <Card style={{ padding: 16, display: "flex", flexDirection: "column", gap: 12 }}>
        <Text weight="semibold" size={400}>1. Mesin Pengikis (Hardened Scraper)</Text>
        <div>
          <Text size={300}>Kecepatan Bawaan</Text>
          <div style={{ marginTop: 4 }}>
            <Dropdown defaultValue="Balanced (Rekomendasi)">
              <Option value="conservative">Conservative (1 Concurrency)</Option>
              <Option value="balanced">Balanced (2 Concurrency)</Option>
              <Option value="fast">Fast Mode (4 Concurrency)</Option>
            </Dropdown>
          </div>
        </div>
        <Switch label="Ekstrak Email Otomatis dari Website Bisnis" />
        <Switch label="Kumpulkan Ulasan Lengkap (Hingga ~300 ulasan)" />
      </Card>

      <Card style={{ padding: 16, display: "flex", flexDirection: "column", gap: 12 }}>
        <Text weight="semibold" size={400}>2. Kunci AI — Multi Provider & Multi Akun (SecureStore)</Text>
        <Text size={200} style={{ color: "gray" }}>
          Secret tersimpan terenkripsi di Windows DPAPI / SecureStore — SQLite hanya simpan metadata (label, hint ****akhir, urutan).
          Bisa tambah banyak key, mis. 3 akun Gemini berbeda untuk rotasi kuota.
        </Text>

        <Text weight="semibold" size={300}>Tambah key baru</Text>
        <Dropdown value={provider === "gemini" ? "Gemini (resmi)" : provider === "openai" ? "OpenAI (resmi)" : "Custom (OpenAI-compatible)"} onOptionSelect={(_, d) => setProvider(String(d.optionValue || "gemini"))} style={{ minWidth: 220 }}>
          <Option value="gemini">Gemini (resmi)</Option>
          <Option value="openai">OpenAI (resmi)</Option>
          <Option value="custom">Custom (OpenAI-compatible)</Option>
        </Dropdown>
        <Input value={label} onChange={(_, d) => setLabel(d.value)} placeholder="Label — contoh: gemini akun 1" />
        <Input type="password" value={keyInput} onChange={(_, d) => setKeyInput(d.value)} placeholder="Tempel API key di sini (tidak pernah ditampilkan lagi)..." />
        {(provider === "custom" || provider === "openai") && (
          <Input value={baseUrl} onChange={(_, d) => setBaseUrl(d.value)} placeholder={provider === "custom" ? "Base URL wajib — contoh: https://gateway.contoh.com/v1" : "Base URL opsional (kosongkan = api.openai.com)"} />
        )}
        <Input value={model} onChange={(_, d) => setModel(d.value)} placeholder="Model opsional — kosongkan = default (gemini-2.0-flash / gpt-4o-mini)" />
        {formError && <Text size={200} style={{ color: "#a80000" }}>{formError}</Text>}
        <div style={{ display: "flex", gap: 8 }}>
          <Button appearance="primary" onClick={handleAddKey}>Simpan Key Aman</Button>
        </div>

        <Divider />
        <Text weight="semibold" size={300}>Urutan fallback — dicoba dari nomor 1 ke bawah</Text>
        <Text size={200} style={{ color: "gray" }}>
          Key yang kuotanya habis / error otomatis dilewati (cooldown 60 detik) ke nomor berikutnya. Nonaktifkan key yang musiman tanpa menghapusnya.
        </Text>
        {slotsError && <Text size={200} style={{ color: "#a80000" }}>{slotsError}</Text>}
        {slots.length === 0 && !slotsError && (
          <Text size={200} style={{ color: "gray" }}>Belum ada key. AI nonaktif sampai minimal 1 key ditambahkan.</Text>
        )}
        {slots.map((s, i) => (
          <Card key={s.id} style={{ padding: 12, display: "flex", flexDirection: "column", gap: 8, opacity: s.enabled ? 1 : 0.6 }}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
              <Text weight="semibold">#{i + 1} {s.label} <Text size={200} style={{ color: "gray" }}>({s.provider}{s.model ? ` • ${s.model}` : ""} • {s.key_hint})</Text></Text>
              <div style={{ display: "flex", gap: 4 }}>
                <Button size="small" appearance="subtle" disabled={i === 0} onClick={() => handleMove(s, -1)} title="Naikkan prioritas">▲</Button>
                <Button size="small" appearance="subtle" disabled={i === slots.length - 1} onClick={() => handleMove(s, 1)} title="Turunkan prioritas">▼</Button>
              </div>
            </div>
            {s.base_url && <Text size={200} style={{ color: "gray" }}>{s.base_url}</Text>}
            <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
              <Button size="small" appearance={s.enabled ? "secondary" : "primary"} onClick={() => handleToggle(s)}>{s.enabled ? "Nonaktifkan" : "Aktifkan"}</Button>
              <Button size="small" appearance="secondary" disabled={testingId === s.id} onClick={() => handleTest(s)}>{testingId === s.id ? "Mengetes..." : "Test key"}</Button>
              <Button size="small" appearance="subtle" onClick={() => handleDelete(s)}>Hapus</Button>
            </div>
          </Card>
        ))}
        {testResult && (
          <Card style={{ padding: 12, background: testResult.startsWith("OK") ? "#e6f4ea" : "#fef0f0" }}>
            <Text size={200}>{testResult}</Text>
          </Card>
        )}
      </Card>

      <Card style={{ padding: 16, display: "flex", flexDirection: "column", gap: 12 }}>
        <Text weight="semibold" size={400}>3. Update Aplikasi (otomatis, tanpa reinstall)</Text>
        <Text size={200} style={{ color: "gray" }}>
          Update diambil dari GitHub Releases moxsenna/msg-desktop, diverifikasi checksum, lalu dipasang otomatis. Install per-user jadi tidak perlu Admin.
        </Text>
        <UpdateSection />
      </Card>

      <Card style={{ padding: 16, display: "flex", flexDirection: "column", gap: 12 }}>
        <Text weight="semibold" size={400}>4. Privasi & Telemetri</Text>
        <Switch
          checked={telemetry}
          onChange={(_, d) => toggleTelemetry(d.checked)}
          label="Kirim telemetri anonim untuk perbaikan aplikasi (Bawaan: Mati / Opt-in)"
        />
        <Divider />
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <Text weight="semibold">Database Lokal</Text>
            <Text size={200} style={{ display: "block", color: "gray" }}>Lokasi: %LOCALAPPDATA%/MSG/msg.db</Text>
          </div>
          <Button appearance="secondary" disabled={backupBusy} onClick={handleBackup}>{backupBusy ? "Mencadangkan..." : "Cadangkan Database (.zip)"}</Button>
        </div>
        {backupMsg && <Text size={200}>{backupMsg}</Text>}
      </Card>

      {saved && (
        <Card style={{ backgroundColor: "#dff6dd", padding: 12 }}>
          <Text style={{ color: "#107c10" }}>Pengaturan berhasil disimpan!</Text>
        </Card>
      )}
    </div>
  );
}
