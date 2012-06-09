//TODO:
// 1. Also for HTTP?: Zero Downtime upgrades of TCP servers in Go http://blog.nella.org/?p=879
// 2. Nicer errror handling, maybe panic?
// 3. Storing pictures of products, resizing
// 4. header function, so I don't have to get login from session every time template is executed. Also different meta keywords, titles.
// 5. template caching
// 6. reCaptcha
// 7. account activation via email
// 8. password and login recover
// 9. oauth for registration
// 10. pretty urls for products
// 11. Test Riak for perfomrance http://blog.airbrake.io/status/planned-airbrake-migration-love-go-love-riak/
package main

import (
	"code.google.com/p/gorilla/mux"
	"code.google.com/p/gorilla/sessions"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"html"
	"html/template"
	"io"
	"log"
	"net/http"

//	"reflect"
)

// If content-type not set manually, it will be guessed by http://golang.org/src/pkg/net/http/sniff.go
// w.Header().Set("Content-Type", "text/html; charset=utf-8")
// io.WriteString(w, v)
// io.WriteString(w, `<br><form method="POST" action="/post"><input name="s"></form>`)
// io.WriteString(w, `<div>hello</div>`)

// HTTP is stateless: a client computer running a web browser must establish a new TCP network connection to the web server with each new HTTP GET or POST request. The web server, therefore, cannot rely on an established TCP network connection for longer than a single HTTP GET or POST operation.
// Session management is the technique used by the web developer to make the stateless HTTP protocol support session state. For example, once a user has been authenticated to the web server, the user's next HTTP request (GET or POST) should not cause the web server to ask for the user's account and password again. For a discussion of the methods used to accomplish this please see HTTP cookie.
// http://en.wikipedia.org/wiki/Session_management
// http://en.wikipedia.org/wiki/HTTP_cookie
var store = sessions.NewCookieStore([]byte("something-very-secret"))

