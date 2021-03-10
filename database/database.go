// /home/krylon/go/src/ticker/database/database.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-03-10 13:38:15 krylon>

// Package database provides the storage/persistence layer,
// using good old SQLite as its backend.
package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"sync"
	"ticker/common"
	"ticker/feed"
	"ticker/logdomain"
	"ticker/query"
	"ticker/tag"
	"time"

	"github.com/blicero/krylib"

	_ "github.com/mattn/go-sqlite3" // Import the database driver
)

var (
	openLock sync.Mutex
	idCnt    int64
)

// ErrTxInProgress indicates that an attempt to initiate a transaction failed
// because there is already one in progress.
var ErrTxInProgress = errors.New("A Transaction is already in progress")

// ErrNoTxInProgress indicates that an attempt was made to finish a
// transaction when none was active.
var ErrNoTxInProgress = errors.New("There is no transaction in progress")

// ErrEmptyUpdate indicates that an update operation would not change any
// values.
var ErrEmptyUpdate = errors.New("Update operation does not change any values")

// ErrInvalidValue indicates that one or more parameters passed to a method
// had values that are invalid for that operation.
var ErrInvalidValue = errors.New("Invalid value for parameter")

// ErrObjectNotFound indicates that an Object was not found in the database.
var ErrObjectNotFound = errors.New("object was not found in database")

// ErrInvalidSavepoint is returned when a user of the Database uses an unkown
// (or expired) savepoint name.
var ErrInvalidSavepoint = errors.New("that save point does not exist")

// If a query returns an error and the error text is matched by this regex, we
// consider the error as transient and try again after a short delay.
var retryPat = regexp.MustCompile("(?i)database is (?:locked|busy)")

// worthARetry returns true if an error returned from the database
// is matched by the retryPat regex.
func worthARetry(e error) bool {
	return retryPat.MatchString(e.Error())
} // func worthARetry(e error) bool

// retryDelay is the amount of time we wait before we repeat a database
// operation that failed due to a transient error.
const retryDelay = 25 * time.Millisecond

func waitForRetry() {
	time.Sleep(retryDelay)
} // func waitForRetry()

// Database is the storage backend for managing Feeds and news.
//
// It is not safe to share a Database instance between goroutines, however
// opening multiple connections to the same Database is safe.
type Database struct {
	id            int64
	db            *sql.DB
	tx            *sql.Tx
	log           *log.Logger
	path          string
	spNameCounter int
	spNameCache   map[string]string
	queries       map[query.ID]*sql.Stmt
}

// Open opens a Database. If the database specified by the path does not exist,
// yet, it is created and initialized.
func Open(path string) (*Database, error) {
	var (
		err      error
		dbExists bool
		db       = &Database{
			path:          path,
			spNameCounter: 1,
			spNameCache:   make(map[string]string),
			queries:       make(map[query.ID]*sql.Stmt),
		}
	)

	openLock.Lock()
	defer openLock.Unlock()
	idCnt++
	db.id = idCnt

	if db.log, err = common.GetLogger(logdomain.Database); err != nil {
		return nil, err
	} else if common.Debug {
		db.log.Printf("[DEBUG] Open database %s\n", path)
	}

	var connstring = fmt.Sprintf("%s?_locking=NORMAL&_journal=WAL&_fk=1&recursive_triggers=0",
		path)

	if dbExists, err = krylib.Fexists(path); err != nil {
		db.log.Printf("[ERROR] Failed to check if %s already exists: %s\n",
			path,
			err.Error())
		return nil, err
	} else if db.db, err = sql.Open("sqlite3", connstring); err != nil {
		db.log.Printf("[ERROR] Failed to open %s: %s\n",
			path,
			err.Error())
		return nil, err
	}

	if !dbExists {
		if err = db.initialize(); err != nil {
			var e2 error
			if e2 = db.db.Close(); e2 != nil {
				db.log.Printf("[CRITICAL] Failed to close database: %s\n",
					e2.Error())
				return nil, e2
			} else if e2 = os.Remove(path); e2 != nil {
				db.log.Printf("[CRITICAL] Failed to remove database file %s: %s\n",
					db.path,
					e2.Error())
			}
			return nil, err
		}
		db.log.Printf("[INFO] Database at %s has been initialized\n",
			path)
	}

	return db, nil
} // func Open(path string) (*Database, error)

func (db *Database) initialize() error {
	var err error
	var tx *sql.Tx

	if common.Debug {
		db.log.Printf("[DEBUG] Initialize fresh database at %s\n",
			db.path)
	}

	if tx, err = db.db.Begin(); err != nil {
		db.log.Printf("[ERROR] Cannot begin transaction: %s\n",
			err.Error())
		return err
	}

	for _, q := range initQueries {
		db.log.Printf("[TRACE] Execute init query:\n%s\n",
			q)
		if _, err = tx.Exec(q); err != nil {
			db.log.Printf("[ERROR] Cannot execute init query: %s\n%s\n",
				err.Error(),
				q)
			if rbErr := tx.Rollback(); rbErr != nil {
				db.log.Printf("[CANTHAPPEN] Cannot rollback transaction: %s\n",
					rbErr.Error())
				return rbErr
			}
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		db.log.Printf("[CANTHAPPEN] Failed to commit init transaction: %s\n",
			err.Error())
		return err
	}

	return nil
} // func (db *Database) initialize() error

// Close closes the database.
// If there is a pending transaction, it is rolled back.
func (db *Database) Close() error {
	// I wonder if would make more snese to panic() if something goes wrong

	var err error

	if db.tx != nil {
		if err = db.tx.Rollback(); err != nil {
			db.log.Printf("[CRITICAL] Cannot roll back pending transaction: %s\n",
				err.Error())
			return err
		}
		db.tx = nil
	}

	for key, stmt := range db.queries {
		if err = stmt.Close(); err != nil {
			db.log.Printf("[CRITICAL] Cannot close statement handle %s: %s\n",
				key,
				err.Error())
			return err
		}
		delete(db.queries, key)
	}

	if err = db.db.Close(); err != nil {
		db.log.Printf("[CRITICAL] Cannot close database: %s\n",
			err.Error())
	}

	db.db = nil
	return nil
} // func (db *Database) Close() error

func (db *Database) getQuery(id query.ID) (*sql.Stmt, error) {
	var (
		stmt  *sql.Stmt
		found bool
		err   error
	)

	if stmt, found = db.queries[id]; found {
		return stmt, nil
	} else if _, found = dbQueries[id]; !found {
		return nil, fmt.Errorf("Unknown Query %d",
			id)
	}

	db.log.Printf("[TRACE] Prepare query %s\n", id)

PREPARE_QUERY:
	if stmt, err = db.db.Prepare(dbQueries[id]); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto PREPARE_QUERY
		}

		db.log.Printf("[ERROR] Cannor parse query %s: %s\n%s\n",
			id,
			err.Error(),
			dbQueries[id])
		return nil, err
	}

	db.queries[id] = stmt
	return stmt, nil
} // func (db *Database) getQuery(query.ID) (*sql.Stmt, error)

