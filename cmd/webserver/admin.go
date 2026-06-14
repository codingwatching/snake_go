package main

// admin.go holds operational endpoints: the feedback dashboard and the Feishu
// notification hook.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/trytobebee/snake_go/pkg/game"
)

// feedbackEntry is a single row rendered on the admin dashboard.
type feedbackEntry struct {
	User    string
	Message string
	Time    string
}

// feedbackPageTemplate auto-escapes all interpolated values, preventing stored
// XSS from user-submitted feedback.
var feedbackPageTemplate = template.Must(template.New("feedback").Parse(`<!DOCTYPE html>
<html><head><title>Admin - Feedback</title>
<meta charset="utf-8">
<style>body{font-family:sans-serif;background:#1a1a2e;color:#fff;padding:20px;}
table{width:100%;border-collapse:collapse;} th,td{border:1px solid #444;padding:12px;text-align:left;} th{background:#333;}</style>
</head><body>
<h1>📩 Recent User Feedback</h1>
<p>Welcome, Admin. Showing last 50 entries.</p>
<table><tr><th>Time</th><th>User</th><th>Message</th></tr>
{{range .}}<tr><td>{{.Time}}</td><td><strong>{{.User}}</strong></td><td>{{.Message}}</td></tr>
{{end}}</table>
</body></html>`))

// adminFeedbackHandler shows recent feedback. It requires ADMIN_SECRET to be
// configured; there is no insecure default.
func adminFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	secret := os.Getenv("ADMIN_SECRET")
	if secret == "" {
		log.Println("⚠️  /admin/feedback requested but ADMIN_SECRET is not configured; refusing.")
		http.Error(w, "Admin endpoint disabled: ADMIN_SECRET is not configured.", http.StatusServiceUnavailable)
		return
	}
	if r.URL.Query().Get("key") != secret {
		http.Error(w, "Unauthorized. Please provide a valid ?key=...", http.StatusUnauthorized)
		return
	}

	rows, err := game.DB.Query("SELECT username, message, created_at FROM feedback ORDER BY created_at DESC LIMIT 50")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var entries []feedbackEntry
	for rows.Next() {
		var e feedbackEntry
		if err := rows.Scan(&e.User, &e.Message, &e.Time); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		entries = append(entries, e)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := feedbackPageTemplate.Execute(w, entries); err != nil {
		log.Printf("❌ Error rendering feedback page: %v\n", err)
	}
}

// notifyFeishu posts a feedback card to the configured Feishu webhook.
func notifyFeishu(username, feedback string) {
	webhookURL := os.Getenv("FEISHU_WEBHOOK_URL")
	if webhookURL == "" {
		return
	}

	// Interactive card format renders most reliably in Feishu.
	payload := map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"header": map[string]interface{}{
				"title": map[string]interface{}{
					"tag":     "plain_text",
					"content": "🐍 贪吃蛇游戏 - 新用户反馈",
				},
				"template": "blue",
			},
			"elements": []map[string]interface{}{
				{
					"tag": "div",
					"text": map[string]interface{}{
						"tag": "lark_md",
						"content": fmt.Sprintf("**用户ID:** %s\n**反馈时间:** %s\n\n**反馈详情:**\n%s",
							username, time.Now().Format("2006-01-02 15:04:05"), feedback),
					},
				},
			},
		},
	}

	jsonPayload, _ := json.Marshal(payload)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("❌ Failed to send Feishu notification: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("⚠️ Feishu returned non-OK status: %s\n", resp.Status)
	} else {
		log.Printf("🔔 Feishu card notification sent successfully!\n")
	}
}
