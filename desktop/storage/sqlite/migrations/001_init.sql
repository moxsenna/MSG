-- 001_init: base schema per 05_DATA_MODEL_V2
-- idempotent via IF NOT EXISTS for re-run safety (runner tracks version)

CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE IF NOT EXISTS search_runs (
    id TEXT PRIMARY KEY,
    query TEXT NOT NULL,
    location_text TEXT NOT NULL DEFAULT '',
    goal TEXT NOT NULL DEFAULT 'any',
    config_json TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'pending',
    started_at TEXT,
    finished_at TEXT,
    result_count INTEGER NOT NULL DEFAULT 0,
    new_business_count INTEGER NOT NULL DEFAULT 0,
    error_code TEXT,
    error_message TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE IF NOT EXISTS source_snapshots (
    id TEXT PRIMARY KEY,
    search_run_id TEXT NOT NULL REFERENCES search_runs(id) ON DELETE CASCADE,
    business_id TEXT REFERENCES businesses(id) ON DELETE SET NULL,
    source TEXT NOT NULL DEFAULT 'gmaps',
    source_key TEXT NOT NULL,
    schema_version TEXT NOT NULL DEFAULT 'v1',
    normalized_json TEXT NOT NULL,
    raw_json TEXT NOT NULL DEFAULT '{}',
    collected_at TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE(search_run_id, content_hash)
);

CREATE TABLE IF NOT EXISTS businesses (
    id TEXT PRIMARY KEY,
    place_id TEXT,
    cid TEXT,
    data_id TEXT,
    source_link TEXT,
    title TEXT NOT NULL,
    primary_category TEXT,
    categories_json TEXT NOT NULL DEFAULT '[]',
    address_text TEXT,
    borough TEXT,
    street TEXT,
    city TEXT,
    postal_code TEXT,
    state TEXT,
    country TEXT,
    website TEXT,
    phone TEXT,
    emails_json TEXT NOT NULL DEFAULT '[]',
    review_count INTEGER NOT NULL DEFAULT 0,
    review_rating REAL NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT '',
    price_range TEXT,
    latitude REAL,
    longitude REAL,
    plus_code TEXT,
    timezone TEXT,
    first_seen_at TEXT NOT NULL,
    last_seen_at TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE IF NOT EXISTS business_contacts (
    id TEXT PRIMARY KEY,
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    value TEXT NOT NULL,
    normalized_value TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'gmaps',
    is_primary INTEGER NOT NULL DEFAULT 0,
    first_seen_at TEXT NOT NULL,
    last_seen_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS website_audits (
    id TEXT PRIMARY KEY,
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    audit_version TEXT NOT NULL DEFAULT 'v1',
    requested_url TEXT NOT NULL,
    final_url TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    evidence_json TEXT NOT NULL DEFAULT '{}',
    screenshot_path TEXT,
    duration_ms INTEGER,
    error_code TEXT,
    audited_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS opportunity_scores (
    id TEXT PRIMARY KEY,
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    rule_set_version TEXT NOT NULL DEFAULT 'v1',
    overall_score INTEGER NOT NULL,
    website_score INTEGER NOT NULL DEFAULT 0,
    app_score INTEGER NOT NULL DEFAULT 0,
    priority TEXT NOT NULL,
    confidence REAL NOT NULL DEFAULT 0,
    evidence_json TEXT NOT NULL DEFAULT '[]',
    calculated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS ai_analyses (
    id TEXT PRIMARY KEY,
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    prompt_version TEXT NOT NULL,
    input_hash TEXT NOT NULL,
    summary TEXT,
    opportunity_type TEXT,
    sales_angle TEXT,
    suggested_offer TEXT,
    reasons_json TEXT NOT NULL DEFAULT '[]',
    confidence REAL,
    raw_response_redacted TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE IF NOT EXISTS lead_states (
    business_id TEXT PRIMARY KEY REFERENCES businesses(id) ON DELETE CASCADE,
    stage TEXT NOT NULL DEFAULT 'new',
    do_not_contact INTEGER NOT NULL DEFAULT 0,
    owner_label TEXT,
    next_follow_up_at TEXT,
    won_value REAL,
    lost_reason TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE IF NOT EXISTS notes (
    id TEXT PRIMARY KEY,
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    color TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE IF NOT EXISTS business_tags (
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    PRIMARY KEY (business_id, tag_id)
);

CREATE TABLE IF NOT EXISTS activities (
    id TEXT PRIMARY KEY,
    business_id TEXT REFERENCES businesses(id) ON DELETE SET NULL,
    search_run_id TEXT REFERENCES search_runs(id) ON DELETE SET NULL,
    type TEXT NOT NULL,
    payload_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE IF NOT EXISTS outreach_drafts (
    id TEXT PRIMARY KEY,
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    channel TEXT NOT NULL,
    tone TEXT NOT NULL DEFAULT 'friendly',
    body TEXT NOT NULL,
    evidence_json TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE IF NOT EXISTS outreach_events (
    id TEXT PRIMARY KEY,
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    draft_id TEXT REFERENCES outreach_drafts(id) ON DELETE SET NULL,
    type TEXT NOT NULL,
    channel TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE IF NOT EXISTS saved_searches (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    query TEXT NOT NULL,
    location_text TEXT NOT NULL DEFAULT '',
    config_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

-- Indexes per spec 05 section 10
CREATE UNIQUE INDEX IF NOT EXISTS idx_businesses_place_id ON businesses(place_id) WHERE place_id IS NOT NULL AND place_id != '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_businesses_cid ON businesses(cid) WHERE cid IS NOT NULL AND cid != '';
CREATE INDEX IF NOT EXISTS idx_businesses_last_seen ON businesses(last_seen_at);
CREATE INDEX IF NOT EXISTS idx_lead_states_stage ON lead_states(stage);
CREATE INDEX IF NOT EXISTS idx_lead_states_followup ON lead_states(next_follow_up_at) WHERE next_follow_up_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_source_snapshots_search_run ON source_snapshots(search_run_id);
CREATE INDEX IF NOT EXISTS idx_source_snapshots_business ON source_snapshots(business_id);
CREATE INDEX IF NOT EXISTS idx_activities_business ON activities(business_id);
CREATE INDEX IF NOT EXISTS idx_activities_type ON activities(type);
CREATE INDEX IF NOT EXISTS idx_opportunity_scores_business ON opportunity_scores(business_id);
CREATE INDEX IF NOT EXISTS idx_business_contacts_normalized ON business_contacts(normalized_value);
