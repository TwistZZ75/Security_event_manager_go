package actions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/smtp"
	"siem-server/alerts"
)

var emailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; }
        .alert { padding: 20px; border-left: 5px solid; }
        .critical { border-color: #d32f2f; background-color: #ffebee; }
        .high { border-color: #f57c00; background-color: #fff3e0; }
        .medium { border-color: #fbc02d; background-color: #fffde7; }
        .low { border-color: #388e3c; background-color: #e8f5e9; }
        .info { color: #666; }
        .data { background-color: #f5f5f5; padding: 10px; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="alert {{.Severity}}">
        <h2>🚨 Security Alert</h2>
        <h3>{{.Title}}</h3>
        <p><strong>Severity:</strong> {{.Severity}}</p>
        <p><strong>Rule:</strong> {{.RuleName}}</p>
        <p><strong>Description:</strong> {{.Description}}</p>
        <p><strong>Time:</strong> {{.CreatedAt.Format "2006-01-02 15:04:05"}}</p>
        
        <h4>Event Details:</h4>
        <div class="data">
            <p><strong>User:</strong> {{index .EventData "username"}}</p>
            <p><strong>Computer:</strong> {{index .EventData "pc_name"}}</p>
            <p><strong>OS:</strong> {{index .EventData "os"}}</p>
            <p><strong>Source:</strong> {{index .EventData "source"}}</p>
            <p><strong>Category:</strong> {{index .EventData "event_category"}}</p>
            <p><strong>Description:</strong> {{index .EventData "event_description"}}</p>
        </div>
        
        <p class="info">This is an automated alert from your SIEM system.</p>
    </div>
</body>
</html>
`

// EmailNotifier отправляет уведомления по email
type EmailNotifier struct {
	SmtpHost     string
	SmtpPort     string
	SmtpUser     string
	SmtpPassword string
	FromEmail    string
	ToEmails     []string
	template     *template.Template
}

// TelegramNotifier отправляет уведомления в Telegram
type TelegramNotifier struct {
	BotToken string
	ChatID   string
	client   *http.Client
}

// MultiNotifier объединяет несколько способов уведомлений
type MultiNotifier struct {
	email    *EmailNotifier
	telegram *TelegramNotifier
}

// NewMultiNotifier создает новый мультинотификатор
func NewMultiNotifier(email *EmailNotifier, telegram *TelegramNotifier) *MultiNotifier {
	return &MultiNotifier{
		email:    email,
		telegram: telegram,
	}
}

//TODO: сделать функцию Validate() проверяющую поля, которые обязательно должны быть заполнены, в противном случае
//падать с ошибкой

// SendNotification отправляет уведомление по всем каналам
func (mn *MultiNotifier) SendNotification(ctx context.Context, alert *alerts.Alert, channels []string) error {
	var errs []error

	for _, channel := range channels {
		var err error
		switch channel {
		case "email":
			if mn.email != nil {
				err = mn.email.Send(ctx, alert)
			}
		case "telegram":
			if mn.telegram != nil {
				err = mn.telegram.Send(ctx, alert)
			}
		default:
			slog.Warn("Unknown notification channel", "channel", channel)
		}

		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %v", channel, err))
		}
	}

	return errors.Join(errs...)
}

// NewEmailNotifier создает новый email нотификатор
func NewEmailNotifier(cfg EmailNotifier) *EmailNotifier {
	tmpl, err := template.New("email").Parse(emailTemplate)
	if err != nil {
		slog.Error("failed to parse email template", "error", err)
	}

	return &EmailNotifier{
		SmtpHost:     cfg.SmtpHost,
		SmtpPort:     cfg.SmtpPort,
		SmtpUser:     cfg.SmtpUser,
		SmtpPassword: cfg.SmtpPassword,
		FromEmail:    cfg.FromEmail,
		ToEmails:     cfg.ToEmails,
		template:     tmpl,
	}
}

// Send отправляет email уведомление
func (en *EmailNotifier) Send(ctx context.Context, alert *alerts.Alert) error {
	if en.template == nil {
		return fmt.Errorf("email template is not initialized")
	}
	if en.SmtpHost == "" || len(en.ToEmails) == 0 {
		return fmt.Errorf("email not configured")
	}

	// Формируем тело письма
	var body bytes.Buffer
	if err := en.template.Execute(&body, alert); err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	// Формируем заголовки
	subject := fmt.Sprintf("[%s] SIEM Alert: %s", alert.Severity, alert.Title)
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		en.FromEmail,
		en.ToEmails[0],
		subject,
		body.String(),
	)

	// Отправляем email
	auth := smtp.PlainAuth("", en.SmtpUser, en.SmtpPassword, en.SmtpHost)
	err := smtp.SendMail(
		en.SmtpHost+":"+en.SmtpPort,
		auth,
		en.FromEmail,
		en.ToEmails,
		[]byte(message),
	)

	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	slog.Info("Email notification sent for alert", "alert_id", alert.ID)
	return nil
}

// NewTelegramNotifier создает новый Telegram нотификатор
func NewTelegramNotifier(cfg TelegramNotifier) *TelegramNotifier {
	return &TelegramNotifier{
		BotToken: cfg.BotToken,
		ChatID:   cfg.ChatID,
		client:   &http.Client{},
	}
}

// Send отправляет уведомление в Telegram
func (tn *TelegramNotifier) Send(ctx context.Context, alert *alerts.Alert) error {
	if tn.BotToken == "" || tn.ChatID == "" {
		return fmt.Errorf("telegram not configured")
	}

	// Формируем сообщение
	severityEmoji := map[string]string{
		"critical": "🔴",
		"high":     "🟠",
		"medium":   "🟡",
		"low":      "🟢",
	}
	emoji := severityEmoji[alert.Severity]

	message := fmt.Sprintf(
		"%s <b>Security Alert</b>\n\n"+
			"<b>%s</b>\n\n"+
			"Severity: %s\n"+
			"Rule: %s\n"+
			"User: %v\n"+
			"Computer: %v\n"+
			"Time: %s",
		emoji,
		alert.Title,
		alert.Severity,
		alert.RuleName,
		alert.EventData["username"],
		alert.EventData["pc_name"],
		alert.CreatedAt.Format("2006-01-02 15:04:05"),
	)

	// Формируем запрос
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", tn.BotToken)
	payload := map[string]interface{}{
		"chat_id":    tn.ChatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := tn.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	slog.Info("Telegram notification sent for alert", "alert_id", alert.ID)
	return nil
}