func login(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	if session.Values["logged"] == true {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		if err := r.ParseForm(); err != nil {
			serveError(w, err)
			return
		}
		login := html.EscapeString(r.Form.Get("login"))
		password := html.EscapeString(r.Form.Get("password"))
		if login == "roman" && password == "pass111" {
			session.Values["logged"] = true
			fmt.Println("session values:", session.Values)
			session.Save(r, w) // before redirect
			http.Redirect(w, r, "/", http.StatusSeeOther)
		} else {
			pageTemplate, err := template.ParseFiles("tpl/login.html", "tpl/header.html", "tpl/footer.html")
			if err != nil {
				log.Fatalf("execution failed: %s", err)
				serveError(w, err)
			}

			tplValues := map[string]interface{}{"Header": "Login", "Copyright": "Roman Frołow"}
			if session.Values["logged"] == true && session.Values["login"] != "" {
				tplValues["login"] = session.Values["login"]
			}

			pageTemplate.Execute(w, tplValues)
			if err != nil {
				log.Fatalf("execution failed: %s", err)
				serveError(w, err)
			}
		}
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	session.Values["logged"] = false
	session.Save(r, w) // before redirect
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func products(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	tplValues := map[string]interface{}{"Header": "Products", "Copyright": "Roman Frołow"}
	db, err := sql.Open("sqlite3", "./db/app.db")
	if err != nil {
		fmt.Println(err)
		serveError(w, err)
		return
	}
	defer db.Close()

	sql := "select title, text, price from products order by title"
	rows, err := db.Query(sql)
	if err != nil {
		fmt.Printf("%q: %s\n", err, sql)
		serveError(w, err)
		return
	}
	defer rows.Close()

	levels := []map[string]string{}
	var title, text, price string
	for rows.Next() {
		rows.Scan(&title, &text, &price)
		levels = append(levels, map[string]string{"title": title, "text": text, "price": price})
	}
	tplValues["levels"] = levels

	rows.Close()

	pageTemplate, err := template.ParseFiles("tpl/products.html", "tpl/header.html", "tpl/footer.html")
	if err != nil {
		log.Fatalf("execution failed: %s", err)
		serveError(w, err)
	}

	if session.Values["logged"] == true && session.Values["login"] != "" {
		tplValues["login"] = session.Values["login"]
	}

	pageTemplate.Execute(w, tplValues)
	if err != nil {
		log.Fatalf("execution failed: %s", err)
		serveError(w, err)
	}
}

func users(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	tplValues := map[string]interface{}{"Header": "Users", "Copyright": "Roman Frołow"}
	db, err := sql.Open("sqlite3", "./db/app.db")
	if err != nil {
		fmt.Println(err)
		serveError(w, err)
		return
	}
	defer db.Close()

	sql := "select name1, surname from users order by surname"
	rows, err := db.Query(sql)
	if err != nil {
		fmt.Printf("%q: %s\n", err, sql)
		serveError(w, err)
		return
	}
	defer rows.Close()

	levels := []map[string]string{}
	var name1, surname string
	for rows.Next() {
		rows.Scan(&name1, &surname)
		levels = append(levels, map[string]string{"name1": name1, "surname": surname})
	}
	tplValues["levels"] = levels
	rows.Close()

	pageTemplate, err := template.ParseFiles("tpl/users.html", "tpl/header.html", "tpl/footer.html")
	if err != nil {
		log.Fatalf("execution failed: %s", err)
		serveError(w, err)
	}

	if session.Values["logged"] == true && session.Values["login"] != "" {
		tplValues["login"] = session.Values["login"]
	}

	pageTemplate.Execute(w, tplValues)
	if err != nil {
		log.Fatalf("execution failed: %s", err)
		serveError(w, err)
	}
}

func productsAdd(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	/*if not session.get('logged_in'):
	      abort(401)
	  g.db.execute('insert into products (title, text, price) values (?, ?, ?)',
	               [request.form['title'], request.form['text'], request.form['price']])
	  g.db.commit()
	  flash('New product was successfully added')
	  return redirect(url_for('show_products'))
	*/
	db, err := sql.Open("sqlite3", "./db/app.db")
	if err != nil {
		fmt.Println(err)
		serveError(w, err)
		return
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		fmt.Println(err)
		serveError(w, err)
		return
	}

	stmt, err := tx.Prepare("insert into products(title, text, price) values(?, ?, ?)")
	if err != nil {
		fmt.Println(err)
		serveError(w, err)
		return
	}

	defer stmt.Close()

	if err := r.ParseForm(); err != nil {
		serveError(w, err)
		return
	}
	title := html.EscapeString(r.Form.Get("title"))
	text := html.EscapeString(r.Form.Get("text"))
	price := html.EscapeString(r.Form.Get("price"))

	res, err := stmt.Exec(title, text, price)
	if err != nil {
		fmt.Println(err)
		serveError(w, err)
		return
	}
	last, err := res.LastInsertId()
	if err != nil {
		fmt.Println(err)
		serveError(w, err)
		return
	}
	fmt.Println("last", last)
	tx.Commit()

	session, _ := store.Get(r, "session-name")
	session.Values["last_product"] = last
	session.Save(r, w)

	// http://en.wikipedia.org/wiki/Post/Redirect/Get
	// http://en.wikipedia.org/wiki/HTTP_303
	// http://stackoverflow.com/questions/46582/response-redirect-with-post-instead-of-get
	// 303 for HTTP 1.1, maybe problem with old corporate proxies, so 302 could be better
	//
	// https://groups.google.com/forum/?fromgroups#!msg/golang-nuts/HeAoybScSTU/qxp1H7mWZVYJ
	// The common practice is to redirect only after successful forms.
	// So forms with errors are treated by the same POST request, and so have
	// access to the data.
	http.Redirect(w, r, "/products", http.StatusSeeOther)
}

func index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Method != "GET" || r.URL.Path != "/" {
		serve404(w)
		return
	}

	session, _ := store.Get(r, "session-name")

	pageTemplate, err := template.ParseFiles("tpl/index.html", "tpl/header.html", "tpl/footer.html")
	if err != nil {
		log.Fatalf("execution failed: %s", err)
		serveError(w, err)
	}

	tplValues := map[string]interface{}{"Header": "Home", "Copyright": "Roman Frołow"}
	if session.Values["logged"] == true && session.Values["login"] != "" {
		tplValues["login"] = session.Values["login"]
	}

	pageTemplate.Execute(w, tplValues)
	if err != nil {
		log.Fatalf("execution failed: %s", err)
		serveError(w, err)
	}
}

func serve404(w http.ResponseWriter) {
	// https://gist.github.com/1075842
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, "Nie ma takiej strony!")
}

func serveError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, "Internal Server Error "+err.Error())
}

func redirectHandler(path string) func(http.ResponseWriter, *http.Request) {
	// http://stackoverflow.com/a/9936937/588759
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, path, http.StatusMovedPermanently)
	}
	// usage: http.HandleFunc("/1", redirectHandler("/one"))
}