func (db *Database) resetSPNamespace() {
	db.spNameCounter = 1
	db.spNameCache = make(map[string]string)
} // func (db *Database) resetSPNamespace()

func (db *Database) generateSPName(name string) string {
	var spname = fmt.Sprintf("Savepoint%05d",
		db.spNameCounter)

	db.spNameCache[name] = spname
	db.spNameCounter++
	return spname
} // func (db *Database) generateSPName() string

// PerformMaintenance performs some maintenance operations on the database.
// It cannot be called while a transaction is in progress and will block
// pretty much all access to the database while it is running.
func (db *Database) PerformMaintenance() error {
	var mQueries = []string{
		"PRAGMA wal_checkpoint(TRUNCATE)",
		"VACUUM",
		"REINDEX",
		"ANALYZE",
	}
	var err error

	if db.tx != nil {
		return ErrTxInProgress
	}

	for _, q := range mQueries {
		if _, err = db.db.Exec(q); err != nil {
			db.log.Printf("[ERROR] Failed to execute %s: %s\n",
				q,
				err.Error())
		}
	}

	return nil
} // func (db *Database) PerformMaintenance() error

// Begin begins an explicit database transaction.
// Only one transaction can be in progress at once, attempting to start one,
// while another transaction is already in progress will yield ErrTxInProgress.
func (db *Database) Begin() error {
	var err error

	db.log.Printf("[DEBUG] Database#%d Begin Transaction\n",
		db.id)

	if db.tx != nil {
		return ErrTxInProgress
	}

BEGIN_TX:
	for db.tx == nil {
		if db.tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				continue BEGIN_TX
			} else {
				db.log.Printf("[ERROR] Failed to start transaction: %s\n",
					err.Error())
				return err
			}
		}
	}

	db.resetSPNamespace()

	return nil
} // func (db *Database) Begin() error

// SavepointCreate creates a savepoint with the given name.
//
// Savepoints only make sense within a running transaction, and just like
// with explicit transactions, managing them is the responsibility of the
// user of the Database.
//
// Creating a savepoint without a surrounding transaction is not allowed,
// even though SQLite allows it.
//
// For details on how Savepoints work, check the excellent SQLite
// documentation, but here's a quick guide:
//
// Savepoints are kind-of-like transactions within a transaction: One
// can create a savepoint, make some changes to the database, and roll
// back to that savepoint, discarding all changes made between
// creating the savepoint and rolling back to it. Savepoints can be
// quite useful, but there are a few things to keep in mind:
//
// - Savepoints exist within a transaction. When the surrounding transaction
//   is finished, all savepoints created within that transaction cease to exist,
//   no matter if the transaction is commited or rolled back.
//
// - When the database is recovered after being interrupted during a
//   transaction, e.g. by a power outage, the entire transaction is rolled back,
//   including all savepoints that might exist.
//
// - When a savepoint is released, nothing changes in the state of the
//   surrounding transaction. That means rolling back the surrounding
//   transaction rolls back the entire transaction, regardless of any
//   savepoints within.
//
// - Savepoints do not nest. Releasing a savepoint releases it and *all*
//   existing savepoints that have been created before it. Rolling back to a
//   savepoint removes that savepoint and all savepoints created after it.
func (db *Database) SavepointCreate(name string) error {
	var err error

	db.log.Printf("[DEBUG] SavepointCreate(%s)\n",
		name)

	if db.tx == nil {
		return ErrNoTxInProgress
	}

SAVEPOINT:
	// It appears that the SAVEPOINT statement does not support placeholders.
	// But I do want to used named savepoints.
	// And I do want to use the given name so that no SQL injection
	// becomes possible.
	// It would be nice if the database package or at least the SQLite
	// driver offered a way to escape the string properly.
	// One possible solution would be to use names generated by the
	// Database instead of user-defined names.
	//
	// But then I need a way to use the Database-generated name
	// in rolling back and releasing the savepoint.
	// I *could* use the names strictly inside the Database, store them in
	// a map or something and hand out a key to that name to the user.
	// Since savepoint only exist within one transaction, I could even
	// re-use names from one transaction to the next.
	//
	// Ha! I could accept arbitrary names from the user, generate a
	// clean name, and store these in a map. That way the user can
	// still choose names that are outwardly visible, but they do
	// not touch the Database itself.
	//
	//if _, err = db.tx.Exec("SAVEPOINT ?", name); err != nil {
	// if _, err = db.tx.Exec("SAVEPOINT " + name); err != nil {
	// 	if worthARetry(err) {
	// 		waitForRetry()
	// 		goto SAVEPOINT
	// 	}

	// 	db.log.Printf("[ERROR] Failed to create savepoint %s: %s\n",
	// 		name,
	// 		err.Error())
	// }

	var internalName = db.generateSPName(name)

	var spQuery = "SAVEPOINT " + internalName

	if _, err = db.tx.Exec(spQuery); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto SAVEPOINT
		}

		db.log.Printf("[ERROR] Failed to create savepoint %s: %s\n",
			name,
			err.Error())
	}

	return err
} // func (db *Database) SavepointCreate(name string) error

// SavepointRelease releases the Savepoint with the given name, and all
// Savepoints created before the one being release.
func (db *Database) SavepointRelease(name string) error {
	var (
		err                   error
		internalName, spQuery string
		validName             bool
	)

	db.log.Printf("[DEBUG] SavepointRelease(%s)\n",
		name)

	if db.tx != nil {
		return ErrNoTxInProgress
	}

	if internalName, validName = db.spNameCache[name]; !validName {
		db.log.Printf("[ERROR] Attempt to release unknown Savepoint %q\n",
			name)
		return ErrInvalidSavepoint
	}

	db.log.Printf("[DEBUG] Release Savepoint %q (%q)",
		name,
		db.spNameCache[name])

	spQuery = "RELEASE SAVEPOINT " + internalName

SAVEPOINT:
	if _, err = db.tx.Exec(spQuery); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto SAVEPOINT
		}

		db.log.Printf("[ERROR] Failed to release savepoint %s: %s\n",
			name,
			err.Error())
	} else {
		delete(db.spNameCache, internalName)
	}

	return err
} // func (db *Database) SavepointRelease(name string) error

