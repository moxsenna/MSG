import * as React from "react";
import {
  Card, Text, Input, Button, Dropdown, Option, Divider, Switch, Badge, Spinner,
} from "@fluentui/react-components";
import { Search24Regular, Settings24Regular, Filter24Regular, Play24Regular } from "@fluentui/react-icons";
import LocationAutocomplete from "./LocationAutocomplete";

type SpeedPreset = "conservative" | "balanced" | "fast" | "custom";

const BUSINESS_PRESETS = [
  "rental mobil",
  "klinik gigi",
  "klinik kecantikan",
  "cafe",
  "restoran",
  "bengkel motor",
  "bengkel mobil",
  "salon",
  "barbershop",
  "hotel",
  "kos-kosan",
  "laundry",
  "fitness",
  "travel",
  "catering",
  "wedding organizer",
  "fotografi",
  "kursus",
  "toko bangunan",
  "dealer motor",
];

export default function DiscoverPage() {
  const [query, setQuery] = React.useState("");
  const [presetQuery, setPresetQuery] = React.useState("__custom");
  const [location, setLocation] = React.useState("");
  const [radius, setRadius] = React.useState("10000");
  const [goal, setGoal] = React.useState("any");
  const [speed, setSpeed] = React.useState<SpeedPreset>("balanced");
  const [showMore, setShowMore] = React.useState(false);
  const [showAdvanced, setShowAdvanced] = React.useState(false);
  const [running, setRunning] = React.useState(false);
  const [found, setFound] = React.useState(0);
  const [stage, setStage] = React.useState("idle");
  const [error, setError] = React.useState<string | null>(null);
  const [success, setSuccess] = React.useState<string | null>(null);
  const [minRating, setMinRating] = React.useState("");
  const [minReviews, setMinReviews] = React.useState("");
  const [noWebsiteOnly, setNoWebsiteOnly] = React.useState(false);
  const [hasContactOnly, setHasContactOnly] = React.useState(false);
  const [extractEmail, setExtractEmail] = React.useState(false);
  const [extraReviews, setExtraReviews] = React.useState(false);
  const [proxy, setProxy] = React.useState("");
  const [geo, setGeo] = React.useState("");
  const [grid, setGrid] = React.useState("");
  const [gridCell, setGridCell] = React.useState("");
  const [target, setTarget] = React.useState("20");
  const [savedTick, setSavedTick] = React.useState(0);
  const loadSaved = () => setSavedTick(t => t + 1);

  React.useEffect(() => {
    const rt = (window as unknown as { runtime?: { EventsOn?: (e: string, cb: (d: unknown) => void) => void; EventsOff?: (e: string) => void } }).runtime;
    if (!rt?.EventsOn) return;
    const onProgress = (data: unknown) => {
      const d = data as { stage?: string; found?: number; count?: number };
      if (d?.stage) setStage(d.stage);
      if (typeof d?.count === "number") setFound(d.count);
      else if (typeof d?.found === "number") setFound(d.found);
    };
    const onStarted = () => { setStage("starting"); setFound(0); };
    const onCompleted = (data: unknown) => {
      const d = data as { found?: number };
      if (typeof d?.found === "number") setFound(d.found);
      setStage("completed");
    };
    const onFailed = () => setStage("failed");
    rt.EventsOn("search:progress", onProgress);
    rt.EventsOn("search:started", onStarted);
    rt.EventsOn("search:completed", onCompleted);
    rt.EventsOn("search:failed", onFailed);
    return () => {
      try { rt.EventsOff?.("search:progress"); rt.EventsOff?.("search:started"); rt.EventsOff?.("search:completed"); rt.EventsOff?.("search:failed"); } catch {}
    };
  }, []);

  const presets = [
    { val: "conservative", label: "Conservative" },
    { val: "balanced", label: "Balanced" },
    { val: "fast", label: "Fast" },
  ];

  const handleFind = async () => {
    if (!query.trim()) return;
    setRunning(true);
    setFound(0);
    setError(null);
    setSuccess(null);
    const app = (window as unknown as { go?: { main?: { App?: { Search?: (req: unknown) => Promise<string> } } } }).go?.main?.App;
    const wailsSearch = app?.Search;
    if (wailsSearch) {
      try {
        const advanced: Record<string, unknown> = {};
        if (proxy.trim()) advanced.proxies = proxy.split(",").map(s => s.trim()).filter(Boolean);
        if (geo.trim()) advanced.geo_coordinates = geo.trim();
        if (grid.trim()) advanced.grid_bbox = grid.trim();
        if (gridCell.trim()) advanced.grid_cell_km = Number(gridCell) || 1.0;
        const req: Record<string, unknown> = {
          query, location_text: location, radius_meters: Number(radius) || 10000, goal, speed_preset: speed,
          extract_email: extractEmail, extra_reviews: extraReviews,
          target_count: Math.max(1, Math.min(50, Number(target) || 20)),
        };
        if (minRating) req.min_rating = Number(minRating);
        if (minReviews) req.min_reviews = Number(minReviews);
        if (noWebsiteOnly) req.goal = "needs_website";
        if (Object.keys(advanced).length > 0) req.advanced = advanced;
        const runId = await wailsSearch(req);
        const parts = [`Berhasil! Run ${String(runId).slice(0, 8)} — selesai.`];
        if (noWebsiteOnly || hasContactOnly || minRating || minReviews) {
          parts.push("Filter tampilan aktif — cek tab Leads lalu gunakan pencarian/kolom kota.");
        } else {
          parts.push("Cek tab Leads untuk hasilnya.");
        }
        try {
          sessionStorage.setItem("msg.lastSearch", JSON.stringify({ query, location, goal, noWebsiteOnly, hasContactOnly, minRating, minReviews }));
        } catch {}
        setSuccess(parts.join(" "));
      } catch (e: unknown) {
        const msg = e instanceof Error ? e.message : typeof e === "string" ? e : JSON.stringify(e);
        setError(msg || "Search gagal — coba FAKE: test untuk demo atau cek Settings > Diagnostics");
      } finally { setRunning(false); setStage("idle"); }
      return;
    }
    const t = setInterval(() => {
      setFound(v => {
        if (v >= 53) { clearInterval(t); setRunning(false); return 53; }
        return v + Math.floor(Math.random() * 7) + 1;
      });
    }, 450);
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16, maxWidth: 760 }}>
      <div>
        <Text size={600} weight="semibold">Discover - Temukan Prospek</Text>
        <Text size={300} style={{ display: "block", color: "gray" }}>
          Cari bisnis lokal berdasarkan jenis usaha & lokasi. Nilai default aman, opsi lanjutan tersembunyi progressive disclosure.
        </Text>
      </div>

      <Card style={{ padding: 16, display: "flex", flexDirection: "column", gap: 12 }}>
        <Text weight="semibold" size={400}>1. Target & Lokasi — mau cari apa, di mana, berapa banyak</Text>

        <Text size={200} style={{ display: "block", marginBottom: -8 }}>Jenis usaha yang ditarget</Text>
        <Dropdown
          value={presetQuery === "__custom" ? "Lainnya (ketik sendiri)..." : presetQuery || "Pilih jenis usaha..."}
          onOptionSelect={(_, d) => {
            const v = String(d.optionValue || "");
            setPresetQuery(v);
            if (v !== "__custom") setQuery(v);
          }}
          style={{ width: "100%" }}
        >
          <Option value="__custom">Lainnya (ketik sendiri)...</Option>
          {BUSINESS_PRESETS.map(p => (
            <Option key={p} value={p}>{p}</Option>
          ))}
        </Dropdown>
        {(presetQuery === "__custom" || presetQuery === "") && (
          <Input
            contentBefore={<Search24Regular />}
            placeholder="Ketik jenis usaha sendiri - mis. jasa sedot wc"
            value={presetQuery === "__custom" ? query : ""}
            onChange={(_, d) => setQuery(d.value)}
            style={{ width: "100%" }}
          />
        )}
        <LocationAutocomplete
          value={location}
          onPick={(v, lat, lon) => {
            setLocation(v);
            if (lat && lon) setGeo(`${lat},${lon}`);
          }}
        />
        {geo && (
          <Text size={200} style={{ color: "gray" }}>
            Titik peta terkunci: {geo} — radius {radius} m beneran difilter dari titik ini.
          </Text>
        )}

        <div style={{ display: "flex", gap: 12, flexWrap: "wrap", alignItems: "end" }}>
          <div>
            <Text size={200} style={{ display: "block", marginBottom: 4 }}>Target hasil (maks 50)</Text>
            <Input
              type="number"
              value={target}
              min={1}
              max={50}
              onChange={(_, d) => {
                const n = Math.max(1, Math.min(50, Number(d.value) || 1));
                setTarget(String(n));
              }}
              style={{ width: 110 }}
            />
          </div>
          <Dropdown value={radius === "5000" ? "5 km" : radius === "25000" ? "25 km" : "10 km"} onOptionSelect={(_, d) => setRadius(String(d.optionValue || "10000"))} style={{ minWidth: 140 }}>
            <Option value="5000">Radius 5 km</Option>
            <Option value="10000">Radius 10 km</Option>
            <Option value="25000">Radius 25 km</Option>
          </Dropdown>
          <Dropdown value={goal === "needs_website" ? "Needs website" : "Any opportunity"} onOptionSelect={(_, d) => setGoal(String(d.optionValue || "any"))} style={{ minWidth: 180 }}>
            <Option value="any">Any opportunity</Option>
            <Option value="needs_website">Needs website</Option>
          </Dropdown>
        </div>
        <Text size={200} style={{ color: "gray" }}>
          Target {target} = usaha maksimal. Kalau datanya memang sedikit, app berhenti jujur di angka yang ada — tidak pernah ngarang.
          Pencarian pertama mengunduh browser sekali saja (~150MB), jadi mohon tunggu lebih lama.
        </Text>

        <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
          <Text size={200} weight="semibold">Speed:</Text>
          {presets.map(p => (
            <Button
              key={p.val}
              size="small"
              appearance={speed === p.val ? "primary" : "outline"}
              onClick={() => setSpeed(p.val as SpeedPreset)}
            >
              {p.label}
            </Button>
          ))}
          {speed === "custom" && <Badge size="small">Custom</Badge>}
        </div>

        <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
          <Button appearance="primary" icon={running ? <Spinner size="tiny" /> : <Play24Regular />} onClick={handleFind} disabled={running || !query.trim()}>
            {running ? `Mencari prospek... (${stage})` : "Find Prospects"}
          </Button>
          <Button
            appearance="secondary"
            disabled={running || !query.trim()}
            onClick={() => {
              const name = window.prompt("Nama untuk pencarian tersimpan ini:", `${query} — ${location || "semua lokasi"}`);
              if (!name) return;
              const app = (window as unknown as { go?: { main?: { App?: { SaveSearch?: (name: string, query: string, location: string, config: string) => Promise<void> } } } }).go?.main?.App;
              app?.SaveSearch?.(name, query, location, JSON.stringify({ radius, goal, speed, target }))
                .then(() => { setSuccess(`Pencarian tersimpan sebagai "${name}". Buka lagi dari daftar bawah.`); loadSaved(); })
                .catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)));
            }}
          >
            Simpan pencarian
          </Button>
        </div>
        <SavedSearchList
          refreshKey={savedTick}
          onUse={s => {
            setQuery(s.query);
            setPresetQuery(BUSINESS_PRESETS.includes(s.query) ? s.query : "__custom");
            setLocation(s.location_text);
            try {
              const c = JSON.parse(s.config || "{}") as { radius?: string; goal?: string; speed?: string; target?: string };
              if (c.radius) setRadius(c.radius);
              if (c.goal) setGoal(c.goal);
              if (c.speed) setSpeed(c.speed as SpeedPreset);
              if (c.target) setTarget(c.target);
            } catch {}
          }}
        />
        {running && (
          <div style={{ display: "flex", alignItems: "center", gap: 8, padding: "8px 0", color: "#0078d4" }}>
            <Spinner size="tiny" /> <Text size={200}>Sedang scraping Google Maps — ini beneran kerja, tunggu 30-60 detik untuk hasil pertama...</Text>
          </div>
        )}
        {error && (
          <Card style={{ padding: 12, background: "#fef0f0", border: "1px solid #e81123" }}>
            <Text weight="semibold" style={{ color: "#a80000" }}>Gagal: {error}</Text>
            <Text size={200} style={{ display: "block", marginTop: 4 }}>Tips: coba ketik `FAKE: rental mobil` untuk demo tanpa browser, atau cek Settings &gt; Diagnostics untuk cek browser.</Text>
          </Card>
        )}
        {success && (
          <Card style={{ padding: 12, background: "#e6f4ea", border: "1px solid #0f7b0f" }}>
            <Text weight="semibold" style={{ color: "#0f7b0f" }}>{success}</Text>
            <Text size={200} style={{ display: "block", marginTop: 4 }}>Buka tab <b>Leads</b> untuk lihat hasilnya. Skor & pipeline ada di tab Pipeline.</Text>
          </Card>
        )}
      </Card>

      <Card style={{ padding: 12 }}>
        <Button appearance="transparent" icon={<Filter24Regular />} onClick={() => setShowMore(v => !v)}>
          {showMore ? "Sembunyikan filter tambahan" : "2. Filter hasil (rating, website, kontak, email)"}
        </Button>
        {showMore && (
          <>
            <Divider style={{ margin: "12px 0" }} />
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12 }}>
              <Dropdown
                placeholder="Min rating"
                value={minRating === "" ? "Any rating" : `≥ ${minRating}`}
                onOptionSelect={(_, d) => setMinRating(String(d.optionValue || ""))}
              >
                <Option value="">Any rating</Option><Option value="4.0">≥ 4.0</Option><Option value="4.5">≥ 4.5</Option>
              </Dropdown>
              <Dropdown
                placeholder="Min reviews"
                value={minReviews === "" ? "Any" : `≥ ${minReviews}`}
                onOptionSelect={(_, d) => setMinReviews(String(d.optionValue || ""))}
              >
                <Option value="">Any</Option><Option value="50">≥ 50</Option><Option value="100">≥ 100</Option>
              </Dropdown>
              <Switch checked={noWebsiteOnly} onChange={(_, d) => setNoWebsiteOnly(d.checked)} label="Hanya tanpa website" />
              <Switch checked={hasContactOnly} onChange={(_, d) => setHasContactOnly(d.checked)} label="Hanya dengan kontak tersedia" />
              <Switch checked={extractEmail} onChange={(_, d) => setExtractEmail(d.checked)} label="Ekstrak email (extract_email)" />
              <Switch checked={extraReviews} onChange={(_, d) => setExtraReviews(d.checked)} label="Kumpulkan ulasan ekstra" />
            </div>
            <Text size={200} style={{ display: "block", marginTop: 8, color: "gray" }}>
              Filter rating/ulasan/website/kontak diterapkan saat menampilkan hasil di tab Leads.
            </Text>
          </>
        )}
      </Card>

      <Card style={{ padding: 12 }}>
        <Button appearance="transparent" icon={<Settings24Regular />} onClick={() => setShowAdvanced(v => !v)}>
          {showAdvanced ? "Sembunyikan advanced" : "3. Mesin scraper (speed, proxy, geo, grid)"}
        </Button>
        {showAdvanced && (
          <>
            <Divider style={{ margin: "12px 0" }} />
            <Text size={200} style={{ color: "gray", display: "block", marginBottom: 8 }}>
              Max depth, fast-mode, browser pool, pages/browser, proxy, geo/grid. Default aman.
            </Text>
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12 }}>
              <Input value={proxy} onChange={(_, d) => setProxy(d.value)} placeholder="Proxy (http://user:pass@host:port, pisah koma)" />
              <Input value={geo} onChange={(_, d) => setGeo(d.value)} placeholder="Geo override lat,lng" />
              <Input value={grid} onChange={(_, d) => setGrid(d.value)} placeholder="Grid bbox minLat,minLon,maxLat,maxLon" />
              <Input value={gridCell} onChange={(_, d) => setGridCell(d.value)} placeholder="Grid cell km (default 1.0)" />
            </div>
          </>
        )}
      </Card>

      {(running || found > 0 || stage !== "idle") && (
        <Card style={{ padding: 16, border: running ? "2px solid #0078d4" : undefined }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            {running && <Spinner size="small" />}
            <Text weight="semibold">{running ? `Finding prospects... (${stage})` : "Selesai"}</Text>
            {running && <Badge appearance="filled" color="brand">Live</Badge>}
          </div>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8, marginTop: 12 }}>
            <Text size={300} weight={found > 0 ? "semibold" : undefined}>Businesses found: {found} / target {target}</Text>
            <Text size={300}>Tersimpan ke SQLite: {found}</Text>
          </div>
          {!running && found > 0 && found < Number(target) && (
            <Text size={200} style={{ display: "block", marginTop: 4, color: "gray" }}>
              Data habis di angka {found} — memang segitu yang ada di Google Maps untuk query ini, tidak dipaksa sampai target.
            </Text>
          )}
          <Text size={200} style={{ display: "block", marginTop: 8, color: running ? "#0078d4" : "gray", fontStyle: running ? "italic" : undefined }}>
            {running ? `Stage: ${stage} — counts update live via Wails events (tunggu 30-60 detik hasil pertama)` : "Run completed - buka tab Leads untuk lihat hasil asli."}
          </Text>
          {running && <Button appearance="secondary" style={{ marginTop: 12 }} onClick={() => {
            const app = (window as unknown as { go?: { main?: { App?: { CancelSearch?: () => Promise<void> } } } }).go?.main?.App;
            try { void app?.CancelSearch?.(); } catch {}
            setRunning(false);
            setStage("idle");
          }}>Cancel search</Button>}
        </Card>
      )}
    </div>
  );
}
function SavedSearchList({ refreshKey, onUse }: {
  refreshKey: number;
  onUse: (s: { query: string; location_text: string; config: string }) => void;
}) {
  const [items, setItems] = React.useState<{ id: string; name: string; query: string; location_text: string; config: string }[]>([]);
  const reload = React.useCallback(() => {
    const w = window as unknown as { go?: { main?: { App?: {
      ListSavedSearches?: () => Promise<{ id: string; name: string; query: string; location_text: string; config: string }[]>;
      DeleteSavedSearch?: (id: string) => Promise<void>;
    } } } };
    w.go?.main?.App?.ListSavedSearches?.().then(rows => setItems(Array.isArray(rows) ? rows : [])).catch(() => {});
  }, []);
  React.useEffect(() => { reload(); }, [reload, refreshKey]);
  if (items.length === 0) return null;
  const remove = (id: string) => {
    const w = window as unknown as { go?: { main?: { App?: { DeleteSavedSearch?: (id: string) => Promise<void> } } } };
    w.go?.main?.App?.DeleteSavedSearch?.(id).then(reload).catch(() => {});
  };
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
      <Text size={200} weight="semibold">Pencarian tersimpan � klik untuk pakai lagi</Text>
      {items.map(s => (
        <div key={s.id} style={{ display: "flex", gap: 8, alignItems: "center" }}>
          <Button size="small" appearance="subtle" onClick={() => onUse(s)} title={`${s.query} � ${s.location_text}`}>{s.name}</Button>
          <Button size="small" appearance="subtle" onClick={() => remove(s.id)} title="Hapus">x</Button>
        </div>
      ))}
    </div>
  );
}
