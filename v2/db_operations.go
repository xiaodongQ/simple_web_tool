package main

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
)

type UserStats struct {
	ID         uint64
	Username   string
	Status     string
	TotalFiles uint64
	TotalSize  float64
	Partitions []PartitionStats
}

type PartitionStats struct {
	UserID   uint64
	Username string
	Part     string
	Count    uint64
	Size     float64

	// TODO: BID和BName应该提到UserStats中作为Slice
	BID   uint64
	BName string
}

type FileInfo struct {
	FID    uint64
	FName  string
	BID    uint64
	FSize  float64
	Status string
}

// 指定bucket查询时，对应的信息
type BucketCondition struct {
	BID   uint64
	BName string
	Part  string
}

func getUserStats(db *sql.DB, bidFilter, bnameFilter, usernameFilter string, limit int) ([]UserStats, error) {
	var users []UserStats

	if bidFilter != "" || bnameFilter != "" {
		// Direct query for specific bucket
		query := "SELECT u.id, u.username, u.status, b.bid, b.bname, b.part FROM users u JOIN buckets b ON u.id = b.user WHERE 1=1"
		args := []interface{}{}

		if usernameFilter != "" {
			query += " AND u.username = ?"
			args = append(args, usernameFilter)
		}

		if bidFilter != "" {
			query += " AND b.bid = ?"
			args = append(args, bidFilter)
		}
		if bnameFilter != "" {
			query += " AND b.bname = ?"
			args = append(args, bnameFilter)
		}

		// 每个用户只查询limit个分区
		if limit > 0 {
			query += " LIMIT ?"
			args = append(args, limit)
		}

		log.Printf("Executing query: %s with args: %v", query, args)
		rows, err := db.Query(query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var (
			mu sync.Mutex
			wg sync.WaitGroup
		)

		userChan := make(chan UserStats, 100) // Buffered channel to send users

		for rows.Next() {
			var user UserStats
			var bucketCond BucketCondition
			if err := rows.Scan(&user.ID, &user.Username, &user.Status, &bucketCond.BID, &bucketCond.BName, &bucketCond.Part); err != nil {
				return nil, err
			}

			wg.Add(1)
			go func(u UserStats, bc BucketCondition) {
				defer wg.Done()
				partitions, err := getUserPartitions(db, bc, u.ID, u.Username, limit)
				if err != nil {
					log.Printf("Error getting partitions for user %s: %v", u.Username, err)
					return
				}

				u.Partitions = partitions
				for _, p := range partitions {
					u.TotalFiles += p.Count
					u.TotalSize += p.Size
				}
				u.TotalSize = u.TotalSize / 1024.0
				userChan <- u
			}(user, bucketCond)
		}

		wg.Wait()
		close(userChan)

		for u := range userChan {
			mu.Lock()
			users = append(users, u)
			mu.Unlock()
		}
	} else {
		// Original logic for all users
		query := "SELECT id, username, status FROM users WHERE 1=1"
		args := []interface{}{}
		if usernameFilter != "" {
			query += " AND username = ?"
			args = append(args, usernameFilter)
		}

		log.Printf("Executing query: %s with args: %v", query, args)
		rows, err := db.Query(query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var (
			mu sync.Mutex
			wg sync.WaitGroup
		)

		userChan := make(chan UserStats, 100) // Buffered channel to send users

		for rows.Next() {
			var user UserStats
			if err := rows.Scan(&user.ID, &user.Username, &user.Status); err != nil {
				return nil, err
			}
			wg.Add(1)
			go func(u UserStats) {
				defer wg.Done()
				partitions, err := getUserPartitions(db, BucketCondition{}, u.ID, u.Username, limit)
				if err != nil {
					log.Printf("Error getting partitions for user %s: %v", u.Username, err)
					return
				}
				u.Partitions = partitions
				for _, p := range partitions {
					u.TotalFiles += p.Count
					u.TotalSize += p.Size
				}
				userChan <- u
			}(user)
		}

		wg.Wait()
		close(userChan)

		for u := range userChan {
			mu.Lock()
			users = append(users, u)
			mu.Unlock()
		}
	}

	return users, nil
}

func getUserPartitions(db *sql.DB, bucketCond BucketCondition, userID uint64, username string, limit int) ([]PartitionStats, error) {
	var partitions []PartitionStats

	if bucketCond.BID > 0 && bucketCond.Part != "" {
		// Get single partition stats for specific bucket
		stats, err := getPartitionStats(db, bucketCond.BID, bucketCond.Part)
		if err != nil {
			return nil, err
		}

		// 组装信息
		var partition PartitionStats
		partition.Count = stats.Count
		partition.Size = stats.Size
		// 其他信息直接填充
		partition.BID = bucketCond.BID
		partition.BName = bucketCond.BName
		partition.Part = bucketCond.Part
		partition.UserID = userID
		partition.Username = username
		partitions = append(partitions, partition)
	} else {
		// 获取该用户下有bucket的分区
		query := "SELECT part FROM buckets WHERE user = ? GROUP BY part"
		args := []interface{}{userID}
		if limit > 0 {
			query += " LIMIT ?"
			args = append(args, limit)
		}

		log.Printf("Executing partition query: %s with args: %v", query, args)
		rows, err := db.Query(query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var (
			mu sync.Mutex
			wg sync.WaitGroup
		)

		partitionChan := make(chan PartitionStats, 100) // Buffered channel to send partitions

		parts := []string{}
		for rows.Next() {
			var part string
			if err := rows.Scan(&part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		}

		for _, part := range parts {
			wg.Add(1)
			go func(p string) {
				defer wg.Done()

				stats, err := getPartitionStats(db, userID, p)
				if err != nil {
					log.Printf("Error getting partition stats for user %d, part %s: %v", userID, p, err)
					return
				}

				if stats.Count == 0 {
					return
				}

				var bid uint64
				var bname string
				err = db.QueryRow("SELECT bid, bname FROM buckets WHERE user = ? AND part = ? LIMIT 1", userID, p).Scan(&bid, &bname)
				if err != nil && err != sql.ErrNoRows {
					log.Printf("Error getting bucket info for user %d, part %s: %v", userID, p, err)
				}

				partitionChan <- PartitionStats{
					Count:    stats.Count,
					Size:     stats.Size,
					UserID:   userID,
					Username: username,
					Part:     p,
					BID:      bid,
					BName:    bname,
				}
			}(part)
		}

		wg.Wait()
		close(partitionChan)

		for p := range partitionChan {
			mu.Lock()
			partitions = append(partitions, p)
			mu.Unlock()
		}
	}

	return partitions, nil
}

// 用户在指定分区的文件统计
func getPartitionStats(db *sql.DB, userID uint64, part string) (*PartitionStats, error) {
	// Query file count and total size for this partition
	query := fmt.Sprintf(
		"SELECT COUNT(*), COALESCE(SUM(fsize), 0) FROM bucket_files_%s WHERE bid IN "+
			"(SELECT bid FROM buckets WHERE user = ? AND part = ?)", part)

	var stats PartitionStats
	stats.Part = part
	stats.UserID = userID

	row := db.QueryRow(query, userID, part)
	if err := row.Scan(&stats.Count, &stats.Size); err != nil {
		if err == sql.ErrNoRows {
			// 没有匹配记录，返回零值
			return &PartitionStats{
				Part:  part,
				Count: 0,
				Size:  0,
			}, nil
		}
		return nil, fmt.Errorf("failed to scan partition stats: %w", err)
	}
	stats.Size = stats.Size / 1024.0 // Convert bytes to KB

	return &stats, nil
}

func getFiles(db *sql.DB, userID uint64, part string, fid uint64, fname string, bucketID uint64) ([]FileInfo, error) {
	// Build query for files in this partition
	query := fmt.Sprintf(
		"SELECT fid, fname, bid, fsize, status FROM bucket_files_%s "+
			"WHERE bid IN (SELECT bid FROM buckets WHERE user = ? AND part = ?)", part)

	args := []interface{}{userID, part}

	// Add filters if provided
	if fid > 0 {
		query += " AND fid = ?"
		args = append(args, fid)
	}
	if fname != "" {
		query += " AND fname LIKE ?"
		args = append(args, "%"+fname+"%")
	}
	if bucketID > 0 {
		query += " AND bid = ?"
		args = append(args, bucketID)
	}

	query += " LIMIT 20"
	log.Printf("Query: %s, Args: %v", query, args)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []FileInfo
	for rows.Next() {
		var file FileInfo
		if err := rows.Scan(&file.FID, &file.FName, &file.BID, &file.FSize, &file.Status); err != nil {
			return nil, err
		}
		file.FSize = file.FSize / 1024.0 // Convert bytes to KB
		files = append(files, file)
	}
	log.Printf("Files: %v", files)

	return files, nil
}