// SavepointRollback rolls back the running transaction to the given savepoint.
func (db *Database) SavepointRollback(name string) error {
	var (
		err                   error
		internalName, spQuery string
		validName             bool
	)

	db.log.Printf("[DEBUG] SavepointRollback(%s)\n",
		name)

	if db.tx != nil {
		return ErrNoTxInProgress
	}

	if internalName, validName = db.spNameCache[name]; !validName {
		return ErrInvalidSavepoint
	}

	spQuery = "ROLLBACK TO SAVEPOINT " + internalName

SAVEPOINT:
	if _, err = db.tx.Exec(spQuery); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto SAVEPOINT
		}

		db.log.Printf("[ERROR] Failed to create savepoint %s: %s\n",
			name,
			err.Error())
	}

	delete(db.spNameCache, name)
	return err
} // func (db *Database) SavepointRollback(name string) error

// Rollback terminates a pending transaction, undoing any changes to the
// database made during that transaction.
// If no transaction is active, it returns ErrNoTxInProgress
func (db *Database) Rollback() error {
	var err error

	db.log.Printf("[DEBUG] Database#%d Roll back Transaction\n",
		db.id)

	if db.tx == nil {
		return ErrNoTxInProgress
	} else if err = db.tx.Rollback(); err != nil {
		return fmt.Errorf("Cannot roll back database transaction: %s",
			err.Error())
	}

	db.tx = nil
	db.resetSPNamespace()

	return nil
} // func (db *Database) Rollback() error

// Commit ends the active transaction, making any changes made during that
// transaction permanent and visible to other connections.
// If no transaction is active, it returns ErrNoTxInProgress
func (db *Database) Commit() error {
	var err error

	db.log.Printf("[DEBUG] Database#%d Commit Transaction\n",
		db.id)

	if db.tx == nil {
		return ErrNoTxInProgress
	} else if err = db.tx.Commit(); err != nil {
		return fmt.Errorf("Cannot commit transaction: %s",
			err.Error())
	}

	db.resetSPNamespace()
	db.tx = nil
	return nil
} // func (db *Database) Commit() error

// FeedAdd adds a Feed to the database.
func (db *Database) FeedAdd(f *feed.Feed) error {
	const qid = query.FeedAdd
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var res sql.Result

EXEC_QUERY:
	if res, err = stmt.Exec(f.Name, f.URL, f.Homepage, int64(f.Interval.Seconds())); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add Feed %s (%s) to database: %s",
				f.Name,
				f.URL,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else {
		var feedID int64

		if feedID, err = res.LastInsertId(); err != nil {
			db.log.Printf("[ERROR] Cannot get ID of new Feed %s: %s\n",
				f.Name,
				err.Error())
			return err
		}

		status = true
		f.ID = feedID
		return nil
	}
} // func (db *Database) FeedAdd(f *feed.Feed) error

// FeedGetAll returns a list of all Feeds stored in the datbase.
func (db *Database) FeedGetAll() ([]feed.Feed, error) {
	const qid = query.FeedGetAll
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var list = make([]feed.Feed, 0, 10)

	for rows.Next() {
		var (
			f               feed.Feed
			interval, stamp int64
		)

		if err = rows.Scan(&f.ID, &f.Name, &f.URL, &f.Homepage, &interval, &stamp, &f.Active); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n", err.Error())
			return nil, err
		} else if stamp != 0 {
			f.LastUpdate = time.Unix(stamp, 0)
		}

		f.Interval = time.Second * time.Duration(interval)

		list = append(list, f)
	}

	return list, nil
} // func (db *Database) FeedGetAll() ([]feed.Feed, error)

// FeedGetMap returns a map of Feeds usable in HTML templates.
func (db *Database) FeedGetMap() (map[int64]feed.Feed, error) {
	const qid = query.FeedGetAll
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var fmap = make(map[int64]feed.Feed, 16)

	for rows.Next() {
		var (
			f               feed.Feed
			interval, stamp int64
		)

		if err = rows.Scan(&f.ID, &f.Name, &f.URL, &f.Homepage, &interval, &stamp, &f.Active); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n", err.Error())
			return nil, err
		} else if stamp != 0 {
			f.LastUpdate = time.Unix(stamp, 0)
		}

		f.Interval = time.Second * time.Duration(interval)

		fmap[f.ID] = f
	}

	return fmap, nil
} // func (db *Database) FeedGetMap() (map[int64]feed.Feed, error)

// FeedGetDue fetches only those Feeds that are due for a refresh.
func (db *Database) FeedGetDue() ([]feed.Feed, error) {
	const qid = query.FeedGetDue
	var (
		err  error
		stmt *sql.Stmt
		now  = time.Now().Unix()
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(now); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var list = make([]feed.Feed, 0, 10)

	for rows.Next() {
		var (
			id                  int64
			name, url, homepage string
			active              bool
			f                   *feed.Feed
			interval, stamp     int64
		)

		// if err = rows.Scan(&f.ID, &f.Name, &f.URL, &f.Homepage, &interval, &stamp, &f.Active); err != nil {
		if err = rows.Scan(&id, &name, &url, &homepage, &interval, &stamp, &active); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n", err.Error())
			return nil, err
		} else if f, err = feed.New(id, name, url, homepage, time.Second*time.Duration(interval), active); err != nil {
			db.log.Printf("[ERROR] Cannot create Feed %s: %s\n",
				name,
				err.Error())
			return nil, err
		}

		if stamp != 0 {
			f.LastUpdate = time.Unix(stamp, 0)
		}

		// f.Interval = time.Second * time.Duration(interval)

		list = append(list, *f)
	}

	return list, nil
} // func (db *Database) FeedGetDue() ([]feed.Feed, error)

// FeedGetByID fetches the Feed with the given ID.
func (db *Database) FeedGetByID(id int64) (*feed.Feed, error) {
	const qid = query.FeedGetByID
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			fd              = &feed.Feed{ID: id}
			stamp, interval int64
		)

		if err = rows.Scan(&fd.Name, &fd.URL, &fd.Homepage, &interval, &stamp, &fd.Active); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		}

		fd.Interval = time.Second * time.Duration(interval)
		if stamp != 0 {
			fd.LastUpdate = time.Unix(stamp, 0)
		}

		return fd, nil
	}

	return nil, nil
} // func (db *Database) FeedGetByID(id int64) (*feed.Feed, error)

// FeedSetActive sets the Feed's Active flag to the given value.
func (db *Database) FeedSetActive(id int64, active bool) error {
	const qid query.ID = query.FeedSetActive
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(active, id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Cannot set active flag for Feed %d: %s\n",
			id,
			err.Error())
		return err
	}

	status = true
	return nil
} // func (db *Database) FeedSetActive(id int64, active bool) error

