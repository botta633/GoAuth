package main

import (
	"log"

	srv "github.com/ahmed/authProject/server"
)

func main() {
	srv.InitCache()

	dummyDB := srv.DBInfo{
		UserName: "root",
		Password: "",
		Address:  "127.0.0.1",
		Name:     "USERINFO",
		Port:     "3306",
	}

	srv.DBInit(dummyDB)
	server := srv.NewHTTPServer(":80")
	log.Fatal(server.ListenAndServe())

}
