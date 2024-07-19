package main

// import (
// 	"database/sql"
// 	"errors"
// 	"log"
// 	"net/http"
// 	"os"
// 	"strings"
// 	"time"

// 	_ "github.com/logoove/sqlite"
// )

// //what about plugins :
// // .so (shared files)
// //entry point is plugin name InitFunc
// //returns an array of plugin pages for editors / accessible pages for anyone
// //https://pkg.go.dev/plugin

// var (
// 	MANDATORYTABLES = []string{"route", "user", "role", "user_role", "route_role", "route_path", "route_path_route", "plugin"}
// 	MANDATORYROLES  = []string{"admin", "member"}
// 	MANDATORYPLUGIN = []string{"administration"}
// 	ROUTES          = make(map[string]*Route)
// 	CONFIGFILE      = "site_config.cfg"
// 	CONFIGMAP       = make(map[string]string)
// )

// func LoadConfig(file string) error {
// 	data, err := os.ReadFile(file)
// 	if err != nil {
// 		return err
// 	}
// 	lines := strings.Split(string(data), "\n")
// 	for _, line := range lines {
// 		line = strings.TrimSpace(line)
// 		if line == "" {
// 			continue
// 		}
// 		parts := strings.Split(line, "=")
// 		if len(parts) != 2 {
// 			return errors.New("Invalid config line: " + line)
// 		}
// 		parts[0] = strings.TrimSpace(parts[0])
// 		parts[1] = strings.TrimSpace(strings.ReplaceAll(parts[1], `"`, ""))

// 		CONFIGMAP[parts[0]] = parts[1]

// 		//todo check if value is valid
// 	}
// 	return nil
// }

// func (bs *BaseServer) Run() error {
// 	return bs.HttpServer.ListenAndServe()
// }

// func (bs *BaseServer) Init() error {
// 	// load site_config.cfg and get all variables (line break separated)
// 	err := LoadConfig(CONFIGFILE)
// 	if err != nil {
// 		return err
// 	}
// 	err = bs.InitDB()
// 	if err != nil {
// 		return err
// 	}
// 	bs.InitRoutes()
// 	bs.HttpServer = &http.Server{
// 		Addr:         ":8080",
// 		Handler:      http.DefaultServeMux,
// 		ReadTimeout:  10 * time.Second,
// 		WriteTimeout: 10 * time.Second,
// 	}
// 	return nil
// }

// func (bs *BaseServer) InitDB() error {
// 	var err error
// 	bs.DB, err = sql.Open("sqlite", "site.db")
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (bs *BaseServer) InitRoutes() error {
// 	var tableNames string
// 	var roleNames string
// 	var routes []*Route
// 	//check if all tables are present
// 	sqlRows, err := bs.DB.Query(`SELECT name FROM sqlite_master
// WHERE type IN ('table','view') AND name NOT LIKE 'sqlite_%'
// UNION ALL
// SELECT name FROM sqlite_temp_master
// WHERE type IN ('table','view')
// ORDER BY 1`)
// 	if err != nil {
// 		return err
// 	}
// 	defer sqlRows.Close()
// 	for sqlRows.Next() {
// 		var tableName string
// 		err = sqlRows.Scan(&tableName)
// 		if err != nil {
// 			return err
// 		}
// 		tableNames += tableName + " "
// 		log.Printf("Found Table: %s", tableName)
// 	}

// 	for _, tableName := range MANDATORYTABLES {
// 		if !strings.Contains(tableNames, tableName) {
// 			return errors.New("Missing table: " + tableName)
// 		}
// 	}

// 	//check if all roles are present
// 	sqlRows, err = bs.DB.Query("SELECT name FROM role")
// 	for sqlRows.Next() {
// 		var name string
// 		err = sqlRows.Scan(&name)
// 		if err != nil {
// 			return err
// 		}
// 		roleNames += name + " "
// 	}
// 	for _, roleName := range MANDATORYROLES {
// 		if !strings.Contains(roleNames, roleName) {
// 			return errors.New("Missing role: " + roleName)
// 		}
// 	}

// 	//load routes
// 	sqlRows, err = bs.DB.Query("SELECT * FROM route")
// 	for sqlRows.Next() {
// 		var route Route
// 		err = sqlRows.Scan(&route.ID, &route.Name, &route.PageID)
// 		if err != nil {
// 			return err
// 		}
// 		ROUTES[route.Name] = &route
// 		routes = append(routes, &route)
// 	}

// 	for _, route := range routes {
// 		// get route file path
// 		err = getPagePaths(bs.DB, route)
// 		if err != nil {
// 			return err
// 		}
// 		http.Handle(route.Name, bs.Logger(final(route.Name)))
// 	}
// 	return nil
// }

// func getPagePaths(db *sql.DB, route *Route) error {

// 	rows, err := db.Query(`
// 	SELECT concat(path.path, '/',page.name) AS full_path FROM page_path
// 	INNER JOIN path ON path.id = page_path.path_id
// 	INNER JOIN page ON page.id = page_path.page_id
// 	WHERE page_id = ?
// 	`, route.PageID)

// 	if err != nil {
// 		return err
// 	}
// 	defer rows.Close()

// 	for rows.Next() {
// 		var pagePath string
// 		err = rows.Scan(&pagePath)
// 		if err != nil {
// 			return err
// 		}
// 		route.PagePath = pagePath
// 	}
// 	return nil
// }

// func (bs *BaseServer) Logger(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		go func() {
// 			//log in db table log id date content
// 			date := time.Now().Format("2006-01-02 15:04:05")
// 			content := "Route " + r.URL.Path + " accessed by " + r.RemoteAddr

// 			// prepare statement
// 			stmt, err := bs.DB.Prepare("INSERT INTO log (date, content) VALUES (?, ?)")
// 			if err != nil {
// 				log.Printf("Error preparing statement: %s", err)
// 				return
// 			}
// 			// execute statement
// 			_, err = stmt.Exec(date, content)
// 			if err != nil {
// 				log.Printf("Error executing statement: %s", err)
// 			}
// 		}()
// 		next.ServeHTTP(w, r)
// 	})
// }

// func final(servingRoute string) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		pagePath := ROUTES[servingRoute].PagePath
// 		rootDirectory := "./files"
// 		if CONFIGMAP["ROOT_DIR"] != "" {
// 		}
// 		print("Serving route: " + servingRoute + " at: " + pagePath + " from: " + rootDirectory + "\n")
// 		http.ServeFile(w, r, rootDirectory+"/"+pagePath)
// 	})
// }

// func main() {
// 	bs := BaseServer{}
// 	bs.Init()
// 	bs.Run()
// }