// FeedSetTimestamp updates the refresh timestamp of the given Feed.
func (db *Database) FeedSetTimestamp(f *feed.Feed, stamp time.Time) error {
	const qid = query.FeedSetTimestamp
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var (
		cnt int64
		res sql.Result
	)

EXEC_QUERY:
	if res, err = stmt.Exec(stamp.Unix(), f.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Cannot update timestamp for Feed %s (%d): %s\n",
			f.Name,
			f.ID,
			err.Error())
		return err
	} else if cnt, err = res.RowsAffected(); err != nil {
		db.log.Printf("[ERROR] Cannot query number of rows affected: %s\n",
			err.Error())
		return err
	} else if cnt != 1 {
		err = fmt.Errorf("Unexpected number of rows affected: %d (expected 1)",
			cnt)
		db.log.Printf("[ERROR] %s\n", err.Error())
		return err
	}

	f.LastUpdate = stamp
	status = true
	return nil
} // func (db *Database) FeedSetTimestamp(f *feed.Feed, stamp time.Time) error

// FeedDelete deletes the Feed with the given ID from the database.
func (db *Database) FeedDelete(id int64) error {
	const qid = query.FeedDelete
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var (
		cnt int64
		res sql.Result
	)

EXEC_QUERY:
	if res, err = stmt.Exec(id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Cannot update timestamp for Feed %d: %s\n",
			id,
			err.Error())
		return err
	} else if cnt, err = res.RowsAffected(); err != nil {
		db.log.Printf("[ERROR] Cannot query number of rows affected: %s\n",
			err.Error())
		return err
	} else if cnt != 1 {
		err = fmt.Errorf("Unexpected number of rows affected: %d (expected 1)",
			cnt)
		db.log.Printf("[ERROR] %s\n", err.Error())
		return err
	}

	status = true
	return nil
} // func (db *Database) FeedDelete(id int64) error

// FeedModify modifies a Feed.
func (db *Database) FeedModify(
	f *feed.Feed,
	name, lnk, homepage string,
	interval time.Duration) error {
	const qid query.ID = query.FeedModify
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(
		name,
		lnk,
		homepage,
		interval.Seconds(),
		f.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Cannot update Feed %s (%d): %s\n",
			f.Name,
			f.ID,
			err.Error())
		return err
	}

	f.Name = name
	f.URL = lnk
	f.Homepage = homepage
	f.Interval = interval
	status = true
	return nil
} // func (db *Database) FeedModify(...) error

// ItemAdd adds an Item to the database.
func (db *Database) ItemAdd(item *feed.Item) error {
	const qid = query.ItemAdd
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var res sql.Result

EXEC_QUERY:
	if res, err = stmt.Exec(item.FeedID, item.URL, item.Title, item.Description, item.Timestamp.Unix()); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add Item %s (%s) to database: %s",
				item.Title,
				item.URL,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else {
		var itemID int64

		if itemID, err = res.LastInsertId(); err != nil {
			db.log.Printf("[ERROR] Cannot get ID of new Item %q: %s\n",
				item.Title,
				err.Error())
			return err
		}

		status = true
		item.ID = itemID
		return nil
	}
} // func (db *Database) ItemAdd(item *feed.Item) error

// ItemGetRecent returns the <limit> most recent news Items.
func (db *Database) ItemGetRecent(limit int) ([]feed.Item, error) {
	const qid = query.ItemGetRecent
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(limit); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var items = make([]feed.Item, 0, limit)

	for rows.Next() {
		var (
			item   feed.Item
			rating *float64
			stamp  int64
		)

		if err = rows.Scan(
			&item.ID,
			&item.FeedID,
			&item.URL,
			&item.Title,
			&item.Description,
			&stamp,
			&item.Read,
			&rating); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		} else if item.Tags, err = db.TagGetByItem(item.ID); err != nil {
			db.log.Printf("[ERROR] Cannot load tags for Item %q (%d): %s\n",
				item.Title,
				item.ID,
				err.Error())
			return nil, err
		}

		if rating != nil {
			item.Rating = *rating
			item.ManuallyRated = true
		} else {
			item.Rating = math.NaN()
		}
		item.Timestamp = time.Unix(stamp, 0)

		items = append(items, item)
	}

	return items, nil
} // func (db *Database) ItemGetRecent(limit int) ([]feed.Item, error)

// ItemGetRated returns all Items that have been rated.
func (db *Database) ItemGetRated() ([]feed.Item, error) {
	const qid = query.ItemGetRated
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var items = make([]feed.Item, 0, 64)

	for rows.Next() {
		var (
			rating *float64
			stamp  int64
			item   = feed.Item{ManuallyRated: true}
		)

		if err = rows.Scan(
			&item.ID,
			&item.FeedID,
			&item.URL,
			&item.Title,
			&item.Description,
			&stamp,
			&item.Read,
			&rating); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		} else if item.Tags, err = db.TagGetByItem(item.ID); err != nil {
			db.log.Printf("[ERROR] Cannot load tags for Item %q (%d): %s\n",
				item.Title,
				item.ID,
				err.Error())
			return nil, err
		}

		if rating != nil {
			item.Rating = *rating
		} else {
			item.Rating = math.NaN()
		}
		item.Timestamp = time.Unix(stamp, 0)
		items = append(items, item)
	}

	return items, nil
} // func (db *Database) ItemGetRated(limit int) ([]feed.Item, error)

// ItemGetByID fetches an Item by its ID.
func (db *Database) ItemGetByID(id int64) (*feed.Item, error) {
	const qid = query.ItemGetByID
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			stamp  int64
			rating *float64
			item   = &feed.Item{ID: id}
		)

		if err = rows.Scan(
			&item.FeedID,
			&item.URL,
			&item.Title,
			&item.Description,
			&stamp,
			&item.Read,
			&rating); err != nil {
			db.log.Printf("[ERROR] Cannot scan Row for Item %d: %s\n",
				id,
				err.Error())
			return nil, err
		} else if item.Tags, err = db.TagGetByItem(item.ID); err != nil {
			db.log.Printf("[ERROR] Cannot load tags for Item %q (%d): %s\n",
				item.Title,
				item.ID,
				err.Error())
			return nil, err
		} else if rating != nil {
			item.ManuallyRated = true
			item.Rating = *rating
		}

		item.Timestamp = time.Unix(stamp, 0)

		return item, nil
	}

	return nil, nil
} // func (db *Database) ItemGetByID(id int64) (*feed.Item, error)

