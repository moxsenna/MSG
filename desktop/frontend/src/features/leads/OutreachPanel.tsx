import * as React from "react";
import { Card, Text, Input, Button, Dropdown, Option, Textarea } from "@fluentui/react-components";
import { Send24Regular } from "@fluentui/react-icons";
import type { LeadItem } from "./index";

interface DraftRow {
  id: string;
  body: string;
  channel: string;
  tone: string;
}

function outreachApp() {
  return (window as unknown as { go?: { main?: { App?: {
    DraftOutreachAI?: (id: string, channel: string, tone: string, offer: string) => Promise<{ draft_id: string; body: string; used_label: string; used_provider: string }>;
    ListOutreachDrafts?: (id: string) => Promise<DraftRow[]>;
    OpenWhatsAppURL?: (id: string, draftId: string) => Promise<string>;
    OpenEmailURL?: (id: string, draftId: string) => Promise<string>;
    MarkOutreachSent?: (id: string, draftId: string, channel: string) => Promise<void>;
  } } } }).go?.main?.App;
}

export default function OutreachPanel({ lead }: { lead: LeadItem }) {
  const [channel, setChannel] = React.useState("whatsapp");
  const [tone, setTone] = React.useState("friendly");
  const [offer, setOffer] = React.useState("");
  const [body, setBody] = React.useState("");
  const [draftId, setDraftId] = React.useState<string | null>(null);
  const [usedInfo, setUsedInfo] = React.useState("");
  const [busy, setBusy] = React.useState(false);
  const [msg, setMsg] = React.useState<string | null>(null);
  const [sent, setSent] = React.useState(false);
  const [templates, setTemplates] = React.useState<{ id: string; name: string; channel: string; tone: string; body: string }[]>([]);

  const fillTemplate = (t: { body: string; channel: string; tone: string }) => {
    const filled = t.body
      .replace(/\{\{\s*nama\s*\}\}/gi, lead.title)
      .replace(/\{\{\s*usaha\s*\}\}/gi, lead.category)
      .replace(/\{\{\s*kota\s*\}\}/gi, lead.city);
    setBody(filled);
    setChannel(t.channel || channel);
    setTone(t.tone || tone);
    setMsg("Template dimasukkan — edit sesukamu sebelum kirim.");
  };

  const saveAsTemplate = () => {
    if (!body.trim()) {
      setMsg("Isi pesan dulu sebelum disimpan jadi template.");
      return;
    }
    const name = window.prompt("Nama template ini:", "Sapaan awal");
    if (!name) return;
    const a = outreachApp() as unknown as { CreateOutreachTemplate?: (name: string, channel: string, tone: string, body: string) => Promise<void> } | null;
    a?.CreateOutreachTemplate?.(name, channel, tone, body)
      .then(() => {
        setMsg(`Template "${name}" tersimpan.`);
        outreachApp()?.ListOutreachDrafts && loadTemplates();
      })
      .catch((e: unknown) => setMsg("Gagal simpan: " + (e instanceof Error ? e.message : String(e))));
  };

  const loadTemplates = () => {
    const a = outreachApp() as unknown as { ListOutreachTemplates?: () => Promise<{ id: string; name: string; channel: string; tone: string; body: string }[]> } | null;
    a?.ListOutreachTemplates?.()
      .then(rows => setTemplates(Array.isArray(rows) ? rows : []))
      .catch(() => undefined);
  };

  React.useEffect(() => {
    setBody("");
    setDraftId(null);
    setUsedInfo("");
    setMsg(null);
    setSent(false);
    loadTemplates();
    outreachApp()?.ListOutreachDrafts?.(lead.id)
      .then(rows => {
        if (Array.isArray(rows) && rows.length > 0) {
          setBody(rows[0].body);
          setDraftId(rows[0].id);
          setChannel(rows[0].channel || "whatsapp");
        }
      })
      .catch(() => undefined);
  }, [lead.id]);

  const generate = () => {
    setBusy(true);
    setMsg(null);
    const p = outreachApp()?.DraftOutreachAI?.(lead.id, channel, tone, offer);
    if (!p) {
      setMsg("Backend belum tersedia.");
      setBusy(false);
      return;
    }
    p.then(r => {
      setBody(r.body);
      setDraftId(r.draft_id);
      setUsedInfo("Dibuat via " + (r.used_label || r.used_provider));
    })
      .catch((e: unknown) => setMsg("Gagal generate: " + (e instanceof Error ? e.message : String(e))))
      .finally(() => setBusy(false));
  };

  const openExternal = () => {
    if (!body.trim()) {
      setMsg("Isi pesan dulu sebelum dibuka di aplikasi luar.");
      return;
    }
    const app = outreachApp();
    const p = channel === "email" ? app?.OpenEmailURL?.(lead.id, draftId || "") : app?.OpenWhatsAppURL?.(lead.id, draftId || "");
    if (!p) {
      setMsg("Backend belum tersedia.");
      return;
    }
    p.then(url => {
      const rt = (window as unknown as { runtime?: { BrowserOpenURL?: (u: string) => void } }).runtime;
      if (rt?.BrowserOpenURL) rt.BrowserOpenURL(url);
      else window.open(url, "_blank");
    })
      .catch((e: unknown) => setMsg("Gagal buka " + channel + ": " + (e instanceof Error ? e.message : String(e))));
  };

  const markSent = () => {
    outreachApp()?.MarkOutreachSent?.(lead.id, draftId || "", channel)
      .then(() => {
        setSent(true);
        setMsg("Ditandai terkirim. Tahap lead otomatis jadi Contacted.");
      })
      .catch((e: unknown) => setMsg("Gagal: " + (e instanceof Error ? e.message : String(e))));
  };

  if (lead.dnc) {
    return (
      <Card style={{ backgroundColor: "#fde7e9", padding: 12 }}>
        <Text style={{ color: "#a80000" }}>Kontak keluar diblokir karena status Do Not Contact (DNC).</Text>
      </Card>
    );
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
      <Text weight="semibold">Buatkan Pesan AI</Text>
      <Text size={200} style={{ color: "gray" }}>AI pakai data lead ini (nama, kategori, kota, rating) + urutan key di Settings.</Text>
      <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
        <Dropdown value={channel === "email" ? "Email" : "WhatsApp"} onOptionSelect={(_, d) => setChannel(String(d.optionValue || "whatsapp"))} style={{ minWidth: 130 }}>
          <Option value="whatsapp">WhatsApp</Option>
          <Option value="email">Email</Option>
        </Dropdown>
        <Dropdown value={tone} onOptionSelect={(_, d) => setTone(String(d.optionValue || "friendly"))} style={{ minWidth: 130 }}>
          <Option value="friendly">Friendly</Option>
          <Option value="formal">Formal</Option>
          <Option value="persuasive">Persuasive</Option>
        </Dropdown>
      </div>
      <Input value={offer} onChange={(_, d) => setOffer(d.value)} placeholder="Penawaran (opsional)" />
      <Button appearance="primary" disabled={busy} onClick={generate}>{busy ? "AI menulis..." : "Buatkan Pesan dengan AI"}</Button>
      {templates.length > 0 && (
        <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
          <Text size={200}>atau pakai template:</Text>
          <Dropdown
            placeholder="Pilih template..."
            onOptionSelect={(_, d) => {
              const t = templates.find(x => x.id === d.optionValue);
              if (t) fillTemplate(t);
            }}
            style={{ minWidth: 180 }}
          >
            {templates.map(t => (
              <Option key={t.id} value={t.id} text={`${t.name} (${t.channel})`}>{t.name} ({t.channel})</Option>
            ))}
          </Dropdown>
        </div>
      )}
      {usedInfo && <Text size={200} style={{ color: "#0f7b0f" }}>{usedInfo}</Text>}
      <Text weight="semibold">Draft Pesan</Text>
      <Textarea rows={6} value={body} onChange={(_, d) => setBody(d.value)} placeholder="Klik Buatkan Pesan dengan AI, atau tulis manual..." />
      <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
        <Button appearance="primary" icon={<Send24Regular />} onClick={openExternal}>{channel === "email" ? "Buka Email" : "Buka WhatsApp"}</Button>
        <Button appearance="secondary" disabled={sent} onClick={markSent}>{sent ? "Sudah Terkirim" : "Tandai Terkirim"}</Button>
        <Button appearance="subtle" onClick={saveAsTemplate}>Simpan jadi template</Button>
      </div>
      <Text size={200} style={{ color: "gray" }}>Template mendukung {"{{nama}}"}, {"{{usaha}}"}, {"{{kota}}"} — otomatis terisi data lead.</Text>
      {msg && <Text size={200}>{msg}</Text>}
    </div>
  );
}
