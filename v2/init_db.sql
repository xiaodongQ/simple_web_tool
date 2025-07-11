-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    username VARCHAR(50) UNIQUE NOT NULL,
    status TINYINT DEFAULT 1 COMMENT '1-正常, 0-禁用'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Buckets表
CREATE TABLE IF NOT EXISTS buckets (
    bid BIGINT UNSIGNED PRIMARY KEY COMMENT '16位无符号整数',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    bname VARCHAR(255) NOT NULL COMMENT 'bucket名称',
    user INT UNSIGNED NOT NULL COMMENT '关联users.id',
    part CHAR(2) NOT NULL COMMENT '分区号(00~FF)',
    INDEX idx_user (user),
    INDEX idx_part (part),
    CONSTRAINT fk_bucket_user FOREIGN KEY (user) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 分区文件表模板
CREATE TABLE IF NOT EXISTS `bucket_files_template` (
    fid BIGINT UNSIGNED PRIMARY KEY COMMENT '16位无符号整数',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    fname VARCHAR(255) NOT NULL COMMENT '文件名',
    bid BIGINT UNSIGNED NOT NULL COMMENT '关联buckets.bid',
    fsize BIGINT UNSIGNED NOT NULL COMMENT '文件大小(字节)',
    status TINYINT DEFAULT 1 COMMENT '1-正常, 0-删除',
    INDEX idx_bid (bid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='分区表模板结构';

-- 创建所有分区表(00~FF)
DELIMITER $$
CREATE PROCEDURE CreateAllPartitionTables()
BEGIN
    DECLARE i INT DEFAULT 0;
    DECLARE hex_part CHAR(2);
    
    WHILE i < 256 DO
        SET hex_part = LPAD(LOWER(HEX(i)), 2, '0');
        SET @tbl_name = CONCAT('bucket_files_', hex_part);
        SET @sql = CONCAT('CREATE TABLE IF NOT EXISTS `', @tbl_name, '` LIKE bucket_files_template;');
        PREPARE stmt FROM @sql;
        EXECUTE stmt;
        SET i = i + 1;
    END WHILE;
END$$
DELIMITER ;

CALL CreateAllPartitionTables();
DROP PROCEDURE IF EXISTS CreateAllPartitionTables;

-- 测试数据初始化
-- 插入测试用户
INSERT INTO users (username, status) VALUES
('admin', 1),
('tester', 1),
('developer', 1);

-- 插入测试bucket
INSERT INTO buckets (bid, bname, user, part) VALUES
(1000000000000001, 'admin-backup', 1, 'a3'),
(1000000000000002, 'tester-data', 2, 'f0'),
(1000000000000003, 'dev-resources', 3, '7b'),
(1000000000000004, 'xxxxxxxxxxxx', 1, 'b1'),
(1000000000000005, 'yyyyyyyyyyyy', 3, 'c2'),
(1000000000000006, 'zzzzzzzzzzzz', 3, 'd3');

-- 插入测试文件数据
-- 分区a3的文件
INSERT INTO bucket_files_a3 (fid, fname, bid, fsize, status) VALUES
(2000000000000001, 'data.csv', 1000000000000001, 307200, 1),
(2000000000000002, 'backup.zip', 1000000000000001, 15728640, 1);

-- 分区f0的文件
INSERT INTO bucket_files_f0 (fid, fname, bid, fsize, status) VALUES
(2000000000000003, 'profile_picture.png', 1000000000000002, 512000, 1),
(2000000000000004, 'report.docx', 1000000000000002, 1843200, 1),
(2000000000000005, 'archive.rar', 1000000000000002, 10485760, 1);

-- 分区7b的文件
INSERT INTO bucket_files_7b (fid, fname, bid, fsize, status) VALUES
(2000000000000006, 'source_code.tar.gz', 1000000000000003, 5242880, 1),
(2000000000000007, 'database_dump.sql', 1000000000000003, 2097152, 1);
