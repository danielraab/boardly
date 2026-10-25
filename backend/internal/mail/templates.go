package mail

import (
	"bytes"
	"embed"
	htmltemplate "html/template"
	"io/fs"
	"strings"
	texttemplate "text/template"
)

//go:embed templates
var templateFiles embed.FS

// DefaultLocale is used when a message is requested in a locale without templates.
const DefaultLocale = "en"

// DefaultAppName names the product when MagicLinkData.AppName is empty.
const DefaultAppName = "Boardly"

// MagicLinkData fills the magic-link templates.
type MagicLinkData struct {
	// AppName is the product name the email shows, DefaultAppName if empty.
	AppName string
	Link    string
	// Code is the one-time code as shown to the reader, such as "K7Q-4MZ".
	Code         string
	ValidMinutes int
}

// RenderMagicLink renders the sign-in email for the locale, falling back to English.
// The txt template defines "subject" and "body"; the html template is the HTML part.
func RenderMagicLink(locale, to string, data MagicLinkData) (Message, error) {
	if _, err := fs.Stat(templateFiles, "templates/"+locale+"/magic_link.txt"); err != nil || strings.ContainsAny(locale, "./") {
		locale = DefaultLocale
	}
	dir := "templates/" + locale + "/"
	if data.AppName == "" {
		data.AppName = DefaultAppName
	}

	txt, err := texttemplate.ParseFS(templateFiles, dir+"magic_link.txt")
	if err != nil {
		return Message{}, err
	}
	html, err := htmltemplate.ParseFS(templateFiles, dir+"magic_link.html")
	if err != nil {
		return Message{}, err
	}

	var subject, text, body bytes.Buffer
	if err := txt.ExecuteTemplate(&subject, "subject", data); err != nil {
		return Message{}, err
	}
	if err := txt.ExecuteTemplate(&text, "body", data); err != nil {
		return Message{}, err
	}
	if err := html.Execute(&body, data); err != nil {
		return Message{}, err
	}
	return Message{
		To:      to,
		Subject: strings.TrimSpace(subject.String()),
		Text:    strings.TrimSpace(text.String()) + "\n",
		HTML:    body.String(),
	}, nil
}
