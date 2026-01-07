CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE tenants (
  id           BIGSERIAL PRIMARY KEY,
  external_id  VARCHAR(255) UNIQUE NOT NULL,
  metadata     JSONB DEFAULT '{}'::jsonb,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE apps (
  id           BIGSERIAL PRIMARY KEY,
  tenant_id    BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  external_id  VARCHAR(255) NOT NULL,
  metadata     JSONB DEFAULT '{}'::jsonb,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT unique_tenant_app UNIQUE (tenant_id, external_id)
);

CREATE INDEX idx_apps_tenant_id ON apps(tenant_id);

CREATE TABLE users (
  id           BIGSERIAL PRIMARY KEY,
  tenant_id    BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  app_id       BIGINT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
  external_id  VARCHAR(255) NOT NULL,
  metadata     JSONB DEFAULT '{}'::jsonb,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT unique_tenant_app_user UNIQUE (tenant_id, app_id, external_id)
);

CREATE INDEX idx_users_tenant_app ON users(tenant_id, app_id);

CREATE TABLE memories (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  app_id                BIGINT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
  user_id               BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  memory_type           VARCHAR(255) NOT NULL,
  content               TEXT NOT NULL,
  importance_score      REAL NOT NULL,
  memory_stability      VARCHAR(255) NOT NULL,
  embedding             vector(1536),
  metadata              JSONB DEFAULT '{}'::jsonb,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT valid_memory_type CHECK (memory_type IN ('profile', 'preference', 'project', 'episodic', 'knowledge', 'other')),
  CONSTRAINT valid_memory_stability CHECK (memory_stability IN ('short_term', 'long_term'))
);

CREATE INDEX idx_memories_tenant_app_user ON memories(tenant_id, app_id, user_id);
CREATE INDEX idx_memories_embedding ON memories USING ivfflat (embedding vector_cosine_ops);

CREATE TABLE events (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  app_id          BIGINT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
  user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  conversation_id TEXT NOT NULL,
  role            TEXT NOT NULL,
  content         TEXT NOT NULL,
  metadata        JSONB DEFAULT '{}'::jsonb,
  timestamp       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_tenant_app_user_conv ON events(tenant_id, app_id, user_id, conversation_id, timestamp);
