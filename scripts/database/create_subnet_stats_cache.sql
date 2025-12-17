-- Cache table for subnet statistics to speed up /api/v1/stats/subnetsnew
CREATE TABLE IF NOT EXISTS subnet_stats_cache (
  subnet_id                VARCHAR(255) PRIMARY KEY,
  subnet_name              VARCHAR(255),
  subnet_icon              VARCHAR(512),
  creator_wallet           VARCHAR(255),
  total_tasks              INT,
  completed_tasks          INT,
  unique_users             INT,
  total_points_distributed INT,
  today_points_distributed INT,
  today_active_users       INT,
  updated_at               DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


