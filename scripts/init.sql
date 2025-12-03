-- Base Schema
CREATE TABLE IF NOT EXISTS workflows (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    version VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,
    states JSONB NOT NULL,
    events JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS workflow_instances (
    id VARCHAR(255) PRIMARY KEY,
    parent_id VARCHAR(255), -- Added for Subprocesses
    workflow_id VARCHAR(255) NOT NULL REFERENCES workflows(id),
    current_state VARCHAR(255) NOT NULL,
    current_sub_state VARCHAR(100), -- Added for R17
    status VARCHAR(50) NOT NULL,
    version INT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    data JSONB,
    variables JSONB,
    current_actor VARCHAR(255),
    "current_role" VARCHAR(255),
    previous_sub_state VARCHAR(100) -- Added for R17
);

CREATE INDEX idx_instances_workflow ON workflow_instances(workflow_id);
CREATE INDEX idx_instances_parent ON workflow_instances(parent_id);

CREATE TABLE IF NOT EXISTS workflow_transitions (
    id VARCHAR(255) PRIMARY KEY,
    instance_id VARCHAR(255) NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,
    event VARCHAR(255) NOT NULL,
    from_state VARCHAR(255) NOT NULL,
    to_state VARCHAR(255) NOT NULL,
    from_sub_state VARCHAR(100), -- Added for R17
    to_sub_state VARCHAR(100),   -- Added for R17
    actor VARCHAR(255),
    actor_role VARCHAR(255),
    data JSONB,
    created_at TIMESTAMP NOT NULL,
    reason TEXT,    -- Added for R17
    feedback TEXT   -- Added for R17
);

CREATE INDEX idx_transitions_instance ON workflow_transitions(instance_id);

-- Additional Tables from Migration 002 (Simplified for Init)
CREATE TABLE IF NOT EXISTS workflow_timers (
    id VARCHAR(255) PRIMARY KEY,
    instance_id VARCHAR(255) NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,
    state VARCHAR(255) NOT NULL,
    event_on_timeout VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    fired_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL
);