// ItemGetByURL fetches an Item by its URL.
func (db *Database) ItemGetByURL(uri string) (*feed.Item, error) {
	const qid = query.ItemGetByURL
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(uri); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			stamp  int64
			rating *float64
			item   = &feed.Item{URL: uri}
		)

		if err = rows.Scan(
			&item.ID,
			&item.FeedID,
			&item.Title,
			&item.Description,
			&stamp,
			&item.Read,
			&rating); err != nil {
			db.log.Printf("[ERROR] Cannot scan Row for Item %s: %s\n",
				uri,
				err.Error())
			return nil, err
		} else if item.Tags, err = db.TagGetByItem(item.ID); err != nil {
			db.log.Printf("[ERROR] Cannot load tags for Item %q (%d): %s\n",
				item.Title,
				item.ID,
				err.Error())
			return nil, err
		} else if rating != nil {
			item.ManuallyRated = true
			item.Rating = *rating
		}

		item.Timestamp = time.Unix(stamp, 0)

		return item, nil
	}

	return nil, nil
} // func (db *Database) ItemGetByURL(uri string) (*feed.Item, error)

// ItemGetByFeed fetches the <limit> most recent Items belonging to the
// given <feedID>.
func (db *Database) ItemGetByFeed(feedID, limit int64) ([]feed.Item, error) {
	const qid query.ID = query.ItemGetByFeed
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(feedID, limit); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var items = make([]feed.Item, 0, limit)

	for rows.Next() {
		var (
			item   = feed.Item{FeedID: feedID}
			rating *float64
			stamp  int64
		)

		if err = rows.Scan(
			&item.ID,
			&item.URL,
			&item.Title,
			&item.Description,
			&stamp,
			&item.Read,
			&rating); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		} else if item.Tags, err = db.TagGetByItem(item.ID); err != nil {
			db.log.Printf("[ERROR] Cannot load tags for Item %q (%d): %s\n",
				item.Title,
				item.ID,
				err.Error())
			return nil, err
		}

		if rating != nil {
			item.ManuallyRated = true
			item.Rating = *rating
		} else {
			item.Rating = math.NaN()
		}
		item.Timestamp = time.Unix(stamp, 0)
		items = append(items, item)
	}

	return items, nil
} // func (db *Database) ItemGetByFeed(feedID, limit int64) ([]feed.Item, error)

// ItemGetAll fetches items from all feeds, ordered by their timestamps in
// descending order, skipping the first <offset> items, returning the next
// <cnt> items.
func (db *Database) ItemGetAll(cnt, offset int64) ([]feed.Item, error) {
	const qid query.ID = query.ItemGetAll
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(cnt, offset); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var items []feed.Item

	if cnt > 0 {
		items = make([]feed.Item, 0, cnt)
	} else {
		items = make([]feed.Item, 0)
	}

	for rows.Next() {
		var (
			item   feed.Item
			rating *float64
			stamp  int64
		)

		if err = rows.Scan(
			&item.ID,
			&item.FeedID,
			&item.URL,
			&item.Title,
			&item.Description,
			&stamp,
			&item.Read,
			&rating); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		} else if item.Tags, err = db.TagGetByItem(item.ID); err != nil {
			db.log.Printf("[ERROR] Cannot load tags for Item %q (%d): %s\n",
				item.Title,
				item.ID,
				err.Error())
			return nil, err
		} else if rating != nil {
			item.ManuallyRated = true
			item.Rating = *rating
		} else {
			item.Rating = math.NaN()
		}
		item.Timestamp = time.Unix(stamp, 0)
		items = append(items, item)
	}

	return items, nil
} // func (db *Database) ItemGetAll(cnt, offset int64) ([]feed.Item, error)

// ItemGetFTS retrieves Items that match a full-text search query.
func (db *Database) ItemGetFTS(fts string) ([]feed.Item, error) {
	const qid query.ID = query.ItemGetFTS
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(fts); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var items = make([]feed.Item, 0, 32)

	for rows.Next() {
		var (
			item   feed.Item
			rating *float64
			stamp  int64
		)

		if err = rows.Scan(
			&item.ID,
			&item.FeedID,
			&item.URL,
			&item.Title,
			&item.Description,
			&stamp,
			&item.Read,
			&rating); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		} else if item.Tags, err = db.TagGetByItem(item.ID); err != nil {
			db.log.Printf("[ERROR] Cannot load tags for Item %q (%d): %s\n",
				item.Title,
				item.ID,
				err.Error())
			return nil, err
		}

		if rating != nil {
			item.ManuallyRated = true
			item.Rating = *rating
		} else {
			item.Rating = math.NaN()
		}
		item.Timestamp = time.Unix(stamp, 0)
		items = append(items, item)
	}

	return items, nil
} // func (db *Database) ItemGetFTS(fts string) ([]feed.Item, error)

// ItemGetByTag fetches all Items the given Tag is attached to.
//
// Currently, this does not take the Tag hierarchy into account.
func (db *Database) ItemGetByTag(t *tag.Tag) ([]feed.Item, error) {
	const qid query.ID = query.ItemGetByTag
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(t.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var items = make([]feed.Item, 0, 32)

	for rows.Next() {
		var (
			item   feed.Item
			rating *float64
			stamp  int64
		)

		if err = rows.Scan(
			&item.ID,
			&item.FeedID,
			&item.URL,
			&item.Title,
			&item.Description,
			&stamp,
			&item.Read,
			&rating); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		} else if item.Tags, err = db.TagGetByItem(item.ID); err != nil {
			db.log.Printf("[ERROR] Cannot load tags for Item %q (%d): %s\n",
				item.Title,
				item.ID,
				err.Error())
			return nil, err
		}

		if rating != nil {
			item.ManuallyRated = true
			item.Rating = *rating
		} else {
			item.Rating = math.NaN()
		}
		item.Timestamp = time.Unix(stamp, 0)
		items = append(items, item)
	}

	return items, nil
} // func (db *Database) ItemGetByTag(t *tag.Tag) ([]feed.Item, error)

// ItemGetByTagRecursive returns all Items marked with the given Tag or any
// of its children (recursively, obviously).
func (db *Database) ItemGetByTagRecursive(t *tag.Tag) ([]feed.Item, error) {
	const qid query.ID = query.ItemGetByTagRecursive
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(t.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var items = make([]feed.Item, 0, 64)

	for rows.Next() {
		var (
			item   feed.Item
			rating *float64
			stamp  int64
		)

		if err = rows.Scan(
			&item.ID,
			&item.FeedID,
			&item.URL,
			&item.Title,
			&item.Description,
			&stamp,
			&item.Read,
			&rating); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		} else if item.Tags, err = db.TagGetByItem(item.ID); err != nil {
			db.log.Printf("[ERROR] Cannot load tags for Item %q (%d): %s\n",
				item.Title,
				item.ID,
				err.Error())
			return nil, err
		} else if rating != nil {
			item.ManuallyRated = true
			item.Rating = *rating
		} else {
			item.Rating = math.NaN()
		}
		item.Timestamp = time.Unix(stamp, 0)
		items = append(items, item)
	}

	return items, nil
} // func (db *Database) ItemGetByTagRecursive(t *tag.Tag) ([]feed.Item, error)