func parseForm(r *http.Request) error {
	// https://groups.google.com/forum/?fromgroups#!topic/golang-nuts/73bqDlejJCQ
	// parseForm calls Request.ParseForm() excluding values from the URL query.
	// It returns an error if the request body was already parsed or if it failed
	// to parse.
	//
	// http://code.google.com/p/go/issues/detail?id=3630#c2
	// If I'm not mistaken, this "exploit" requires controlling the form's action.  If an attacker can control that, they could also probably redirect the user to their own server and steal all of the information and then redirect them back to the original action with properly-formed (but compromised) POST data.  If you are concerned about this in your webapps, it is probably trivial to add a quick `if r.Method = "POST" { r.URL.RawQuery = "" }`, though I would personally recommend auditing where the form tags get their action (in my own apps, it's always hard-coded in the template).
	// I think it's poor design to care where you get your form values.  I wouldn't mind if FormValues only got the data from the canonical source for the current method (GET -> query params, POST -> form body), but putting that in your code doesn't seem like the correct approach.  The PHP language (from my view) encourages people to care about the difference, but as soon as you do you make it harder to do simple things like unit test your code.  Often it is super easy to control form responses in query parameters for testing and they also are very good for creating links to pre-populate a form (akin to "mailto" links that provide the subject for you).  When you start caring where the data came from, the logic here becomes much more difficult.
	// Assuming that an established, authenticated and secure connection's $_POST could be trusted bit me once.... Never again.
	//
	// https://groups.google.com/forum/?fromgroups#!topic/golang-nuts/ke_JP5IkofA
	// In the ParseForm method the values in the url query are overwritten by any values submitted via post
	// Nothing gets overwritten... Both values are added to req.Form. So with req.Form.Get you get the first value associated with the key - the one from your url query. The value from the post form is number two in the slice: req.Form["user"][1]
	if r.Form != nil {
		return errors.New("Request body was parsed already.")
	}
	tmp := r.URL.RawQuery
	r.URL.RawQuery = ""
	if err := r.ParseForm(); err != nil {
		return err
	}
	r.URL.RawQuery = tmp
	return nil
}

func notlsHandler(w http.ResponseWriter, r *http.Request) {
	fullUrl := "https://localhost:9999" + r.RequestURI
	http.Redirect(w, r, fullUrl, http.StatusMovedPermanently)
}

var addr = flag.String("addr", ":9999", "http service address") // Q=17, R=18

func main() {
	store.Options.Secure = true
	flag.Parse()
	// If only one version can be returned (i.e., the other redirects to it), that’s great!
	// http://googlewebmastercentral.blogspot.com/2010/04/to-slash-or-not-to-slash.html
	r := mux.NewRouter().StrictSlash(true)
	// If a site is accessed over HTTPS and loads some parts of a page over insecure HTTP, the user might still be vulnerable to some attacks or some kinds of surveillance. For instance, the New York Times makes many HTML pages available in HTTPS, but other resources such as images, CSS, JavaScript, or third party ads and tracking beacons, are only loadable over HTTP. That means that these resources are sent unencrypted, and someone spying on you could probably infer the article you were reading.
	// There are also potential vulnerabilities when parts of a page are loaded over HTTP because an attacker might replace them with versions containing false information, or Javascript code that helps the attacker spy on the user or take over an account.
	//
	// In order to [enable HTTPS by default for Gmail] we had to deploy
	// no additional machines and no special hardware. On our production
	// frontend machines, SSL/TLS accounts for less than 1% of the CPU load, less
	// than 10KB of memory per connection and less than 2% of network overhead.
	// www.eff.org/https-everywhere/faq

	// no refferal with https?

	//Not needed, because there is redirecting
	//s := r.Schemes("https").Subrouter()
	r.HandleFunc("/", http.HandlerFunc(index))
	// trailing slash denotes a directory, while the lack of it denotes a file/resource
	// http://techblog.bozho.net/?p=401
	r.HandleFunc("/login", http.HandlerFunc(login))
	r.HandleFunc("/logout", http.HandlerFunc(logout))
	r.HandleFunc("/products", http.HandlerFunc(products))
	r.HandleFunc("/products/add", http.HandlerFunc(productsAdd))
	r.HandleFunc("/users", http.HandlerFunc(users))
	http.Handle("/", r)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// starting in goroutines with error reporting, thanks to davecheney from #go-nuts
	go func() {
		log.Fatalf("ListenAndServe: %v", http.ListenAndServe(":8080", http.HandlerFunc(notlsHandler)))
	}()
	go func() {
		log.Fatalf("ListenAndServeTLS: %v", http.ListenAndServeTLS(*addr, "tls/cert.pem", "tls/key.pem", nil))
	}()
	select {}

	// Testing redirecting
	// curl -v http://localhost:8080/products
	// Or if you made forwarding or run as superuser so you have access to ports below 1024
	// curl -v http://localhost/products
}
