CREATE TABLE workflows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

ALTER TABLE jobs ADD COLUMN workflow_id UUID REFERENCES workflows(id) ON DELETE SET NULL;

CREATE TABLE job_dependencies (
    upstream_job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
    downstream_job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    PRIMARY KEY (upstream_job_id, downstream_job_id)
);

CREATE TABLE workflow_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status VARCHAR(50) DEFAULT 'RUNNING',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    finished_at TIMESTAMP WITH TIME ZONE
);

ALTER TABLE executions ADD COLUMN workflow_run_id UUID REFERENCES workflow_runs(id) ON DELETE SET NULL;