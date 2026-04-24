package username

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"time"
)

// ReservedUsernameChecker handles reserved username validation
type ReservedUsernameChecker struct {
	store       ReservedUsernameStore
	reservedMap map[string]bool
	mu          sync.RWMutex
	lastRefresh time.Time
}

// ReservedUsernameStore defines the interface for reserved username data access
type ReservedUsernameStore interface {
	GetAllReservedUsernames(ctx context.Context) ([]string, error)
	IsReserved(ctx context.Context, username string) (bool, error)
}

// NewReservedUsernameChecker creates a new reserved username checker
func NewReservedUsernameChecker(store ReservedUsernameStore) *ReservedUsernameChecker {
	return &ReservedUsernameChecker{
		store:       store,
		reservedMap: make(map[string]bool),
	}
}

// IsReserved checks if a username is reserved (case-insensitive)
func (c *ReservedUsernameChecker) IsReserved(ctx context.Context, username string) (bool, error) {
	if username == "" {
		return false, nil
	}

	// Normalize for comparison
	normalized := strings.ToLower(strings.TrimSpace(username))

	// Check in-memory cache first
	c.mu.RLock()
	if reserved, ok := c.reservedMap[normalized]; ok {
		c.mu.RUnlock()
		return reserved, nil
	}
	c.mu.RUnlock()

	// Check database
	isReserved, err := c.store.IsReserved(ctx, normalized)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}

	// Cache result
	c.mu.Lock()
	c.reservedMap[normalized] = isReserved
	c.mu.Unlock()

	return isReserved, nil
}

// RefreshCache reloads the reserved usernames from the database
func (c *ReservedUsernameChecker) RefreshCache(ctx context.Context) error {
	reserved, err := c.store.GetAllReservedUsernames(ctx)
	if err != nil {
		return err
	}

	newMap := make(map[string]bool, len(reserved))
	for _, username := range reserved {
		newMap[strings.ToLower(username)] = true
	}

	c.mu.Lock()
	c.reservedMap = newMap
	c.lastRefresh = time.Now()
	c.mu.Unlock()

	return nil
}

// GetLastRefreshTime returns when the cache was last refreshed
func (c *ReservedUsernameChecker) GetLastRefreshTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastRefresh
}

// HardcodedSystemReserved contains system-critical reserved usernames
// These are always blocked regardless of database state
var HardcodedSystemReserved = map[string]bool{
	"admin":         true,
	"administrator": true,
	"root":          true,
	"system":        true,
	"superuser":     true,
	"support":       true,
	"help":          true,
	"official":      true,
	"locolive":      true,
	"locoliv":       true,
	"privacy":       true,
	"security":      true,
	"moderator":     true,
	"mod":           true,
	"staff":         true,
	"team":          true,
	"api":           true,
	"www":           true,
	"mail":          true,
	"ftp":           true,
	"sftp":          true,
	"ssh":           true,
	"smtp":          true,
	"imap":          true,
	"pop":           true,
	"pop3":          true,
	"http":          true,
	"https":         true,
	"localhost":     true,
	"127_0_0_1":     true,
	"0_0_0_0":       true,
	"null":          true,
	"nil":           true,
	"undefined":     true,
	"NaN":           true,
	"Infinity":      true,
	"console":       true,
	"log":           true,
	"debug":         true,
	"test":          true,
	"testing":       true,
	"example":       true,
	"sample":        true,
	"demo":          true,
	"user":          true,
	"users":         true,
	"account":       true,
	"accounts":      true,
	"login":         true,
	"logout":        true,
	"signup":        true,
	"register":      true,
	"signin":        true,
	"signout":       true,
	"auth":          true,
	"oauth":         true,
	"sso":           true,
	"password":      true,
	"reset":         true,
	"verify":        true,
	"confirm":       true,
	"email":         true,
	"phone":         true,
	"mobile":        true,
	"contact":       true,
	"about":         true,
	"faq":           true,
	"terms":         true,
	"policy":        true,
	"legal":         true,
	"cookie":        true,
	"cookies":       true,
	"settings":      true,
	"config":        true,
	"configuration": true,
	"dashboard":     true,
	"home":          true,
	"index":         true,
	"main":          true,
	"start":         true,
	"getstarted":    true,
	"welcome":       true,
	"hello":         true,
	"hi":            true,
	"news":          true,
	"blog":          true,
	"careers":       true,
	"jobs":          true,
	"press":         true,
	"media":         true,
	"partner":       true,
	"partners":      true,
	"affiliate":     true,
	"business":      true,
	"enterprise":    true,
	"pro":           true,
	"premium":       true,
	"plus":          true,
	"gold":          true,
	"vip":           true,
	"elite":         true,
	"featured":      true,
	"popular":       true,
	"trending":      true,
	"explore":       true,
	"discover":      true,
	"search":        true,
	"find":          true,
	"lookup":        true,
	"all":           true,
	"everyone":      true,
	"notifications": true,
	"messages":      true,
	"chat":          true,
	"calls":         true,
	"video":         true,
	"voice":         true,
	"stories":       true,
	"reels":         true,
	"posts":         true,
	"photos":        true,
	"videos":        true,
	"files":         true,
	"uploads":       true,
	"downloads":     true,
	"share":         true,
	"send":          true,
	"create":        true,
	"edit":          true,
	"delete":        true,
	"remove":        true,
	"block":         true,
	"report":        true,
	"feedback":      true,
	"status":        true,
	"online":        true,
	"offline":       true,
	"away":          true,
	"busy":          true,
	"available":     true,
	"invisible":     true,
	"ghost":         true,
	"anonymous":     true,
	"guest":         true,
	"unknown":       true,
	"deleted":       true,
	"deactivated":   true,
	"suspended":     true,
	"banned":        true,
	"blocked":       true,
	"restricted":    true,
	"limited":       true,
}

// IsHardcodedReserved checks against hardcoded system reserved names
func IsHardcodedReserved(username string) bool {
	if username == "" {
		return false
	}
	lower := strings.ToLower(username)
	return HardcodedSystemReserved[lower]
}
