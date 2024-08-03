package main

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
	"time"

	lib "github.com/Louis-de-Lavenne-de-Choulot/gowp_objects"
	_ "github.com/logoove/sqlite"
)

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
		// all data
		var fullData = make(map[string]interface{})
		fullData["routes"] = &ROUTES
		fullData["config"] = &CONFIGMAP
		fullData["menupages"] = &MENUPAGES
		fullData["URL"] = &actualRoute.URLS

		tokenCookie, err := r.Cookie("token")
		var user lib.User = lib.User{Role: -1}
		if err != nil && actualRoute.Role != -1 {
			callback := r.URL.Path
			var q url.Values = url.Values{}
			q.Set("callback", callback)
			http.Redirect(w, r, "/admin/admin-login?"+q.Encode(), http.StatusSeeOther)
			return
		} else if err == nil {
			userID, ok := COOKIEROLEFUNC(tokenCookie)
			if !ok {
				//remove cookie
				tokenCookie.MaxAge = -1
				http.SetCookie(w, tokenCookie)
				callback := r.URL.Path
				var q url.Values = url.Values{}
				q.Set("callback", callback)
				http.Redirect(w, r, "/admin/admin-login?"+q.Encode(), http.StatusSeeOther)
				return
			}

			//extract User
			user, err = getUser(userID)
			if err != nil {
				HandleError(w, err)
				return
			}
		}

		if actualRoute.Role != -1 && user.Role > actualRoute.Role {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("You are not allowed to access this url"))
			return
		}

		if actualRoute.BoundFunctions != nil {
			err := plugingCalls(w, r, actualRoute, &fullData)

			if actualRoute.PagePath == "" || err != nil {
				w.Write([]byte("Error: " + err.Error()))
				return
			}
		}

		//path to page that is mapped to the serving route
		pagePath := actualRoute.PagePath

		tmpl, err := template.ParseFiles(CONFIGMAP["ROOT_DIR"] + pagePath)
		if err != nil {
			HandleError(w, err)
			return
		}

		//Add admin template if user has role
		if user.Role != -1 {
			if actualRoute.Role != -1 {
				pagePathAdmin := CONFIGMAP["ROOT_DIR"] + "/template/left-menu.html"
				// true because we want it wrapped in the content
				tmpl, err = AddMenuToTemplate(pagePathAdmin, tmpl, &fullData, true)
				if err != nil {
					HandleError(w, err)
					return
				}
			}
			pagePathAdmin := CONFIGMAP["ROOT_DIR"] + "/template/top-menu.html"
			tmpl, err = AddMenuToTemplate(pagePathAdmin, tmpl, &fullData, false)
			if err != nil {
				HandleError(w, err)
				return
			}
		}

		err = tmpl.Execute(w, fullData)
		if err != nil {
			HandleError(w, err)
			return
		}
	})
}
