-- 建表语句
CREATE TABLE IF NOT EXISTS users (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    username VARCHAR(50) UNIQUE NOT NULL,
    status TINYINT DEFAULT 1
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS buckets (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    bucket_id VARCHAR(32) UNIQUE NOT NULL,
    user_id INT UNSIGNED NOT NULL,
    `partition` VARCHAR(20) NOT NULL COMMENT '存储分区',
    storage BIGINT DEFAULT 0,
    INDEX idx_user (user_id)
) ENGINE=InnoDB;

-- 测试数据初始化
START TRANSACTION;
INSERT INTO users (created_at, updated_at, username, status) VALUES
(NOW(), NOW(), 'admin', 1),
(NOW(), NOW(), 'tester', 1);

INSERT INTO buckets (created_at, updated_at, bucket_id, user_id, `partition`) VALUES
(NOW(), NOW(), 'B001', 1, 'A3'),
(NOW(), NOW(), 'B002', 2, 'F0');

-- 自动创建分区子表
CREATE TABLE IF NOT EXISTS `bucket_files_template` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,
  `bucket_id` VARCHAR(36) NOT NULL,
  `file_name` VARCHAR(255) NOT NULL,
  `file_size` BIGINT NOT NULL,
  `status` TINYINT DEFAULT 1,
  PRIMARY KEY (`id`),
  INDEX `idx_bucket_id` (`bucket_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='分区表模板结构';

-- 添加模板表存在性检查
SET @template_exists = (SELECT COUNT(*) FROM information_schema.tables 
WHERE table_schema = 'testdb' AND table_name = 'bucket_files_template');

SELECT IF(@template_exists = 1, '模板表创建成功', '模板表创建失败') AS verification_result;

DELIMITER $$
CREATE PROCEDURE CreateAllPartitions()
BEGIN
  DECLARE i INT DEFAULT 0;
  WHILE i < 256 DO
    SET @tbl_name = CONCAT('bucket_files_', LPAD(HEX(i), 2, '0'));
    SET @sql = CONCAT('CREATE TABLE IF NOT EXISTS `', @tbl_name, '` LIKE bucket_files_template;');
    PREPARE stmt FROM @sql;
    EXECUTE stmt;
    SET i = i + 1;
  END WHILE;
END$$
DELIMITER ;
CALL CreateAllPartitions();

-- 验证分区表数量
SELECT COUNT(*) AS total_partitions FROM information_schema.tables 
WHERE table_schema = 'testdb' AND table_name LIKE 'bucket_files_%';
DROP PROCEDURE CreateAllPartitions;

COMMIT;

-- First bucket files
INSERT INTO bucket_files_A3 
(created_at, updated_at, bucket_id, file_name, file_size, status)
VALUES
(NOW() - INTERVAL 3 DAY, NOW() - INTERVAL 3 DAY, 'B001', 'data.csv', 307200, 1),
(NOW() - INTERVAL 1 DAY, NOW() - INTERVAL 1 DAY, 'B001', 'backup.zip', 15728640, 1);

-- Second bucket files
INSERT INTO bucket_files_F0 
(created_at, updated_at, bucket_id, file_name, file_size, status)
VALUES
(NOW() - INTERVAL 9 DAY, NOW() - INTERVAL 9 DAY, 'B002', 'profile_picture.png', 512000, 1),
(NOW() - INTERVAL 7 DAY, NOW() - INTERVAL 7 DAY, 'B002', 'report.docx', 1843200, 1),
(NOW(), NOW(), 'B002', 'archive.rar', 10485760, 1);
