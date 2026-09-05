import * as React from "react";
import {
  Card, Text, Input, Button, Badge, Table, TableHeader, TableRow, TableHeaderCell,
  TableBody, TableCell, Drawer, DrawerHeader, DrawerHeaderTitle, DrawerBody,
  Divider, Textarea, Dropdown, Option, TabList, Tab
} from "@fluentui/react-components";
import { Search24Regular, Filter24Regular, Dismiss24Regular, Globe24Regular, Call24Regular } from "@fluentui/react-icons";
import OutreachPanel from "./OutreachPanel";

export interface LeadItem {
  id: string;
  title: string;
  category: string;
  city: string;
  address?: string;
  maps_link?: string;
  website: string;
  phone: string;
  stage: string;
  score?: number;
  dnc: boolean;
  is_mobile?: boolean;
  follow_up_at?: string;
}

export interface LeadDetail extends LeadItem {
  address: string;
  maps_link: string;
  socials: string[];
  review_rating: number;
  review_count: number;
}

export default function LeadsPage() {
  const [search, setSearch] = React.useState("");
  const [selectedLead, setSelectedLead] = React.useState<LeadItem | null>(null);
  const [activeTab, setActiveTab] = React.useState<string>("overview");
  const [noteText, setNoteText] = React.useState("");
  const [analysis, setAnalysis] = React.useState<{
    summary: string; opportunity_type: string; sales_angle: string;
    suggested_offer: string; reasons: string[]; confidence: number;
    used_label: string; used_provider: string;
  } | null>(null);
  const [analyzing, setAnalyzing] = React.useState(false);
  const [analysisError, setAnalysisError] = React.useState<string | null>(null);

  const [detail, setDetail] = React.useState<LeadDetail | null>(null);

  React.useEffect(() => {
    setAnalysis(null);
    setAnalysisError(null);
    setDetail(null);
    if (!selectedLead) return;
    const w = window as unknown as { go?: { main?: { App?: { GetLead?: (id: string) => Promise<LeadDetail> } } } };
    w.go?.main?.App?.GetLead?.(selectedLead.id)
      .then(d => setDetail(d))
      .catch(() => {});
  }, [selectedLead?.id]);

  const handleAnalyze = () => {
    if (!selectedLead || analyzing) return;
    const w = window as unknown as { go?: { main?: { App?: { AnalyzeLead?: (id: string) => Promise<{
      summary: string; opportunity_type: string; sales_angle: string;
      suggested_offer: string; reasons: string[]; confidence: number;
      used_label: string; used_provider: string;
    }> } } } };
    const fn = w.go?.main?.App?.AnalyzeLead;
    if (!fn) {
      setAnalysisError("Backend belum tersedia (jalankan via aplikasi desktop).");
      return;
    }
    setAnalyzing(true);
    setAnalysisError(null);
    fn(selectedLead.id)
      .then(r => setAnalysis(r))
      .catch((e: unknown) => setAnalysisError(e instanceof Error ? e.message : String(e)))
      .finally(() => setAnalyzing(false));
  };

  const [leads, setLeads] = React.useState<LeadItem[]>([]);
  const [total, setTotal] = React.useState(0);
  const [loading, setLoading] = React.useState(false);
  const [loadError, setLoadError] = React.useState<string | null>(null);
  const [page, setPage] = React.useState(0);
  const [stageFilter, setStageFilter] = React.useState("");
  const [activeFilter, setActiveFilter] = React.useState<{ minRating: string; minReviews: string; noWebsiteOnly: boolean; hasContactOnly: boolean; goal: string } | null>(null);
  const pageSize = 50;

  React.useEffect(() => {
    try {
      const raw = sessionStorage.getItem("msg.lastSearch");
      if (raw) {
        const f = JSON.parse(raw) as { minRating?: string; minReviews?: string; noWebsiteOnly?: boolean; hasContactOnly?: boolean; goal?: string };
        if (f && (f.minRating || f.minReviews || f.noWebsiteOnly || f.hasContactOnly || (f.goal && f.goal !== "any"))) {
          setActiveFilter({ minRating: f.minRating || "", minReviews: f.minReviews || "", noWebsiteOnly: !!f.noWebsiteOnly, hasContactOnly: !!f.hasContactOnly, goal: f.goal || "any" });
        } else {
          setActiveFilter(null);
        }
      }
    } catch {}
  }, []);

  const reload = React.useCallback(() => {
    const wails = (window as unknown as { go?: { main?: { App?: { ListLeads?: (q: unknown) => Promise<{ items: LeadItem[]; total: number } | [LeadItem[], number]> } } } }).go;
    const listLeads = wails?.main?.App?.ListLeads;
    if (!listLeads) {
      setLoadError("Backend belum tersedia (jalankan via aplikasi desktop, bukan preview web).");
      return;
    }
    setLoading(true);
    setLoadError(null);
    const q: Record<string, unknown> = { search, limit: pageSize, offset: page * pageSize, sort_by: "updated" };
    if (stageFilter) q.stage = stageFilter;
    if (activeFilter?.minRating) q.min_rating = Number(activeFilter.minRating);
    if (activeFilter?.minReviews) q.min_reviews = Number(activeFilter.minReviews);
    if (activeFilter?.noWebsiteOnly || activeFilter?.goal === "needs_website") q.has_website = false;
    if (activeFilter?.hasContactOnly) q.has_contact = true;
    listLeads(q)
      .then((res) => {
        if (Array.isArray(res)) {
          const items = res[0] || [];
          setLeads(items);
          setTotal(typeof res[1] === "number" ? res[1] : items.length);
        } else if (res && typeof res === "object") {
          const items = (res as { items?: LeadItem[] }).items || [];
          const t = (res as { total?: number }).total;
          setLeads(items);
          setTotal(typeof t === "number" ? t : items.length);
        } else {
          setLeads([]);
          setTotal(0);
        }
      })
      .catch((e: unknown) => {
        const msg = e instanceof Error ? e.message : String(e);
        setLoadError(msg || "Gagal memuat leads dari SQLite.");
      })
      .finally(() => setLoading(false));
  }, [search, page, stageFilter, activeFilter]);

  React.useEffect(() => {
    reload();
  }, [reload]);

  React.useEffect(() => {
    const onFocus = () => reload();
    window.addEventListener("focus", onFocus);
    return () => window.removeEventListener("focus", onFocus);
  }, [reload]);

  const filtered = leads.filter(l =>
    l.title.toLowerCase().includes(search.toLowerCase()) ||
    l.city.toLowerCase().includes(search.toLowerCase()) ||
    l.category.toLowerCase().includes(search.toLowerCase())
  );

  const [selected, setSelected] = React.useState<Set<string>>(new Set());
  const [bulkBusy, setBulkBusy] = React.useState(false);
  const [bulkMsg, setBulkMsg] = React.useState<string | null>(null);
  const [bulkStage, setBulkStage] = React.useState("qualified");
  const [bulkDate, setBulkDate] = React.useState("");

  React.useEffect(() => {
    setSelected(new Set());
  }, [search, page, stageFilter]);

  const toggleOne = (id: string) => {
    setSelected(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleAll = () => {
    if (selected.size === filtered.length && filtered.length > 0) {
      setSelected(new Set());
    } else {
      setSelected(new Set(filtered.map(l => l.id)));
    }
  };

  const bulkApp = () => (window as unknown as { go?: { main?: { App?: {
    UpdateLeadStage?: (id: string, stage: string) => Promise<void>;
    SetLeadDNC?: (id: string, dnc: boolean) => Promise<void>;
    SetLeadFollowUp?: (id: string, at: string) => Promise<void>;
    DeleteLead?: (id: string) => Promise<void>;
    AnalyzeLead?: (id: string) => Promise<{ summary: string }>;
  } } } }).go?.main?.App;

  const bulkAnalyze = () => {
    const fn = bulkApp()?.AnalyzeLead;
    if (!fn) { setBulkMsg("Backend belum tersedia."); return; }
    const ids = filtered.filter(l => selected.has(l.id)).map(l => l.id);
    if (ids.length === 0) return;
    if (!window.confirm(`Jalankan Analisa AI untuk ${ids.length} lead? Tiap lead 1 panggilan AI sesuai urutan fallback.`)) return;
    setBulkBusy(true);
    setBulkMsg(null);
    let ok = 0;
    const errors: string[] = [];
    (async () => {
      for (let i = 0; i < ids.length; i++) {
        setBulkMsg(`Menganalisa ${i + 1}/${ids.length}...`);
        try {
          await fn(ids[i]);
          ok++;
        } catch (e: unknown) {
          errors.push(`${ids[i].slice(0, 6)}: ${e instanceof Error ? e.message : String(e)}`.slice(0, 80));
        }
      }
      setBulkBusy(false);
      if (errors.length === 0) {
        setBulkMsg(`Analisa AI: ${ok}/${ids.length} berhasil. Buka tiap lead untuk lihat hasilnya.`);
      } else {
        setBulkMsg(`Analisa AI: ${ok}/${ids.length} berhasil, ${errors.length} gagal (${errors.slice(0, 2).join("; ")}${errors.length > 2 ? "..." : ""}).`);
      }
      reload();
    })();
  };

  const runBulk = async (label: string, fn: (id: string) => Promise<void>, onDone?: (ids: string[]) => void) => {
    const ids = filtered.filter(l => selected.has(l.id)).map(l => l.id);
    if (ids.length === 0) return;
    setBulkBusy(true);
    setBulkMsg(null);
    let ok = 0;
    const errors: string[] = [];
    for (const id of ids) {
      try {
        await fn(id);
        ok++;
      } catch (e: unknown) {
        errors.push(`${id.slice(0, 6)}: ${e instanceof Error ? e.message : String(e)}`);
      }
    }
    setBulkBusy(false);
    if (onDone) onDone(ids);
    if (errors.length === 0) {
      setBulkMsg(`${label}: ${ok}/${ids.length} berhasil.`);
    } else {
      setBulkMsg(`${label}: ${ok}/${ids.length} berhasil, ${errors.length} gagal (${errors.slice(0, 2).join("; ")}${errors.length > 2 ? "..." : ""}).`);
    }
    reload();
  };

  const bulkSetStage = () => {
    const fn = bulkApp()?.UpdateLeadStage;
    if (!fn) { setBulkMsg("Backend belum tersedia."); return; }
    const stage = bulkStage;
    void runBulk(`Pindah tahap ke ${stage}`, id => fn(id, stage), ids => {
      setLeads(prev => prev.map(l => (ids.includes(l.id) ? { ...l, stage } : l)));
    });
  };

  const bulkSetDNC = (dnc: boolean) => {
    const fn = bulkApp()?.SetLeadDNC;
    if (!fn) { setBulkMsg("Backend belum tersedia."); return; }
    void runBulk(dnc ? "Tandai DNC" : "Buka blokir DNC", id => fn(id, dnc), ids => {
      setLeads(prev => prev.map(l => (ids.includes(l.id) ? { ...l, dnc } : l)));
    });
  };

  const bulkSetFollowUp = () => {
    const fn = bulkApp()?.SetLeadFollowUp;
    if (!fn) { setBulkMsg("Backend belum tersedia."); return; }
    if (!bulkDate) { setBulkMsg("Isi tanggal follow-up dulu."); return; }
    const at = bulkDate;
    void runBulk(`Jadwal follow-up ${at}`, id => fn(id, at), ids => {
      setLeads(prev => prev.map(l => (ids.includes(l.id) ? { ...l, follow_up_at: at } : l)));
    });
  };

  const bulkDelete = () => {
    const n = selected.size;
    if (n === 0) return;
    if (!window.confirm(`Hapus ${n} lead permanen beserta data CRM-nya? Tidak bisa dibatalkan.`)) return;
    const fn = bulkApp()?.DeleteLead;
    if (!fn) { setBulkMsg("Backend belum tersedia."); return; }
    void runBulk("Hapus", id => fn(id), ids => {
      setLeads(prev => prev.filter(l => !ids.includes(l.id)));
      setTotal(t => Math.max(0, t - ids.length));
      setSelected(new Set());
      if (selectedLead && ids.includes(selectedLead.id)) setSelectedLead(null);
    });
  };

  const bulkExport = () => {
    const rows = filtered.filter(l => selected.has(l.id));
    if (rows.length === 0) return;
    const esc = (v: unknown) => `"${String(v ?? "").replace(/"/g, '""')}"`;
    const csv = [["title", "category", "city", "address", "maps_link", "website", "phone", "stage", "score", "dnc", "follow_up_at"],
      ...rows.map(l => [l.title, l.category, l.city, l.address ?? "", l.maps_link ?? "", l.website, l.phone, l.stage, l.score ?? "", l.dnc ? "1" : "0", l.follow_up_at ?? ""])];
    const blob = new Blob([csv.map(r => r.map(esc).join(",")).join("\n")], { type: "text/csv" });
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = `msg-leads-selected-${rows.length}-${new Date().toISOString().slice(0, 10)}.csv`;
    a.click();
    setTimeout(() => URL.revokeObjectURL(a.href), 5000);
    setBulkMsg(`Export ${rows.length} lead terpilih ke CSV.`);
  };

  const handleStageChange = (newStage: string) => {
    if (!selectedLead) return;
    const w = window as unknown as { go?: { main?: { App?: { UpdateLeadStage?: (id: string, stage: string) => Promise<void> } } } };
    const fn = w.go?.main?.App?.UpdateLeadStage;
    if (fn) {
      fn(selectedLead.id, newStage).then(() => {
        setLeads(prev => prev.map(l => l.id === selectedLead.id ? { ...l, stage: newStage } : l));
        setSelectedLead(prev => prev ? { ...prev, stage: newStage } : null);
      }).catch(() => {});
      return;
    }
    setLeads(prev => prev.map(l => l.id === selectedLead.id ? { ...l, stage: newStage } : l));
    setSelectedLead(prev => prev ? { ...prev, stage: newStage } : null);
  };

  const handleToggleDNC = () => {
    if (!selectedLead) return;
    const nextDNC = !selectedLead.dnc;
    const w = window as unknown as { go?: { main?: { App?: { SetLeadDNC?: (id: string, dnc: boolean) => Promise<void> } } } };
    const fn = w.go?.main?.App?.SetLeadDNC;
    if (fn) {
      fn(selectedLead.id, nextDNC).then(() => {
        setLeads(prev => prev.map(l => l.id === selectedLead.id ? { ...l, dnc: nextDNC } : l));
        setSelectedLead(prev => prev ? { ...prev, dnc: nextDNC } : null);
      }).catch(() => {});
      return;
    }
    setLeads(prev => prev.map(l => l.id === selectedLead.id ? { ...l, dnc: nextDNC } : l));
    setSelectedLead(prev => prev ? { ...prev, dnc: nextDNC } : null);
  };

  const handleAddNote = () => {
    if (!selectedLead || !noteText.trim()) return;
    const w = window as unknown as { go?: { main?: { App?: { AddLeadNote?: (id: string, body: string) => Promise<void> } } } };
    const fn = w.go?.main?.App?.AddLeadNote;
    if (fn) { fn(selectedLead.id, noteText).then(() => setNoteText("")).catch(() => {}); return; }
    setNoteText("");
  };

  const handleDeleteLead = () => {
    if (!selectedLead) return;
    if (!window.confirm(`Hapus lead "${selectedLead.title}" beserta semua data CRM-nya? Tidak bisa dibatalkan.`)) return;
    const w = window as unknown as { go?: { main?: { App?: { DeleteLead?: (id: string) => Promise<void> } } } };
    const fn = w.go?.main?.App?.DeleteLead;
    if (!fn) return;
    fn(selectedLead.id)
      .then(() => {
        setLeads(prev => prev.filter(l => l.id !== selectedLead.id));
        setTotal(t => Math.max(0, t - 1));
        setSelectedLead(null);
      })
      .catch((e: unknown) => setLoadError(e instanceof Error ? e.message : String(e)));
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <div>
          <Text size={600} weight="semibold">Leads Workspace</Text>
          <Text size={300} style={{ display: "block", color: "gray" }}>Kelola dan kualifikasi prospek bisnis lokal Anda</Text>
        </div>
        <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
          <Dropdown
            placeholder="Semua tahap"
            value={stageFilter === "" ? "Semua tahap" : stageFilter}
            onOptionSelect={(_, d) => { setStageFilter(String(d.optionValue || "")); setPage(0); }}
            style={{ minWidth: 150 }}
          >
            <Option value="">Semua tahap</Option>
            <Option value="new">New</Option>
            <Option value="qualified">Qualified</Option>
            <Option value="shortlisted">Shortlisted</Option>
            <Option value="contacted">Contacted</Option>
            <Option value="replied">Replied</Option>
            <Option value="meeting">Meeting</Option>
            <Option value="proposal">Proposal</Option>
            <Option value="won">Won</Option>
            <Option value="lost">Lost</Option>
          </Dropdown>
          {activeFilter && (
            <Badge appearance="filled" color="brand">
              Filter Discover: {[(activeFilter.goal === "needs_website" || activeFilter.noWebsiteOnly) && "tanpa website", activeFilter.minRating && `≥${activeFilter.minRating}★`, activeFilter.minReviews && `≥${activeFilter.minReviews} ulasan`, activeFilter.hasContactOnly && "ada kontak"].filter(Boolean).join(" • ")}
            </Badge>
          )}
          {activeFilter && (
            <Button size="small" appearance="subtle" onClick={() => { setActiveFilter(null); try { sessionStorage.removeItem("msg.lastSearch"); } catch {} }}>Hapus filter</Button>
          )}
          <Button appearance="secondary" onClick={reload} disabled={loading}>{loading ? "Memuat..." : "Muat ulang"}</Button>
          <Button
            appearance="secondary"
            disabled={total === 0 || loading}
            onClick={() => {
              const w = window as unknown as { go?: { main?: { App?: { ListLeads?: (q: unknown) => Promise<{ items: LeadItem[]; total: number }> } } } };
              const fn = w.go?.main?.App?.ListLeads;
              if (!fn) return;
              setLoading(true);
              const all: LeadItem[] = [];
              const pageSizeAll = 200;
              let off = 0;
              const totalPages = Math.max(1, Math.ceil(total / pageSizeAll));
              const step = (o: number): Promise<void> => {
                const q: Record<string, unknown> = { search, limit: pageSizeAll, offset: o, sort_by: "updated" };
                if (stageFilter) q.stage = stageFilter;
                return fn(q).then(res => {
                  const items = res.items || [];
                  all.push(...items);
                  if (all.length < (res.total || 0) && o + pageSizeAll < totalPages * pageSizeAll) {
                    return step(o + pageSizeAll);
                  }
                });
              };
              step(0).then(() => {
                const esc = (v: unknown) => `"${String(v ?? "").replace(/"/g, '""')}"`;
                const rows = [["title", "category", "city", "address", "maps_link", "website", "phone", "stage", "score", "dnc", "follow_up_at"],
                  ...all.map(l => [l.title, l.category, l.city, l.address ?? "", l.maps_link ?? "", l.website, l.phone, l.stage, l.score ?? "", l.dnc ? "1" : "0", l.follow_up_at ?? ""])];
                const blob = new Blob([rows.map(r => r.map(esc).join(",")).join("\n")], { type: "text/csv" });
                const a = document.createElement("a");
                a.href = URL.createObjectURL(blob);
                a.download = `msg-leads-semua-${all.length}-${new Date().toISOString().slice(0, 10)}.csv`;
                a.click();
                setTimeout(() => URL.revokeObjectURL(a.href), 5000);
                setBulkMsg(`Export semua: ${all.length} baris.`);
              }).catch((e: unknown) => {
                setLoadError(e instanceof Error ? e.message : String(e));
              }).finally(() => setLoading(false));
            }}
          >
            Export semua ({total})
          </Button>
          <Button
            appearance="primary"
            disabled={leads.length === 0}
            onClick={() => {
              const esc = (v: unknown) => `"${String(v ?? "").replace(/"/g, '""')}"`;
              const rows = [["title", "category", "city", "address", "maps_link", "website", "phone", "stage", "score", "dnc", "follow_up_at"],
                ...leads.map(l => [l.title, l.category, l.city, l.address ?? "", l.maps_link ?? "", l.website, l.phone, l.stage, l.score ?? "", l.dnc ? "1" : "0", l.follow_up_at ?? ""])];
              const blob = new Blob([rows.map(r => r.map(esc).join(",")).join("\n")], { type: "text/csv" });
              const a = document.createElement("a");
              a.href = URL.createObjectURL(blob);
              a.download = `msg-leads-${new Date().toISOString().slice(0, 10)}.csv`;
              a.click();
              setTimeout(() => URL.revokeObjectURL(a.href), 5000);
            }}
          >
            Export CSV
          </Button>
        </div>
      </div>

      {/* Command Bar */}
      <Card style={{ padding: 12 }}>
        <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
          <Input
            contentBefore={<Search24Regular />}
            placeholder="Cari nama bisnis, kategori, atau kota..."
            value={search}
            onChange={(e, d) => setSearch(d.value)}
            style={{ minWidth: 320 }}
          />
          <Text size={200} style={{ color: "gray" }}>
            {loading ? "Memuat dari SQLite..." : `Menampilkan ${filtered.length} dari ${total} prospek`}
          </Text>
        </div>
        {loadError && (
          <Text size={200} style={{ display: "block", marginTop: 8, color: "#a80000" }}>
            Gagal memuat: {loadError} — pastikan aplikasi dibuka dari installer (bukan preview web) lalu klik Muat ulang.
          </Text>
        )}
        {!loading && !loadError && leads.length === 0 && (
          <Text size={200} style={{ display: "block", marginTop: 8, color: "gray" }}>
            Belum ada leads. Buka tab Discover → jalankan Find Prospects (mis. rental mobil / cirebon) lalu kembali ke sini dan klik Muat ulang.
          </Text>
        )}
      </Card>

      {selected.size > 0 && (
        <Card style={{ padding: 12, border: "2px solid #0078d4" }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
            <Text weight="semibold">{selected.size} terpilih</Text>
            <Dropdown value={bulkStage} onOptionSelect={(_, d) => setBulkStage(String(d.optionValue || "qualified"))} style={{ minWidth: 140 }}>
              <Option value="new">New</Option>
              <Option value="qualified">Qualified</Option>
              <Option value="shortlisted">Shortlisted</Option>
              <Option value="contacted">Contacted</Option>
              <Option value="replied">Replied</Option>
              <Option value="meeting">Meeting</Option>
              <Option value="proposal">Proposal</Option>
              <Option value="won">Won</Option>
              <Option value="lost">Lost</Option>
            </Dropdown>
            <Button size="small" appearance="secondary" disabled={bulkBusy} onClick={bulkSetStage}>Terapkan tahap</Button>
            <Button size="small" appearance="secondary" disabled={bulkBusy} onClick={() => bulkSetDNC(true)}>Tandai DNC</Button>
            <Button size="small" appearance="secondary" disabled={bulkBusy} onClick={() => bulkSetDNC(false)}>Buka DNC</Button>
            <Input type="date" value={bulkDate} onChange={(_, d) => setBulkDate(d.value)} style={{ maxWidth: 160 }} />
            <Button size="small" appearance="secondary" disabled={bulkBusy || !bulkDate} onClick={bulkSetFollowUp}>Jadwalkan</Button>
            <Button size="small" appearance="secondary" disabled={bulkBusy} onClick={bulkExport}>Export terpilih</Button>
            <Button size="small" appearance="primary" disabled={bulkBusy} onClick={bulkAnalyze}>Analisa AI terpilih</Button>
            <Button size="small" appearance="subtle" disabled={bulkBusy} onClick={bulkDelete} style={{ color: "#a80000" }}>Hapus terpilih</Button>
            <Button size="small" appearance="subtle" onClick={() => setSelected(new Set())}>Batal</Button>
          </div>
          {bulkBusy && <Text size={200} style={{ color: "#0078d4" }}>Memproses bulk...</Text>}
          {bulkMsg && <Text size={200}>{bulkMsg}</Text>}
        </Card>
      )}

      {/* Leads Grid */}
      <Card style={{ padding: 0, overflow: "hidden" }}>
        <Table aria-label="Leads table">
          <TableHeader>
            <TableRow>
              <TableHeaderCell>
                <input
                  type="checkbox"
                  aria-label="Pilih semua"
                  checked={filtered.length > 0 && selected.size === filtered.length}
                  onChange={toggleAll}
                />
              </TableHeaderCell>
              <TableHeaderCell>Bisnis</TableHeaderCell>
              <TableHeaderCell>Kategori</TableHeaderCell>
              <TableHeaderCell>Kota</TableHeaderCell>
              <TableHeaderCell>Skor Peluang</TableHeaderCell>
              <TableHeaderCell>Tahap CRM</TableHeaderCell>
              <TableHeaderCell>Kontak</TableHeaderCell>
              <TableHeaderCell>Aksi</TableHeaderCell>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filtered.map(item => (
              <TableRow
                key={item.id}
                onClick={() => setSelectedLead(item)}
                style={{ cursor: "pointer", backgroundColor: selectedLead?.id === item.id ? "rgba(0,120,212,0.1)" : undefined }}
              >
                <TableCell onClick={e => e.stopPropagation()}>
                  <input
                    type="checkbox"
                    aria-label={`Pilih ${item.title}`}
                    checked={selected.has(item.id)}
                    onChange={() => toggleOne(item.id)}
                  />
                </TableCell>
                <TableCell>
                  <Text weight="semibold">{item.title}</Text>
                  {item.dnc && <Badge color="danger" size="small" style={{ marginLeft: 6 }}>DNC</Badge>}
                </TableCell>
                <TableCell>{item.category}</TableCell>
                <TableCell>{item.city}</TableCell>
                <TableCell>
                  <Badge color={item.score && item.score >= 80 ? "success" : item.score && item.score >= 50 ? "warning" : "informative"}>
                    {item.score || 0} / 100
                  </Badge>
                </TableCell>
                <TableCell>
                  <Badge appearance="outline">{item.stage.toUpperCase()}</Badge>
                </TableCell>
                <TableCell>
                  <div style={{ display: "flex", gap: 8, alignItems: "center", whiteSpace: "nowrap" }}>
                    {item.phone ? (
                      <>
                        <Text size={200} style={{ whiteSpace: "nowrap" }}>{item.phone}</Text>
                        {item.is_mobile === false ? (
                          <Badge size="small" appearance="outline" title="Nomor kabel/rumah — WhatsApp kemungkinan gagal">kabel</Badge>
                        ) : (
                          <Button
                            size="small"
                            appearance="subtle"
                            icon={<Call24Regular style={{ color: "green" }} />}
                            title={`Chat WhatsApp ${item.phone}`}
                            onClick={e => {
                              e.stopPropagation();
                              const w = window as unknown as { go?: { main?: { App?: { OpenWhatsAppURL?: (id: string, draftId: string) => Promise<string> } } } };
                              w.go?.main?.App?.OpenWhatsAppURL?.(item.id, "")
                                .then(url => {
                                  const rt = (window as unknown as { runtime?: { BrowserOpenURL?: (u: string) => void } }).runtime;
                                  if (rt?.BrowserOpenURL) rt.BrowserOpenURL(url);
                                  else window.open(url, "_blank");
                                })
                                .catch(() => {});
                            }}
                          />
                        )}
                      </>
                    ) : (
                      <Text size={200} style={{ color: "gray" }}>—</Text>
                    )}
                    {item.website && <Globe24Regular style={{ fontSize: 16, color: "#0078d4" }} />}
                  </div>
                </TableCell>
                <TableCell>
                  <Button size="small" appearance="subtle" onClick={(e) => { e.stopPropagation(); setSelectedLead(item); }}>Detail</Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Card>
      <div style={{ display: "flex", gap: 8, alignItems: "center", justifyContent: "flex-end" }}>
        <Text size={200} style={{ color: "gray" }}>Halaman {page + 1}{total > 0 ? ` dari ${Math.max(1, Math.ceil(total / pageSize))}` : ""} • {total} prospek</Text>
        <Button size="small" appearance="secondary" disabled={page === 0 || loading} onClick={() => setPage(p => Math.max(0, p - 1))}>Sebelumnya</Button>
        <Button size="small" appearance="secondary" disabled={loading || (page + 1) * pageSize >= total} onClick={() => setPage(p => p + 1)}>Berikutnya</Button>
      </div>

      {/* Right Drawer Detail */}
      {selectedLead && (
        <Drawer position="end" open={!!selectedLead} onOpenChange={(_, { open }) => !open && setSelectedLead(null)} style={{ width: 480 }}>
          <DrawerHeader>
            <DrawerHeaderTitle
              action={<Button appearance="subtle" icon={<Dismiss24Regular />} onClick={() => setSelectedLead(null)} />}
            >
              <Text weight="semibold" size={500}>{selectedLead.title}</Text>
            </DrawerHeaderTitle>
          </DrawerHeader>
          <DrawerBody style={{ display: "flex", flexDirection: "column", gap: 16 }}>
            <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
              <Badge color={selectedLead.score && selectedLead.score >= 80 ? "success" : "warning"} size="large">
                Skor Peluang: {selectedLead.score || 0}
              </Badge>
              <ScoreWhy leadId={selectedLead.id} />
              {selectedLead.dnc ? (
                <Badge color="danger">Do Not Contact (DNC)</Badge>
              ) : (
                <Badge color="success">Siap Dikontak</Badge>
              )}
            </div>

            <TabList selectedValue={activeTab} onTabSelect={(_, d) => setActiveTab(d.value as string)}>
              <Tab value="overview">Ringkasan</Tab>
              <Tab value="outreach">Pesan Outreach</Tab>
              <Tab value="notes">Catatan & CRM</Tab>
            </TabList>

            <Divider />

            {activeTab === "overview" && (
              <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
                <div>
                  <Text weight="semibold">Kategori & Lokasi</Text>
                  <Text size={300} style={{ display: "block" }}>{selectedLead.category} • {selectedLead.city}</Text>
                </div>
                <div>
                  <Text weight="semibold">Alamat</Text>
                  <Text size={300} style={{ display: "block" }}>{detail?.address || selectedLead.address || "Belum ada alamat detail"}</Text>
                  {(detail?.maps_link || selectedLead.maps_link) && (
                    <a href={detail?.maps_link || selectedLead.maps_link} target="_blank" rel="noreferrer" style={{ fontSize: 12 }}>Buka di Google Maps</a>
                  )}
                </div>
                <div>
                  <Text weight="semibold">Media Sosial</Text>
                  {(detail?.socials || []).length > 0 ? (
                    <div style={{ display: "flex", flexDirection: "column", gap: 4, marginTop: 4 }}>
                      {(detail?.socials || []).map(s => (
                        <a key={s} href={s} target="_blank" rel="noreferrer" style={{ fontSize: 12 }}>{s.replace(/^https?:\/\/(www\.)?/, "")}</a>
                      ))}
                    </div>
                  ) : (
                    <Text size={300} style={{ display: "block", color: "gray" }}>Tidak terdeteksi — sosmed diambil dari website bisnis saat scrape.</Text>
                  )}
                </div>
                <div>
                  <Text weight="semibold">Website</Text>
                  <Text size={300} style={{ display: "block" }}>
                    {selectedLead.website ? (
                      <a href={selectedLead.website} target="_blank" rel="noreferrer">{selectedLead.website}</a>
                    ) : (
                      <Text style={{ color: "#d13438" }}>Tidak ada website terdeteksi (Peluang Utama)</Text>
                    )}
                  </Text>
                </div>
                <div>
                  <Text weight="semibold">Nomor Telepon</Text>
                  <Text size={300} style={{ display: "block" }}>{selectedLead.phone || "Tidak tersedia"}</Text>
                </div>
                <AuditPanel leadId={selectedLead.id} website={selectedLead.website} />
                <Divider />
                <div>
                  <Text weight="semibold">Tahap CRM Saat Ini</Text>
                  <div style={{ marginTop: 6 }}>
                    <Dropdown
                      value={selectedLead.stage}
                      onOptionSelect={(_, d) => d.optionValue && handleStageChange(d.optionValue)}
                    >
                      <Option value="new">New</Option>
                      <Option value="qualified">Qualified</Option>
                      <Option value="shortlisted">Shortlisted</Option>
                      <Option value="contacted">Contacted</Option>
                      <Option value="replied">Replied</Option>
                      <Option value="meeting">Meeting</Option>
                      <Option value="proposal">Proposal</Option>
                      <Option value="won">Won</Option>
                      <Option value="lost">Lost</Option>
                    </Dropdown>
                  </div>
                </div>
                <Button appearance={selectedLead.dnc ? "primary" : "secondary"} onClick={handleToggleDNC}>
                  {selectedLead.dnc ? "Buka Blokir DNC" : "Tandai Do Not Contact (DNC)"}
                </Button>
                <Button appearance="subtle" onClick={handleDeleteLead} style={{ color: "#a80000" }}>
                  Hapus Lead Permanen
                </Button>
                <Divider />
                <div>
                  <Text weight="semibold">Analisa AI (chain fallback)</Text>
                  <Text size={200} style={{ display: "block", color: "gray", marginBottom: 8 }}>
                    Jalanin key AI sesuai urutan fallback di Settings. Hasil tersimpan ke riwayat.
                  </Text>
                  <Button appearance="primary" disabled={analyzing} onClick={handleAnalyze}>
                    {analyzing ? "Menganalisa..." : "Analisa AI"}
                  </Button>
                  {analysisError && (
                    <Card style={{ marginTop: 8, padding: 12, background: "#fef0f0", border: "1px solid #e81123" }}>
                      <Text size={200} style={{ color: "#a80000" }}>{analysisError}</Text>
                      <Text size={200} style={{ display: "block", marginTop: 4, color: "gray" }}>
                        Belum ada key? Tambah di Settings → Kunci AI. Key kuota habis otomatis lanjut ke key berikut.
                      </Text>
                    </Card>
                  )}
                  {analysis && (
                    <Card style={{ marginTop: 8, padding: 12, background: "#e6f4ea", border: "1px solid #0f7b0f" }}>
                      <Text weight="semibold">{analysis.summary}</Text>
                      <div style={{ display: "flex", gap: 6, marginTop: 6, flexWrap: "wrap" }}>
                        <Badge appearance="outline">{analysis.opportunity_type || "opportunity"}</Badge>
                        <Badge appearance="outline">via {analysis.used_label || analysis.used_provider}</Badge>
                        <Badge appearance="outline">confidence {Math.round((analysis.confidence || 0) * 100)}%</Badge>
                      </div>
                      <Text size={200} style={{ display: "block", marginTop: 8 }}><b>Angle:</b> {analysis.sales_angle}</Text>
                      <Text size={200} style={{ display: "block", marginTop: 4 }}><b>Tawaran:</b> {analysis.suggested_offer}</Text>
                      {(analysis.reasons || []).length > 0 && (
                        <ul style={{ margin: "8px 0 0", paddingLeft: 18 }}>
                          {(analysis.reasons || []).map((r, i) => <li key={i}><Text size={200}>{r}</Text></li>)}
                        </ul>
                      )}
                    </Card>
                  )}
                </div>
              </div>
            )}

            {activeTab === "outreach" && (
              <OutreachPanel lead={selectedLead} />
            )}

            {activeTab === "notes" && (
              <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
                <Text weight="semibold">Tambah Catatan</Text>
                <Textarea
                  value={noteText}
                  onChange={(e, d) => setNoteText(d.value)}
                  placeholder="Tulis catatan prospek, hasil telpon, dll..."
                />
                <Button appearance="primary" onClick={handleAddNote}>Simpan Catatan</Button>
                <Divider />
                <DealOutcomePanel
                  leadId={selectedLead.id}
                  stage={selectedLead.stage}
                  wonValue={(selectedLead as { won_value?: number }).won_value}
                  lostReason={(selectedLead as { lost_reason?: string }).lost_reason}
                  onSaved={(stage, value, reason) => {
                    setSelectedLead(prev => prev ? ({ ...prev, stage, won_value: value, lost_reason: reason } as typeof prev) : null);
                    setLeads(prev => prev.map(l => (l.id === selectedLead.id ? { ...l, stage } : l)));
                  }}
                />
                <Divider />
                <TagPanel leadId={selectedLead.id} />
                <Divider />
                <Text weight="semibold">Jadwal Follow-up</Text>
                <Text size={200} style={{ color: "gray" }}>
                  {selectedLead.follow_up_at ? `Terjadwal: ${selectedLead.follow_up_at}` : "Belum terjadwal — isi tanggal agar muncul di tab Follow-ups."}
                </Text>
                <Input
                  type="date"
                  id="followup-date"
                  defaultValue={selectedLead.follow_up_at ? selectedLead.follow_up_at.slice(0, 10) : ""}
                />
                <Button
                  appearance="secondary"
                  onClick={() => {
                    const el = document.getElementById("followup-date") as HTMLInputElement | null;
                    const v = el?.value || "";
                    if (!v || !selectedLead) return;
                    const w = window as unknown as { go?: { main?: { App?: { SetLeadFollowUp?: (id: string, at: string) => Promise<void> } } } };
                    const fn = w.go?.main?.App?.SetLeadFollowUp;
                    if (!fn) return;
                    fn(selectedLead.id, v).then(() => {
                      setSelectedLead(prev => prev ? { ...prev, follow_up_at: v } : null);
                    }).catch(() => {});
                  }}
                >
                  Simpan Jadwal
                </Button>
              </div>
            )}
          </DrawerBody>
        </Drawer>
      )}
    </div>
  );
}
function AuditPanel({ leadId, website }: { leadId: string; website: string }) {
  const [busy, setBusy] = React.useState(false);
  const [result, setResult] = React.useState<{
    reachable: boolean; https: boolean; has_contact: boolean; has_whatsapp: boolean;
    has_form: boolean; title: string; final_url: string; error_code: string; new_score: number;
  } | null>(null);
  const [err, setErr] = React.useState<string | null>(null);

  if (!website) return null;

  const run = () => {
    setBusy(true);
    setErr(null);
    const w = window as unknown as { go?: { main?: { App?: { RunWebsiteAudit?: (id: string) => Promise<{
      reachable: boolean; https: boolean; has_contact: boolean; has_whatsapp: boolean;
      has_form: boolean; title: string; final_url: string; error_code: string; new_score: number;
    }> } } } };
    w.go?.main?.App?.RunWebsiteAudit?.(leadId)
      .then(r => setResult(r))
      .catch((e: unknown) => setErr(e instanceof Error ? e.message : String(e)))
      .finally(() => setBusy(false));
  };

  return (
    <div>
      <Text weight="semibold">Audit Website (bukti pitch)</Text>
      <div style={{ marginTop: 6 }}>
        <Button size="small" appearance="secondary" disabled={busy} onClick={run}>
          {busy ? "Meng-audit..." : "Audit website sekarang"}
        </Button>
      </div>
      {err && <Text size={200} style={{ display: "block", marginTop: 6, color: "#a80000" }}>{err}</Text>}
      {result && (
        <Card style={{ marginTop: 8, padding: 12 }}>
          <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
            <Badge appearance="outline" color={result.reachable ? "success" : "danger"}>{result.reachable ? "Online" : "Mati"}</Badge>
            <Badge appearance="outline" color={result.https ? "success" : "warning"}>{result.https ? "HTTPS" : "Tanpa HTTPS"}</Badge>
            <Badge appearance="outline" color={result.has_contact ? "success" : "warning"}>{result.has_contact ? "Ada kontak" : "Tanpa kontak"}</Badge>
            <Badge appearance="outline" color={result.has_whatsapp ? "success" : "informative"}>{result.has_whatsapp ? "Ada WA" : "Tanpa WA"}</Badge>
            <Badge appearance="outline">{result.has_form ? "Ada form" : "Tanpa form"}</Badge>
            {result.new_score > 0 && <Badge appearance="filled" color="brand">Skor baru: {result.new_score}</Badge>}
          </div>
          {result.title && <Text size={200} style={{ display: "block", marginTop: 6 }}>{result.title}</Text>}
          {result.error_code && <Text size={200} style={{ display: "block", color: "#a80000" }}>{result.error_code}</Text>}
          <Text size={200} style={{ display: "block", marginTop: 4, color: "gray" }}>
            Pakai temuan ini buat pitch: tanpa HTTPS / tanpa kontak / tanpa WA = alasan kuat mereka butuh website baru.
          </Text>
        </Card>
      )}
    </div>
  );
}
function DealOutcomePanel({ leadId, stage, wonValue, lostReason, onSaved }: {
  leadId: string; stage: string; wonValue?: number; lostReason?: string;
  onSaved: (stage: string, value: number, reason: string) => void;
}) {
  const [value, setValue] = React.useState(wonValue ? String(wonValue) : "");
  const [reason, setReason] = React.useState(lostReason || "");
  const [msg, setMsg] = React.useState<string | null>(null);
  const [busy, setBusy] = React.useState(false);
  const save = (s: string) => {
    setBusy(true);
    setMsg(null);
    const w = window as unknown as { go?: { main?: { App?: { SetDealOutcome?: (id: string, stage: string, value: number, reason: string) => Promise<void> } } } };
    const fn = w.go?.main?.App?.SetDealOutcome;
    if (!fn) { setMsg("Backend belum tersedia."); setBusy(false); return; }
    fn(leadId, s, Number(value) || 0, reason)
      .then(() => { onSaved(s, Number(value) || 0, reason); setMsg(s === "won" ? "Deal menang tercatat." : "Alasan kalah tercatat."); })
      .catch((e: unknown) => setMsg("Gagal: " + (e instanceof Error ? e.message : String(e))))
      .finally(() => setBusy(false));
  };
  return (
    <div>
      <Text weight="semibold">Hasil Deal (closing)</Text>
      {(stage === "won" && wonValue) ? <Text size={200} style={{ display: "block", color: "#0f7b0f" }}>Menang: Rp {Number(wonValue).toLocaleString("id-ID")}</Text> : null}
      {(stage === "lost" && lostReason) ? <Text size={200} style={{ display: "block", color: "#a80000" }}>Kalah: {lostReason}</Text> : null}
      <div style={{ display: "flex", gap: 8, marginTop: 6, flexWrap: "wrap" }}>
        <Input type="number" min={0} value={value} onChange={(_, d) => setValue(d.value)} placeholder="Nilai deal Rp" style={{ maxWidth: 170 }} />
        <Input value={reason} onChange={(_, d) => setReason(d.value)} placeholder="Alasan (khusus kalah)" style={{ flex: 1, minWidth: 170 }} />
      </div>
      <div style={{ display: "flex", gap: 8, marginTop: 8 }}>
        <Button size="small" appearance="primary" disabled={busy} onClick={() => save("won")}>Tandai Menang</Button>
        <Button size="small" appearance="secondary" disabled={busy} onClick={() => save("lost")}>Tandai Kalah</Button>
      </div>
      {msg && <Text size={200} style={{ display: "block", marginTop: 4 }}>{msg}</Text>}
    </div>
  );
}

function TagPanel({ leadId }: { leadId: string }) {
  const [tags, setTags] = React.useState<string[]>([]);
  const [input, setInput] = React.useState("");
  const [msg, setMsg] = React.useState<string | null>(null);
  const refresh = React.useCallback(() => {
    const w = window as unknown as { go?: { main?: { App?: { GetLead?: (id: string) => Promise<{ tags?: string[] }> } } } };
    w.go?.main?.App?.GetLead?.(leadId).then(d => setTags(d.tags || [])).catch(() => {});
  }, [leadId]);
  React.useEffect(() => { refresh(); }, [refresh]);
  const add = () => {
    const t = input.trim();
    if (!t) return;
    const w = window as unknown as { go?: { main?: { App?: { AddLeadTag?: (id: string, tag: string) => Promise<void> } } } };
    const fn = w.go?.main?.App?.AddLeadTag;
    if (!fn) { setMsg("Backend belum tersedia."); return; }
    fn(leadId, t)
      .then(() => { setInput(""); refresh(); })
      .catch((e: unknown) => setMsg("Gagal: " + (e instanceof Error ? e.message : String(e))));
  };
  return (
    <div>
      <Text weight="semibold">Label / Tag</Text>
      <div style={{ display: "flex", gap: 6, flexWrap: "wrap", marginTop: 6 }}>
        {tags.length === 0 ? <Text size={200} style={{ color: "gray" }}>Belum ada label.</Text> : null}
        {tags.map(t => <Badge key={t} appearance="outline" size="small">{t}</Badge>)}
      </div>
      <div style={{ display: "flex", gap: 8, marginTop: 8 }}>
        <Input value={input} onChange={(_, d) => setInput(d.value)} placeholder="mis. prioritas, resto, followup-minggu" style={{ flex: 1 }} />
        <Button size="small" appearance="secondary" onClick={add}>Tambah</Button>
      </div>
      {msg && <Text size={200} style={{ display: "block", marginTop: 4 }}>{msg}</Text>}
    </div>
  );
}
function ScoreWhy({ leadId }: { leadId: string }) {
  const [open, setOpen] = React.useState(false);
  const [detail, setDetail] = React.useState<{
    overall: number; website: number; app: number; priority: string; confidence: number;
    evidence: { label: string; points: number; detail: string }[];
  } | null>(null);
  const [err, setErr] = React.useState<string | null>(null);
  const toggle = () => {
    if (open || detail) { setOpen(!open); return; }
    const w = window as unknown as { go?: { main?: { App?: { GetScoreDetail?: (id: string) => Promise<{
      overall: number; website: number; app: number; priority: string; confidence: number;
      evidence: { label: string; points: number; detail: string }[];
    }> } } } };
    w.go?.main?.App?.GetScoreDetail?.(leadId)
      .then(d => { setDetail(d); setOpen(true); })
      .catch((e: unknown) => setErr(e instanceof Error ? e.message : String(e)));
  };
  return (
    <span>
      <Button size="small" appearance="subtle" onClick={toggle}>Kenapa skor ini?</Button>
      {err ? <Text size={200} style={{ color: "#a80000" }}>{err}</Text> : null}
      {open && detail ? (
        <Card style={{ marginTop: 8, padding: 12 }}>
          <Text size={200} weight="semibold">Website {detail.website} � Sistem {detail.app} � {detail.priority}</Text>
          {detail.evidence.map((e, i) => (
            <div key={i} style={{ display: "flex", gap: 6, marginTop: 4 }}>
              <Badge size="small" appearance="outline">+{e.points}</Badge>
              <Text size={200}>{e.label}{e.detail ? ` (${e.detail})` : ""}</Text>
            </div>
          ))}
          <Text size={200} style={{ display: "block", marginTop: 6, color: "gray" }}>
            Pakai rincian ini buat bahan omongan ke klien.
          </Text>
        </Card>
      ) : null}
    </span>
  );
}
