import * as React from "react";
import { Card, Text, Badge, Button, Table, TableHeader, TableRow, TableHeaderCell, TableBody, TableCell } from "@fluentui/react-components";
import { Call24Regular, Checkmark24Regular } from "@fluentui/react-icons";

interface FollowUpRow {
  business_id: string;
  title: string;
  phone: string;
  stage: string;
  follow_up_at: string;
  is_overdue: boolean;
}

export default function FollowUpsPage() {
  const [items, setItems] = React.useState<FollowUpRow[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [loadError, setLoadError] = React.useState<string | null>(null);

  const reload = React.useCallback(() => {
    const fn = (window as unknown as { go?: { main?: { App?: { ListFollowUps?: () => Promise<FollowUpRow[]> } } } }).go?.main?.App?.ListFollowUps;
    if (!fn) {
      setLoadError("Backend belum tersedia (jalankan via aplikasi desktop).");
      return;
    }
    setLoading(true);
    setLoadError(null);
    fn().then(rows => {
      setItems(Array.isArray(rows) ? rows : []);
    }).catch((e: unknown) => {
      setLoadError(e instanceof Error ? e.message : String(e));
    }).finally(() => setLoading(false));
  }, []);

  React.useEffect(() => {
    reload();
  }, [reload]);

  React.useEffect(() => {
    const onFocus = () => reload();
    window.addEventListener("focus", onFocus);
    return () => window.removeEventListener("focus", onFocus);
  }, [reload]);

  const fuApp = () => (window as unknown as { go?: { main?: { App?: {
    OpenWhatsAppURL?: (id: string, draftId: string) => Promise<string>;
    SetLeadFollowUp?: (id: string, at: string) => Promise<void>;
  } } } }).go?.main?.App;

  const callItem = (it: FollowUpRow) => {
    fuApp()?.OpenWhatsAppURL?.(it.business_id, "")
      .then(url => {
        const rt = (window as unknown as { runtime?: { BrowserOpenURL?: (u: string) => void } }).runtime;
        if (rt?.BrowserOpenURL) rt.BrowserOpenURL(url);
        else window.open(url, "_blank");
      })
      .catch((e: unknown) => setLoadError(e instanceof Error ? e.message : String(e)));
  };

  const completeItem = (_id: string) => {
    fuApp()?.SetLeadFollowUp?.(_id, "")
      .then(() => setItems(prev => prev.filter(it => it.business_id !== _id)))
      .catch((e: unknown) => setLoadError(e instanceof Error ? e.message : String(e)));
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <div>
          <Text size={600} weight="semibold">Antrean Follow-up</Text>
          <Text size={300} style={{ display: "block", color: "gray" }}>Jadwal tindak lanjut prospek agar konversi penjualan maksimal</Text>
        </div>
        <Button size="small" appearance="secondary" onClick={reload} disabled={loading}>{loading ? "Memuat..." : "Muat ulang"}</Button>
      </div>
      {loadError && (
        <Text size={200} style={{ color: "#a80000" }}>Gagal memuat: {loadError}</Text>
      )}
      {!loading && !loadError && items.length === 0 && (
        <Text size={200} style={{ color: "gray" }}>
          Belum ada jadwal follow-up. Follow-up muncul setelah Anda mengatur tanggal tindak lanjut dari detail lead (terjadwal di lead_states).
        </Text>
      )}

      <Card style={{ padding: 0 }}>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHeaderCell>Bisnis</TableHeaderCell>
              <TableHeaderCell>Telepon</TableHeaderCell>
              <TableHeaderCell>Tahap CRM</TableHeaderCell>
              <TableHeaderCell>Jadwal</TableHeaderCell>
              <TableHeaderCell>Status</TableHeaderCell>
              <TableHeaderCell>Aksi</TableHeaderCell>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map(it => (
              <TableRow key={it.business_id}>
                <TableCell><Text weight="semibold">{it.title}</Text></TableCell>
                <TableCell>{it.phone}</TableCell>
                <TableCell><Badge appearance="outline">{it.stage.toUpperCase()}</Badge></TableCell>
                <TableCell>{it.follow_up_at}</TableCell>
                <TableCell>
                  {it.is_overdue ? (
                    <Badge color="danger">Terlambat</Badge>
                  ) : (
                    <Badge color="informative">Terjadwal</Badge>
                  )}
                </TableCell>
                <TableCell>
                  <div style={{ display: "flex", gap: 6 }}>
                    <Button size="small" appearance="primary" icon={<Call24Regular />} onClick={() => callItem(it)}>Hubungi</Button>
                    <Button size="small" appearance="subtle" icon={<Checkmark24Regular />} onClick={() => completeItem(it.business_id)}>Selesai</Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Card>
    </div>
  );
}
