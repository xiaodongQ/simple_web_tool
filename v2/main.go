package main

import (
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

//go:embed templates/*
var templates embed.FS

// Config holds database connection parameters
type Config struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
}

var (
	currentDB *sql.DB
	configs   []Config
)

func main() {
	err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/config", configHandler)
	http.HandleFunc("/user-stats", userStatsHandler)
	http.HandleFunc("/files", filesHandler)

	fmt.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	// tmpl, err := template.ParseFS(templates, "templates/base.html")
	tmpl, err := template.ParseFS(templates, "templates/*.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func configHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling config request, method:", r.Method)
	switch r.Method {
	case http.MethodGet:
		tmpl, err := template.ParseFS(templates, "templates/base.html", "templates/config.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tmpl.Execute(w, map[string]interface{}{
			"Configs": configs,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		fmt.Printf("form: %+v\n", r.Form)

		// Parse form data
		var newConfigs []Config
		for i := 0; i < 3; i++ {
			host := r.FormValue(fmt.Sprintf("host_%d", i))
			port := r.FormValue(fmt.Sprintf("port_%d", i))
			user := r.FormValue(fmt.Sprintf("user_%d", i))
			pass := r.FormValue(fmt.Sprintf("pass_%d", i))
			dbname := r.FormValue(fmt.Sprintf("dbname_%d", i))

			if host != "" {
				newConfigs = append(newConfigs, Config{
					Host:     host,
					Port:     port,
					User:     user,
					Password: pass,
					DBName:   dbname,
				})
			}
			if i < len(newConfigs) {
				fmt.Printf("new config: %+v\n", newConfigs[i])
			}
		}

		configs = newConfigs
		if err := saveConfig(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			w.Write([]byte("Configuration saved successfully"))
		} else {
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	}
}

func userStatsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling user stats request")

	dbIndex := r.URL.Query().Get("db")
	if dbIndex == "" {
		dbIndex = "0"
	}

	index, err := strconv.Atoi(dbIndex)
	if err != nil || index < 0 || index >= len(configs) {
		http.Error(w, "Invalid database index", http.StatusBadRequest)
		return
	}

	fmt.Printf("using db: %d, host: %s\n", index, configs[index].Host)
	db, err := connectDB(configs[index])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	users, err := getUserStats(db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		tmpl, err := template.ParseFS(templates, "templates/user_stats.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "content", map[string]interface{}{
			"Users":   users,
			"Configs": configs,
		})
		return
	}

	tmpl, err := template.ParseFS(templates, "templates/base.html", "templates/user_stats.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, map[string]interface{}{
		"Users":   users,
		"Configs": configs,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func filesHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling files request")

	userID := r.URL.Query().Get("user")
	part := r.URL.Query().Get("part")
	fidStr := r.URL.Query().Get("fid")
	fname := r.URL.Query().Get("fname")
	fmt.Printf("req userID: %s, part: %s, fidStr: %s, fname: %s\n", userID, part, fidStr, fname)

	if userID == "" || part == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Parse parameters
	uid, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var fid uint64
	if fidStr != "" {
		fid, err = strconv.ParseUint(fidStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid bucket ID", http.StatusBadRequest)
			return
		}
	}

	// Get database index
	dbIndex := r.URL.Query().Get("db")
	if dbIndex == "" {
		dbIndex = "0"
	}
	index, err := strconv.Atoi(dbIndex)
	if err != nil || index < 0 || index >= len(configs) {
		http.Error(w, "Invalid database index", http.StatusBadRequest)
		return
	}

	// Connect to database
	db, err := connectDB(configs[index])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Query files
	files, err := getFiles(db, uid, part, fid, fname)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFS(templates, "templates/base.html", "templates/files.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, map[string]interface{}{
		"Files":   files,
		"Configs": configs,
		"UserID":  uid,
		"Part":    part,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func initConfig() error {
	// TODO: implement config loading
	return nil
}
