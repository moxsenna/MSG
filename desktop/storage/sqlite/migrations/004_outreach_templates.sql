-- 004_outreach_templates: reusable message templates (placeholders: {{nama}}, {{usaha}}, {{kota}})
CREATE TABLE IF NOT EXISTS outreach_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    channel TEXT NOT NULL DEFAULT 'whatsapp',
    tone TEXT NOT NULL DEFAULT 'friendly',
    body TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
