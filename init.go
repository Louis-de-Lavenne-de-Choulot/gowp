package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"plugin"
	"strings"
	"time"

	lib "github.com/Louis-de-Lavenne-de-Choulot/gowp_objects"
	_ "github.com/logoove/sqlite"
)

var (
	// BASIC SERVER INIT INFOS
	// tables mandatory in database for the program to work
	MANDATORYTABLES = []string{"route", "user", "role", "user_role", "route_role", "route_path", "route_path_route", "plugin"}
	// roles mandatory in database for the program to work
	MANDATORYROLES = []string{"admin", "member"}
	// plugins mandatory for the program to work
	MANDATORYPLUGIN = []string{"administration"}
	// config file to load mandatory for the program to work
	CONFIGFILE = "site_config.cfg"

	//RESULTING GLOBAL MAPS
	// all routes
	ROUTES = make(map[string]*lib.Route)
	// all settings
	CONFIGMAP = make(map[string]string)
	// all MENUPAGES string -> []string
	MENUPAGES = make(map[string][][]string)

	//SERVER
	SERVER = &lib.BaseServer{}

	//COOKIE CHECK FUNCTION (comes from administration plugin)
	COOKIEROLEFUNC func(c *http.Cookie) (int, bool)
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

	if CONFIGMAP["ROOT_DIR"] == "" {
		CONFIGMAP["ROOT_DIR"] = "./files/"
	}
	err = initDB()
	if err != nil {
		return err
	}
	err = initRoutes()
	if err != nil {
		return err
	}
	err = initAdminPlugin()
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

func initAdminPlugin() error {
	var pluginPath string
	err := SERVER.DB.QueryRow("SELECT path FROM plugin WHERE name = ?", CONFIGMAP["ADMIN_PLUGIN"]).Scan(&pluginPath)
	if err != nil {
		return err
	}

	pluginFile, err := plugin.Open(pluginPath)
	if err != nil {
		addLog("Error opening admin plugin file "+err.Error(), time.Now().Format("2006-01-02 15:04:05"))
		return err
	}
	init, err := pluginFile.Lookup("CookieLogin")
	if err != nil {
		addLog("Error looking up admin plugin cookie login function", time.Now().Format("2006-01-02 15:04:05"))
		return err
	}
	var ok bool
	COOKIEROLEFUNC, ok = init.(func(c *http.Cookie) (int, bool))
	if !ok {
		addLog("Error casting admin plugin cookiecheck function", time.Now().Format("2006-01-02 15:04:05"))
		return err
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
		var boundPluginName sql.NullString
		var boundFunction sql.NullString
		var pageID sql.NullInt64
		var MainMenuPageID sql.NullInt64
		err = sqlRows.Scan(&route.ID, &route.Title, &route.Slug, &pageID, &MainMenuPageID, &route.InMenu)
		if err != nil {
			return err
		}
		if pageID.Valid {
			route.PageID = pageID.Int64
		}
		if route.InMenu {
			if !MainMenuPageID.Valid && MENUPAGES[route.Title] == nil {
				MENUPAGES[route.Title] = [][]string{}
				MENUPAGES[route.Title] = append(MENUPAGES[route.Title], []string{route.Slug, route.Title})
			} else {
				var mainPage string
				var mainPageSlug string
				err = SERVER.DB.QueryRow("SELECT title, slug FROM route WHERE id = ?", MainMenuPageID.Int64).Scan(&mainPage, &mainPageSlug)
				if err != nil {
					return err
				}
				if MENUPAGES[mainPage] == nil {
					MENUPAGES[mainPage] = [][]string{}
					MENUPAGES[mainPage] = append(MENUPAGES[mainPage], []string{mainPageSlug, mainPage})

				}
				MENUPAGES[mainPage] = append(MENUPAGES[mainPage], []string{route.Slug, route.Title})
			}
		}

		rows, err := SERVER.DB.Query(`
	SELECT route_role.role_id, plugin.path, route_path_plugin.bound_function, concat(route_path.path, route.route_name) AS full_path FROM route_path_route
  	LEFT  JOIN route_path ON route_path.id = route_path_route.route_path_id
  	LEFT  JOIN route_role ON route_role.route_id = route_path_route.route_id
  	LEFT  JOIN route ON route.id = route_path_route.route_id
  	LEFT  JOIN route_path_plugin ON route_path_plugin.route_path_id = route_path_route.route_id
  	LEFT JOIN plugin ON plugin.id = route_path_plugin.plugin_id
  	WHERE route_path_route.route_id =?`, route.ID)
		if err != nil {
			return err
		}
		for rows.Next() {
			err = rows.Scan(&route.Role, &boundPluginName, &boundFunction, &route.Slug)
			if err != nil {
				return err
			}
		}
		if len(route.Slug) > 0 && route.Slug[0] != '/' {
			route.Slug = "/" + route.Slug
		}
		if boundPluginName.Valid && boundFunction.Valid {
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
				println("route " + route.Slug + " bound pages: " + fmt.Sprintf("%v", route.URLS) + " " + fmt.Sprintf("%v", boundFunction.String))

				route.BoundFunctions[boundPluginName.String] = append(route.BoundFunctions[boundPluginName.String], boundFunction.String)
				boundPluginName = sql.NullString{}
				boundFunction = sql.NullString{}
			}
		}

		ROUTES[route.Slug] = &route
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
		println("Route " + route.Slug)
		http.Handle(route.Slug, logger(final(route.Slug)))
	}
	return nil
}

func getRoutePaths(db *sql.DB, route *lib.Route) error {
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
