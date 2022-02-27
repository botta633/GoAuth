package server

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	_ "encoding/json"
	"fmt"
	_ "html/template"
	"log"
	"net/http"
	"text/template"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

var cache redis.Conn

func InitCache() {
	// Initialize the redis connection to a redis instance running on your local machine
	conn, err := redis.DialURL("redis://localhost")
	if err != nil {
		panic(err)
	}
	// Assign the connection to the package level `cache` variable
	cache = conn
}

var db *sql.DB

type User struct {
	Email     string `db:"EMAIL"`
	Password  string `db:"PASSWORD"`
	Telephone string `db:"TELEPHONE"`
	Address   string `db:"ADDRESS"`
	FullName  string `db:"FULLNAME"`
}

type DBInfo struct {
	UserName string
	Name     string
	Password string
	Address  string
	Port     string
}
type httpServer struct {
	User *User
}

type fileRenderer struct {
	Header     string
	PageHeader string
}

type infoPage struct {
	Email     string `db:"EMAIL"`
	Address   string `db:"ADDRESS"`
	Telephone string `db:"TELEPHONE"`
	FullName  string `db:"FULLNAME"`
}

func newUser() *User {
	return &User{}
}

func newHTTPServer() *httpServer {
	return &httpServer{
		User: newUser(),
	}

}
func generate_token(w http.ResponseWriter, r *http.Request) {

	sessionToken := uuid.NewV4().String()
	_, err := cache.Do("SETEX", sessionToken, "120", r.FormValue("email"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		panic(err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   sessionToken,
		Expires: time.Now().Add(120 * time.Second),
		Path:    "/",
	})
}

func NewHTTPServer(addr string) *http.Server {
	httpsrv := newHTTPServer()
	r := mux.NewRouter()
	//r.HandleFunc("/", Home).Methods("GET")
	r.HandleFunc("/signin", SignInProduce).Methods("POST")
	r.HandleFunc("/signin", SignInConsume).Methods("GET")
	r.HandleFunc("/signup", httpsrv.SignUpProduce).Methods("POST")
	r.HandleFunc("/signup", httpsrv.SignUpConsume).Methods("GET")
	r.HandleFunc("/info", httpsrv.InfoConsume).Methods("GET")
	r.HandleFunc("/infoedit", httpsrv.InfoEditProduce).Methods("POST")
	r.HandleFunc("/infoedit", httpsrv.InfoEditConsume).Methods("GET")
	r.HandleFunc("/info", httpsrv.InfoProduce).Methods("POST")
	r.HandleFunc("/reset", httpsrv.ResetConsume).Methods("GET")
	r.HandleFunc("/reset", httpsrv.ResetProduce).Methods("POST")

	return &http.Server{
		Addr:    addr,
		Handler: r,
	}
}

func DBInit(dummyDB DBInfo) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dummyDB.UserName, dummyDB.Password,
		dummyDB.Address, dummyDB.Port, dummyDB.Name)
	var err error
	db, err = sql.Open("mysql", dsn)

	if err != nil {
		panic(err.Error())
	}

	err = db.Ping()
	if err != nil {
		panic(err)

	}
	fmt.Println(db.Stats().InUse)

}

