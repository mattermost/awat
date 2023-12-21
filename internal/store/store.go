// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"context"
	"database/sql"
	"net/url"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	// enable the pq driver
	_ "github.com/lib/pq"
)

// SQLStore abstracts access to the database.
type SQLStore struct {
	db     *sqlx.DB
	logger logrus.FieldLogger
}

// New creates and initializes a new SQLStore instance. It takes a database connection string (DSN)
// and a logger instance. It returns an initialized SQLStore or an error if the connection to the database fails.
func New(dsn string, logger *logrus.Logger) (*SQLStore, error) {
	dbURL, err := url.Parse(dsn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse dsn as an url")
	}

	db, err := sqlx.Connect("postgres", dbURL.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to postgres database")
	}

	return &SQLStore{
		db,
		logger,
	}, nil
}

// queryer is an interface describing a resource that can query.
//
// It exactly matches sqlx.Queryer, existing simply to constrain sqlx usage to this file.
type queryer interface {
	sqlx.Queryer
}

// get queries for a single row, writing the result into dest.
//
// Use this to simplify querying for a single row or column. Dest may be a pointer to a simple
// type, or a struct with fields to be populated from the returned columns.
func (sqlStore *SQLStore) get(q sqlx.Queryer, dest interface{}, query string, args ...interface{}) error {
	query = sqlStore.db.Rebind(query)

	return sqlx.Get(q, dest, query, args...)
}

// builder is an interface describing a resource that can construct SQL and arguments.
//
// It exists to allow consuming any squirrel.*Builder type.
type builder interface {
	ToSql() (string, []interface{}, error)
}

// get queries for a single row, building the sql, and writing the result into dest.
//
// Use this to simplify querying for a single row or column. Dest may be a pointer to a simple
// type, or a struct with fields to be populated from the returned columns.
func (sqlStore *SQLStore) getBuilder(q sqlx.Queryer, dest interface{}, b builder) error {
	sql, args, err := b.ToSql()
	if err != nil {
		return errors.Wrap(err, "failed to build sql")
	}

	sql = sqlStore.db.Rebind(sql)

	err = sqlx.Get(q, dest, sql, args...)
	if err != nil {
		return err
	}

	return nil
}

// selectBuilder queries for one or more rows, building the sql, and writing the result into dest.
//
// Use this to simplify querying for multiple rows (and possibly columns). Dest may be a slice of
// a simple, or a slice of a struct with fields to be populated from the returned columns.
func (sqlStore *SQLStore) selectBuilder(q sqlx.Queryer, dest interface{}, b builder) error {
	sql, args, err := b.ToSql()
	if err != nil {
		return errors.Wrap(err, "failed to build sql")
	}

	sql = sqlStore.db.Rebind(sql)

	err = sqlx.Select(q, dest, sql, args...)
	if err != nil {
		return err
	}

	return nil
}

// execer is an interface describing a resource that can execute write queries.
//
// It allows the use of *sqlx.Db and *sqlx.Tx.
type execer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	DriverName() string
}

// exec executes the given query using positional arguments, automatically rebinding for the db.
func (sqlStore *SQLStore) exec(e execer, sql string, args ...interface{}) (sql.Result, error) {
	sql = sqlStore.db.Rebind(sql)
	return e.Exec(sql, args...)
}

// exec executes the given query, building the necessary sql.
func (sqlStore *SQLStore) execBuilder(e execer, b builder) (sql.Result, error) {
	sql, args, err := b.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build sql")
	}

	return sqlStore.exec(e, sql, args...)
}

// dbInterface is an interface describing a resource that can execute read and write queries.
//
// It allows the use of *sqlx.Db and *sqlx.Tx.
type dbInterface interface {
	execer
	queryer
}

type transactionStarter interface {
	BeginTxx(context.Context, *sql.TxOptions) (*sqlx.Tx, error)
}

func (sqlStore *SQLStore) beginTransaction(tr transactionStarter) (*Transaction, error) {
	return sqlStore.beginCustomTransaction(tr, nil)
}

func (sqlStore *SQLStore) beginCustomTransaction(tr transactionStarter, opts *sql.TxOptions) (*Transaction, error) {
	tx, err := tr.BeginTxx(context.Background(), opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}

	return &Transaction{
		Tx:        tx,
		sqlStore:  sqlStore,
		committed: false,
	}, nil
}

// Transaction is a wrapper around *sqlx.Tx providing convenience methods.
type Transaction struct {
	*sqlx.Tx
	sqlStore  *SQLStore
	committed bool
}

// Commit commits the pending transaction.
func (t *Transaction) Commit() error {
	err := t.Tx.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit the transaction")
	}
	t.committed = true
	return nil
}

// RollbackUnlessCommitted rollback the transaction if it is not committed.
func (t *Transaction) RollbackUnlessCommitted() {
	if !t.committed {
		err := t.Tx.Rollback()
		if err != nil {
			t.sqlStore.logger.Errorf("error: failed to rollback uncommitted transaction: %s", err.Error())
		}
	}
}

// tableExists determines if the given table name exists in the database.
func (sqlStore *SQLStore) tableExists(tableName string) (bool, error) {
	var tableExists bool

	err := sqlStore.get(sqlStore.db, &tableExists,
		"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = 'system')",
	)
	if err != nil {
		return false, errors.Wrapf(err, "failed to check if %s table exists", tableName)
	}

	return tableExists, nil
}
