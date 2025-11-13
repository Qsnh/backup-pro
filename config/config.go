package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 主配置结构
type Config struct {
	Backup       BackupConfig       `yaml:"backup"`
	S3           S3Config           `yaml:"s3"`
	Schedule     ScheduleConfig     `yaml:"schedule"`
	Notification NotificationConfig `yaml:"notification"`
	Logging      LoggingConfig      `yaml:"logging"`
}

// BackupConfig 备份配置
type BackupConfig struct {
	SourceDir      string `yaml:"source_dir"`      // 要备份的目录
	TempDir        string `yaml:"temp_dir"`        // 临时目录，用于存放打包文件
	RetentionDays  int    `yaml:"retention_days"`  // 保留天数（可选）
	RetentionCount int    `yaml:"retention_count"` // 保留份数
	NamePrefix     string `yaml:"name_prefix"`     // 备份文件名前缀
}

// S3Config S3 配置
type S3Config struct {
	Endpoint        string `yaml:"endpoint"`          // S3 端点（可选，用于兼容 MinIO 等）
	Region          string `yaml:"region"`            // AWS 区域
	Bucket          string `yaml:"bucket"`            // S3 桶名
	AccessKeyID     string `yaml:"access_key_id"`     // 访问密钥 ID
	SecretAccessKey string `yaml:"secret_access_key"` // 访问密钥
	Prefix          string `yaml:"prefix"`            // S3 对象前缀（路径）
	UsePathStyle    bool   `yaml:"use_path_style"`    // 是否使用路径样式（MinIO 需要）
}

// ScheduleConfig 定时任务配置
type ScheduleConfig struct {
	Enabled    bool   `yaml:"enabled"`      // 是否启用定时任务
	CronExpr   string `yaml:"cron_expr"`    // Cron 表达式
	RunAtStart bool   `yaml:"run_at_start"` // 启动时立即执行一次
}

// NotificationConfig 通知配置
type NotificationConfig struct {
	Enabled bool          `yaml:"enabled"` // 是否启用通知
	Email   EmailConfig   `yaml:"email"`   // 邮件通知配置
	Webhook WebhookConfig `yaml:"webhook"` // Webhook 通知配置
}

// EmailConfig 邮件配置
type EmailConfig struct {
	Enabled  bool     `yaml:"enabled"`
	SMTPHost string   `yaml:"smtp_host"`
	SMTPPort int      `yaml:"smtp_port"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	From     string   `yaml:"from"`
	To       []string `yaml:"to"`
}

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
	Method  string `yaml:"method"` // POST, GET
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Output string `yaml:"output"` // stdout, file
	File   string `yaml:"file"`   // 日志文件路径（当 output 为 file 时）
}

// LoadConfig 从 YAML 文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	if cfg.Backup.TempDir == "" {
		cfg.Backup.TempDir = "./tmp"
	}
	if cfg.Backup.NamePrefix == "" {
		cfg.Backup.NamePrefix = "backup"
	}
	if cfg.S3.Region == "" {
		cfg.S3.Region = "us-east-1"
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Output == "" {
		cfg.Logging.Output = "stdout"
	}
	if cfg.Notification.Webhook.Method == "" {
		cfg.Notification.Webhook.Method = "POST"
	}

	// 验证必填字段
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &cfg, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Backup.SourceDir == "" {
		return fmt.Errorf("backup.source_dir 不能为空")
	}
	if c.S3.Bucket == "" {
		return fmt.Errorf("s3.bucket 不能为空")
	}
	if c.S3.AccessKeyID == "" {
		return fmt.Errorf("s3.access_key_id 不能为空")
	}
	if c.S3.SecretAccessKey == "" {
		return fmt.Errorf("s3.secret_access_key 不能为空")
	}
	if c.Backup.RetentionCount <= 0 {
		return fmt.Errorf("backup.retention_count 必须大于 0")
	}
	if c.Schedule.Enabled && c.Schedule.CronExpr == "" {
		return fmt.Errorf("schedule.cron_expr 不能为空（当启用定时任务时）")
	}

	return nil
}