// ItemRatingSet sets an Item's Rating.
func (db *Database) ItemRatingSet(i *feed.Item, rating float64) error {
	const qid = query.ItemRatingSet
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	// db.log.Printf("[TRACE] Rate Item %d (%q): %f\n",
	// 	i.ID,
	// 	i.Title,
	// 	rating)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}
		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(rating, i.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot rate Item %s (%s): %s",
				i.Title,
				i.URL,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	i.Rating = rating
	i.ManuallyRated = true
	return nil
} // func (db *Database) ItemRatingSet(i *feed.Item, rating float64) error

// ItemRatingClear clears an Item's Rating.
func (db *Database) ItemRatingClear(i *feed.Item) error {
	const qid = query.ItemRatingClear
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(i.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot rate Item %s (%s): %s",
				i.Title,
				i.URL,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	i.Rating = math.NaN()
	i.ManuallyRated = false
	return nil
} // func (db *Database) ItemRatingClear(i *feed.Item) error

// ItemHasDuplicate checks if a possible duplicate of the given Item already
// exists in the database.
func (db *Database) ItemHasDuplicate(i *feed.Item) (bool, error) {
	const qid query.ID = query.ItemHasDuplicate
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return false, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(i.URL, i.FeedID, i.Title); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return false, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var cnt int64

		if err = rows.Scan(&cnt); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return false, err
		}

		return cnt > 0, nil
	}

	err = fmt.Errorf("query %s should always return exactly one row",
		qid)

	db.log.Printf("[CANTHAPPEN] %s\n",
		err.Error())

	return false, err
} // func (db *Database) ItemHasDuplicate(i *feed.Item) (bool, error)

// FTSRebuild rebuilds the index used in the full-text search.
func (db *Database) FTSRebuild() error {
	const (
		qsel query.ID = query.ItemGetContent
		qins query.ID = query.ItemInsertFTS
		qdel query.ID = query.FTSClear
	)
	var (
		err           error
		status        bool
		sel, ins, del *sql.Stmt
	)

	if sel, err = db.getQuery(qsel); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qsel,
			err.Error())
		return err
	} else if ins, err = db.getQuery(qins); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qins,
			err.Error())
		return err
	} else if del, err = db.getQuery(qdel); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qdel,
			err.Error())
		return err
	} else if err = db.Begin(); err != nil {
		db.log.Printf("[ERROR] Cannot begin transaction: %s\n",
			err.Error())
		return err
	}

	defer func() {
		var x error
		if status {
			if x = db.Commit(); x != nil {
				db.log.Printf("[ERROR] Cannot commit transaction: %s\n",
					err.Error())
			}
		} else {
			if x = db.Rollback(); x != nil {
				db.log.Printf("[ERROR] Cannot roll back transaction: %s\n",
					err.Error())
			}
		}
	}()

	sel = db.tx.Stmt(sel)
	ins = db.tx.Stmt(ins)
	del = db.tx.Stmt(del)

	if _, err = del.Exec(); err != nil {
		db.log.Printf("[ERROR] Cannot clear FTS index: %s\n",
			err.Error())
		return err
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = sel.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			db.log.Printf("[ERROR] Cannot execute query %s: %s\n",
				qsel,
				err.Error())
			return err
		}
	}

	defer rows.Close() // nolint: errcheck

	for rows.Next() {
		var link, body string

		if err = rows.Scan(&link, &body); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return err
		} else if _, err = ins.Exec(link, body); err != nil {
			db.log.Printf("[ERROR] Cannot insert Item %s into full text index: %s\n",
				link,
				err.Error())
			return err
		}
	}

	status = true
	return nil
} // func (db *Database) FTSRebuild() error

// TagCreate creates a new Tag. <parent> is the ID of the parent tag. A value
// of 0 means no parent.
func (db *Database) TagCreate(name, desc string, parentID int64) (*tag.Tag, error) {
	const qid = query.TagCreate
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return nil, err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return nil, errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var res sql.Result
	var parent *int64

	if parentID != 0 {
		parent = &parentID
	}

EXEC_QUERY:
	if res, err = stmt.Exec(name, desc, parent); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add Tag %s (%s) to database: %s",
				name,
				desc,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return nil, err
		}
	} else {
		var tagID int64

		if tagID, err = res.LastInsertId(); err != nil {
			db.log.Printf("[ERROR] Cannot get ID of new Tag %q: %s\n",
				name,
				err.Error())
			return nil, err
		}

		status = true
		return &tag.Tag{
			ID:          tagID,
			Name:        name,
			Description: desc,
			Parent:      parentID,
		}, nil
	}
} // func (db *Database) TagCreate(name, desc string, parentID int64) (*tag.Tag, error)

// TagDelete removes the Tag with the given ID from the database.
//
// This will automatically remove all links of the given Tag to Items as well.
func (db *Database) TagDelete(id int64) error {
	const qid = query.TagDelete
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot delete Tag %d: %s",
				id,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	return nil
} // func (db *Database) TagDelete(id int64) error

// TagGetAll fetches all Tags from the database, in no particular order.
func (db *Database) TagGetAll() ([]tag.Tag, error) {
	const qid query.ID = query.TagGetAll
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var tags = make([]tag.Tag, 0, 16)

	for rows.Next() {
		var (
			t      tag.Tag
			parent *int64
		)

		if err = rows.Scan(&t.ID, &t.Name, &t.Description, &parent); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		}

		if parent != nil {
			t.Parent = *parent
		}

		tags = append(tags, t)
	}

	return tags, nil
} // func (db *Database) TagGetAll() ([]tag.Tag, error)

// TagGetHierarchy retrieves a slice of all Tags, organized in a hierarchical
// fashion.
func (db *Database) TagGetHierarchy() ([]tag.Tag, error) {
	const (
		qroot     query.ID = query.TagGetRoots
		qchildren query.ID = query.TagGetChildren
	)

	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qroot); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qroot,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var tags = make([]tag.Tag, 0, 16)

	for rows.Next() {
		var t tag.Tag

		if err = rows.Scan(&t.ID, &t.Name, &t.Description); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		} else if t.Children, err = db.TagGetChildrenImmediate(t.ID); err != nil {
			db.log.Printf("[ERROR] Cannot get children of Tag %s (%d): %s\n",
				t.Name,
				t.ID,
				err.Error())
			return nil, err
		}

		tags = append(tags, t)
	}

	return tags, nil
} // func (db *Database) TagGetHierarchy() ([]tag.Tag, error)

