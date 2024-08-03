package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"plugin"
	"strings"
	"time"

	lib "github.com/Louis-de-Lavenne-de-Choulot/gowp_objects"
)

func HandleError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}

func AddMenuToTemplate(pagePathToAdd string, tmpl *template.Template, fullData *map[string]interface{}, wrap bool) (*template.Template, error) {

	tmplAdmin, err := template.ParseFiles(pagePathToAdd)
	if err != nil {
		return tmpl, err
	}
	// Insert the content from the first template into the second template just after <body>

	// Parse the HTML
	// Find the index of the <body> tag
	bufAdmin := new(strings.Builder)
	err = tmplAdmin.Execute(bufAdmin, fullData)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return tmpl, err
	}

	buf := new(strings.Builder)
	err = tmpl.Execute(buf, fullData)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return tmpl, err
	}
	templateString := buf.String()

	bodyStartIndex := strings.Index(templateString, "<body")
	if bodyStartIndex == -1 {
		fmt.Println("No <body> tag found in the template")
		return tmpl, err
	}

	// Find the index of the closing ">" of the <body> tag
	bodyStartEndIndex := bodyStartIndex + len("body>") + 1

	// Find the index of the closing </body> tag
	bodyEndIndex := strings.Index(templateString, "</body")
	if bodyEndIndex == -1 {
		fmt.Println("No <body> tag found in the template")
		return tmpl, err
	}

	newHTML := string(templateString[:bodyStartEndIndex])
	if wrap {
		wrapperStr := "<div class=\"gowp-main-container\">"
		userHTMLWrapper := "<div class=\"gowp-container\">"
		newHTML += wrapperStr + bufAdmin.String() + userHTMLWrapper + string(templateString[bodyStartEndIndex:bodyEndIndex]) + "</div></div>" + string(templateString[bodyEndIndex:])

	} else {
		newHTML += bufAdmin.String() + string(templateString[bodyStartEndIndex:])
	}
	// Reassign the modified template to the original `tmpl` variable
	tmpl, err = template.New("modified").Parse(newHTML)
	if err != nil {
		fmt.Println("Error creating modified template:", err)
		return tmpl, err
	}
	return tmpl, nil
}

func getUser(userID int) (lib.User, error) {
	var user lib.User
	err := SERVER.DB.QueryRow("SELECT user.id, username, user.name, email, role.unique_key FROM user LEFT JOIN user_role ON user_id = user.id LEFT JOIN role ON role.id = user_role.role_id  WHERE user.id = ?", userID).Scan(&user.ID, &user.Username, &user.Name, &user.Email, &user.Role)
	if err != nil {
		println("Error getting user: " + err.Error())
		return user, err
	}
	return user, nil
}

func plugingCalls(w http.ResponseWriter, r *http.Request, actualRoute *lib.Route, fullData *map[string]interface{}) error {
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
			fn, ok := init.(func(s *lib.BaseServer, x http.ResponseWriter, y *http.Request, fD *map[string]interface{}) error)
			if !ok {
				w.WriteHeader(http.StatusInternalServerError)
				addLog("Error casting plugin "+function+"() function", time.Now().Format("2006-01-02 15:04:05"))
				return err
			}
			err = fn(SERVER, w, r, fullData)
			if err != nil {
				addLog("Error inside plugin "+function+"() function with error : "+err.Error(), time.Now().Format("2006-01-02 15:04:05"))
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
	os.Remove(CONFIGMAP["ROOT_DIR"] + templatesPath)
	for i := len(removeCalls) - 1; i > 0; i-- {
		_, err := SERVER.DB.Exec(removeCalls[i])
		if err != nil {
			fmt.Printf("Error removing plugin: %s with Query '''\n%s'''\n\n", err, removeCalls[i])
		}
	}
}
