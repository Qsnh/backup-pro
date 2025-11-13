package scheduler

import (
	"context"
	"fmt"

	"github.com/Qsnh/backup-pro/config"
	"github.com/Qsnh/backup-pro/logger"
	"github.com/robfig/cron/v3"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	cron    *cron.Cron
	cfg     config.ScheduleConfig
	jobFunc func() error
}

// NewScheduler 创建调度器
func NewScheduler(cfg config.ScheduleConfig, jobFunc func() error) *Scheduler {
	// 创建 cron 实例，使用秒级精度
	c := cron.New(cron.WithSeconds())

	return &Scheduler{
		cron:    c,
		cfg:     cfg,
		jobFunc: jobFunc,
	}
}

// Start 启动调度器
func (s *Scheduler) Start(ctx context.Context) error {
	log := logger.GetLogger()

	if !s.cfg.Enabled {
		log.Info("定时任务未启用")
		return nil
	}

	// 添加定时任务
	_, err := s.cron.AddFunc(s.cfg.CronExpr, func() {
		log.Info("定时任务触发，开始执行备份...")
		if err := s.jobFunc(); err != nil {
			log.Errorf("定时任务执行失败: %v", err)
		}
	})

	if err != nil {
		return fmt.Errorf("添加定时任务失败: %w", err)
	}

	// 启动 cron
	s.cron.Start()

	log.Infof("定时任务已启动，Cron 表达式: %s", s.cfg.CronExpr)

	// 如果配置为启动时执行，立即执行一次
	if s.cfg.RunAtStart {
		log.Info("启动时立即执行备份...")
		if err := s.jobFunc(); err != nil {
			log.Errorf("启动时备份执行失败: %v", err)
		}
	}

	// 等待上下文取消
	<-ctx.Done()
	log.Info("正在停止定时任务...")
	s.Stop()

	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	if s.cron != nil {
		ctx := s.cron.Stop()
		<-ctx.Done()
		logger.GetLogger().Info("定时任务已停止")
	}
}

// GetNextRunTime 获取下次执行时间
func (s *Scheduler) GetNextRunTime() string {
	if !s.cfg.Enabled || s.cron == nil {
		return "定时任务未启用"
	}

	entries := s.cron.Entries()
	if len(entries) == 0 {
		return "无计划任务"
	}

	return entries[0].Next.Format("2006-01-02 15:04:05")
}
