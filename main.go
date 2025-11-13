package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Qsnh/backup-pro/backup"
	"github.com/Qsnh/backup-pro/config"
	"github.com/Qsnh/backup-pro/logger"
	"github.com/Qsnh/backup-pro/notification"
	"github.com/Qsnh/backup-pro/scheduler"
)

var (
	configPath = flag.String("config", "config.yaml", "配置文件路径")
	runOnce    = flag.Bool("once", false, "立即执行一次备份后退出（忽略定时任务配置）")
	version    = "1.0.0"
)

func main() {
	flag.Parse()

	fmt.Printf("BackupPro - S3 备份工具 v%s\n\n", version)

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	if err := logger.InitLogger(cfg.Logging); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}

	log := logger.GetLogger()
	log.Info("BackupPro 启动成功")
	log.Infof("配置文件: %s", *configPath)

	// 创建 S3 上传器
	uploader, err := backup.NewS3Uploader(cfg.S3)
	if err != nil {
		log.Fatalf("创建 S3 上传器失败: %v", err)
	}

	// 创建通知器
	notifier := notification.NewNotifier(cfg.Notification)

	// 定义备份任务
	backupJob := func() error {
		return runBackup(cfg, uploader, notifier)
	}

	// 如果是单次运行模式
	if *runOnce {
		log.Info("单次运行模式，执行备份...")
		if err := backupJob(); err != nil {
			log.Fatalf("备份失败: %v", err)
		}
		log.Info("备份完成，程序退出")
		return
	}

	// 创建上下文和信号处理
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听系统信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Info("收到退出信号，准备停止...")
		cancel()
	}()

	// 创建并启动调度器
	sched := scheduler.NewScheduler(cfg.Schedule, backupJob)

	log.Info("BackupPro 运行中... 按 Ctrl+C 退出")

	if cfg.Schedule.Enabled {
		log.Infof("下次执行时间: %s", sched.GetNextRunTime())
	}

	// 启动调度器（阻塞直到收到停止信号）
	if err := sched.Start(ctx); err != nil {
		log.Fatalf("调度器启动失败: %v", err)
	}

	log.Info("BackupPro 已停止")
}

// runBackup 执行备份任务
func runBackup(cfg *config.Config, uploader *backup.S3Uploader, notifier *notification.Notifier) error {
	log := logger.GetLogger()
	ctx := context.Background()

	log.Info("===== 开始备份任务 =====")

	// 1. 打包压缩
	log.Infof("备份源目录: %s", cfg.Backup.SourceDir)
	archiveResult, err := backup.CreateTarGz(
		cfg.Backup.SourceDir,
		cfg.Backup.TempDir,
		cfg.Backup.NamePrefix,
	)
	if err != nil {
		notifier.NotifyFailure(fmt.Sprintf("打包失败: %v", err))
		return fmt.Errorf("打包失败: %w", err)
	}

	// 确保清理临时文件
	defer func() {
		if err := backup.CleanupTempFile(archiveResult.FilePath); err != nil {
			log.Warnf("清理临时文件失败: %v", err)
		}
	}()

	// 2. 上传到 S3
	if err := uploader.Upload(ctx, archiveResult.FilePath, archiveResult.FileName); err != nil {
		notifier.NotifyFailure(fmt.Sprintf("上传失败: %v", err))
		return fmt.Errorf("上传失败: %w", err)
	}

	// 3. 清理旧备份
	log.Infof("清理旧备份，保留最新 %d 份", cfg.Backup.RetentionCount)
	if err := uploader.CleanupOldBackups(ctx, cfg.Backup.RetentionCount); err != nil {
		log.Warnf("清理旧备份失败: %v", err)
		// 不中断流程
	}

	// 4. 获取备份信息
	backupInfo, err := uploader.GetBackupInfo(ctx)
	if err != nil {
		log.Warnf("获取备份信息失败: %v", err)
	} else {
		log.Info(backupInfo)
	}

	// 5. 发送成功通知
	notifier.NotifySuccess(archiveResult.FileName, archiveResult.Size, archiveResult.Checksum)

	log.Info("===== 备份任务完成 =====")
	return nil
}
