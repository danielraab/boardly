package notes

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// The note document format, version 1. Every block, item and inline type has
// a Markdown equivalent (see the note-document spec); a new type must define
// one too, so notes can always be converted to Markdown.
//
// Keep in sync with frontend/src/features/notes/document/types.ts.
const (
	DocVersion = 1
	// MaxDocSize bounds a document's JSON form, in bytes.
	MaxDocSize = 512 * 1024
	// MaxListDepth bounds how deeply lists nest; a top-level list is depth 1.
	MaxListDepth = 6
)

// Block types.
const (
	BlockParagraph = "paragraph"
	BlockHeading   = "heading"
	BlockQuote     = "quote"
	BlockCode      = "code"
	BlockDivider   = "divider"
	BlockList      = "list"
	// BlockAttachment shows one of the owner's attachments. It has no text.
	BlockAttachment = "attachment"
)

// List styles.
const (
	ListBullet  = "bullet"
	ListOrdered = "ordered"
	ListTask    = "task"
)

// Inline types.
const (
	InlineText  = "text"
	InlineLink  = "link"
	InlineBreak = "break"
)

// ErrInvalidDocument is returned for content that isn't a valid document.
var ErrInvalidDocument = errors.New("invalid document")

var (
	marks       = map[string]bool{"bold": true, "italic": true, "strike": true, "code": true}
	linkSchemes = map[string]bool{"http": true, "https": true, "mailto": true}
	taskIDRe    = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
	// attachmentIDRe matches the shape of an attachment ID; whether it is
	// one of the owner's attachments is checked when the note is saved.
	attachmentIDRe = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)
)

// Doc is a note's content.
type Doc struct {
	V      int     `json:"v"`
	Blocks []Block `json:"blocks"`
}

// Block is one block of a document. Which fields are set depends on Type.
type Block struct {
	Type    string    `json:"type"`
	Level   int       `json:"level,omitempty"`
	Content *[]Inline `json:"content,omitempty"`
	Text    *string   `json:"text,omitempty"`
	Style   string    `json:"style,omitempty"`
	Items   *[]Item   `json:"items,omitempty"`
	ID      string    `json:"id,omitempty"`
}

// Item is a list item. ID and Checked are only set in task lists.
type Item struct {
	ID       string   `json:"id,omitempty"`
	Checked  *bool    `json:"checked,omitempty"`
	Content  []Inline `json:"content"`
	Children *Block   `json:"children,omitempty"`
}

// Inline is a text run, a link around text runs, or a line break.
type Inline struct {
	Type    string    `json:"type"`
	Text    *string   `json:"text,omitempty"`
	Marks   []string  `json:"marks,omitempty"`
	URL     string    `json:"url,omitempty"`
	Content *[]Inline `json:"content,omitempty"`
}

// EmptyDoc returns a document without blocks.
func EmptyDoc() Doc {
	return Doc{V: DocVersion, Blocks: []Block{}}
}

// ParseDoc decodes and validates a document and normalizes its task IDs (see
// normalizeTaskIDs). Empty text runs are dropped.
func ParseDoc(raw []byte) (Doc, error) {
	if len(raw) > MaxDocSize {
		return Doc{}, fmt.Errorf("%w: larger than %d bytes", ErrInvalidDocument, MaxDocSize)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var d Doc
	if err := dec.Decode(&d); err != nil {
		return Doc{}, fmt.Errorf("%w: %v", ErrInvalidDocument, err)
	}
	if dec.More() {
		return Doc{}, fmt.Errorf("%w: trailing data", ErrInvalidDocument)
	}
	if d.V != DocVersion {
		return Doc{}, fmt.Errorf("%w: unknown version %d", ErrInvalidDocument, d.V)
	}
	if d.Blocks == nil {
		d.Blocks = []Block{}
	}
	for i := range d.Blocks {
		if err := checkBlock(&d.Blocks[i], 0); err != nil {
			return Doc{}, fmt.Errorf("%w: %v", ErrInvalidDocument, err)
		}
	}
	normalizeTaskIDs(&d)
	return d, nil
}

// checkBlock validates b, which is nested in depth lists.
func checkBlock(b *Block, depth int) error {
	if b.Type != BlockAttachment && b.ID != "" {
		return fmt.Errorf("%s with an ID", b.Type)
	}
	switch b.Type {
	case BlockParagraph, BlockQuote, BlockHeading:
		if b.Text != nil || b.Style != "" || b.Items != nil {
			return fmt.Errorf("%s with list or code fields", b.Type)
		}
		if b.Type == BlockHeading {
			if b.Level < 1 || b.Level > 3 {
				return fmt.Errorf("heading level %d", b.Level)
			}
		} else if b.Level != 0 {
			return fmt.Errorf("%s with a level", b.Type)
		}
		content, err := checkInlines(b.Content, true)
		if err != nil {
			return err
		}
		b.Content = &content
	case BlockCode:
		if b.Level != 0 || b.Content != nil || b.Style != "" || b.Items != nil {
			return errors.New("code with other fields")
		}
		if b.Text == nil {
			b.Text = new(string)
		}
	case BlockDivider:
		if b.Level != 0 || b.Content != nil || b.Text != nil || b.Style != "" || b.Items != nil {
			return errors.New("divider with fields")
		}
	case BlockList:
		return checkList(b, depth+1)
	case BlockAttachment:
		if b.Level != 0 || b.Content != nil || b.Text != nil || b.Style != "" || b.Items != nil {
			return errors.New("attachment with other fields")
		}
		if !attachmentIDRe.MatchString(b.ID) {
			return fmt.Errorf("invalid attachment ID %q", b.ID)
		}
	default:
		return fmt.Errorf("unknown block type %q", b.Type)
	}
	return nil
}

// AttachmentIDs returns the IDs of the attachments in a document, each once,
// in the order they first appear.
func AttachmentIDs(d Doc) []string {
	ids := []string{}
	seen := map[string]bool{}
	for _, b := range d.Blocks {
		if b.Type == BlockAttachment && !seen[b.ID] {
			seen[b.ID] = true
			ids = append(ids, b.ID)
		}
	}
	return ids
}

// SplitTrailingAttachments splits off the attachment blocks at the end of d,
// after its last block of another type, for connected apps that keep
// attachments next to the text. It returns the rest of d and the IDs of the
// split-off attachments, each once, in the order they first appear.
func SplitTrailingAttachments(d Doc) (Doc, []string) {
	end := len(d.Blocks)
	for end > 0 && d.Blocks[end-1].Type == BlockAttachment {
		end--
	}
	body := Doc{V: d.V, Blocks: d.Blocks[:end:end]}
	return body, AttachmentIDs(Doc{Blocks: d.Blocks[end:]})
}

// JoinTrailingAttachments is d followed by an attachment block for each of
// ids, each once, the opposite of SplitTrailingAttachments.
func JoinTrailingAttachments(d Doc, ids []string) Doc {
	blocks := append([]Block{}, d.Blocks...)
	seen := map[string]bool{}
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			blocks = append(blocks, Block{Type: BlockAttachment, ID: id})
		}
	}
	return Doc{V: d.V, Blocks: blocks}
}

