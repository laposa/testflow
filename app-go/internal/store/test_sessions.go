package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// NewTestSession is a test session to create with its services, suites and
// tests (spec §8.2).
type NewTestSession struct {
	IssuerID    int64
	IssuerName  string // for the session_created activity entry
	Name        string
	Environment string
	Services    []NewService
}

// NewService is a session_services row with its suites (spec §6.4).
type NewService struct {
	Type           string // automated or manual
	Name           string
	Path           string
	RepositoryID   int64
	RepositoryName string // owner/repo
	WorkflowID     int64  // 0 is stored as NULL
	CommitSHA      string
	Branch         string // empty means master
	Suites         []NewSuite
}

// NewSuite is a service_suites row with its tests (spec §6.5).
type NewSuite struct {
	Name  string
	Path  string
	Tests []NewTest
}

// NewTest is a suite_tests row (spec §6.6).
type NewTest struct {
	Name string
	Path string
}

// TestSession is a test_sessions row with its issuer and selected tests.
type TestSession struct {
	ID          int64
	IssuerID    int64
	IssuerName  string
	Name        string
	Environment string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Services    []Service // ordered by id
}

// Service is a session_services row (spec §6.4).
type Service struct {
	ID             int64
	SessionID      int64
	Type           string
	Name           string
	Path           string
	RepositoryID   int64
	RepositoryName string
	WorkflowID     int64 // 0 if none
	CommitSHA      string
	Branch         string
	Suites         []Suite // ordered by id
}

// DisplayName is "<repo without owner>/<name> (<type>)"
// (legacy: SessionService::displayName).
func (s Service) DisplayName() string {
	_, repo, ok := strings.Cut(s.RepositoryName, "/")
	if !ok {
		repo = s.RepositoryName
	}
	return repo + "/" + s.Name + " (" + s.Type + ")"
}

// Suite is a service_suites row (spec §6.5).
type Suite struct {
	ID    int64
	Name  string
	Path  string
	Tests []Test // ordered by id
}

// Test is a suite_tests row (spec §6.6).
type Test struct {
	ID   int64
	Name string
	Path string
}

// SessionCreatedBody is the session_created activity text (spec §8.2).
func SessionCreatedBody(user string) string { return user + " created the session." }

// CreateTestSession stores a session with its services, suites and tests
// and logs session_created, all in one transaction. It returns the new id.
func (s *Store) CreateTestSession(ctx context.Context, ns NewTestSession) (int64, error) {
	if ns.Name == "" || ns.Environment == "" || len(ns.Services) == 0 {
		return 0, errors.New("create test session: name, environment and services are required")
	}
	var id int64
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`INSERT INTO test_sessions (issuer_id, name, environment) VALUES ($1, $2, $3) RETURNING id`,
			ns.IssuerID, ns.Name, ns.Environment).Scan(&id); err != nil {
			return err
		}
		for _, svc := range ns.Services {
			if err := insertService(ctx, tx, id, svc); err != nil {
				return err
			}
		}
		return insertActivity(ctx, tx, NewActivity{
			SessionID: id,
			UserID:    ns.IssuerID,
			Type:      ActivitySessionCreated,
			Body:      SessionCreatedBody(ns.IssuerName),
		})
	})
	if err != nil {
		return 0, fmt.Errorf("create test session: %w", err)
	}
	return id, nil
}

