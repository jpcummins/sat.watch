package api

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func createDb() (*API, sqlmock.Sqlmock) {
	_, mock, _ := sqlmock.New()
	db := API{}
	// dialector := postgres.New(postgres.Config{
	// 	Conn:       mockDb,
	// 	DriverName: "postgres",
	// })

	// err := db.init(dialector)
	// if err != nil {
	// 	log.Panicf("%v", err)
	// }

	// rows := sqlmock.NewRows([]string{})
	// mock.ExpectQuery("SELECT").WillReturnRows(rows)
	return &db, mock
}

func TestCreateAddress(t *testing.T) {
	createDb()
}
