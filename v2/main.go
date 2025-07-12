package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

//go:embed templates/*
var templates embed.FS

var (
	appConfig     AppConfig
	dbConnections = make(map[int]*sql.DB) // 存储数据库连接池
)

func main() {
	// 日志打印行号
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// 支持自定义监听端口
	port := flag.String("port", "8888", "server listen port")
	flag.Parse()

	err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// 初始化默认数据库连接
	if appConfig.DefaultDBIndex != -1 && appConfig.DefaultDBIndex < len(appConfig.Configs) {
		cfg := appConfig.Configs[appConfig.DefaultDBIndex]
		db, err := connectDB(cfg)
		if err != nil {
			log.Fatalf("Failed to connect to default database %d: %v", appConfig.DefaultDBIndex, err)
		}
		dbConnections[appConfig.DefaultDBIndex] = db
	}

	http.HandleFunc("/", userStatsHandler) // 修改根路由
	http.HandleFunc("/config", configHandler)
	http.HandleFunc("/user-stats", userStatsHandler)
	http.HandleFunc("/files", filesHandler)

	log.Printf("Starting server on port %s", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", *port), nil))
}

func configHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling config request, method:", r.Method)
	switch r.Method {
	case http.MethodGet:
		tmpl, err := template.ParseFS(templates, "templates/base.html", "templates/config.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tmpl.Execute(w, map[string]interface{}{
			"Configs":        appConfig.Configs,
			"DefaultDBIndex": appConfig.DefaultDBIndex,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	case http.MethodPost:
		var req AppConfig
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 验证DefaultDBIndex是否在有效范围内
		if req.DefaultDBIndex < 0 || req.DefaultDBIndex >= len(req.Configs) {
			http.Error(w, "DefaultDBIndex out of bounds.", http.StatusBadRequest)
			return
		}

		// 测试默认数据库连接
		if req.DefaultDBIndex >= 0 && req.DefaultDBIndex < len(req.Configs) {
			if err := testDBConnection(req.Configs[req.DefaultDBIndex]); err != nil {
				http.Error(w, "无法连接到默认数据库: "+err.Error(), http.StatusBadRequest)
				return
			}
		}

		appConfig.Configs = req.Configs
		appConfig.DefaultDBIndex = req.DefaultDBIndex

		if err := saveConfig(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 关闭所有旧的数据库连接
		for _, db := range dbConnections {
			if db != nil {
				db.Close()
			}
		}
		// 清空连接池
		dbConnections = make(map[int]*sql.DB)

		// 重新初始化默认数据库连接
		if appConfig.DefaultDBIndex != -1 && appConfig.DefaultDBIndex < len(appConfig.Configs) {
			cfg := appConfig.Configs[appConfig.DefaultDBIndex]
			db, err := connectDB(cfg)
			if err != nil {
				log.Printf("Failed to connect to default database %d after config update: %v", appConfig.DefaultDBIndex, err)
				// 即使连接失败，也尝试保存已有的连接，避免程序崩溃
				dbConnections[appConfig.DefaultDBIndex] = nil // 标记为无效连接
			} else {
				dbConnections[appConfig.DefaultDBIndex] = db
			}
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

func userStatsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling user stats request")

	dbIndexStr := r.URL.Query().Get("db")
	var selectedIndex int = -1 // Use an int to track the actual index

	// Determine the effective database index
	if dbIndexStr == "" {
		// If no db parameter, use the default config
		if appConfig.DefaultDBIndex != -1 && appConfig.DefaultDBIndex < len(appConfig.Configs) {
			selectedIndex = appConfig.DefaultDBIndex
			dbIndexStr = strconv.Itoa(selectedIndex)
		} else if len(appConfig.Configs) > 0 {
			// If no valid default found, use the first config if available
			selectedIndex = 0
			dbIndexStr = "0"
		}
	} else {
		idx, err := strconv.Atoi(dbIndexStr)
		if err != nil || idx < 0 || idx >= len(appConfig.Configs) {
			http.Error(w, "Invalid database index", http.StatusBadRequest)
			return
		}
		selectedIndex = idx
	}

	// If a valid configuration is selected (or defaulted to)
	if selectedIndex == -1 {
		http.Error(w, "No database selected or configured.", http.StatusBadRequest)
		return
	}

	selectedConfig := appConfig.Configs[selectedIndex]

	log.Printf("using db: %d, host: %s\n", selectedIndex, selectedConfig.Host)
	db, ok := dbConnections[selectedIndex]
	if !ok || db == nil {
		http.Error(w, "Database connection not found or invalid.", http.StatusInternalServerError)
		return
	}

	typeParam := r.URL.Query().Get("type")

	if typeParam == "bucket" {
		bidFilter := r.URL.Query().Get("bid")
		bnameFilter := r.URL.Query().Get("bname")

		bucketStats, err := getUserStats(db, bidFilter, bnameFilter)
		if err != nil {
			http.Error(w, "Error getting bucket stats: "+err.Error(), http.StatusInternalServerError)
			return
		}

		data := struct {
			Users           []UserStats
			Configs         []Config
			SelectedDBIndex string
		}{
			Users:           bucketStats,
			Configs:         appConfig.Configs,
			SelectedDBIndex: dbIndexStr,
		}

		if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			tmpl, err := template.ParseFS(templates, "templates/bucket_stats_content.html")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			tmpl.Execute(w, data)
			return
		}

		tmpl, err := template.ParseFS(templates, "templates/base.html", "templates/user_stats.html", "templates/bucket_stats_content.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Default behavior for general user stats
		userStats, err := getUserStats(db, "", "") // Pass empty filters for general user stats
		if err != nil {
			http.Error(w, "Error getting user stats: "+err.Error(), http.StatusInternalServerError)
			return
		}

		data := struct {
			Users           []UserStats
			Configs         []Config
			SelectedDBIndex string
		}{
			Users:           userStats,
			Configs:         appConfig.Configs,
			SelectedDBIndex: dbIndexStr,
		}

		if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			tmpl, err := template.ParseFS(templates, "templates/user_stats_content.html")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			tmpl.Execute(w, data)
			return
		}

		tmpl, err := template.ParseFS(templates, "templates/base.html", "templates/user_stats.html", "templates/user_stats_content.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func filesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling files request")

	userIDStr := r.URL.Query().Get("user")
	part := r.URL.Query().Get("part")
	fidStr := r.URL.Query().Get("fid")
	fname := r.URL.Query().Get("fname")
	bucketIDStr := r.URL.Query().Get("bucket")
	log.Printf("req userID: %s, part: %s, fidStr: %s, fname: %s, bucketID: %s\n", userIDStr, part, fidStr, fname, bucketIDStr)

	if userIDStr == "" || part == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Parse parameters
	uid, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var fid uint64
	if fidStr != "" {
		fid, err = strconv.ParseUint(fidStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid file ID", http.StatusBadRequest)
			return
		}
	}

	var bucketID uint64
	if bucketIDStr != "" {
		bucketID, err = strconv.ParseUint(bucketIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid bucket ID", http.StatusBadRequest)
			return
		}
	}

	dbIndexStr := r.URL.Query().Get("db")
	var selectedIndex int = -1

	if dbIndexStr == "" {
		if appConfig.DefaultDBIndex != -1 && appConfig.DefaultDBIndex < len(appConfig.Configs) {
			selectedIndex = appConfig.DefaultDBIndex
		} else if len(appConfig.Configs) > 0 {
			selectedIndex = 0
		}
	} else {
		idx, err := strconv.Atoi(dbIndexStr)
		if err != nil || idx < 0 || idx >= len(appConfig.Configs) {
			http.Error(w, "Invalid database index", http.StatusBadRequest)
			return
		}
		selectedIndex = idx
	}

	if selectedIndex == -1 {
		http.Error(w, "No database configured or selected", http.StatusInternalServerError)
		return
	}

	db, ok := dbConnections[selectedIndex]
	if !ok || db == nil {
		http.Error(w, "Database connection not found or invalid.", http.StatusInternalServerError)
		return
	}

	// Query files
	files, err := getFiles(db, uid, part, fid, fname, bucketID) // Modified: added bucketID
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFS(templates, "templates/base.html", "templates/files.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		tmpl, err := template.ParseFS(templates, "templates/files.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := tmpl.ExecuteTemplate(w, "content", map[string]interface{}{
			"Files":    files,
			"UserID":   uid,
			"Part":     part,
			"FID":      fidStr,
			"FName":    fname,
			"BucketID": bucketID,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if err := tmpl.Execute(w, map[string]interface{}{
		"Files":    files,
		"Configs":  appConfig.Configs,
		"UserID":   uid,
		"Part":     part,
		"FID":      fidStr,
		"FName":    fname,
		"BucketID": bucketID,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
