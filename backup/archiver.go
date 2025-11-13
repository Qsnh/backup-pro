package backup

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Qsnh/backup-pro/logger"
)

// ArchiveResult 打包结果
type ArchiveResult struct {
	FilePath string // 打包文件路径
	FileName string // 打包文件名
	Checksum string // SHA256 校验和
	Size     int64  // 文件大小（字节）
}

// CreateTarGz 创建 tar.gz 压缩包
func CreateTarGz(sourceDir, tempDir, namePrefix string) (*ArchiveResult, error) {
	log := logger.GetLogger()

	// 确保临时目录存在
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 生成文件名：prefix_YYYYMMDD_HHMMSS.tar.gz
	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("%s_%s.tar.gz", namePrefix, timestamp)
	filePath := filepath.Join(tempDir, fileName)

	log.Infof("开始打包目录: %s", sourceDir)
	log.Infof("打包文件: %s", filePath)

	// 创建打包文件
	outFile, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("创建打包文件失败: %w", err)
	}
	defer outFile.Close()

	// 创建 gzip writer
	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	// 创建 tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// 遍历目录并添加到 tar
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warnf("访问路径失败 %s: %v", path, err)
			return nil // 继续处理其他文件
		}

		// 获取相对路径
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// 跳过根目录
		if relPath == "." {
			return nil
		}

		// 创建 tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			log.Warnf("创建文件头失败 %s: %v", path, err)
			return nil
		}
		header.Name = relPath

		// 写入 header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// 如果是文件，写入内容
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				log.Warnf("打开文件失败 %s: %v", path, err)
				return nil
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				log.Warnf("写入文件内容失败 %s: %v", path, err)
				return nil
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("打包失败: %w", err)
	}

	// 关闭 writer 以确保数据写入
	tarWriter.Close()
	gzWriter.Close()
	outFile.Close()

	log.Info("打包完成，计算校验和...")

	// 计算校验和
	checksum, err := calculateChecksum(filePath)
	if err != nil {
		return nil, fmt.Errorf("计算校验和失败: %w", err)
	}

	// 获取文件大小
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	result := &ArchiveResult{
		FilePath: filePath,
		FileName: fileName,
		Checksum: checksum,
		Size:     fileInfo.Size(),
	}

	log.Infof("打包完成: 文件=%s, 大小=%d bytes, 校验和=%s", fileName, result.Size, checksum)

	return result, nil
}

// calculateChecksum 计算文件的 SHA256 校验和
func calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// VerifyChecksum 验证文件校验和
func VerifyChecksum(filePath, expectedChecksum string) (bool, error) {
	actualChecksum, err := calculateChecksum(filePath)
	if err != nil {
		return false, err
	}
	return actualChecksum == expectedChecksum, nil
}

// CleanupTempFile 清理临时文件
func CleanupTempFile(filePath string) error {
	log := logger.GetLogger()
	if err := os.Remove(filePath); err != nil {
		log.Warnf("清理临时文件失败 %s: %v", filePath, err)
		return err
	}
	log.Infof("已清理临时文件: %s", filePath)
	return nil
}
