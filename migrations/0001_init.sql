CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  external_id  VARCHAR(255) UNIQUE NOT NULL,
  metadata     JSONB DEFAULT '{}'::jsonb,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE memories (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id               UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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

CREATE INDEX idx_memories_user_id ON memories(user_id);
CREATE INDEX idx_memories_embedding ON memories USING ivfflat (embedding vector_cosine_ops);

CREATE TABLE events (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  conversation_id TEXT NOT NULL,
  role            TEXT NOT NULL,
  content         TEXT NOT NULL,
  metadata        JSONB DEFAULT '{}'::jsonb,
  timestamp       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_user_conv ON events(user_id, conversation_id, timestamp);
