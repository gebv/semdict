package query

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"

	"github.com/budden/semdict/pkg/apperror"
	"github.com/budden/semdict/pkg/sddb"
	"github.com/budden/semdict/pkg/shared"

	"github.com/budden/semdict/pkg/user"
	"github.com/gin-gonic/gin"
)

// Params of a request to show a sense, including implicit Sduserid, obtained from session
type senseViewParamsType struct {
	Sduserid int64
	Senseid  int64
}

// senseDataForEditType is obtained from the DB also used for a view.
type senseDataForEditType struct {
	Id             int64
	OWord          string
	Theme          string
	Phrase         string
	OwnerId        int64
	Sdusernickname sql.NullString // owner (direct or implied)
	Allth          []*ThemeRecord
}

type senseEditTemplateParams struct {
	Ad *senseDataForEditType
}

// SenseViewHTMLTemplateParamsType are params for senseview.t.html
type SenseViewHTMLTemplateParamsType struct {
	Svp    *senseViewParamsType
	Sdfe   *senseDataForEditType
	Phrase template.HTML
}

func SenseByIdViewDirHandler(c *gin.Context) {
	senseID := extractIdFromRequest(c, "senseid")

	senseDataList := readSenseWithWordLanguageFromDb(senseID)

	if len(senseDataList) > 0 {
		c.HTML(http.StatusOK,
			"senseview.t.html",
			senseDataList)
	} else {
		apperror.Panic500AndErrorIf(apperror.ErrDummy, "Sorry, no sense (yet?) with id = «%d»", senseID)
	}
}

// read the sense, see the vsense view and senseViewParamsType for the explanation
func readSenseFromDb(svp *senseViewParamsType) (dataFound bool, ad *senseDataForEditType) {
	reply, err1 := sddb.NamedReadQuery(
		`select * from tsense where id = :senseid`, &svp)
	apperror.Panic500AndErrorIf(err1, "Failed to extract a sense, sorry")
	defer sddb.CloseRows(reply)()
	ad = &senseDataForEditType{}
	for reply.Next() {
		err1 = reply.StructScan(ad)
		dataFound = true
	}
	sddb.FatalDatabaseErrorIf(err1, "Error obtaining data of sense: %#v", err1)
	return
}

type senseDataWithWordLanguage struct {
	SenseID            int64          `db:"sense_id"`
	OWord              string         `db:"oword"`
	Theme              string         `db:"theme"`
	Phrase             template.HTML  `db:"phrase"`
	SenseOwnerID       int64          `db:"sense_owner_id"`
	LanguageID         *int64         `db:"language_id"`
	LanguageSlug       *string        `db:"language_slug"`
	LanguageCommentary *string        `db:"language_commentary"`
	LanguageOwnerID    *int64         `db:"language_owner_id"`
	WordID             *int64         `db:"word_id"`
	Word               *string        `db:"word"`
	WordCommentary     *template.HTML `db:"word_commentary"`
	WordOwnerID        *int64         `db:"word_owner_id"`
}

// read the sense with words by language.
func readSenseWithWordLanguageFromDb(senseID int64) (d []*senseDataWithWordLanguage) {
	reply, err := sddb.NamedReadQuery(
		`
SELECT ts.id         sense_id,
       ts.oword,
       ts.theme,
       ts.phrase,
       ts.ownerid    sense_owner_id,
       t.languageid  language_id,
       tl.slug       language_slug,
       tl.commentary language_commentary,
       tl.ownerid    language_owner_id,
       t.id          word_id,
       t.word,
       t.commentary  word_commentary,
       t.ownerid     word_owner_id
FROM tsense AS ts
         LEFT JOIN tlws AS t ON t.senseid = ts.id
         LEFT JOIN tlanguage tl on tl.id = t.languageid
WHERE ts.id = :senseid
ORDER BY tl.id;
`, &struct {
			Senseid int64
		}{
			Senseid: senseID,
		})
	apperror.Panic500AndErrorIf(err, "Failed to extract a sense, sorry")
	defer sddb.CloseRows(reply)()
	for reply.Next() {
		v := &senseDataWithWordLanguage{}
		err = reply.StructScan(v)
		sddb.FatalDatabaseErrorIf(err, "Error obtaining data of sense: %#v", err)
		d = append(d, v)
	}
	return
}

//  SenseEditDirHandler serves /senseedit/:senseid
func SenseEditDirHandler(c *gin.Context) {
	user.EnsureLoggedIn(c)
	Senseid := extractIdFromRequest(c, "senseid")
	svp := &senseViewParamsType{
		Sduserid: int64(user.GetSDUserIdOrZero(c)),
		Senseid:  Senseid}

	var dataFound bool
	var ad *senseDataForEditType
	dataFound, ad = readSenseFromDb(svp)

	if !dataFound {
		c.HTML(http.StatusBadRequest,
			"general.t.html",
			shared.GeneralTemplateParams{Message: fmt.Sprintf("Sorry, no sense (yet?) for «%d»", svp.Senseid)})
		return
	}

	allThemes := AllKnownThemes()
	ad.Allth = allThemes

	aetp := &senseEditTemplateParams{Ad: ad}
	c.HTML(http.StatusOK,
		"senseedit.t.html",
		aetp)
}
