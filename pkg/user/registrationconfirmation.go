package user

import (
	"net/http"

	"github.com/budden/a/pkg/apperror"
	"github.com/jmoiron/sqlx"

	"github.com/budden/a/pkg/database"
	"github.com/budden/a/pkg/shared"
	"github.com/gin-gonic/gin"
)

// RegistrationConfirmationPageHandler processes a registration confirmation
func RegistrationConfirmationPageHandler(c *gin.Context) {
	var rd RegistrationData
	// fill nickname and confirmationkey
	extractNicknameAndConfirmationKeyFromRequest(c, &rd)
	// update sdusers_db, fill userid
	processRegistrationConfirmationWithSDUsersDbStage1(&rd)
	// promote the user to Sd Db. If we crash here, user will be able to login,
	// (and unable to register again), but wil be missing from the main content db
	promoteUserToSdDb(&rd)
	c.HTML(http.StatusMovedPermanently,
		"general.html",
		shared.GeneralTemplateParams{Message: "Registration confirmed. Now you can proceed to the <a href=/>Login page</a>"})
}

func extractNicknameAndConfirmationKeyFromRequest(c *gin.Context, rd *RegistrationData) {
	query := c.Request.URL.Query()
	nicknames, ok1 := query["nickname"]
	confirmationkeys, ok2 := query["confirmationkey"]

	if !ok1 || !ok2 ||
		len(nicknames) == 0 ||
		len(confirmationkeys) == 0 {
		apperror.Panic500If(apperror.ErrDummy, "Bad registration confirmation URL")
	}
	rd.Nickname = nicknames[0]
	rd.ConfirmationKey = confirmationkeys[0]
}

func processRegistrationConfirmationWithSDUsersDbStage1(rd *RegistrationData) {
	err := WithTransaction(
		database.SDUsersDb,
		func(trans *database.TransactionType) (err1 error) {
			database.CheckDbAlive(trans.Conn)
			var reply *sqlx.Rows
			reply, err1 = trans.Tx.NamedQuery(
				`select * from process_registrationconfirmation(:confirmationkey, :nickname)`,
				rd)
			apperror.Panic500If(err1, "Failed to confirm registration, sorry")
			for reply.Next() {
				err1 = reply.Scan(&rd.UserID)
				//fmt.Printf("UserID = %v\n", rd.UserID)
				database.FatalDatabaseErrorIf(err1, database.SDUsersDb, "Error obtaining id of a new user, err = %#v", err1)
			}
			// hence err1 == nil
			return
		})
	// if we have error here, it is an error in commit, so is fatal
	database.FatalDatabaseErrorIf(err, database.SDUsersDb, "Failed around registrationconfirmation, error is %#v", err)
	return
}

// Copy user from the sdusers_db to sd_db. We run it as a part of
// registration process
func promoteUserToSdDb(rd *RegistrationData) {
	conn := database.SDDb
	database.CheckDbAlive(conn)
	_, err1 := conn.Db.NamedExec(
		"insert into sduser2 (id, nickname) values (:userid, :nickname)",
		rd)
	// here we rely upon the reliability of connection pool and don't crash the app
	apperror.Panic500If(err1, "Failed to add user to the dict database")
	return

	// Fault-tolerant approach would be to replicate the user
	// asynchronously (w/o error reported to the user) at the registration confirmation
	// as well as after the login, as well as after the user change
	// special field's in the sdusers_db should be used to decide efficiently if
	// we need to replicate

}
