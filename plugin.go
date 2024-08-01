package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"plugin"
	"strconv"
	"strings"
	"time"

	lib "github.com/Louis-de-Lavenne-de-Choulot/gowp_objects"
)

func addPlugin(w http.ResponseWriter, r *http.Request) {
	//create a list of db remove calls that will append 1remove per insert
	var removeCalls []string

	// zip file is uploaded, now we can save it
	// get the file from the request
	file, header, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		w.Write([]byte("Error getting file"))
		return
	}
	defer file.Close()
	pluginTempPath := "./temp/" + header.Filename
	// create a new file
	out, err := os.Create(pluginTempPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error creating temporary file"))
		return
	}
	defer out.Close()

	// write the file
	_, err = io.Copy(out, file)
	if err != nil {
		cleanUpPlugin(0, header.Filename, "", nil)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error copying to temporary file"))
		return
	}

	//check if ./plugins/header.Filename is already

	// unzip the file
	err = unzip("./temp/"+header.Filename, "./temp")
	if err != nil {
		cleanUpPlugin(1, header.Filename, "", nil)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error unzipping file"))
		return
	}
	headerFN := header.Filename[:len(header.Filename)-4]
	var pluginInfosMap = loadPluginInfos("./temp/" + headerFN + "/plugin_infos.json")
	if pluginInfosMap == nil {
		cleanUpPlugin(1, header.Filename, "", nil)
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Error getting plugin infos with path ./temp/" + headerFN + "/plugin_infos.json"))
		return
	}
	// get the name of the plugin
	pluginName := pluginInfosMap["name"].(string)
	if pluginName == "" {
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("No plugin name"))
		return
	}
	// get the base route of the plugin
	baseRoute := pluginInfosMap["base_route"].(string)
	if baseRoute == "" {
		baseRoute = pluginName
	}
	// if url path does not start with /, add one
	if strings.HasPrefix(baseRoute, "/") {
		baseRoute = "/" + baseRoute
	}
	// get the version of the plugin
	version := pluginInfosMap["version"].(string)
	// get the author of the plugin
	author := pluginInfosMap["author"].(string)
	// get the description of the plugin
	description := pluginInfosMap["description"].(string)
	// get plugin filename
	pluginFileName := pluginInfosMap["plugin_file_name"].(string)
	if pluginFileName == "" {
		pluginFileName = pluginName
	}
	// // get the languages supported by the plugin
	languages := pluginInfosMap["languages"].([]interface{})

	// create array of Route
	var routes []lib.Route
	// create map[string]int64
	var tempMap map[string]int64 = make(map[string]int64)
	// check if plugin already exists and is same version
	var idVersion int64 = -1
	var foundVersion string = ""
	SERVER.DB.QueryRow("SELECT id, version FROM plugin WHERE name = ?", pluginName).Scan(&idVersion, &foundVersion)
	if foundVersion != "" && foundVersion == version {
		cleanUpPlugin(1, header.Filename, "", nil)
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Plugin with same version already exists"))
		return
	}

	// check if base_route already exists
	var dbRouteName string
	SERVER.DB.QueryRow("SELECT name FROM route WHERE name = ?", baseRoute).Scan(&dbRouteName)
	if dbRouteName != "" {
		cleanUpPlugin(1, header.Filename, "", nil)
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Base route already exists"))
		return
	}
	// get file .so from plugin
	if len(languages) == 0 {
		cleanUpPlugin(1, header.Filename, "", nil)
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("no language added"))
		return
	}
	file_language := fmt.Sprintf("%v", languages[0])
	pluginPath := "./temp/" + headerFN + "/" + file_language + "/" + pluginFileName
	pluginFile, err := plugin.Open(pluginPath)
	// Run Init and wait for []lib.RouteImport
	if err != nil {
		cleanUpPlugin(1, header.Filename, "", nil)
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Error opening plugin file" + pluginPath + "\n\n ERROR : " + err.Error()))
		return
	}
	init, err := pluginFile.Lookup("Init")
	if err != nil {
		cleanUpPlugin(1, header.Filename, "", nil)
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Error looking up plugin init function"))
		return
	}
	initFunc, ok := init.(func() lib.RootImport)
	if !ok {
		cleanUpPlugin(1, header.Filename, "", nil)
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Error casting plugin init function"))
		return
	}
	routesImport := initFunc()
	//push plugin to db
	res, err := SERVER.DB.Exec("INSERT INTO plugin (name, version, author, description, path) VALUES (?, ?, ?, ?, ?)", pluginName, version, author, description, pluginPath)
	if err != nil {
		cleanUpPlugin(1, header.Filename, "", nil)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	pluginID, _ := res.LastInsertId()
	removeCalls = append(removeCalls, "DELETE FROM plugin WHERE id ="+strconv.Itoa(int(pluginID)))

	// add route_path
	res, err = SERVER.DB.Exec("INSERT INTO route_path (path) VALUES (?)", baseRoute)
	if err != nil {
		cleanUpPlugin(2, header.Filename, pluginName, removeCalls)
		w.WriteHeader(http.StatusInternalServerError)
		println("INSERT INTO route_path (path) VALUES (?)", baseRoute)
		w.Write([]byte("Error inserting route path into database ERROR: " + err.Error()))
		return
	}
	routePATHID, _ := res.LastInsertId()
	removeCalls = append(removeCalls, "DELETE FROM route_path WHERE id ="+strconv.Itoa(int(routePATHID)))

	for _, routePathFunction := range routesImport.BoundFunctions {
		// add route_path_plugin
		res, err = SERVER.DB.Exec("INSERT INTO route_path_plugin (route_path_id, plugin_id, bound_function) VALUES (?,?,?)", routePATHID, pluginID, routePathFunction)
		if err != nil {
			cleanUpPlugin(2, header.Filename, pluginName, removeCalls)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error inserting route_path_plugin into database ERROR: " + err.Error()))
			return
		}

		routePATHPluginID, _ := res.LastInsertId()
		removeCalls = append(removeCalls, "DELETE FROM route_path_plugin WHERE id ="+strconv.Itoa(int(routePATHPluginID)))
	}

	// FOREACH route import
	for _, routeImport := range routesImport.Routes {
		// anyone == file is under DEFAULT_ROUTE/route
		var routePath int64 = -1
		// IF route permissions
		if routeImport.Role != 0 {
			// other == file is under /base_route/route
			routePath = routePATHID
		}

		newPath := ROOT_DIR + pluginName + routeImport.PagePath
		os.MkdirAll(newPath[:strings.LastIndex(newPath, "/")], 0755)
		//new io writer
		writer, err := os.OpenFile(newPath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			cleanUpPlugin(2, header.Filename, pluginName, removeCalls)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error creating page file " + newPath + " ERROR: " + err.Error()))
			return
		}
		//new io reader
		reader, err := os.Open("./temp/" + headerFN + routeImport.PagePath)
		if err != nil {
			cleanUpPlugin(2, header.Filename, pluginName, removeCalls)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error opening page file " + "./temp/" + headerFN + routeImport.PagePath + " ERROR: " + err.Error()))
			return
		}
		//move page to ROOT_DIR/plugin_name/
		_, err = io.Copy(writer, reader)
		if err != nil {
			cleanUpPlugin(2, header.Filename, pluginName, removeCalls)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error moving page to plugin folder " + "./temp" + routeImport.PagePath + " to " + newPath))
			return
		}

		//remove ROOT_DIR
		newPath = strings.Replace(newPath, ROOT_DIR, "", 1)
		// add route to page table
		res, err = SERVER.DB.Exec("INSERT INTO page (name) VALUES (?)", newPath)
		if err != nil {
			cleanUpPlugin(2, header.Filename, pluginName, removeCalls)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error inserting page into database ERROR: " + err.Error()))
			return
		}
		// get page ID
		routeImport.PageID, _ = res.LastInsertId()
		removeCalls = append(removeCalls, "DELETE FROM page WHERE id ="+strconv.Itoa(int(routeImport.PageID)))

		// add route to routes table
		res, err = SERVER.DB.Exec("INSERT INTO route (route_name, page_id) VALUES (?, ?)", routeImport.Name, routeImport.PageID)
		if err != nil {
			cleanUpPlugin(2, header.Filename, pluginName, removeCalls)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error inserting route into database ERROR: " + err.Error()))
			return
		}
		// get page ID
		routeID, _ := res.LastInsertId()
		removeCalls = append(removeCalls, "DELETE FROM route WHERE id ="+strconv.Itoa(int(routeID)))

		for _, routePathFunction := range routeImport.BoundFunctions {
			// add route_path_plugin
			res, err = SERVER.DB.Exec("INSERT INTO route_plugin (route_id, plugin_id, bound_function) VALUES (?,?,?)", routeID, pluginID, routePathFunction)
			if err != nil {
				cleanUpPlugin(2, header.Filename, pluginName, removeCalls)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Error inserting route_plugin into database ERROR: " + err.Error()))
				return
			}

			routePluginID, _ := res.LastInsertId()
			removeCalls = append(removeCalls, "DELETE FROM route_plugin WHERE id ="+strconv.Itoa(int(routePluginID)))
		}
		println("added route " + routeImport.Name + " with ID " + strconv.Itoa(int(routeID)))
		tempMap[routeImport.Name] = routeID

		if routePath != -1 {
			routeImport.Name = baseRoute + "/" + routeImport.Name
			res, err = SERVER.DB.Exec("INSERT INTO route_path_route (route_path_id, route_id) VALUES (?, ?)", routePath, routeID)
			if err != nil {
				cleanUpPlugin(2, header.Filename, pluginName, removeCalls)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Error inserting route_path_route route into database ERROR: " + err.Error()))
				return
			}

			routeLinkID, _ := res.LastInsertId()
			removeCalls = append(removeCalls, "DELETE FROM route_path_route WHERE route_id ="+strconv.Itoa(int(routeLinkID)))
		} else {
			routeImport.Name = CONFIGMAP["DEFAULT_ROUTE"] + "/" + routeImport.Name
		}

		// get assigned route ID and add it to map[oldName]newID and to Route.PageID
		route := lib.Route{
			ID:             routeID,
			Name:           routeImport.Name,
			PagePath:       newPath,
			PageID:         routeID,
			PageReferences: make(map[string]int64),
			Role:           routeImport.Role,
		}
		for _, strRef := range routeImport.PageReferences {
			route.PageReferences[strRef] = 0
		}
		// add route to []Route
		routes = append(routes, route)
	}

	// FOREACH route import
	for _, route := range routes {
		// FOREACH References in route import[i]
		for strRef := range route.PageReferences {
			route.PageReferences[strRef] = tempMap[strRef]

			// push to table route_to_route_reference
			res, err = SERVER.DB.Exec("INSERT INTO route_to_route_reference (route_id, old_route_name, new_route_id) VALUES (?, ?, ?)", route.ID, strRef, tempMap[strRef])
			if err != nil {
				cleanUpPlugin(2, header.Filename, pluginName, removeCalls)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Error inserting route to route reference into database, err " + err.Error()))
				return
			}
			routeRef, _ := res.LastInsertId()
			removeCalls = append(removeCalls, "DELETE FROM route_path_route WHERE id ="+strconv.Itoa(int(routeRef)))

		}
		// ENDFOREACH
	}
	//copy plugin from ./temp to ./plugins
	err = copyDir("./temp/"+headerFN, "./plugins/"+headerFN)
	if err != nil {
		cleanUpPlugin(2, header.Filename, pluginName, removeCalls)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error moving plugin to plugins folder"))
		return
	}
	cleanUpPlugin(1, header.Filename, "", nil)

	addLog("Added Plugin "+pluginName+" by "+author+" version "+version+" description "+description, time.Now().Format("2006-01-02 15:04:05"))
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Plugin " + pluginName + " added successfully"))
}

