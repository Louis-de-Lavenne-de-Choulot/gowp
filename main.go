package main

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"plugin"
	"strings"
	"time"

	lib "github.com/Louis-de-Lavenne-de-Choulot/gowp_objects"
	_ "github.com/logoove/sqlite"
)

//what about plugins :
// .so (shared files)
//entry point is plugin name InitFunc
//returns an array of plugin pages for editors / accessible pages for anyone
//https://pkg.go.dev/plugin

// 2hr of sleep in 3 days, some repetitions may have slipped in
var (
	MANDATORYTABLES = []string{"route", "user", "role", "user_role", "route_role", "route_path", "route_path_route", "plugin"}
	MANDATORYROLES  = []string{"admin", "member"}
	MANDATORYPLUGIN = []string{"administration"}
	ROUTES          = make(map[string]*lib.Route)
	ADMINROUTES     = "/admin"
	ROOT_DIR        = "./files/"
	CONFIGFILE      = "site_config.cfg"
	//result from loading config file
	CONFIGMAP                 = make(map[string]string)
	SERVER    *lib.BaseServer = &lib.BaseServer{}
)

// #Serveur Start
func loadConfig(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "=")
		if len(parts) != 2 {
			return errors.New("Invalid config line: " + line)
		}
		parts[0] = strings.TrimSpace(parts[0])
		parts[1] = strings.TrimSpace(strings.ReplaceAll(parts[1], `"`, ""))

		CONFIGMAP[parts[0]] = parts[1]

		//todo check if value is valid
	}
	return nil
}

func run() error {
	println("listening on port 8080")
	return SERVER.HttpServer.ListenAndServe()
}

