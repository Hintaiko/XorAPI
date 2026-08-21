package service

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

func SendMail(to, subject, body string) error {
	host := GetConfig("smtp_host")
	user := GetConfig("smtp_user")
	pass := GetConfig("smtp_pass")
	from := GetConfig("smtp_from")
	if host == "" || user == "" {
		return fmt.Errorf("SMTP 未配置，请联系管理员在后台设置邮箱服务")
	}
	if from == "" {
		from = user
	}
	port := GetConfigInt("smtp_port", 465)
	addr := fmt.Sprintf("%s:%d", host, port)

	headers := map[string]string{
		"From":         from,
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/plain; charset=UTF-8",
	}
	var b strings.Builder
	for k, v := range headers {
		fmt.Fprintf(&b, "%s: %s\r\n", k, v)
	}
	b.WriteString("\r\n" + body)

	auth := smtp.PlainAuth("", user, pass, host)
	if port == 465 {
		return sendMailSSL(addr, host, auth, from, to, b.String())
	}
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(b.String()))
}

func sendMailSSL(addr, host string, auth smtp.Auth, from, to, msg string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer c.Close()
	if err = c.Auth(auth); err != nil {
		return err
	}
	if err = c.Mail(from); err != nil {
		return err
	}
	if err = c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write([]byte(msg)); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