// TagGetChildren fetches all the children - recursively - of the given Tag.
func (db *Database) TagGetChildren(id int64) ([]tag.Tag, error) {
	const qid query.ID = query.TagGetChildren
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(id, id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var tags = make([]tag.Tag, 0, 16)

	for rows.Next() {
		var (
			t      tag.Tag
			parent *int64
		)

		if err = rows.Scan(&t.ID, &t.Name, &t.Description, &parent); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		}

		if parent != nil {
			t.Parent = *parent
		}

		tags = append(tags, t)
	}

	return tags, nil
} // func (db *Database) TagGetChildren(id int64) ([]tag.Tag, error)

// TagGetChildrenImmediate fetches all Tags that are directly descended from
// the given parent Tag, i.e. Tags whose parent ID equals the argument.
func (db *Database) TagGetChildrenImmediate(id int64) ([]tag.Tag, error) {
	const qid query.ID = query.TagGetChildrenImmediate
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var tags = make([]tag.Tag, 0, 16)

	for rows.Next() {
		var t tag.Tag

		if err = rows.Scan(&t.ID, &t.Name, &t.Description); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		}

		t.Parent = id

		if t.Children, err = db.TagGetChildrenImmediate(t.ID); err != nil {
			db.log.Printf("[ERROR] Cannot get immediate children of Tag %s (%d): %s\n",
				t.Name,
				t.ID,
				err.Error())
			return nil, err
		}

		tags = append(tags, t)
	}

	return tags, nil
} // func (db *Database) TagGetChildrenImmediate(id int64) ([]tag.Tag, error)

// TagGetByID loads a Tag by its database ID.
func (db *Database) TagGetByID(id int64) (*tag.Tag, error) {
	const qid query.ID = query.TagGetByID

	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Cannot load Tag #%d: %s\n",
			id,
			err.Error())
		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			t      = &tag.Tag{ID: id}
			parent *int64
		)

		if err = rows.Scan(&t.Name, &t.Description, &parent); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		}

		if parent != nil {
			t.Parent = *parent
		}

		return t, nil
	}

	return nil, nil
} // func (db *Database) TagGetByID(id int64) (*tag.Tag, error)

// TagGetByName loads a Tag by its database ID.
func (db *Database) TagGetByName(name string) (*tag.Tag, error) {
	const qid query.ID = query.TagGetByName

	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(name); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Cannot load Tag %s: %s\n",
			name,
			err.Error())
		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			t      = &tag.Tag{Name: name}
			parent *int64
		)

		if err = rows.Scan(&t.ID, &t.Description, &parent); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		}

		if parent != nil {
			t.Parent = *parent
		}

		return t, nil
	}

	return nil, nil
} // func (db *Database) TagGetByName(id int64) (*tag.Tag, error)

// TagGetByItem returns a (possibly empty) slice of all Tags attached to an
// Item.
func (db *Database) TagGetByItem(itemID int64) ([]tag.Tag, error) {
	const qid query.ID = query.TagGetByItem
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(itemID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Cannot load Tags for Item #%d: %s\n",
			itemID,
			err.Error())
		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec
	var tags = make([]tag.Tag, 0, 4)

	for rows.Next() {
		var (
			t      tag.Tag
			parent *int64
		)

		if err = rows.Scan(&t.ID, &t.Name, &t.Description, &parent); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		}

		if parent != nil {
			t.Parent = *parent
		}

		tags = append(tags, t)
	}

	return tags, nil
} // func (db *Database) TagGetByItem(itemID int64) ([]tag.Tag, error)

