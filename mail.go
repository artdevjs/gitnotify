package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"time"
)

// MailContent ..
type MailContent struct {
	WebsiteURL string
	User       string // provider/username
	Name       string
	Data       []*gitRepoDiffs
}

func processForMail(diff []*gitRepoDiffs, conf *Setting) error {
	mailContent := &MailContent{
		WebsiteURL: config.ServerProto + config.ServerHost,
		User:       fmt.Sprintf("%s/%s", conf.Auth.Provider, conf.Auth.UserName),
		Name:       conf.usersName(),
	}
	mailContent.Data = diff

	htmlBuffer := &bytes.Buffer{}
	displayPage(htmlBuffer, "changes_mail", mailContent)
	html, _ := ioutil.ReadAll(htmlBuffer)

	textBuffer := &bytes.Buffer{}
	displayPage(textBuffer, "changes_mail_text", mailContent)
	text, _ := ioutil.ReadAll(textBuffer)
	textContent := strings.Replace(string(text), "\n\n", "\n", -1)
	textContent = strings.Replace(textContent, "\n\n", "\n", -1)

	loc, _ := time.LoadLocation(conf.User.TimeZoneName)
	t := time.Now().In(loc)
	subject := "[GitNotify] New Updates from your Repositories - " + t.Format("02 Jan 2006 | 15 Hrs")

	to := &recepient{
		Name:     conf.usersName(),
		Address:  conf.usersEmail(),
		UserName: conf.Auth.UserName,
		Provider: conf.Auth.Provider,
	}

	ctx := &emailCtx{
		Subject:  subject,
		TextBody: textContent,
		HTMLBody: string(html),
	}

	sendEmail(to, ctx)
	return nil
}
