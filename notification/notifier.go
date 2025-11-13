package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/Qsnh/backup-pro/config"
	"github.com/Qsnh/backup-pro/logger"
)

// Notifier 通知器
type Notifier struct {
	cfg config.NotificationConfig
}

// NotificationMessage 通知消息
type NotificationMessage struct {
	Success  bool   `json:"success"`
	Title    string `json:"title"`
	Message  string `json:"message"`
	FileName string `json:"file_name,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
	Checksum string `json:"checksum,omitempty"`
	Error    string `json:"error,omitempty"`
	Time     string `json:"time"`
}

// NewNotifier 创建通知器
func NewNotifier(cfg config.NotificationConfig) *Notifier {
	return &Notifier{cfg: cfg}
}

// NotifySuccess 发送成功通知
func (n *Notifier) NotifySuccess(fileName string, fileSize int64, checksum string) {
	if !n.cfg.Enabled {
		return
	}

	msg := NotificationMessage{
		Success:  true,
		Title:    "备份成功",
		Message:  fmt.Sprintf("备份已成功上传到 S3: %s", fileName),
		FileName: fileName,
		FileSize: fileSize,
		Checksum: checksum,
		Time:     time.Now().Format(time.RFC3339),
	}

	n.send(msg)
}

// NotifyFailure 发送失败通知
func (n *Notifier) NotifyFailure(errMsg string) {
	if !n.cfg.Enabled {
		return
	}

	msg := NotificationMessage{
		Success: false,
		Title:   "备份失败",
		Message: "备份过程中发生错误",
		Error:   errMsg,
		Time:    time.Now().Format(time.RFC3339),
	}

	n.send(msg)
}

// send 发送通知
func (n *Notifier) send(msg NotificationMessage) {
	log := logger.GetLogger()

	// 发送邮件
	if n.cfg.Email.Enabled {
		if err := n.sendEmail(msg); err != nil {
			log.Errorf("发送邮件通知失败: %v", err)
		}
	}

	// 发送 Webhook
	if n.cfg.Webhook.Enabled {
		if err := n.sendWebhook(msg); err != nil {
			log.Errorf("发送 Webhook 通知失败: %v", err)
		}
	}
}

// sendEmail 发送邮件
func (n *Notifier) sendEmail(msg NotificationMessage) error {
	log := logger.GetLogger()

	if n.cfg.Email.SMTPHost == "" || len(n.cfg.Email.To) == 0 {
		return fmt.Errorf("邮件配置不完整")
	}

	// 构建邮件内容
	subject := msg.Title
	body := n.formatEmailBody(msg)

	// 构建邮件
	message := fmt.Sprintf("From: %s\r\n", n.cfg.Email.From)
	message += fmt.Sprintf("To: %s\r\n", strings.Join(n.cfg.Email.To, ","))
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "Content-Type: text/plain; charset=UTF-8\r\n"
	message += "\r\n"
	message += body

	// 连接 SMTP 服务器
	addr := fmt.Sprintf("%s:%d", n.cfg.Email.SMTPHost, n.cfg.Email.SMTPPort)
	auth := smtp.PlainAuth("", n.cfg.Email.Username, n.cfg.Email.Password, n.cfg.Email.SMTPHost)

	log.Infof("发送邮件通知到: %s", strings.Join(n.cfg.Email.To, ","))

	// 发送邮件
	err := smtp.SendMail(addr, auth, n.cfg.Email.From, n.cfg.Email.To, []byte(message))
	if err != nil {
		return fmt.Errorf("发送邮件失败: %w", err)
	}

	log.Info("邮件通知发送成功")
	return nil
}

// formatEmailBody 格式化邮件正文
func (n *Notifier) formatEmailBody(msg NotificationMessage) string {
	var body strings.Builder

	body.WriteString(fmt.Sprintf("状态: %s\n", msg.Title))
	body.WriteString(fmt.Sprintf("时间: %s\n", msg.Time))
	body.WriteString(fmt.Sprintf("消息: %s\n", msg.Message))

	if msg.FileName != "" {
		body.WriteString(fmt.Sprintf("文件名: %s\n", msg.FileName))
	}

	if msg.FileSize > 0 {
		body.WriteString(fmt.Sprintf("文件大小: %d bytes (%.2f MB)\n", msg.FileSize, float64(msg.FileSize)/1024/1024))
	}

	if msg.Checksum != "" {
		body.WriteString(fmt.Sprintf("校验和: %s\n", msg.Checksum))
	}

	if msg.Error != "" {
		body.WriteString(fmt.Sprintf("错误: %s\n", msg.Error))
	}

	return body.String()
}

// sendWebhook 发送 Webhook
func (n *Notifier) sendWebhook(msg NotificationMessage) error {
	log := logger.GetLogger()

	if n.cfg.Webhook.URL == "" {
		return fmt.Errorf("Webhook URL 未配置")
	}

	// 序列化消息
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	log.Infof("发送 Webhook 通知到: %s", n.cfg.Webhook.URL)

	// 发送 HTTP 请求
	method := strings.ToUpper(n.cfg.Webhook.Method)
	if method != "POST" && method != "GET" {
		method = "POST"
	}

	var req *http.Request
	if method == "POST" {
		req, err = http.NewRequest(method, n.cfg.Webhook.URL, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("创建请求失败: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, n.cfg.Webhook.URL, nil)
		if err != nil {
			return fmt.Errorf("创建请求失败: %w", err)
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送 Webhook 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Webhook 返回错误状态: %d", resp.StatusCode)
	}

	log.Info("Webhook 通知发送成功")
	return nil
}