// TagNameUpdate renames the given Tag to the new name.
func (db *Database) TagNameUpdate(t *tag.Tag, name string) error {
	const qid = query.TagNameUpdate
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(name, t.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot rename Tag %s (%d) to %s: %s",
				t.Name,
				t.ID,
				name,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	t.Name = name
	return nil
} // func (db *Database) TagNameUpdate(t *tag.Tag, name string) error

// TagDescriptionUpdate changes the given Tag's Description.
func (db *Database) TagDescriptionUpdate(t *tag.Tag, desc string) error {
	const qid = query.TagDescriptionUpdate
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(desc, t.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("cannot update description of Tag %s (%d) to %q: %s",
				t.Name,
				t.ID,
				desc,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	t.Description = desc
	return nil
} // func (db *Database) TagDescriptionUpdate(t *tag.Tag, desc string) error

// TagParentSet updates the given Tag's Parent field.
func (db *Database) TagParentSet(t *tag.Tag, parent int64) error {
	const qid = query.TagParentSet
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(parent, t.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("cannot set Parent of Tag %s (%d) to %d: %s",
				t.Name,
				t.ID,
				parent,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	t.Parent = parent
	return nil
} // func (db *Database) TagParentSet(t *tag.Tag, parent int64) error

// TagParentClear clears the given Tag's Parent field.
func (db *Database) TagParentClear(t *tag.Tag) error {
	const qid = query.TagParentClear
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(t.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("cannot clear Parent of Tag %s (%d): %s",
				t.Name,
				t.ID,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	t.Parent = 0
	return nil
} // func (db *Database) TagParentClear(t *tag.Tag) error

// TagLinkCreate attaches the given Tag to the given Item.
func (db *Database) TagLinkCreate(itemID, tagID int64) error {
	const qid query.ID = query.TagLinkCreate
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(tagID, itemID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("cannot attach Tag %d to Item %d: %s",
				tagID,
				itemID,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	return nil
} // func (db *Database) TagLinkCreate(itemID, tagID int64) error

// TagLinkDelete detaches a Tag from an Item.
func (db *Database) TagLinkDelete(itemID, tagID int64) error {
	const qid query.ID = query.TagLinkDelete
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(tagID, itemID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("cannot attach Tag %d to Item %d: %s",
				tagID,
				itemID,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	return nil
} // func (db *Database) TagLinkDelete(itemID, tagID int64) error

// TagLinkGetByItem returns a slice of *the IDs* of all the Tags attached
// to a given Item.
func (db *Database) TagLinkGetByItem(itemID int64) ([]int64, error) {
	const qid query.ID = query.TagLinkGetByItem
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(itemID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Cannot load Tags for Item #%d: %s\n",
			itemID,
			err.Error())
		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec
	var tags = make([]int64, 0, 4)

	for rows.Next() {
		var tagID int64

		if err = rows.Scan(&tagID); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		}

		tags = append(tags, tagID)
	}

	return tags, nil
} // func (db *Database) TagLinkGetByItem(itemID int64) ([]int64, error)

// ReadLaterAdd adds a ReadLater note to the database.
func (db *Database) ReadLaterAdd(item *feed.Item, note string, deadline time.Time) (*feed.ReadLater, error) {
	const qid query.ID = query.ReadLaterAdd
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return nil, err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return nil, errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var (
		now = time.Now()
		res sql.Result
	)

EXEC_QUERY:
	if res, err = stmt.Exec(item.ID, note, now.Unix(), deadline.Unix()); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add Item %s (%s) to database: %s",
				item.Title,
				item.URL,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return nil, err
		}
	} else {
		var later = &feed.ReadLater{
			Item:     item,
			ItemID:   item.ID,
			Note:     note,
			Deadline: deadline,
		}

		if later.ID, err = res.LastInsertId(); err != nil {
			db.log.Printf("[ERROR] Cannot get ID of new Item %q: %s\n",
				item.Title,
				err.Error())
			return nil, err
		}

		status = true
		return later, nil
	}
} // func (db *Database) ReadLaterAdd(item *feed.Item, note string, deadline time.Time) (*feed.ReadLater, error)

// ReadLaterGetByItem returns the ReadLater note for the given Item.
func (db *Database) ReadLaterGetByItem(item *feed.Item) (*feed.ReadLater, error) {
	const qid query.ID = query.ReadLaterGetByItem
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(item.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			stamp    int64
			deadline *int64
			note     *string
			later    = &feed.ReadLater{
				Item:   item,
				ItemID: item.ID,
			}
		)

		if err = rows.Scan(
			&later.ID,
			&note,
			&stamp,
			&deadline,
			&later.Read); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		}

		if deadline != nil {
			later.Deadline = time.Unix(*deadline, 0)
		}

		if note != nil {
			later.Note = *note
		}

		later.Timestamp = time.Unix(stamp, 0)

		return later, nil
	}

	return nil, nil
} // func (db *Database) ReadLaterGetByItem(itemID int64) (*feed.ReadLater, error)

// ReadLaterGetAll returns all ReadLater notes.
func (db *Database) ReadLaterGetAll() ([]feed.ReadLater, error) {
	const qid query.ID = query.ReadLaterGetAll
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var items = make([]feed.ReadLater, 0, 32)
	// var items = make(map[int64]feed.ReadLater, 32)

	for rows.Next() {
		var (
			lstamp, istamp int64
			deadline       *int64
			note           *string
			rating         *float64
			later          feed.ReadLater
			item           = new(feed.Item)
			read           *bool
		)

		if err = rows.Scan(
			&later.ID,
			&later.ItemID,
			&note,
			&lstamp,
			&deadline,
			&read,
			&item.FeedID,
			&item.URL,
			&item.Title,
			&item.Description,
			&istamp,
			&item.Read,
			&rating); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		} else if item.Tags, err = db.TagGetByItem(later.ItemID); err != nil {
			db.log.Printf("[ERROR] Cannot load tags for Item %d: %s\n",
				later.ItemID,
				err.Error())
			return nil, err
		}

		if read != nil {
			later.Read = *read
		}

		if deadline != nil {
			later.Deadline = time.Unix(*deadline, 0)
		}

		if note != nil {
			later.Note = *note
		}

		if rating != nil {
			item.Rating = *rating
			item.ManuallyRated = true
		} else {
			item.Rating = math.NaN()
		}

		later.Item = item
		later.Timestamp = time.Unix(lstamp, 0)
		item.ID = later.ItemID
		item.Timestamp = time.Unix(istamp, 0)

		items = append(items, later)
		// items[later.ItemID] = later
	}

	return items, nil
} // func (db *Database) ReadLaterGetAll() ([]feed.ReadLater, error)

// ReadLaterGetUnread returns all ReadLater items that are not marked a read.
func (db *Database) ReadLaterGetUnread() (map[int64]feed.ReadLater, error) {
	const qid query.ID = query.ReadLaterGetUnread
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	// var items = make([]feed.ReadLater, 0, 32)
	var items = make(map[int64]feed.ReadLater, 32)

	if rows.Next() {
		var (
			lstamp, istamp int64
			deadline       *int64
			note           *string
			rating         *float64
			later          = feed.ReadLater{Read: false}
			item           = new(feed.Item)
		)

		if err = rows.Scan(
			&later.ID,
			&later.ItemID,
			&note,
			&lstamp,
			&deadline,
			&item.FeedID,
			&item.URL,
			&item.Title,
			&item.Description,
			&istamp,
			&rating); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n",
				err.Error())
			return nil, err
		}

		if deadline != nil {
			later.Deadline = time.Unix(*deadline, 0)
		}

		if note != nil {
			later.Note = *note
		}

		if rating != nil {
			item.Rating = *rating
		} else {
			item.Rating = math.NaN()
		}

		later.Timestamp = time.Unix(lstamp, 0)
		item.Timestamp = time.Unix(istamp, 0)
		item.ID = later.ItemID
		later.Item = item

		// items = append(items, later)
		items[later.ItemID] = later
	}

	return items, nil
} // func (db *Database) ReadLaterGetUnread() ([]feed.ReadLater, error)

// ReadLaterMarkRead marks a ReadLater note as read.
func (db *Database) ReadLaterMarkRead(itemID int64) error {
	const qid query.ID = query.ReadLaterMarkRead
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(itemID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("cannot mark ReadLater Note %d as Read: %s",
				itemID,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	return nil
} // func (db *Database) ReadLaterMarkRead(itemID int64) error

// ReadLaterMarkUnread marks a ReadLater note as read.
func (db *Database) ReadLaterMarkUnread(itemID int64) error {
	const qid query.ID = query.ReadLaterMarkUnread
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(itemID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("cannot mark ReadLater Note %d as Read: %s",
				itemID,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	return nil
} // func (db *Database) ReadLaterMarkUnread(itemID int64) error

// ReadLaterDelete deletes a ReadLater note from the database.
func (db *Database) ReadLaterDelete(id int64) error {
	const qid query.ID = query.ReadLaterDelete
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("cannot delete ReadLater note %d: %s",
				id,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	return nil
} // func (db *Database) ReadLaterDelete(id int64) error

// ReadLaterSetDeadline updates the deadline of a ReadLater note.
func (db *Database) ReadLaterSetDeadline(l *feed.ReadLater, deadline time.Time) error {
	const qid query.ID = query.ReadLaterSetDeadine
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(deadline.Unix(), l.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("cannot set deadline of ReadLater note %d to %s: %s",
				l.ID,
				l.Timestamp.Format(common.TimestampFormat),
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	l.Deadline = deadline
	return nil
} // func (db *Database) ReadLaterSetDeadline(l *feed.ReadLater, deadline time.Time) error

// ReadLaterSetNote updates the Note on a ReadLater item.
func (db *Database) ReadLaterSetNote(l *feed.ReadLater, note string) error {
	const qid query.ID = query.ReadLaterSetNote
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(note, l.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("cannot set Note of ReadLater note %d: %s",
				l.ID,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	l.Note = note
	return nil
} // func (db *Database) ReadLaterSetNote(l *feed.ReadLater, note string) error
