-- 0001_init: the complete TestFlow schema (spec §6).
-- Tables are created unqualified, in the connection's current schema.

-- Keeps updated_at current on every UPDATE, so queries don't have to.
CREATE FUNCTION set_updated_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$;

-- §6.1
CREATE TABLE users (
    id          bigserial PRIMARY KEY,
    github_id   bigint NOT NULL UNIQUE,
    name        text NOT NULL,
    email       text,
    login       text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- §6.2 Server-side login sessions. Expired rows are deleted by the syncer.
CREATE TABLE web_sessions (
    id          text PRIMARY KEY,
    user_id     bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    expires_at  timestamptz NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX web_sessions_user_id_idx ON web_sessions (user_id);
CREATE INDEX web_sessions_expires_at_idx ON web_sessions (expires_at);

-- §6.3
CREATE TABLE test_sessions (
    id           bigserial PRIMARY KEY,
    issuer_id    bigint NOT NULL REFERENCES users,
    name         text NOT NULL,
    environment  text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- §6.4
CREATE TABLE session_services (
    id               bigserial PRIMARY KEY,
    session_id       bigint NOT NULL REFERENCES test_sessions ON DELETE CASCADE,
    type             text NOT NULL CHECK (type IN ('automated', 'manual')),
    name             text NOT NULL,
    path             text NOT NULL,
    repository_id    bigint NOT NULL,
    repository_name  text NOT NULL,
    workflow_id      bigint,
    commit_sha       text NOT NULL,
    branch           text NOT NULL DEFAULT 'master',
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX session_services_session_id_idx ON session_services (session_id);

-- §6.5
CREATE TABLE service_suites (
    id          bigserial PRIMARY KEY,
    service_id  bigint NOT NULL REFERENCES session_services ON DELETE CASCADE,
    name        text NOT NULL,
    path        text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX service_suites_service_id_idx ON service_suites (service_id);

-- §6.6
CREATE TABLE suite_tests (
    id          bigserial PRIMARY KEY,
    suite_id    bigint NOT NULL REFERENCES service_suites ON DELETE CASCADE,
    name        text NOT NULL,
    path        text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX suite_tests_suite_id_idx ON suite_tests (suite_id);

-- §6.7 Status values are listed in §8.4.
CREATE TABLE service_runs (
    id             bigserial PRIMARY KEY,
    service_id     bigint NOT NULL REFERENCES session_services ON DELETE CASCADE,
    type           text NOT NULL CHECK (type IN ('automated', 'manual')),
    status         text NOT NULL CHECK (status IN (
                       'unknown', 'queued', 'in_progress', 'success', 'failure', 'neutral',
                       'cancelled', 'skipped', 'timed_out', 'action_required', 'stale')),
    passed         integer,
    failed         integer,
    skipped        integer,
    duration       double precision,
    result_log     text,
    run_log        text,
    github_run_id  bigint,
    commit_sha     text,
    started_at     timestamptz,
    finished_at    timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX service_runs_service_id_idx ON service_runs (service_id, id DESC);
-- The syncer's work list (§8.5): automated runs without a run log yet.
CREATE INDEX service_runs_unsynced_idx ON service_runs (created_at)
    WHERE type = 'automated' AND run_log IS NULL;

-- §6.8 One row per test per run, for automated and manual runs (D12).
CREATE TABLE run_test_results (
    run_id           bigint NOT NULL REFERENCES service_runs ON DELETE CASCADE,
    test_id          bigint NOT NULL REFERENCES suite_tests ON DELETE CASCADE,
    status           text NOT NULL CHECK (status IN ('pass', 'fail', 'skipped', 'error', 'notRun')),
    comment          text,
    failure_message  text,
    duration         double precision,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, test_id)
);
CREATE INDEX run_test_results_test_id_idx ON run_test_results (test_id);

-- §6.9 Append-only, so no updated_at. user_id NULL means the system (e.g. the
-- syncer). body is plain text; HTML is produced only by the views (D13).
CREATE TABLE activity_log (
    id          bigserial PRIMARY KEY,
    session_id  bigint NOT NULL REFERENCES test_sessions ON DELETE CASCADE,
    user_id     bigint REFERENCES users ON DELETE SET NULL,
    type        text NOT NULL CHECK (type IN (
                    'session_created', 'run_dispatched', 'run_status_changed',
                    'manual_run_started', 'manual_run_finished', 'review_requested',
                    'review_approved', 'review_rejected', 'comment')),
    body        text NOT NULL,
    metadata    jsonb NOT NULL DEFAULT '{}',
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX activity_log_session_id_idx ON activity_log (session_id, id DESC);

-- §6.10
CREATE TABLE review_requests (
    id            bigserial PRIMARY KEY,
    session_id    bigint NOT NULL REFERENCES test_sessions ON DELETE CASCADE,
    requester_id  bigint NOT NULL REFERENCES users,
    reviewer_id   bigint NOT NULL REFERENCES users,
    status        text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    completed_at  timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX review_requests_session_id_idx ON review_requests (session_id);
-- A reviewer has at most one pending request per session (§8.9).
CREATE UNIQUE INDEX review_requests_one_pending_idx ON review_requests (session_id, reviewer_id)
    WHERE status = 'pending';

CREATE TRIGGER users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER test_sessions_updated_at BEFORE UPDATE ON test_sessions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER session_services_updated_at BEFORE UPDATE ON session_services
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER service_suites_updated_at BEFORE UPDATE ON service_suites
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER suite_tests_updated_at BEFORE UPDATE ON suite_tests
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER service_runs_updated_at BEFORE UPDATE ON service_runs
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER run_test_results_updated_at BEFORE UPDATE ON run_test_results
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER review_requests_updated_at BEFORE UPDATE ON review_requests
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
