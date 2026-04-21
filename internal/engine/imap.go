package engine

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-sasl"

	"github.com/termite-mail/termite/internal/providers"
)

// Envelope holds parsed envelope data from an IMAP message.
type Envelope struct {
	From       []string
	To         []string
	Cc         []string
	Subject    string
	Date       time.Time
	MessageID  string
	InReplyTo  []string
	References []string
}

// RawMessage holds a fetched IMAP message's metadata and body.
type RawMessage struct {
	UID      imap.UID
	Envelope Envelope
	Body     []byte
	Flags    []imap.Flag
}

// IMAPConn wraps a go-imap v2 client connection.
type IMAPConn struct {
	client *imapclient.Client
}

// Connect establishes an IMAP connection and authenticates.
// For port 993 with TLS enabled, it uses DialTLS (implicit TLS).
// For other ports with TLS enabled, it uses DialStartTLS.
// For non-TLS, it uses DialInsecure.
func (c *IMAPConn) Connect(host string, port int, tlsEnabled bool, creds providers.Credentials) error {
	addr := fmt.Sprintf("%s:%d", host, port)

	var client *imapclient.Client
	var err error

	if tlsEnabled {
		if port == 993 {
			client, err = imapclient.DialTLS(addr, nil)
		} else {
			client, err = imapclient.DialStartTLS(addr, nil)
		}
	} else {
		client, err = imapclient.DialInsecure(addr, nil)
	}
	if err != nil {
		return fmt.Errorf("imap: failed to connect to %s: %w", addr, err)
	}

	c.client = client

	// Authenticate using XOAUTH2 or regular Login.
	if creds.AuthMethod == "XOAUTH2" {
		saslClient := newXOAuth2Client(creds.Username, creds.AccessToken)
		if err := c.client.Authenticate(saslClient); err != nil {
			c.client.Close()
			return fmt.Errorf("imap: XOAUTH2 authentication failed: %w", err)
		}
	} else {
		password := creds.Password
		if password == "" {
			password = creds.AccessToken
		}
		if err := c.client.Login(creds.Username, password).Wait(); err != nil {
			c.client.Close()
			return fmt.Errorf("imap: login failed: %w", err)
		}
	}

	return nil
}

// ListFolders returns all mailbox names visible to the authenticated user.
func (c *IMAPConn) ListFolders() ([]string, error) {
	listCmd := c.client.List("", "*", nil)

	var folders []string
	for {
		mbox := listCmd.Next()
		if mbox == nil {
			break
		}
		folders = append(folders, mbox.Mailbox)
	}
	if err := listCmd.Close(); err != nil {
		return nil, fmt.Errorf("imap: list folders failed: %w", err)
	}
	return folders, nil
}

// FetchUIDsSince returns UIDs in the given folder that are greater than sinceUID.
// If sinceUID is 0, it returns UIDs for all messages in the folder.
func (c *IMAPConn) FetchUIDsSince(folder string, sinceUID uint32) ([]uint32, error) {
	if _, err := c.client.Select(folder, nil).Wait(); err != nil {
		return nil, fmt.Errorf("imap: select %q failed: %w", folder, err)
	}

	criteria := &imap.SearchCriteria{}
	if sinceUID > 0 {
		// Search for UIDs greater than sinceUID by specifying a UID range.
		var uidSet imap.UIDSet
		uidSet.AddRange(imap.UID(sinceUID+1), 0) // 0 means * (latest)
		criteria.UID = append(criteria.UID, uidSet)
	}

	searchData, err := c.client.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("imap: uid search in %q failed: %w", folder, err)
	}

	imapUIDs := searchData.AllUIDs()
	uids := make([]uint32, len(imapUIDs))
	for i, uid := range imapUIDs {
		uids[i] = uint32(uid)
	}
	return uids, nil
}

