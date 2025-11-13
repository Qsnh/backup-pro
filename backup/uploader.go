package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	cfg "github.com/Qsnh/backup-pro/config"
	"github.com/Qsnh/backup-pro/logger"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Uploader S3 上传器
type S3Uploader struct {
	client *s3.Client
	bucket string
	prefix string
	cfg    cfg.S3Config
}

// NewS3Uploader 创建 S3 上传器
func NewS3Uploader(s3Cfg cfg.S3Config) (*S3Uploader, error) {
	log := logger.GetLogger()

	ctx := context.TODO()

	// 配置 AWS SDK
	var awsCfg aws.Config
	var err error

	if s3Cfg.Endpoint != "" {
		// 自定义端点（如 MinIO）
		customResolver := aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               s3Cfg.Endpoint,
					HostnameImmutable: true,
				}, nil
			})

		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(s3Cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				s3Cfg.AccessKeyID,
				s3Cfg.SecretAccessKey,
				"",
			)),
			config.WithEndpointResolverWithOptions(customResolver),
		)
	} else {
		// 标准 AWS S3
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(s3Cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				s3Cfg.AccessKeyID,
				s3Cfg.SecretAccessKey,
				"",
			)),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("加载 AWS 配置失败: %w", err)
	}

	// 创建 S3 客户端
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = s3Cfg.UsePathStyle
	})

	log.Infof("S3 客户端初始化成功: bucket=%s, region=%s", s3Cfg.Bucket, s3Cfg.Region)

	return &S3Uploader{
		client: client,
		bucket: s3Cfg.Bucket,
		prefix: s3Cfg.Prefix,
		cfg:    s3Cfg,
	}, nil
}

// Upload 上传文件到 S3
func (u *S3Uploader) Upload(ctx context.Context, filePath, fileName string) error {
	log := logger.GetLogger()

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 构建 S3 key
	key := fileName
	if u.prefix != "" {
		key = filepath.Join(u.prefix, fileName)
	}

	log.Infof("开始上传到 S3: s3://%s/%s", u.bucket, key)

	// 上传文件
	_, err = u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
		Body:   file,
	})

	if err != nil {
		return fmt.Errorf("上传到 S3 失败: %w", err)
	}

	log.Infof("上传成功: s3://%s/%s", u.bucket, key)

	return nil
}

// ListBackups 列出 S3 中的备份文件
func (u *S3Uploader) ListBackups(ctx context.Context) ([]types.Object, error) {
	log := logger.GetLogger()

	prefix := u.prefix
	if prefix != "" && prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}

	log.Debugf("列出 S3 备份文件: prefix=%s", prefix)

	// 列出对象
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(u.bucket),
	}

	if prefix != "" && prefix != "/" {
		input.Prefix = aws.String(prefix)
	}

	result, err := u.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("列出 S3 对象失败: %w", err)
	}

	// 按时间排序（最新的在前）
	objects := result.Contents
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].LastModified.After(*objects[j].LastModified)
	})

	log.Infof("找到 %d 个备份文件", len(objects))

	return objects, nil
}

// CleanupOldBackups 清理旧备份，保留指定数量
func (u *S3Uploader) CleanupOldBackups(ctx context.Context, retentionCount int) error {
	log := logger.GetLogger()

	// 列出所有备份
	backups, err := u.ListBackups(ctx)
	if err != nil {
		return err
	}

	// 如果备份数量未超过保留数量，无需清理
	if len(backups) <= retentionCount {
		log.Infof("当前备份数 %d <= 保留数 %d，无需清理", len(backups), retentionCount)
		return nil
	}

	// 删除超出的备份
	toDelete := backups[retentionCount:]
	log.Infof("需要删除 %d 个旧备份", len(toDelete))

	for _, obj := range toDelete {
		log.Infof("删除旧备份: %s (最后修改: %s)", *obj.Key, obj.LastModified.Format(time.RFC3339))

		_, err := u.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(u.bucket),
			Key:    obj.Key,
		})

		if err != nil {
			log.Errorf("删除对象失败 %s: %v", *obj.Key, err)
			// 继续删除其他文件
			continue
		}

		log.Infof("已删除: %s", *obj.Key)
	}

	return nil
}

// GetBackupInfo 获取备份信息摘要
func (u *S3Uploader) GetBackupInfo(ctx context.Context) (string, error) {
	backups, err := u.ListBackups(ctx)
	if err != nil {
		return "", err
	}

	if len(backups) == 0 {
		return "S3 中暂无备份", nil
	}

	latest := backups[0]
	info := fmt.Sprintf("S3 备份信息: 共 %d 个文件, 最新备份: %s (大小: %d bytes, 时间: %s)",
		len(backups),
		*latest.Key,
		latest.Size,
		latest.LastModified.Format(time.RFC3339))

	return info, nil
}
