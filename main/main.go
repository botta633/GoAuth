package main

import (
	"log"

	srv "github.com/ahmed/authProject/server"
)

func main() {
	srv.InitCache()

	dummyDB := srv.DBInfo{
		UserName: "sql11471156",
		Password: "DGbDwx2xY1",
		Address:  "sql11.freemysqlhosting.net",
		Name:     "sql11471156",
		Port:     "3306",
	}

	srv.DBInit(dummyDB)
	server := srv.NewHTTPServer(":80")
	log.Fatal(server.ListenAndServe())

}
