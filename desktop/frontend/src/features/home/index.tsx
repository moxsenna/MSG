import * as React from "react";
import { Card, Text, Button, Badge } from "@fluentui/react-components";
import { useNavigate } from "react-router-dom";
import { Search24Regular, People24Regular, Board24Regular, Checkmark24Regular } from "@fluentui/react-icons";

interface HomeLead {
  id: string;
  title: string;
  city: string;
  website: string;
  score?: number;
}

export default function HomePage() {
  const nav = useNavigate();
  const [newLeads, setNewLeads] = React.useState(0);
  const [qualified, setQualified] = React.useState(0);
  const [overdue, setOverdue] = React.useState(0);
  const [replied, setReplied] = React.useState(0);
  const [top, setTop] = React.useState<HomeLead[]>([]);
  const [loaded, setLoaded] = React.useState(false);

  React.useEffect(() => {
    const app = (window as unknown as { go?: { main?: { App?: {
      ListLeads?: (q: unknown) => Promise<[HomeLead[], number]>;
      GetKanbanBoard?: () => Promise<Record<string, { title: string }[]>>;
      ListFollowUps?: () => Promise<{ is_overdue: boolean }[]>;
    } } } }).go?.main?.App;
    if (!app?.ListLeads) return;
    (async () => {
      try {
        const res = await app.ListLeads!({ limit: 50, offset: 0, sort_by: "score" });
        const list: HomeLead[] = Array.isArray(res) ? (res[0] || []) : ((res as { items?: HomeLead[] }).items || []);
        setNewLeads(list.length);
        setTop(list.slice(0, 2));
        if (app.GetKanbanBoard) {
          const board = await app.GetKanbanBoard();
          setQualified(((board as Record<string, unknown[]>).qualified || []).length);
          setReplied(((board as Record<string, unknown[]>).replied || []).length);
        }
        if (app.ListFollowUps) {
          const fu = await app.ListFollowUps();
          const late = (Array.isArray(fu) ? fu : []).filter(f => f.is_overdue).length;
          setOverdue(late);
          if (late > 0) {
            try {
              if ("Notification" in window) {
                if (Notification.permission === "granted") {
                  new Notification(`MSG Desktop: ${late} follow-up terlambat`, { body: "Buka tab Follow-ups sebelum prospek dingin." });
                } else if (Notification.permission === "default") {
                  void Notification.requestPermission();
                }
              }
            } catch {}
          }
        }
      } catch {}
      finally { setLoaded(true); }
    })();
  }, []);

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16, maxWidth: 760 }}>
      <div>
        <Text size={700} weight="semibold">Find and qualify local business opportunities</Text>
        <Text size={300} style={{ display: "block", color: "gray", marginTop: 4 }}>
          Discover → Qualify → Understand → Reach → Track — tanpa Docker, tanpa server, data lokal di %LOCALAPPDATA%/MSG.
        </Text>
      </div>

      <OnboardingCard />
      {loaded && overdue > 0 && (
        <Card style={{ padding: 12, background: "#fef0f0", border: "1px solid #e81123" }}>
          <Text weight="semibold" style={{ color: "#a80000" }}>Ada {overdue} follow-up terlambat — duit jangan sampai hilang.</Text>
          <div style={{ marginTop: 8 }}>
            <Button size="small" appearance="primary" onClick={() => nav("/followups")}>Buka Follow-ups</Button>
          </div>
        </Card>
      )}

      <Card style={{ padding: 16 }}>
        <Text weight="semibold">Find your next client</Text>
        <Text size={300} style={{ display: "block", color: "gray", marginBottom: 12 }}>
          Mulai dari jenis usaha + lokasi. Gunakan Discover untuk pencarian penuh dengan filter & advanced scraper settings.
        </Text>
        <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
          <Button appearance="primary" icon={<Search24Regular />} onClick={() => nav("/discover")}>Open Discover</Button>
          <Button appearance="secondary" icon={<People24Regular />} onClick={() => nav("/leads")}>Open Leads</Button>
          <Button appearance="outline" icon={<Board24Regular />} onClick={() => nav("/pipeline")}>Open Pipeline</Button>
        </div>
      </Card>

      <Card style={{ padding: 16 }}>
        <Text weight="semibold">Daily summary</Text>
        {!loaded && <Text size={200} style={{ color: "gray" }}>Memuat dari SQLite...</Text>}
        <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 12, marginTop: 8 }}>
          <Card style={{ padding: 12 }}><Text size={500} weight="semibold">{newLeads}</Text><Text size={200} style={{ display: "block" }}>New leads</Text></Card>
          <Card style={{ padding: 12 }}><Text size={500} weight="semibold">{qualified}</Text><Text size={200} style={{ display: "block" }}>Qualified</Text></Card>
          <Card style={{ padding: 12 }}><Text size={500} weight="semibold">{overdue}</Text><Text size={200} style={{ display: "block" }}>Overdue follow-ups</Text>{overdue > 0 && <Badge color="danger" style={{ marginTop: 4 }}>{overdue}</Badge>}</Card>
          <Card style={{ padding: 12 }}><Text size={500} weight="semibold">{replied}</Text><Text size={200} style={{ display: "block" }}>Replied</Text></Card>
        </div>
        {loaded && newLeads === 0 && (
          <Text size={200} style={{ display: "block", marginTop: 8, color: "gray" }}>
            Belum ada data. Buka Discover → Find Prospects untuk mengisi leads pertama.
          </Text>
        )}
      </Card>

      <Card style={{ padding: 16 }}>
        <Text weight="semibold">Top opportunities (evidence-first)</Text>
        <div style={{ display: "flex", flexDirection: "column", gap: 8, marginTop: 8 }}>
          {top.map(t => (
            <Card key={t.id} style={{ padding: 12, display: "flex", justifyContent: "space-between", alignItems: "center" }}>
              <div><Text weight="semibold">{t.title} — {t.city || "lokasi scraper"}</Text><Text size={200} style={{ display: "block", color: "gray" }}>{t.website ? t.website : "No website terdeteksi"}</Text></div>
              <Badge color={(t.score || 0) >= 80 ? "success" : "warning"}>{t.score || 0}</Badge>
            </Card>
          ))}
          {loaded && top.length === 0 && (
            <Text size={200} style={{ color: "gray" }}>Belum ada prospek — hasil Discover akan muncul di sini.</Text>
          )}
        </div>
      </Card>

      <WeeklyReportCard />

      <Card style={{ padding: 12, display: "flex", gap: 8, alignItems: "center" }}>
        <Checkmark24Regular style={{ color: "#107c10" }} />
        <Text size={200}>Engine Ready — SQLite WAL, SecureStore, hardened scraper in-process. AI optional, DNC enforced.</Text>
      </Card>
    </div>
  );
}
function OnboardingCard() {
  const [show, setShow] = React.useState(false);
  React.useEffect(() => {
    try {
      if (!localStorage.getItem("msg.onboarded")) setShow(true);
    } catch { setShow(true); }
  }, []);
  if (!show) return null;
  const dismiss = () => {
    try { localStorage.setItem("msg.onboarded", "1"); } catch {}
    setShow(false);
  };
  const steps = [
    ["1. Discover", "Pilih jenis usaha + lokasi + target ? Find Prospects. Pencarian pertama download browser sekali saja (~150MB)."],
    ["2. Leads", "Hasil asli masuk sini + skor + no HP. Centang banyak untuk aksi bulk, buka Detail buat analisa AI & outreach."],
    ["3. Pipeline", "Geser kartu New ? Qualified ? ... ? Won. Catat nominal deal pas menang."],
    ["4. Follow-ups", "Atur tanggal tindak lanjut dari detail lead. Yang merah = terlambat, langsung hubungi."],
  ];
  return (
    <Card style={{ padding: 16, border: "2px solid #0078d4" }}>
      <Text weight="semibold" size={400}>Mulai dalam 4 langkah</Text>
      {steps.map(([t, d]) => (
        <div key={t} style={{ marginTop: 8 }}>
          <Text weight="semibold" size={300}>{t}</Text>
          <Text size={200} style={{ display: "block", color: "gray" }}>{d}</Text>
        </div>
      ))}
      <div style={{ marginTop: 12 }}>
        <Button size="small" appearance="primary" onClick={dismiss}>Mengerti, jangan tampilkan lagi</Button>
      </div>
    </Card>
  );
}
function WeeklyReportCard() {
  const [rep, setRep] = React.useState<{
    new_leads: number; contacted: number; replied: number; won: number;
    won_value: number; lost: number; overdue: number; search_runs: number; activities: number;
  } | null>(null);
  React.useEffect(() => {
    const w = window as unknown as { go?: { main?: { App?: { WeeklyReport?: () => Promise<{
      new_leads: number; contacted: number; replied: number; won: number;
      won_value: number; lost: number; overdue: number; search_runs: number; activities: number;
    }> } } } };
    w.go?.main?.App?.WeeklyReport?.().then(setRep).catch(() => {});
  }, []);
  if (!rep) return null;
  const items: [string, string][] = [
    ["Prospek baru", String(rep.new_leads)],
    ["Dihubungi", String(rep.contacted)],
    ["Dibalas", String(rep.replied)],
    ["Menang", `${rep.won} (Rp ${Math.round(rep.won_value).toLocaleString("id-ID")})`],
    ["Kalah", String(rep.lost)],
    ["Terlambat", String(rep.overdue)],
    ["Pencarian", String(rep.search_runs)],
    ["Aktivitas", String(rep.activities)],
  ];
  return (
    <Card style={{ padding: 16 }}>
      <Text weight="semibold">Laporan 7 hari terakhir</Text>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 12, marginTop: 8 }}>
        {items.map(([label, val]) => (
          <Card key={label} style={{ padding: 12 }}>
            <Text size={500} weight="semibold">{val}</Text>
            <Text size={200} style={{ display: "block" }}>{label}</Text>
          </Card>
        ))}
      </div>
    </Card>
  );
}
