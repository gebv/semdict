package user

import (
	"sync"

	"github.com/budden/a/pkg/apperror"
	"github.com/budden/a/pkg/database"
	"github.com/jmoiron/sqlx"
)

// Mutex we lock for any writes to sdusers_db to minimize parallelism at the db level
var writeSDUsersMutex sync.Mutex

// PostgresqlErrorCodeUniqueViolation is a unique_violation,
// https://postgrespro.ru/docs/postgrespro/9.5/errcodes-appendix
const PostgresqlErrorCodeUniqueViolation = "23505"

// PostgresqlErrorCodeNoData = no_data warning
const PostgresqlErrorCodeNoData = "02000"

// WithSDUsersDbTransaction opens a transaction in the sdusers_db, then runs body
// Then, if there is no error, and transaction is still active, commit transaction and returns commit's error
// If there was an error or panic while executing body, tries to rollback the tran transaction. If rollback fails,
// and there were no panic, panics. If rollback failed and there was panic, writes a message that rollback failed and
// continues to panic
func WithSDUsersDbTransaction(body func(tx *sqlx.Tx) (err error)) (err error) {
	writeSDUsersMutex.Lock()
	defer writeSDUsersMutex.Unlock()

	var tx *sqlx.Tx
	tx, err = database.SDUsersDb.Beginx()
	apperror.ExitAppIf(err, "Unable to start transaction")
	defer func() { database.RollbackIfActive(tx) }()
	_, err = tx.Exec(`set transaction isolation level repeatable read`)
	apperror.ExitAppIf(err, "Unable to start transaction")

	err = body(tx)
	if err == nil {
		err = database.CommitIfActive(tx)
	}
	return
}
