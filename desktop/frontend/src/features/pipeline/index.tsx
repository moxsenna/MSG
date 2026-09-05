import * as React from "react";
import { Card, Text, Badge, Button } from "@fluentui/react-components";
import { ArrowRight16Regular } from "@fluentui/react-icons";

interface PipelineCard {
  id: string;
  title: string;
  city: string;
  category: string;
  score: number;
}

const STAGES = [
  { id: "new", label: "New" },
  { id: "qualified", label: "Qualified" },
  { id: "shortlisted", label: "Shortlisted" },
  { id: "contacted", label: "Contacted" },
  { id: "replied", label: "Replied" },
  { id: "meeting", label: "Meeting" },
  { id: "proposal", label: "Proposal" },
  { id: "won", label: "Won" },
];

const emptyBoard = (): Record<string, PipelineCard[]> => ({
  new: [], qualified: [], shortlisted: [], contacted: [], replied: [], meeting: [], proposal: [], won: [],
});

export default function PipelinePage() {
  const [cards, setCards] = React.useState<Record<string, PipelineCard[]>>(emptyBoard);
  const [loading, setLoading] = React.useState(false);
  const [loadError, setLoadError] = React.useState<string | null>(null);

  const reload = React.useCallback(() => {
    const boardFn = (window as unknown as { go?: { main?: { App?: { GetKanbanBoard?: () => Promise<Record<string, PipelineCard[]>> } } } }).go?.main?.App?.GetKanbanBoard;
    if (!boardFn) {
      setLoadError("Backend belum tersedia (jalankan via aplikasi desktop).");
      return;
    }
    setLoading(true);
    setLoadError(null);
    boardFn().then(board => {
      const next = emptyBoard();
      if (board) {
        for (const st of Object.keys(next)) {
          const list = (board as Record<string, PipelineCard[]>)[st];
          if (Array.isArray(list)) next[st] = list;
        }
      }
      setCards(next);
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

  const moveCard = (card: PipelineCard, fromStage: string, toStage: string) => {
    if (fromStage === toStage) return;
    if (!STAGES.some(s => s.id === toStage)) return;
    const moveFn = (window as unknown as { go?: { main?: { App?: { MovePipelineCard?: (id: string, stage: string) => Promise<void> } } } }).go?.main?.App?.MovePipelineCard;
    if (moveFn) {
      moveFn(card.id, toStage).then(() => {
        setCards(prev => ({
          ...prev,
          [fromStage]: (prev[fromStage] || []).filter(c => c.id !== card.id),
          [toStage]: [...(prev[toStage] || []), { ...card }],
        }));
      }).catch(() => {});
      return;
    }
    setCards(prev => ({
      ...prev,
      [fromStage]: (prev[fromStage] || []).filter(c => c.id !== card.id),
      [toStage]: [...(prev[toStage] || []), card],
    }));
  };

  const moveNext = (card: PipelineCard, fromStage: string) => {
    const idx = STAGES.findIndex(s => s.id === fromStage);
    if (idx < 0 || idx >= STAGES.length - 1) return;
    moveCard(card, fromStage, STAGES[idx + 1].id);
  };

  const onDropTo = (e: React.DragEvent, toStage: string) => {
    e.preventDefault();
    try {
      const data = JSON.parse(e.dataTransfer.getData("application/msg-card")) as { id: string; from: string };
      if (!data?.id || !data?.from) return;
      for (const [st, list] of Object.entries(cards)) {
        const found = (list || []).find(c => c.id === data.id);
        if (found) {
          moveCard(found, st, toStage);
          return;
        }
      }
    } catch {}
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <div>
          <Text size={600} weight="semibold">Sales Pipeline Kanban</Text>
          <Text size={300} style={{ display: "block", color: "gray" }}>Pantau progres transaksi dan konversi penawaran bisnis lokal</Text>
        </div>
        <Button size="small" appearance="secondary" onClick={reload} disabled={loading}>{loading ? "Memuat..." : "Muat ulang"}</Button>
      </div>
      {loadError && (
        <Text size={200} style={{ color: "#a80000" }}>Gagal memuat board: {loadError}</Text>
      )}
      {!loading && !loadError && STAGES.every(s => (cards[s.id] || []).length === 0) && (
        <Text size={200} style={{ color: "gray" }}>
          Board masih kosong. Leads baru dari Discover masuk ke kolom New — buka detail lead di tab Leads lalu pindahkan tahap CRM ke Qualified untuk mulai pipeline.
        </Text>
      )}

      <div style={{ display: "flex", gap: 12, overflowX: "auto", paddingBottom: 16 }}>
        {STAGES.map(stage => {
          const list = cards[stage.id] || [];
          return (
            <div key={stage.id} style={{ minWidth: 240, width: 260, display: "flex", flexDirection: "column", gap: 8 }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", padding: "4px 8px" }}>
                <Text weight="semibold" size={300}>{stage.label}</Text>
                <Badge size="small" appearance="filled">{list.length}</Badge>
              </div>

              <div
                style={{ backgroundColor: "#f5f5f5", borderRadius: 8, padding: 8, minHeight: 400, display: "flex", flexDirection: "column", gap: 8 }}
                onDragOver={e => e.preventDefault()}
                onDrop={e => onDropTo(e, stage.id)}
              >
                {list.map(card => (
                  <Card
                    key={card.id}
                    style={{ padding: 12, cursor: "grab" }}
                    draggable
                    onDragStart={e => {
                      e.dataTransfer.setData("application/msg-card", JSON.stringify({ id: card.id, from: stage.id }));
                      e.dataTransfer.effectAllowed = "move";
                    }}
                  >
                    <Text weight="semibold">{card.title}</Text>
                    <Text size={200} style={{ color: "gray" }}>{card.category} • {card.city}</Text>
                    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginTop: 8 }}>
                      <Badge size="small" color={card.score >= 80 ? "success" : "warning"}>{card.score}</Badge>
                      <Button
                        size="small"
                        appearance="subtle"
                        icon={<ArrowRight16Regular />}
                        onClick={() => moveNext(card, stage.id)}
                        title="Pindah ke tahap berikutnya"
                      />
                    </div>
                  </Card>
                ))}
                {list.length === 0 && (
                  <div style={{ display: "flex", justifyContent: "center", alignItems: "center", height: 100 }}>
                    <Text size={200} style={{ color: "gray" }}>Kosong</Text>
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
