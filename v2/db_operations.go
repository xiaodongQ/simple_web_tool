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
	TotalSize  float64
	Partitions []PartitionStats
}

type PartitionStats struct {
	UserID   uint64
	Username string
	BID      uint64
	BName    string
	Part     string
	Count    uint64
	Size     float64
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

		for rows.Next() {
			var user UserStats
			var bucketCond BucketCondition
			if err := rows.Scan(&user.ID, &user.Username, &user.Status, &bucketCond.BID, &bucketCond.BName, &bucketCond.Part); err != nil {
				return nil, err
			}

			partitions, err := getUserPartitions(db, bucketCond, user.ID, user.Username, limit)
			if err != nil {
				return nil, err
			}

			user.Partitions = partitions
			for _, p := range partitions {
				user.TotalFiles += p.Count
				user.TotalSize += p.Size
			}

			user.TotalSize = user.TotalSize / 1024.0
			users = append(users, user)
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

		for rows.Next() {
			var user UserStats
			if err := rows.Scan(&user.ID, &user.Username, &user.Status); err != nil {
				return nil, err
			}

			// 每个用户都查询limit个分区
			partitions, err := getUserPartitions(db, BucketCondition{}, user.ID, user.Username, limit)
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

		// 用户在存在bucket的各分区的文件统计
		for rows.Next() {
			var part string
			if err := rows.Scan(&part); err != nil {
				return nil, err
			}

			// 统计该分区下的文件数量和大小
			stats, err := getPartitionStats(db, userID, part)
			if err != nil {
				return nil, err
			}

			if stats.Count == 0 {
				// TODO 暂时注释
				// 该分区下没有文件，跳过
				continue
			}

			// 属于该用户和该分区下的bucket统计信息
			// Get bucket info for this partition
			bucketRows, err := db.Query("SELECT b.bid, b.bname, u.username FROM buckets b JOIN users u ON b.user = u.id WHERE b.user = ? AND b.part = ?", userID, part)
			if err != nil {
				return nil, err
			}
			defer bucketRows.Close()

			for bucketRows.Next() {
				var partition PartitionStats
				if err := bucketRows.Scan(&partition.BID, &partition.BName, &partition.Username); err != nil {
					return nil, err
				}

				partition.Part = part
				partition.Count = stats.Count
				partition.Size = stats.Size
				partition.UserID = userID
				partition.Username = username
				partitions = append(partitions, partition)
			}
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
