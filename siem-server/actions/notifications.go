package actions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"siem-server/alerts"
)

// EmailNotifier отправляет уведомления по email
type EmailNotifier struct {
	smtpHost     string
	smtpPort     string
	smtpUser     string
	smtpPassword string
	fromEmail    string
	toEmails     []string
	template     *template.Template
}

// TelegramNotifier отправляет уведомления в Telegram
type TelegramNotifier struct {
	botToken string
	chatID   string
	client   *http.Client
}

// MultiNotifier объединяет несколько способов уведомлений
type MultiNotifier struct {
	email    *EmailNotifier
	telegram *TelegramNotifier
}

// NewMultiNotifier создает новый мультинотификатор
func NewMultiNotifier() *MultiNotifier {
	return &MultiNotifier{
		email:    NewEmailNotifier(),
		telegram: NewTelegramNotifier(),
	}
}

// SendNotification отправляет уведомление по всем каналам
func (mn *MultiNotifier) SendNotification(ctx context.Context, alert *alerts.Alert, channels []string) error {
	var errors []error

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
			log.Printf("Unknown notification channel: %s", channel)
		}

		if err != nil {
			errors = append(errors, fmt.Errorf("%s: %v", channel, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("notification errors: %v", errors)
	}

	return nil
}

// NewEmailNotifier создает новый email нотификатор
func NewEmailNotifier() *EmailNotifier {
	tmpl, _ := template.New("email").Parse(emailTemplate)

	return &EmailNotifier{
		smtpHost:     os.Getenv("SMTP_HOST"),
		smtpPort:     os.Getenv("SMTP_PORT"),
		smtpUser:     os.Getenv("SMTP_USER"),
		smtpPassword: os.Getenv("SMTP_PASSWORD"),
		fromEmail:    os.Getenv("SMTP_FROM"),
		toEmails:     []string{os.Getenv("ALERT_EMAIL")},
		template:     tmpl,
	}
}

// Send отправляет email уведомление
func (en *EmailNotifier) Send(ctx context.Context, alert *alerts.Alert) error {
	if en.smtpHost == "" || en.toEmails == nil || len(en.toEmails) == 0 {
		return fmt.Errorf("email not configured")
	}

	// Формируем тело письма
	var body bytes.Buffer
	if err := en.template.Execute(&body, alert); err != nil {
		return fmt.Errorf("failed to render email template: %v", err)
	}

	// Формируем заголовки
	subject := fmt.Sprintf("[%s] SIEM Alert: %s", alert.Severity, alert.Title)
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		en.fromEmail,
		en.toEmails[0],
		subject,
		body.String(),
	)

	// Отправляем email
	auth := smtp.PlainAuth("", en.smtpUser, en.smtpPassword, en.smtpHost)
	err := smtp.SendMail(
		en.smtpHost+":"+en.smtpPort,
		auth,
		en.fromEmail,
		en.toEmails,
		[]byte(message),
	)

	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	log.Printf("Email notification sent for alert %d", alert.ID)
	return nil
}

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

// NewTelegramNotifier создает новый Telegram нотификатор
func NewTelegramNotifier() *TelegramNotifier {
	return &TelegramNotifier{
		botToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		chatID:   os.Getenv("TELEGRAM_CHAT_ID"),
		client:   &http.Client{},
	}
}

// Send отправляет уведомление в Telegram
func (tn *TelegramNotifier) Send(ctx context.Context, alert *alerts.Alert) error {
	if tn.botToken == "" || tn.chatID == "" {
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
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", tn.botToken)
	payload := map[string]interface{}{
		"chat_id":    tn.chatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := tn.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	log.Printf("Telegram notification sent for alert %d", alert.ID)
	return nil
}
