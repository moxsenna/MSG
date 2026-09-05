import * as React from "react";
import { Input, Text, Card } from "@fluentui/react-components";

export interface LocationOption {
  label: string;
  short: string;
  lat: string;
  lon: string;
}

async function fetchOptions(q: string): Promise<LocationOption[]> {
  const url = `https://nominatim.openstreetmap.org/search?format=jsonv2&countrycodes=id&addressdetails=1&limit=6&q=${encodeURIComponent(q)}`;
  const resp = await fetch(url, { headers: { Accept: "application/json" } });
  if (!resp.ok) return [];
  const data = (await resp.json()) as {
    display_name?: string;
    lat?: string;
    lon?: string;
    address?: Record<string, string>;
  }[];
  if (!Array.isArray(data)) return [];
  return data.slice(0, 6).map(d => {
    const a = d.address || {};
    const parts = [a.road, a.suburb, a.village, a.town, a.city_district, a.city, a.county, a.state].filter(Boolean);
    const seen: string[] = [];
    for (const p of parts) {
      if (p && !seen.includes(p)) seen.push(p);
    }
    return { label: d.display_name || "", short: seen.slice(0, 3).join(", ") || d.display_name || "", lat: d.lat || "", lon: d.lon || "" };
  }).filter(o => o.short);
}

export default function LocationAutocomplete({ value, onPick }: { value: string; onPick: (v: string, lat?: string, lon?: string) => void }) {
  const [text, setText] = React.useState(value);
  const [options, setOptions] = React.useState<LocationOption[]>([]);
  const [open, setOpen] = React.useState(false);
  const [loading, setLoading] = React.useState(false);
  const timer = React.useRef<number | null>(null);

  React.useEffect(() => { setText(value); }, [value]);

  const onChange = (v: string) => {
    setText(v);
    onPick(v);
    if (timer.current) window.clearTimeout(timer.current);
    if (v.trim().length < 3) {
      setOptions([]);
      setOpen(false);
      return;
    }
    timer.current = window.setTimeout(() => {
      setLoading(true);
      fetchOptions(v.trim())
        .then(opts => {
          setOptions(opts);
          setOpen(opts.length > 0);
        })
        .catch(() => {})
        .finally(() => setLoading(false));
    }, 400);
  };

  return (
    <div style={{ position: "relative", width: "100%" }}>
      <Input
        placeholder="Lokasi — ketik kelurahan / kecamatan / jalan, mis. pamengkang"
        value={text}
        onChange={(_, d) => onChange(d.value)}
        onFocus={() => { if (options.length > 0) setOpen(true); }}
        onBlur={() => setTimeout(() => setOpen(false), 150)}
        style={{ width: "100%" }}
      />
      {loading && <Text size={200} style={{ color: "gray" }}>Mencari lokasi...</Text>}
      {open && (
        <Card style={{ position: "absolute", zIndex: 10, width: "100%", padding: 4, marginTop: 4, maxHeight: 240, overflow: "auto" }}>
          {options.map(o => (
            <div
              key={o.label}
              onMouseDown={() => {
                onPick(o.short, o.lat, o.lon);
                setText(o.short);
                setOpen(false);
              }}
              style={{ padding: "8px 10px", cursor: "pointer", borderRadius: 4 }}
              onMouseEnter={e => { (e.target as HTMLElement).style.background = "#f0f6ff"; }}
              onMouseLeave={e => { (e.target as HTMLElement).style.background = "transparent"; }}
            >
              <Text size={300} weight="semibold">{o.short}</Text>
              <Text size={200} style={{ display: "block", color: "gray" }}>{o.label}</Text>
            </div>
          ))}
        </Card>
      )}
    </div>
  );
}