// FetchMessages fetches complete messages (envelope + full body) for the given UIDs.
func (c *IMAPConn) FetchMessages(folder string, uids []imap.UID) ([]RawMessage, error) {
	if len(uids) == 0 {
		return nil, nil
	}

	if _, err := c.client.Select(folder, nil).Wait(); err != nil {
		return nil, fmt.Errorf("imap: select %q failed: %w", folder, err)
	}

	uidSet := imap.UIDSetNum(uids...)
	bodySection := &imap.FetchItemBodySection{Peek: true}
	fetchOptions := &imap.FetchOptions{
		UID:      true,
		Envelope: true,
		Flags:    true,
		BodySection: []*imap.FetchItemBodySection{
			bodySection,
		},
	}

	messages, err := c.client.Fetch(uidSet, fetchOptions).Collect()
	if err != nil {
		return nil, fmt.Errorf("imap: fetch messages failed: %w", err)
	}

	result := make([]RawMessage, 0, len(messages))
	for _, msg := range messages {
		raw := RawMessage{
			UID:   msg.UID,
			Flags: msg.Flags,
		}

		if msg.Envelope != nil {
			raw.Envelope = convertEnvelope(msg.Envelope)
		}

		body := msg.FindBodySection(bodySection)
		if body != nil {
			raw.Body = body
		}

		result = append(result, raw)
	}

	return result, nil
}

// FetchHeadersOnly fetches only envelope (header) data for the given UIDs.
func (c *IMAPConn) FetchHeadersOnly(folder string, uids []imap.UID) ([]RawMessage, error) {
	if len(uids) == 0 {
		return nil, nil
	}

	if _, err := c.client.Select(folder, nil).Wait(); err != nil {
		return nil, fmt.Errorf("imap: select %q failed: %w", folder, err)
	}

	uidSet := imap.UIDSetNum(uids...)
	headerSection := &imap.FetchItemBodySection{
		Specifier: imap.PartSpecifierHeader,
		Peek:      true,
	}
	fetchOptions := &imap.FetchOptions{
		UID:      true,
		Envelope: true,
		Flags:    true,
		BodySection: []*imap.FetchItemBodySection{
			headerSection,
		},
	}

	messages, err := c.client.Fetch(uidSet, fetchOptions).Collect()
	if err != nil {
		return nil, fmt.Errorf("imap: fetch headers failed: %w", err)
	}

	result := make([]RawMessage, 0, len(messages))
	for _, msg := range messages {
		raw := RawMessage{
			UID:   msg.UID,
			Flags: msg.Flags,
		}

		if msg.Envelope != nil {
			raw.Envelope = convertEnvelope(msg.Envelope)
		}

		// Store raw headers as the body for header-only fetches.
		headerBytes := msg.FindBodySection(headerSection)
		if headerBytes != nil {
			raw.Body = headerBytes
		}

		result = append(result, raw)
	}

	return result, nil
}

// MarkRead sets the \Seen flag on the specified messages.
func (c *IMAPConn) MarkRead(folder string, uids []imap.UID) error {
	return c.storeFlags(folder, uids, imap.StoreFlagsAdd, imap.FlagSeen)
}

// MarkUnread removes the \Seen flag from the specified messages.
func (c *IMAPConn) MarkUnread(folder string, uids []imap.UID) error {
	return c.storeFlags(folder, uids, imap.StoreFlagsDel, imap.FlagSeen)
}

// Move moves the specified messages to the destination folder.
func (c *IMAPConn) Move(folder string, uids []imap.UID, dest string) error {
	if len(uids) == 0 {
		return nil
	}

	if _, err := c.client.Select(folder, nil).Wait(); err != nil {
		return fmt.Errorf("imap: select %q failed: %w", folder, err)
	}

	uidSet := imap.UIDSetNum(uids...)
	if _, err := c.client.Move(uidSet, dest).Wait(); err != nil {
		return fmt.Errorf("imap: move to %q failed: %w", dest, err)
	}

	return nil
}