func insertService(ctx context.Context, tx pgx.Tx, sessionID int64, svc NewService) error {
	branch := svc.Branch
	if branch == "" {
		branch = "master"
	}
	var serviceID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO session_services
		    (session_id, type, name, path, repository_id, repository_name, workflow_id, commit_sha, branch)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7::bigint, 0), $8, $9)
		RETURNING id`,
		sessionID, svc.Type, svc.Name, svc.Path, svc.RepositoryID, svc.RepositoryName,
		svc.WorkflowID, svc.CommitSHA, branch).Scan(&serviceID); err != nil {
		return err
	}
	for _, suite := range svc.Suites {
		var suiteID int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO service_suites (service_id, name, path) VALUES ($1, $2, $3) RETURNING id`,
			serviceID, suite.Name, suite.Path).Scan(&suiteID); err != nil {
			return err
		}
		names := make([]string, len(suite.Tests))
		paths := make([]string, len(suite.Tests))
		for i, t := range suite.Tests {
			names[i], paths[i] = t.Name, t.Path
		}
		// unnest keeps the array order, so ids follow the given order.
		if _, err := tx.Exec(ctx, `
			INSERT INTO suite_tests (suite_id, name, path)
			SELECT $1, t.name, t.path FROM unnest($2::text[], $3::text[]) WITH ORDINALITY AS t(name, path, n)
			ORDER BY t.n`,
			suiteID, names, paths); err != nil {
			return err
		}
	}
	return nil
}

// TestSession returns the session with id, its issuer's name and its
// services, suites and tests, or ErrNotFound.
func (s *Store) TestSession(ctx context.Context, id int64) (TestSession, error) {
	var ts TestSession
	err := s.db.QueryRow(ctx, `
		SELECT ts.id, ts.issuer_id, u.name, ts.name, ts.environment, ts.created_at, ts.updated_at
		FROM test_sessions ts JOIN users u ON u.id = ts.issuer_id
		WHERE ts.id = $1`, id).
		Scan(&ts.ID, &ts.IssuerID, &ts.IssuerName, &ts.Name, &ts.Environment, &ts.CreatedAt, &ts.UpdatedAt)
	if err != nil {
		return TestSession{}, notFound(err)
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, session_id, type, name, path, repository_id, repository_name,
		       COALESCE(workflow_id, 0), commit_sha, branch
		FROM session_services WHERE session_id = $1 ORDER BY id`, id)
	if err != nil {
		return TestSession{}, fmt.Errorf("test session services: %w", err)
	}
	ts.Services, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (Service, error) {
		var svc Service
		err := row.Scan(&svc.ID, &svc.SessionID, &svc.Type, &svc.Name, &svc.Path, &svc.RepositoryID,
			&svc.RepositoryName, &svc.WorkflowID, &svc.CommitSHA, &svc.Branch)
		return svc, err
	})
	if err != nil {
		return TestSession{}, fmt.Errorf("test session services: %w", err)
	}
	index := make(map[int64]int, len(ts.Services))
	for i, svc := range ts.Services {
		index[svc.ID] = i
	}

	rows, err = s.db.Query(ctx, `
		SELECT su.service_id, su.id, su.name, su.path, t.id, t.name, t.path
		FROM service_suites su
		JOIN session_services ss ON ss.id = su.service_id
		LEFT JOIN suite_tests t ON t.suite_id = su.id
		WHERE ss.session_id = $1
		ORDER BY su.id, t.id`, id)
	if err != nil {
		return TestSession{}, fmt.Errorf("test session tests: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			serviceID int64
			suite     Suite
			testID    *int64
			testName  *string
			testPath  *string
		)
		if err := rows.Scan(&serviceID, &suite.ID, &suite.Name, &suite.Path, &testID, &testName, &testPath); err != nil {
			return TestSession{}, fmt.Errorf("test session tests: %w", err)
		}
		svc := &ts.Services[index[serviceID]]
		if n := len(svc.Suites); n == 0 || svc.Suites[n-1].ID != suite.ID {
			svc.Suites = append(svc.Suites, suite)
		}
		if testID != nil {
			last := &svc.Suites[len(svc.Suites)-1]
			last.Tests = append(last.Tests, Test{ID: *testID, Name: *testName, Path: *testPath})
		}
	}
	if err := rows.Err(); err != nil {
		return TestSession{}, fmt.Errorf("test session tests: %w", err)
	}
	return ts, nil
}