func checkToken(w http.ResponseWriter, r *http.Request) (*http.Cookie, error) {
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			return c, err
		}

	}

	return c, nil

}
func (s *httpServer) ResetConsume(w http.ResponseWriter, r *http.Request) {
	temp := template.Must(template.ParseFiles("../resetPassword.html"))
	temp.Execute(w, nil)

}
func (s *httpServer) ResetProduce(w http.ResponseWriter, r *http.Request) {
	c, err := checkToken(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	sessionToken := c.Value

	response, err := cache.Do("GET", sessionToken)

	result := fmt.Sprintf("%s", response)
	query, _ := db.Prepare("SELECT * FROM USER WHERE EMAIL = ?;")

	_, err = query.Exec(result)
	if err != nil {
		if err.Error()[:5] == "Error" {
			log.Println("This email is not registered")
			http.Redirect(w, r, "/signup", http.StatusSeeOther)
			return
		}
		panic(err)

	}

	


}
func (s *httpServer) InfoEditProduce(w http.ResponseWriter, r *http.Request) {
	c, err := checkToken(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	sessionToken := c.Value

	response, err := cache.Do("GET", sessionToken)

	fmt.Println(response)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
	}

	if response == nil {
		w.WriteHeader(http.StatusUnauthorized)
		http.Redirect(w, r, "/signin", http.StatusSeeOther)

	}

	submit := r.FormValue("submit")

	err = r.ParseForm()

	if err != nil {
		panic(err)
	}

	if submit == "Save" {
		data := infoPage{
			Email:     r.FormValue("Email"),
			FullName:  r.FormValue("FullName"),
			Address:   r.FormValue("Address"),
			Telephone: r.FormValue("Telephone"),
		}
		res := fmt.Sprintf("%s", response)

		query, err := db.Prepare("UPDATE USER SET EMAIL = ?, FULLNAME = ?, TELEPHONE = ?, ADDRESS = ? WHERE EMAIL = ?;")

		if err != nil {
			panic(err)
		}

		_, err = query.Exec(data.Email, data.FullName, data.Telephone, data.Address, res)

		if err != nil {

			fmt.Println(err.Error())

		}
		_, err = cache.Do("SETEX", sessionToken, "120", data.Email)
		if err != nil {
			panic(err)

		}
	}

	http.Redirect(w, r, "/info", http.StatusSeeOther)

}
func (s *httpServer) InfoEditConsume(w http.ResponseWriter, r *http.Request) {
	c, err := checkToken(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	sessionToken := c.Value

	response, err := cache.Do("GET", sessionToken)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
	}

	if response == nil {
		w.WriteHeader(http.StatusUnauthorized)
		http.Redirect(w, r, "/signin", http.StatusSeeOther)

	}
	fmt.Println("Came here before query")
	selDB, err := db.Query("SELECT EMAIL, FULLNAME, ADDRESS, TELEPHONE FROM USER WHERE EMAIL=?", response)
	fmt.Println("Came here after edit")
	if err != nil {
		fmt.Println(err.Error())
	}

	data := infoPage{}
	fmt.Println("Came here after query")

	for selDB.Next() {
		err = selDB.Scan(&data.Email, &data.FullName, &data.Address, &data.Telephone)
		if err != nil {
			panic(err)

		}
	}
	tmpl := template.Must(template.ParseFiles("../editinfo.html"))

	tmpl.Execute(w, data)

}
func (s *httpServer) InfoProduce(w http.ResponseWriter, r *http.Request) {
	c, err := checkToken(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	sessionToken := c.Value

	response, err := cache.Do("GET", sessionToken)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
	}

	if response == nil {
		w.WriteHeader(http.StatusUnauthorized)
		http.Redirect(w, r, "/signin", http.StatusSeeOther)

	}

	submit := r.FormValue("submit")

	if submit == "Edit" {
		fmt.Println("Came here in Edit")
		http.Redirect(w, r, "/infoedit", http.StatusSeeOther)
		return
	}

	if submit == "Logout" {
		response, err = cache.Do("DEL", sessionToken)
		if response != nil {
			fmt.Println("res is not null")
		} else {
			fmt.Println("Response is  null")
		}
		if err != nil {

			panic(err)

		}
		http.SetCookie(w, &http.Cookie{
			Name:    "session_token",
			Value:   sessionToken,
			Expires: time.Now(),
			Path:    "/",
		})

		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

}

func (s *httpServer) InfoConsume(w http.ResponseWriter, r *http.Request) {

	c, err := checkToken(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	sessionToken := c.Value

	response, err := cache.Do("GET", sessionToken)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
	}

	if response == nil {
		w.WriteHeader(http.StatusUnauthorized)
		http.Redirect(w, r, "/signin", http.StatusSeeOther)

	}

	res := fmt.Sprintf("%s", response)

	fmt.Println(res)

	selDB, err := db.Query("SELECT * FROM USER WHERE EMAIL=?", res)

	if err != nil {
		fmt.Println(err.Error())
	}
	user := User{}

	for selDB.Next() {
		err = selDB.Scan(&user.Email, &user.Telephone, &user.Address, &user.Password, &user.FullName)
		fmt.Println(user.Address)
		if err != nil {
			panic(err)
		}
	}

	tmpl := template.Must(template.ParseFiles("../information.html"))

	data := infoPage{
		Email:     user.Email,
		Telephone: user.Telephone,
		Address:   user.Address,
		FullName:  user.FullName,
	}

	tmpl.Execute(w, data)

}

func (s *httpServer) SignUpProduce(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	h := sha256.Sum256([]byte(r.FormValue("password")))
	hasher := base64.StdEncoding.EncodeToString(h[:])
	data := User{
		Email:     r.FormValue("email"),
		Password:  hasher,
		FullName:  "",
		Address:   "",
		Telephone: "",
	}

	fmt.Println(data.Email)

	query, _ := db.Prepare("INSERT INTO USER (EMAIL, PASSWORD, FULLNAME, ADDRESS,  TELEPHONE) VALUES (?, ?, ?, ?, ?);")

	_, err = query.Exec(data.Email, data.Password, data.FullName, data.Address, data.Telephone)
	if err != nil {
		if err.Error()[:10] == "Error 1062" {
			fmt.Println("This email is already registered")
			http.Redirect(w, r, "/signup", http.StatusSeeOther)
			return
		}
		panic(err)

	}
	generate_token(w, r)
	http.Redirect(w, r, "/info", http.StatusSeeOther)

}
func (s *httpServer) SignUpConsume(w http.ResponseWriter, r *http.Request) {

	c, err := checkToken(w, r)

	if err == nil {

		sessionToken := c.Value

		response, err := cache.Do("GET", sessionToken)

		if err == nil {
			fmt.Println("came here in sigunp")

			if response != nil {
				http.Redirect(w, r, "/info", http.StatusSeeOther)
				return

			}
		}
	}
	tmpl := template.Must(template.ParseFiles("../signup-signin.html"))
	data := fileRenderer{
		Header:     "Sign up",
		PageHeader: "Regsiter!",
	}

	tmpl.Execute(w, data)

}

func SignInProduce(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}
	h := sha256.Sum256([]byte(r.FormValue("password")))
	hasher := base64.StdEncoding.EncodeToString(h[:])

	data := User{
		Email:     r.FormValue("email"),
		Password:  hasher,
		FullName:  "",
		Address:   "",
		Telephone: "",
	}
	err = db.QueryRow("SELECT * FROM USER WHERE EMAIL = ?", data.Email).Scan(&data.Email,
		&data.Telephone, &data.Address, &data.Password, &data.FullName)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Redirect(w, r, "/signin", http.StatusSeeOther)
			return
		}
		panic(err)

	}

	if data.Password != hasher {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	generate_token(w, r)

	http.Redirect(w, r, "/info", http.StatusSeeOther)

}
func SignInConsume(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("../signin.html"))
	tmpl.Execute(w, nil)

}
