import * as React from "react";
import { Card, Text, Badge } from "@fluentui/react-components";
import { History24Regular } from "@fluentui/react-icons";

interface ActivityRow {
  id: string;
  business_id: string;
  title: string;
  type: string;
  detail: string;
  at: string;
}

export default function ActivityPage() {
  const [activities, setActivities] = React.useState<ActivityRow[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [loadError, setLoadError] = React.useState<string | null>(null);

  const reload = React.useCallback(() => {
    const fn = (window as unknown as { go?: { main?: { App?: { ListActivities?: (limit: number) => Promise<ActivityRow[]> } } } }).go?.main?.App?.ListActivities;
    if (!fn) {
      setLoadError("Backend belum tersedia (jalankan via aplikasi desktop).");
      return;
    }
    setLoading(true);
    setLoadError(null);
    fn(50).then(rows => {
      setActivities(Array.isArray(rows) ? rows : []);
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

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <div>
          <Text size={600} weight="semibold">Riwayat Aktivitas</Text>
          <Text size={300} style={{ display: "block", color: "gray" }}>Catatan audit dan linimasa operasional prospeksi</Text>
        </div>
        <button onClick={reload} disabled={loading} style={{ fontSize: 12, padding: "4px 10px", cursor: "pointer" }}>{loading ? "Memuat..." : "Muat ulang"}</button>
      </div>
      {loadError && (
        <Text size={200} style={{ color: "#a80000" }}>Gagal memuat: {loadError}</Text>
      )}
      {!loading && !loadError && activities.length === 0 && (
        <Text size={200} style={{ color: "gray" }}>
          Belum ada aktivitas. Aktivitas muncul otomatis setelah Find Prospects, pindah tahap, atau tambah catatan.
        </Text>
      )}

      <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
        {activities.map(a => (
          <Card key={a.id} style={{ padding: 14, display: "flex", flexDirection: "row", alignItems: "center", gap: 12 }}>
            <History24Regular style={{ color: "#0078d4" }} />
            <div style={{ flex: 1 }}>
              <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
                <Text weight="semibold">{a.title}</Text>
                <Badge size="small" appearance="outline">{a.type}</Badge>
              </div>
              <Text size={300} style={{ color: "gray" }}>{a.detail}</Text>
            </div>
            <Text size={200} style={{ color: "gray" }}>{a.at}</Text>
          </Card>
        ))}
      </div>
    </div>
  );
}
