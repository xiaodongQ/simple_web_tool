package main

import (
	"database/sql"
	"fmt"
	"log"
)

type UserStats struct {
	ID         uint64
	Username   string
	Status     string
	TotalFiles uint64
	TotalSize  uint64
	Partitions []PartitionStats
}

type PartitionStats struct {
	Part  string
	Count uint64
	Size  uint64
}

type FileInfo struct {
	FID    uint64
	FName  string
	BID    uint64
	FSize  uint64
	Status string
}

func getUserStats(db *sql.DB) ([]UserStats, error) {
	// Query users
	rows, err := db.Query("SELECT id, username, status FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserStats
	for rows.Next() {
		var user UserStats
		if err := rows.Scan(&user.ID, &user.Username, &user.Status); err != nil {
			return nil, err
		}

		// Get bucket partitions for this user
		partitions, err := getUserPartitions(db, user.ID)
		if err != nil {
			return nil, err
		}

		user.Partitions = partitions
		for _, p := range partitions {
			user.TotalFiles += p.Count
			user.TotalSize += p.Size
		}

		users = append(users, user)
	}

	return users, nil
}

func getUserPartitions(db *sql.DB, userID uint64) ([]PartitionStats, error) {
	// Query distinct partitions for this user
	rows, err := db.Query(
		"SELECT DISTINCT part FROM buckets WHERE user = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var partitions []PartitionStats
	for rows.Next() {
		var part string
		if err := rows.Scan(&part); err != nil {
			return nil, err
		}

		stats, err := getPartitionStats(db, userID, part)
		if err != nil {
			return nil, err
		}

		// 过滤掉没有文件的分区
		if stats.Count == 0 && stats.Size == 0 {
			// 暂时去掉
			// continue
		}

		partitions = append(partitions, *stats)
	}

	return partitions, nil
}

func getPartitionStats(db *sql.DB, userID uint64, part string) (*PartitionStats, error) {
	// Query file count and total size for this partition
	query := fmt.Sprintf(
		"SELECT COUNT(*), COALESCE(SUM(fsize), 0) FROM bucket_files_%s WHERE bid IN "+
			"(SELECT bid FROM buckets WHERE user = ? AND part = ?)", part)

	var stats PartitionStats
	stats.Part = part

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

	return &stats, nil
}

func getFiles(db *sql.DB, userID uint64, part string, fid uint64, fname string) ([]FileInfo, error) {
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

	query += " LIMIT 10"
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
		files = append(files, file)
	}
	log.Printf("Files: %v", files)

	return files, nil
}