// checkList validates a list at depth (1 for a top-level list).
func checkList(b *Block, depth int) error {
	if b.Type != BlockList {
		return fmt.Errorf("children of type %q", b.Type)
	}
	if depth > MaxListDepth {
		return fmt.Errorf("lists nested deeper than %d", MaxListDepth)
	}
	if b.Level != 0 || b.Content != nil || b.Text != nil {
		return errors.New("list with other fields")
	}
	switch b.Style {
	case ListBullet, ListOrdered, ListTask:
	default:
		return fmt.Errorf("unknown list style %q", b.Style)
	}
	if b.Items == nil || len(*b.Items) == 0 {
		return errors.New("list without items")
	}
	for i := range *b.Items {
		it := &(*b.Items)[i]
		if b.Style == ListTask {
			if it.ID != "" && !taskIDRe.MatchString(it.ID) {
				return fmt.Errorf("invalid task ID %q", it.ID)
			}
			if it.Checked == nil {
				it.Checked = new(bool)
			}
		} else if it.ID != "" || it.Checked != nil {
			return errors.New("ID or checked state outside a task list")
		}
		content, err := checkInlines(&it.Content, true)
		if err != nil {
			return err
		}
		it.Content = content
		if it.Children != nil {
			if err := checkList(it.Children, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkInlines validates inline content and drops empty text runs. Links and
// line breaks are only allowed where allowLinks is set; a link's content is
// text runs only.
func checkInlines(content *[]Inline, allowLinks bool) ([]Inline, error) {
	out := []Inline{}
	if content == nil {
		return out, nil
	}
	for _, in := range *content {
		switch in.Type {
		case InlineText:
			if in.URL != "" || in.Content != nil {
				return nil, errors.New("text with link fields")
			}
			if in.Text == nil {
				return nil, errors.New("text without text")
			}
			seen := map[string]bool{}
			for _, m := range in.Marks {
				if !marks[m] || seen[m] {
					return nil, fmt.Errorf("unknown or repeated mark %q", m)
				}
				seen[m] = true
			}
			if *in.Text == "" {
				continue
			}
		case InlineLink:
			if !allowLinks {
				return nil, errors.New("link inside a link")
			}
			if in.Text != nil || in.Marks != nil {
				return nil, errors.New("link with text fields")
			}
			u, err := url.Parse(in.URL)
			if err != nil || !linkSchemes[strings.ToLower(u.Scheme)] || len(in.URL) > 2048 {
				return nil, fmt.Errorf("invalid link URL %q", in.URL)
			}
			inner, err := checkInlines(in.Content, false)
			if err != nil {
				return nil, err
			}
			for _, t := range inner {
				if t.Type != InlineText {
					return nil, errors.New("link with content other than text")
				}
			}
			in.Content = &inner
		case InlineBreak:
			if !allowLinks {
				return nil, errors.New("line break inside a link")
			}
			if in.Text != nil || in.Marks != nil || in.URL != "" || in.Content != nil {
				return nil, errors.New("line break with fields")
			}
		default:
			return nil, fmt.Errorf("unknown inline type %q", in.Type)
		}
		out = append(out, in)
	}
	return out, nil
}

// normalizeTaskIDs gives every task item without an ID, or with an ID that
// already appeared earlier in the document, a new ID.
func normalizeTaskIDs(d *Doc) {
	seen := map[string]bool{}
	var visit func(b *Block)
	visit = func(b *Block) {
		if b.Type != BlockList || b.Items == nil {
			return
		}
		for i := range *b.Items {
			it := &(*b.Items)[i]
			if b.Style == ListTask {
				for it.ID == "" || seen[it.ID] {
					it.ID = NewTaskID()
				}
				seen[it.ID] = true
			}
			if it.Children != nil {
				visit(it.Children)
			}
		}
	}
	for i := range d.Blocks {
		visit(&d.Blocks[i])
	}
}

// NewTaskID returns a random task ID of 12 base64url characters.
func NewTaskID() string {
	b := make([]byte, 9)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