// Delete marks the specified messages as deleted and expunges them.
func (c *IMAPConn) Delete(folder string, uids []imap.UID) error {
	if len(uids) == 0 {
		return nil
	}

	if _, err := c.client.Select(folder, nil).Wait(); err != nil {
		return fmt.Errorf("imap: select %q failed: %w", folder, err)
	}

	uidSet := imap.UIDSetNum(uids...)

	// Set the \Deleted flag.
	storeFlags := &imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Flags:  []imap.Flag{imap.FlagDeleted},
		Silent: true,
	}
	if err := c.client.Store(uidSet, storeFlags, nil).Close(); err != nil {
		return fmt.Errorf("imap: store deleted flag failed: %w", err)
	}

	// Expunge the deleted messages.
	expungeCmd := c.client.UIDExpunge(uidSet)
	if _, err := expungeCmd.Collect(); err != nil {
		return fmt.Errorf("imap: expunge failed: %w", err)
	}

	return nil
}

// Close logs out and closes the IMAP connection.
func (c *IMAPConn) Close() error {
	if c.client == nil {
		return nil
	}
	if err := c.client.Logout().Wait(); err != nil {
		slog.Warn("imap: logout error", "err", err)
	}
	return c.client.Close()
}

// storeFlags is a helper that selects a folder and applies a flag operation.
func (c *IMAPConn) storeFlags(folder string, uids []imap.UID, op imap.StoreFlagsOp, flag imap.Flag) error {
	if len(uids) == 0 {
		return nil
	}

	if _, err := c.client.Select(folder, nil).Wait(); err != nil {
		return fmt.Errorf("imap: select %q failed: %w", folder, err)
	}

	uidSet := imap.UIDSetNum(uids...)
	storeFlags := &imap.StoreFlags{
		Op:     op,
		Flags:  []imap.Flag{flag},
		Silent: true,
	}
	if err := c.client.Store(uidSet, storeFlags, nil).Close(); err != nil {
		return fmt.Errorf("imap: store flags failed: %w", err)
	}

	return nil
}

// convertEnvelope converts an imap.Envelope to our internal Envelope type.
func convertEnvelope(env *imap.Envelope) Envelope {
	return Envelope{
		From:       addressesToStrings(env.From),
		To:         addressesToStrings(env.To),
		Cc:         addressesToStrings(env.Cc),
		Subject:    env.Subject,
		Date:       env.Date,
		MessageID:  env.MessageID,
		InReplyTo:  env.InReplyTo,
		References: parseReferences(env),
	}
}

// addressesToStrings converts a slice of imap.Address to "name <mailbox@host>" strings.
func addressesToStrings(addrs []imap.Address) []string {
	if len(addrs) == 0 {
		return nil
	}
	result := make([]string, 0, len(addrs))
	for _, a := range addrs {
		result = append(result, a.Addr())
	}
	return result
}

// parseReferences extracts References from the envelope.
// go-imap v2 does not expose References in the Envelope struct directly,
// so we rely on InReplyTo and caller-side header parsing for full references.
func parseReferences(env *imap.Envelope) []string {
	// The References header is not part of the IMAP ENVELOPE response.
	// Full references must be extracted from raw headers during parsing.
	// We return InReplyTo as a starting point; the parser fills in full references.
	return env.InReplyTo
}

// xoauth2Client implements sasl.Client for the XOAUTH2 SASL mechanism.
// XOAUTH2 is used by Gmail and other providers for OAuth2-based IMAP/SMTP auth.
// See https://developers.google.com/gmail/imap/xoauth2-protocol
type xoauth2Client struct {
	username    string
	accessToken string
}

// newXOAuth2Client creates a SASL client for XOAUTH2 authentication.
func newXOAuth2Client(username, accessToken string) sasl.Client {
	return &xoauth2Client{
		username:    username,
		accessToken: accessToken,
	}
}

func (c *xoauth2Client) Start() (string, []byte, error) {
	// XOAUTH2 initial response format:
	// "user=" + user + "\x01" + "auth=Bearer " + token + "\x01\x01"
	resp := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", c.username, c.accessToken)
	return "XOAUTH2", []byte(resp), nil
}

func (c *xoauth2Client) Next(challenge []byte) ([]byte, error) {
	// XOAUTH2 sends an empty response to any server challenge (error response).
	return []byte{}, nil
}

// Ensure xoauth2Client implements sasl.Client at compile time.
var _ sasl.Client = (*xoauth2Client)(nil)