// func deletePlugin(w http.ResponseWriter, r *http.Request) {
// 	//TODO
// }

func loadPluginInfos(file string) map[string]interface{} {
	pluginInfos, err := os.Open(file)
	if err != nil {
		return nil
	}
	defer pluginInfos.Close()
	// read the file
	pluginInfosBytes, err := ioutil.ReadAll(pluginInfos)
	if err != nil {
		return nil
	}
	// unmarshal the json
	var pluginInfosMap map[string]interface{}
	err = json.Unmarshal(pluginInfosBytes, &pluginInfosMap)
	if err != nil {
		return nil
	}
	return pluginInfosMap
}

func unzip(src, dest string) error {
	// Open the ZIP file
	zipFile, err := zip.OpenReader(src)
	if err != nil {
		addLog("ZIP: Error opening zip file: "+err.Error(), time.Now().Format("2006-01-02 15:04:05"))
		return err
	}
	defer zipFile.Close()

	//check if destination already exists if so then throw
	if _, err := os.Stat(dest); err != nil && !os.IsNotExist(err) {
		addLog("ZIP: Destination already exists: "+err.Error(), time.Now().Format("2006-01-02 15:04:05"))
		return err
	}

	if err = os.MkdirAll(dest, 0755); err != nil {
		addLog("ZIP: Error creating end folder: "+err.Error(), time.Now().Format("2006-01-02 15:04:05"))
		return err
	}

	// Extract the contents of the ZIP file
	for _, file := range zipFile.File {
		destPath := filepath.Join(dest, file.Name)
		// Create the destination directory if it doesn't exist
		if file.FileInfo().IsDir() {
			err := os.MkdirAll(destPath, file.Mode())
			if err != nil {
				addLog("ZIP: Error creating a destination folder: "+err.Error(), time.Now().Format("2006-01-02 15:04:05"))
			}
			continue
		}

		// Open the file for reading
		fileReader, err := file.Open()
		if err != nil {
			addLog("ZIP: Error opening file to copy: "+err.Error(), time.Now().Format("2006-01-02 15:04:05"))
			continue
		}
		defer fileReader.Close()

		// Create the destination file
		destFile, err := os.Create(destPath)
		if err != nil {
			addLog("ZIP: Error creating a destination file: "+err.Error(), time.Now().Format("2006-01-02 15:04:05"))
			continue
		}
		defer destFile.Close()

		// Copy the contents of the file to the destination file
		if _, err = io.Copy(destFile, fileReader); err != nil {
			addLog("ZIP: Error copying file: "+err.Error(), time.Now().Format("2006-01-02 15:04:05"))
			continue
		}
	}

	fmt.Println("ZIP file extracted successfully!")
	return nil
}

// File copies a single file from src to dst
func copyFile(src, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File
	var srcinfo os.FileInfo

	if srcfd, err = os.Open(src); err != nil {
		return err
	}
	defer srcfd.Close()

	if dstfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}
	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcinfo.Mode())
}

// Dir copies a whole directory recursively
func copyDir(src string, dst string) error {
	var err error
	var fds []os.FileInfo
	var srcinfo os.FileInfo

	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}

	if err = os.MkdirAll(dst, srcinfo.Mode()); err != nil {
		return err
	}

	if fds, err = ioutil.ReadDir(src); err != nil {
		return err
	}
	for _, fd := range fds {
		srcfp := path.Join(src, fd.Name())
		dstfp := path.Join(dst, fd.Name())

		if fd.IsDir() {
			if err = copyDir(srcfp, dstfp); err != nil {
				fmt.Println(err)
			}
		} else {
			if err = copyFile(srcfp, dstfp); err != nil {
				fmt.Println(err)
			}
		}
	}
	return nil
}