func Init() error {
	// load site_config.cfg and get all variables (line break separated)
	err := loadConfig(CONFIGFILE)
	if err != nil {
		return err
	}

	if CONFIGMAP["ROOT_DIR"] != "" {
		ROOT_DIR = CONFIGMAP["ROOT_DIR"]
	}
	err = initDB()
	if err != nil {
		return err
	}
	err = initRoutes()
	if err != nil {
		return err
	}
	SERVER.HttpServer = &http.Server{

		Addr:         ":8080",
		Handler:      http.DefaultServeMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return nil
}

func initDB() error {
	var err error
	SERVER.DB, err = sql.Open("sqlite", "site.db")
	if err != nil {
		return err
	}
	return nil
}

func initRoutes() error {
	var tableNames string
	var roleNames string
	var routes []*lib.Route
	//check if all tables are present
	sqlRows, err := SERVER.DB.Query(`SELECT name FROM sqlite_master
WHERE type IN ('table','view') AND name NOT LIKE 'sqlite_%'
UNION ALL
SELECT name FROM sqlite_temp_master
WHERE type IN ('table','view')
ORDER BY 1`)
	if err != nil {
		return err
	}
	defer sqlRows.Close()
	for sqlRows.Next() {
		var tableName string
		err = sqlRows.Scan(&tableName)
		if err != nil {
			return err
		}
		tableNames += tableName + " "
		log.Printf("Found Table: %s", tableName)
	}

	for _, tableName := range MANDATORYTABLES {
		if !strings.Contains(tableNames, tableName) {
			return errors.New("Missing table: " + tableName)
		}
	}

	//check if all roles are present
	sqlRows, err = SERVER.DB.Query("SELECT name FROM role")
	if err != nil {
		return err
	}
	for sqlRows.Next() {
		var name string
		err = sqlRows.Scan(&name)
		if err != nil {
			return err
		}
		roleNames += name + " "
	}
	for _, roleName := range MANDATORYROLES {
		if !strings.Contains(roleNames, roleName) {
			return errors.New("Missing role: " + roleName)
		}
	}

	//load routes
	sqlRows, err = SERVER.DB.Query("SELECT * FROM route")
	if err != nil {
		return err
	}
	for sqlRows.Next() {
		var route lib.Route = lib.Route{}
		route.BoundFunctions = make(map[string][]string)
		route.URLS = make(map[string]string)
		route.PageReferences = make(map[string]int64)
		var boundPluginName sql.NullString
		var boundFunction sql.NullString
		err = sqlRows.Scan(&route.ID, &route.Name, &route.PageID)
		if err != nil {
			return err
		}

		rows, err := SERVER.DB.Query(`
SELECT plugin.path, route_path_plugin.bound_function, concat(route_path.path, route.route_name) AS full_path FROM route_path_route
  LEFT  JOIN route_path ON route_path.id = route_path_route.route_path_id
  LEFT  JOIN route ON route.id = route_path_route.route_id
  LEFT  JOIN route_path_plugin ON route_path_plugin.route_path_id = route_path_route.route_id
  LEFT JOIN plugin ON plugin.id = route_path_plugin.plugin_id
  WHERE route_path_route.route_id =?`, route.ID)
		if err != nil {
			return err
		}
		for rows.Next() {
			err = rows.Scan(&boundPluginName, &boundFunction, &route.Name)
			if err != nil {
				return err
			}
		}
		if len(route.Name) > 0 && route.Name[0] != '/' {
			route.Name = "/" + route.Name
		}
		if boundPluginName.Valid && boundFunction.Valid {
			println("routepath bound function: " + boundPluginName.String + " " + boundFunction.String)
			route.BoundFunctions[boundPluginName.String] = append(route.BoundFunctions[boundPluginName.String], boundFunction.String)
			boundPluginName = sql.NullString{}
			boundFunction = sql.NullString{}
		}

		rows, err = SERVER.DB.Query(`
SELECT route_to_route_reference.old_route_name, concat(route_path.path, route.route_name) FROM route_to_route_reference
    LEFT  JOIN route_path_route ON route_path_route.route_id = route_to_route_reference.route_id
    LEFT JOIN route_path ON route_path.id = route_path_route.route_path_id
	LEFT JOIN route ON route.id = route_to_route_reference.new_route_id
  WHERE route_to_route_reference.route_id =?`, route.ID)
		if err != nil {
			return err
		}

		for rows.Next() {
			var oldName, newName string
			err = rows.Scan(&oldName, &newName)
			if err != nil {
				return err
			}
			if len(newName) > 0 && newName[0] != '/' {
				newName = "/" + newName
			}
			route.URLS[oldName] = newName
		}

		rows, err = SERVER.DB.Query(`SELECT plugin.path, route_plugin.bound_function FROM route_plugin
  LEFT JOIN plugin ON plugin.id = route_plugin.plugin_id
  WHERE route_id = ?`, route.ID)
		if err != nil {
			return err
		}

		for rows.Next() {
			err = rows.Scan(&boundPluginName, &boundFunction)
			if err != nil {
				return err
			}
			if boundPluginName.Valid && boundFunction.Valid {
				println("bound function: " + boundPluginName.String + " " + boundFunction.String)

				route.BoundFunctions[boundPluginName.String] = append(route.BoundFunctions[boundPluginName.String], boundFunction.String)
				boundPluginName = sql.NullString{}
				boundFunction = sql.NullString{}
			}
		}

		ROUTES[route.Name] = &route
		routes = append(routes, &route)
	}
	println("routes :")
	for _, route := range routes {
		// get route file path
		err = getRoutePaths(SERVER.DB, route)
		if err != nil {
			println(err.Error())
			return err
		}
		println("Route " + route.Name)
		http.Handle(route.Name, logger(final(route.Name)))
	}
	return nil
}

func getRoutePaths(db *sql.DB, route *lib.Route) error {

	// rows, err := db.Query(`
	// SELECT concat(route.route_name, '/',route_path.path) AS full_path FROM route_path_route
	// LEFT JOIN route_path ON route_path.id = route_path_route.route_path_id
	// LEFT JOIN route ON route.id = route_path_route.route_id
	// WHERE route_id = ?
	// `, route.PageID)
	rows, err := db.Query(`SELECT name FROM page WHERE id = ?`, route.PageID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var pagePath string
		err = rows.Scan(&pagePath)
		if err != nil {
			return err
		}
		route.PagePath = pagePath
	}

	return nil
}

// #Serveur Running
func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		go func() {
			//log in db table log id date content
			date := time.Now().Format("2006-01-02 15:04:05")
			content := "lib.Route " + r.URL.Path + " accessed by " + r.RemoteAddr
			err := addLog(content, date)
			if err != nil {
				log.Printf("Error adding log: %s", err)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func addLog(content, date string) error {
	// prepare statement
	stmt, err := SERVER.DB.Prepare("INSERT INTO log (date, content) VALUES (?, ?)")
	if err != nil {
		log.Printf("Error preparing statement: %s", err)
		return err
	}
	// execute statement
	_, err = stmt.Exec(date, content)
	if err != nil {
		log.Printf("Error executing statement: \n\n'''\n%s\n'''\n\n with Error: \n'''\n%s\n'''", content, err)
		return err
	}
	return nil
}

type URLS struct {
	URL map[string]string
}

func final(servingRoute string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/admin/plugin/add" {
			addPlugin(w, r)
			return
		}

		actualRoute := ROUTES[servingRoute]
		fmt.Println(actualRoute)
		if actualRoute.BoundFunctions != nil {
			err := plugingCalls(w, r, actualRoute)
			if actualRoute.PagePath == "" || err != nil {
				return
			}
		}

		var urls URLS = URLS{}
		urls.URL = actualRoute.URLS
		//path to page that is mapped to the serving route
		pagePath := actualRoute.PagePath

		tmpl, err := template.ParseFiles(ROOT_DIR + pagePath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		print("Serving route: " + servingRoute + " at: " + pagePath + " from: " + ROOT_DIR + " with call " + r.URL.Path + "\n")
		err = tmpl.Execute(w, urls)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

	})
}

func plugingCalls(w http.ResponseWriter, r *http.Request, actualRoute *lib.Route) error {
	for plugin_name, functions := range actualRoute.BoundFunctions {

		pluginFile, err := plugin.Open(plugin_name)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			addLog("Error opening plugin file "+err.Error(), time.Now().Format("2006-01-02 15:04:05"))
			return err
		}

		for _, function := range functions {
			init, err := pluginFile.Lookup(function)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				addLog("Error looking up plugin init function", time.Now().Format("2006-01-02 15:04:05"))
				return err
			}
			fn, ok := init.(func(x http.ResponseWriter, y *http.Request, route *lib.Route) error)
			if !ok {
				w.WriteHeader(http.StatusInternalServerError)
				addLog("Error casting plugin init function", time.Now().Format("2006-01-02 15:04:05"))
				return err
			}
			err = fn(w, r, actualRoute)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				addLog("Error calling plugin function", time.Now().Format("2006-01-02 15:04:05"))
				return err
			}
		}
	}
	return nil
}

func cleanUpPlugin(stepN int8, headerFN string, templatesPath string, removeCalls []string) {
	os.Remove("./temp/" + headerFN)
	if stepN == 0 {
		return
	}
	headerFN = headerFN[:len(headerFN)-4]
	os.RemoveAll("./temp/" + headerFN)
	if stepN == 1 {
		return
	}
	//print list of remove calls
	os.Remove(ROOT_DIR + templatesPath)
	for i := len(removeCalls) - 1; i > 0; i-- {
		_, err := SERVER.DB.Exec(removeCalls[i])
		if err != nil {
			fmt.Printf("Error removing plugin: %s with Query '''\n%s'''\n\n", err, removeCalls[i])
		}
	}
}

// #Main
func main() {
	err := Init()
	if err != nil {
		log.Fatal(err)
	}
	run()
}
