-- Main provider table: links an assistant to an observability backend type.
-- Connection details live in the child options table (no vault dependency).
CREATE TABLE IF NOT EXISTS assistant_telemetry_providers (
    id              BIGINT       NOT NULL PRIMARY KEY,
    project_id      BIGINT       NOT NULL,
    organization_id BIGINT       NOT NULL,
    assistant_id    BIGINT       NOT NULL,
    provider_type   VARCHAR(50)  NOT NULL,
    enabled         BOOLEAN      NOT NULL DEFAULT TRUE,
    created_date    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_date    TIMESTAMPTZ           DEFAULT NULL
);

CREATE INDEX IF NOT EXISTS idx_atp_assistant_id ON assistant_telemetry_providers(assistant_id);
CREATE INDEX IF NOT EXISTS idx_atp_project_id   ON assistant_telemetry_providers(project_id);

-- Options table: key-value configuration pairs for each provider.
-- Examples: endpoint, headers, insecure, region, connection_string, etc.
CREATE TABLE IF NOT EXISTS assistant_telemetry_provider_options (
    id                              BIGINT        NOT NULL PRIMARY KEY,
    assistant_telemetry_provider_id BIGINT        NOT NULL REFERENCES assistant_telemetry_providers(id) ON DELETE CASCADE,
    key                             VARCHAR(200)  NOT NULL,
    value                           VARCHAR(1000) NOT NULL DEFAULT '',
    status                          VARCHAR(50)   NOT NULL DEFAULT 'active',
    created_by                      BIGINT,
    updated_by                      BIGINT,
    created_date                    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_date                    TIMESTAMPTZ            DEFAULT NULL
);

CREATE INDEX IF NOT EXISTS idx_atpo_provider_id ON assistant_telemetry_provider_options(assistant_telemetry_provider_id);
