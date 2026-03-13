package plugins

import (
	"database/sql"
	"encoding/json"
	"slices"
	"strings"
	"sync"

	db "github.com/zkyrnx11/mack/src/sql"
)

type Mode string

const (
	ModePublic  Mode = "public"
	ModePrivate Mode = "private"
)

type Settings struct {
	mu             sync.RWMutex
	Prefixes       []string
	Sudoers        []string
	BannedUsers    []string
	Mode           Mode
	Language       string
	DisabledCmds   []string
	GCDisabled     bool
	AutoSaveStatus bool
	AutoLikeStatus bool
	AntiDelete     bool
}

var BotSettings = &Settings{
	Prefixes: []string{"."},
	Sudoers:  []string{},
	Mode:     ModePublic,
	Language: "en",
}

var settingsUser string

func InitDB(rawDB *sql.DB) error {
	if err := db.InitDB(rawDB); err != nil {
		return err
	}
	loadAnticallSettings()
	return nil
}

func InitSettings(user string) error {
	settingsUser = user
	if err := LoadSettings(); err != nil {
		return err
	}

	return SaveSettings()
}

func LoadSettings() error {
	if settingsUser == "" {
		return nil
	}
	rows, err := db.DB().Query(
		`SELECT key, value FROM bot_settings WHERE user = ?`, settingsUser)
	if err != nil {
		return err
	}
	defer rows.Close()

	BotSettings.mu.Lock()
	defer BotSettings.mu.Unlock()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return err
		}
		switch key {
		case "prefixes":
			var p []string
			if json.Unmarshal([]byte(value), &p) == nil {
				BotSettings.Prefixes = p
			}
		case "sudoers":
			var s []string
			if json.Unmarshal([]byte(value), &s) == nil {
				BotSettings.Sudoers = s
			}
		case "mode":
			BotSettings.Mode = Mode(value)
		case "language":
			BotSettings.Language = value
		case "banned_users":
			var b []string
			if json.Unmarshal([]byte(value), &b) == nil {
				BotSettings.BannedUsers = b
			}
		case "disabled_cmds":
			var d []string
			if json.Unmarshal([]byte(value), &d) == nil {
				BotSettings.DisabledCmds = d
			}
		case "gc_disabled":
			BotSettings.GCDisabled = value == "true"
		case "auto_save_status":
			BotSettings.AutoSaveStatus = value == "true"
		case "auto_like_status":
			BotSettings.AutoLikeStatus = value == "true"
		case "anti_delete":
			BotSettings.AntiDelete = value == "true"
		}
	}
	return rows.Err()
}

func SaveSettings() error {
	if settingsUser == "" {
		return nil
	}

	BotSettings.mu.RLock()
	prefixes := BotSettings.Prefixes
	sudoers := BotSettings.Sudoers
	bannedUsers := BotSettings.BannedUsers
	mode := BotSettings.Mode
	language := BotSettings.Language
	disabledCmds := BotSettings.DisabledCmds
	gcDisabled := BotSettings.GCDisabled
	autoSaveStatus := BotSettings.AutoSaveStatus
	autoLikeStatus := BotSettings.AutoLikeStatus
	antiDelete := BotSettings.AntiDelete
	BotSettings.mu.RUnlock()

	pData, _ := json.Marshal(prefixes)
	sData, _ := json.Marshal(sudoers)
	bData, _ := json.Marshal(bannedUsers)
	dData, _ := json.Marshal(disabledCmds)
	gcStr := "false"
	if gcDisabled {
		gcStr = "true"
	}
	autoSaveStr := "false"
	if autoSaveStatus {
		autoSaveStr = "true"
	}
	autoLikeStr := "false"
	if autoLikeStatus {
		autoLikeStr = "true"
	}
	antiDeleteStr := "false"
	if antiDelete {
		antiDeleteStr = "true"
	}

	upsert := `INSERT INTO bot_settings (user, key, value) VALUES (?, ?, ?)
		ON CONFLICT(user, key) DO UPDATE SET value = excluded.value`

	tx, err := db.DB().Begin()
	if err != nil {
		return err
	}
	for _, row := range [][2]string{
		{"prefixes", string(pData)},
		{"sudoers", string(sData)},
		{"banned_users", string(bData)},
		{"mode", string(mode)},
		{"language", language},
		{"disabled_cmds", string(dData)},
		{"gc_disabled", gcStr},
		{"auto_save_status", autoSaveStr},
		{"auto_like_status", autoLikeStr},
		{"anti_delete", antiDeleteStr},
	} {
		if _, err = tx.Exec(upsert, settingsUser, row[0], row[1]); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Settings) IsSudo(phone string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.Sudoers {
		if p == phone {
			return true
		}
	}
	return false
}

func (s *Settings) GetPrefixes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]string, len(s.Prefixes))
	copy(result, s.Prefixes)
	return result
}

func (s *Settings) GetMode() Mode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Mode
}

func (s *Settings) SetPrefixes(raw string) {
	parts := strings.Fields(raw)
	for i, p := range parts {
		if strings.ToLower(p) == "empty" {
			parts[i] = ""
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Prefixes = parts
}

func (s *Settings) AddSudo(phone string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.Sudoers {
		if p == phone {
			return
		}
	}
	s.Sudoers = append(s.Sudoers, phone)
}

func (s *Settings) RemoveSudo(phone string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, p := range s.Sudoers {
		if p == phone {
			s.Sudoers = append(s.Sudoers[:i], s.Sudoers[i+1:]...)
			return true
		}
	}
	return false
}

func (s *Settings) SetMode(m Mode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Mode = m
}

func (s *Settings) GetLanguage() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Language == "" {
		return "en"
	}
	return s.Language
}

func (s *Settings) SetLanguage(lang string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Language = lang
}

func (s *Settings) DisableCmd(name string) {
	name = strings.ToLower(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.DisabledCmds {
		if d == name {
			return
		}
	}
	s.DisabledCmds = append(s.DisabledCmds, name)
}

func (s *Settings) EnableCmd(name string) bool {
	name = strings.ToLower(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, d := range s.DisabledCmds {
		if d == name {
			s.DisabledCmds = append(s.DisabledCmds[:i], s.DisabledCmds[i+1:]...)
			return true
		}
	}
	return false
}

func (s *Settings) IsCmdDisabled(name string) bool {
	name = strings.ToLower(name)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, d := range s.DisabledCmds {
		if d == name {
			return true
		}
	}
	return false
}

func (s *Settings) SetGCDisabled(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.GCDisabled = v
}

func (s *Settings) IsGCDisabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.GCDisabled
}

func (s *Settings) BanUser(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if slices.Contains(s.BannedUsers, id) {
		return
	}
	s.BannedUsers = append(s.BannedUsers, id)
}

func (s *Settings) UnbanUser(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, b := range s.BannedUsers {
		if b == id {
			s.BannedUsers = append(s.BannedUsers[:i], s.BannedUsers[i+1:]...)
			return true
		}
	}
	return false
}

func (s *Settings) IsBanned(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, b := range s.BannedUsers {
		if b == id {
			return true
		}
	}
	return false
}
