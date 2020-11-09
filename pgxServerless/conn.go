package pgxServerless

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"net/url"
	"reflect"
	"strings"
	"time"
)

type SlsConn struct {
	config     slsConnConfig
	delay      delay
	tempConfig SlsConnConfigParams
	conn       *pgx.Conn
	logger     Logger
	connCred   connCred
}

func New(config SlsConnConfigParams) *SlsConn {
	return &SlsConn{
		tempConfig: config,
	}
}

func (s *SlsConn) parseURL(connString string) error {
	parse, err := url.Parse(connString)
	if err != nil {
		return err
	}

	s.connCred.user = parse.User.Username()
	s.connCred.database = strings.Replace(parse.Path, "/", "", 2)

	return nil
}

func (s *SlsConn) Connect(ctx context.Context, connectionString string) (*SlsConn, error) {
	s.config = newDefaultConfig()
	if err := s.config.mergeAndValidate(s.tempConfig); err != nil {
		return nil, err
	}

	if err := s.parseURL(connectionString); err != nil {
		return nil, err
	}

	s.logger = newLogger(s.config.Debug)
	s.delay = newDelay(delayConfig{
		backoffCapMs:   s.config.BackoffCapMs,
		backoffBaseMs:  s.config.BackoffBaseMs,
		backoffDelayMs: s.config.BackoffDelayMs,
	})

	for i := 1; i < s.config.BackoffMaxRetries+1; i++ {
		conn, err := pgx.Connect(ctx, connectionString)
		if err != nil {
			if containsError(connectionErrors, err) {
				delay := s.delay.getDelay()
				time.Sleep(delay)
				s.logger.Info(fmt.Sprintf("Retry connection...Retry attempt: %v with delay: %v", i, delay))

				if i == s.config.BackoffMaxRetries {
					return nil, err
				}

				continue
			}

			return nil, err
		}

		s.conn = conn
		s.connCred.url = connectionString
		break
	}

	s.logger.Info("Connected")

	return s, nil
}

func (s *SlsConn) getIdleProcessesListByMinimumTimeout(ctx context.Context) ([]statActivity, error) {
	query := `
    WITH processes AS(
      SELECT
         EXTRACT(EPOCH FROM (Now() - state_change)) AS idle_time,
         pid
      FROM pg_stat_activity
      WHERE usename=$1
        AND datname=$2
        AND state='idle'
    )
    SELECT pid
    FROM processes
    WHERE idle_time > $3
    LIMIT $4;`

	rows, err := s.conn.Query(
		ctx,
		query,
		s.connCred.user,
		s.connCred.database,
		s.config.MinConnectionIdleTimeSec,
		s.config.MaxIdleConnectionsToKill,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	stats := make([]statActivity, 0)

	for rows.Next() {
		var stat statActivity
		if err := rows.Scan(&stat.pid); err != nil {
			return nil, err
		}

		stats = append(stats, stat)
	}

	return stats, nil
}

func (s *SlsConn) killProcesses(ctx context.Context, pids []int) error {
	query := `
	SELECT pg_terminate_backend(pid)
    FROM pg_stat_activity
    WHERE pid = ANY ($1) AND state='idle'`

	ids := &pgtype.Int4Array{}
	if err := ids.Set(pids); err != nil {
		return err
	}

	if _, err := s.conn.Exec(ctx, query, ids); err != nil {
		return err
	}

	return nil
}

func (s SlsConn) getProcessCount(ctx context.Context) (uint8, error) {
	query := `
	SELECT COUNT(pid)
    FROM pg_stat_activity
    WHERE datname=$1
      AND usename=$2;`
	var count uint8

	err := s.conn.QueryRow(
		ctx,
		query,
		s.connCred.database,
		s.connCred.user,
	).Scan(&count)

	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *SlsConn) Clean(ctx context.Context) error {
	count, err := s.getProcessCount(ctx)
	if err != nil {
		return err
	}
	s.logger.Info(fmt.Sprintf("Total processes: %v", count))

	if float32(count) > float32(s.config.MaxConnections)*s.config.ConnUtilization {
		processList, err := s.getIdleProcessesListByMinimumTimeout(ctx)
		if err != nil {
			return err
		}

		pidLst := make([]int, 0)
		for _, activity := range processList {
			pidLst = append(pidLst, activity.pid)
		}
		if err := s.killProcesses(ctx, pidLst); err != nil {
			return err
		}

		s.logger.Info(fmt.Sprintf("Killed processes: %v", len(processList)))
	}

	return nil
}

func (s SlsConn) GetConnection() *pgx.Conn {
	return s.conn
}

func (s SlsConn) Close(ctx context.Context) error {
	return s.conn.Close(ctx)
}

func (s SlsConn) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	res, err := s.retry(ctx, "Query", sql, args...)
	if err != nil {
		return nil, err
	}
	rows, ok := res.Interface().(pgx.Rows)
	if !ok {
		return nil, errors.New("type mismatch, should be of type pgx.Rows")
	}

	return rows, nil
}

func (s SlsConn) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	res, err := s.retry(ctx, "Exec", sql, args...)
	if err != nil {
		return nil, err
	}

	commandTag, ok := res.Interface().(pgconn.CommandTag)
	if !ok {
		return nil, errors.New("type mismatch, should be of type pgx.CommandTag")
	}

	return commandTag, nil
}

// Re-usable method to retry any pgx method
func (s SlsConn) retry(ctx context.Context, function string, sql string, args ...interface{}) (reflect.Value, error) {
	var res reflect.Value
	for i := 1; i < s.config.BackoffMaxRetries+1; i++ {
		allArgs := []interface{}{ctx, sql}
		allArgs = append(allArgs, args...)
		out, err := callFuncByName(s.conn, function, allArgs...)
		if err != nil {
			if containsError(queryErrors, err) {
				delay := s.delay.getDelay()
				time.Sleep(delay)

				conn, err := pgx.Connect(ctx, s.connCred.url)
				if err != nil {
					if containsError(connectionErrors, err) {
						continue
					}

					return reflect.Value{}, err
				}

				s.conn = conn
				s.logger.Info(fmt.Sprintf("Retry query...Retry attempt: %v with delay: %v", i, delay))

				if i == s.config.BackoffMaxRetries {
					return reflect.Value{}, err
				}

				continue
			}

			return reflect.Value{}, err
		}

		res = out
		break
	}

	return res, nil
}
