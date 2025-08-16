package main

import (
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"unicode"

	"github.com/Vadim_Makhnev/receipt-parser/internal/config"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/joho/godotenv"
	"golang.org/x/net/html"
)

type ReceiptParseService struct {
	config *config.IMAPConfig
}

func NewReceiptParseService(cfg *config.IMAPConfig) *ReceiptParseService {
	return &ReceiptParseService{
		config: cfg,
	}
}

// Loading environment
func LoadEnv() error {
	if err := godotenv.Load("local.env"); err != nil {
		return fmt.Errorf("Не удалось загрузить env окружение %w:", err)
	}
	return nil
}

func (s *ReceiptParseService) Run() error {

	log.Println("Connecting to server...")

	c, err := client.DialTLS(s.config.Server, nil)
	if err != nil {
		return err
	}
	log.Println("Connected")

	defer c.Logout()

	if err := c.Login(s.config.Username, s.config.Password); err != nil {
		return err
	}

	log.Println("Logged in")

	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	if err := <-done; err != nil {
		return err
	}

	mbox, err := c.Select("INBOX", false)
	if err != nil {
		return err
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(mbox.Messages - 3)

	messages := make(chan *imap.Message, 10)
	done = make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchRFC822}, messages)
	}()

	if err := <-done; err != nil {
		return err
	}

	for msg := range messages {
		if msg == nil {
			continue
		}

		r := msg.GetBody(&imap.BodySectionName{})
		if r == nil {
			log.Println("Тело сообщения пустое")
		}

		mr, err := mail.CreateReader(r)
		if err != nil {
			log.Printf("Ошибка парсинга MIME: %v", err)
			continue
		}
		defer mr.Close()

		textHtml, err := ExtractHTML(mr)
		if err != nil {
			log.Printf("Ошибка извлечения текста: %v", err)
			continue
		}

		total := totalSumParser(textHtml)
		fmt.Printf("Итоговая сумма: %s\n", total)
		return nil

	}
	return nil
}

// Parsing text/html parts into string
// Recursively going through the DOM tree
// Writing DOM text nodes into strings.Builder
func HTMLToText(htmlStr string) string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return htmlStr
	}

	var sb strings.Builder
	var extractText func(*html.Node)

	extractText = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractText(c)
		}
	}
	extractText(doc)

	return strings.TrimSpace(sb.String())
}

// Finding text/html parts
// io.ReadAll to read text/html
// Writing readed text/html part into strings.Builder
// separated by space
func ExtractHTML(mr *mail.Reader) (string, error) {
	var fullText strings.Builder

	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}

		if err != nil {
			return "", err
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			contentType, _, _ := h.ContentType()
			if strings.Contains(contentType, "text/html") {
				body, err := io.ReadAll(p.Body)
				if err != nil {
					return "", err
				}
				fullText.WriteString(" ")
				fullText.WriteString(HTMLToText(string(body)))
			}
		}
	}

	return fullText.String(), nil
}

// Delete all spaces in the string
// Default spaces and invisible spaces
func DeleteAllSpaces(text string) string {
	spaces := regexp.MustCompile(`\s+`)

	str := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, text)

	return string(spaces.ReplaceAll([]byte(str), []byte("")))

}

// Parsing string into total sum
func totalSumParser(text string) string {
	total := regexp.MustCompile(`итого\d+\.\d+`)

	text = DeleteAllSpaces(text)
	text = strings.ReplaceAll(text, ",", ".")
	grid := total.Find([]byte(strings.ToLower(text)))

	return string(grid)
}

func main() {
	if err := LoadEnv(); err != nil {
		log.Fatal(err)
	}

	imapconfig, err := config.NewIMAPConfig()
	if err != nil {
		log.Fatal(err)
	}

	service := NewReceiptParseService(imapconfig)

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
