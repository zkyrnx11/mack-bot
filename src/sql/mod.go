package db

import (
	"strings"
	"time"
)

type AFKStatus struct {
	Message string
	SetAt   time.Time
}

func AddWarn(chatJID, userID string) int {
	settingsDB.Exec(
		`INSERT INTO warns (chat_jid, user_id, count) VALUES (?, ?, 1)
		 ON CONFLICT(chat_jid, user_id) DO UPDATE SET count = count + 1`,
		chatJID, userID,
	)
	return GetWarnCount(chatJID, userID)
}

func ResetWarns(chatJID, userID string) {
	settingsDB.Exec(`DELETE FROM warns WHERE chat_jid = ? AND user_id = ?`, chatJID, userID)
}

func GetWarnCount(chatJID, userID string) int {
	var n int
	settingsDB.QueryRow(
		`SELECT count FROM warns WHERE chat_jid = ? AND user_id = ?`, chatJID, userID,
	).Scan(&n)
	return n
}

func IsShhed(chatJID, userID string) bool {
	var dummy string
	err := settingsDB.QueryRow(
		`SELECT user_id FROM shh_users WHERE chat_jid = ? AND user_id = ?`, chatJID, userID,
	).Scan(&dummy)
	return err == nil
}

func SetShh(chatJID, userID string) {
	settingsDB.Exec(
		`INSERT OR IGNORE INTO shh_users (chat_jid, user_id) VALUES (?, ?)`, chatJID, userID,
	)
}

func UnShh(chatJID, userID string) {
	settingsDB.Exec(
		`DELETE FROM shh_users WHERE chat_jid = ? AND user_id = ?`, chatJID, userID,
	)
}

func GetAntilinkMode(chatJID string) string {
	var mode string
	if err := settingsDB.QueryRow(
		`SELECT mode FROM antilink_settings WHERE chat_jid = ?`, chatJID,
	).Scan(&mode); err != nil {
		return "off"
	}
	return mode
}

func SetAntilinkMode(chatJID, mode string) {
	settingsDB.Exec(
		`INSERT INTO antilink_settings (chat_jid, mode) VALUES (?, ?)
		 ON CONFLICT(chat_jid) DO UPDATE SET mode = excluded.mode`,
		chatJID, mode,
	)
}

func GetAntiwords(chatJID string) []string {
	rows, err := settingsDB.Query(
		`SELECT word FROM antiword_settings WHERE chat_jid = ?`, chatJID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var words []string
	for rows.Next() {
		var w string
		if rows.Scan(&w) == nil {
			words = append(words, w)
		}
	}
	return words
}

func AddAntiword(chatJID, word string) {
	settingsDB.Exec(
		`INSERT OR IGNORE INTO antiword_settings (chat_jid, word) VALUES (?, ?)`, chatJID, word,
	)
}

func RemoveAntiword(chatJID, word string) {
	settingsDB.Exec(
		`DELETE FROM antiword_settings WHERE chat_jid = ? AND word = ?`, chatJID, word,
	)
}

func GetAntispamMode(chatJID string) string {
	var mode string
	if err := settingsDB.QueryRow(
		`SELECT mode FROM antispam_settings WHERE chat_jid = ?`, chatJID,
	).Scan(&mode); err != nil {
		return "off"
	}
	return mode
}

func SetAntispamMode(chatJID, mode string) {
	settingsDB.Exec(
		`INSERT INTO antispam_settings (chat_jid, mode) VALUES (?, ?)
		 ON CONFLICT(chat_jid) DO UPDATE SET mode = excluded.mode`,
		chatJID, mode,
	)
}

func IsAntispamWhitelisted(chatJID, userID string) bool {
	var dummy string
	err := settingsDB.QueryRow(
		`SELECT user_id FROM antispam_whitelist WHERE chat_jid = ? AND user_id = ?`, chatJID, userID,
	).Scan(&dummy)
	return err == nil
}

func SetAntispamWhitelist(chatJID, userID string, allow bool) {
	if allow {
		settingsDB.Exec(
			`INSERT OR IGNORE INTO antispam_whitelist (chat_jid, user_id) VALUES (?, ?)`, chatJID, userID,
		)
	} else {
		settingsDB.Exec(
			`DELETE FROM antispam_whitelist WHERE chat_jid = ? AND user_id = ?`, chatJID, userID,
		)
	}
}

func GetAFK(userID string) *AFKStatus {
	var msg string
	var setAt int64
	err := settingsDB.QueryRow(
		`SELECT message, set_at FROM afk_status WHERE user_id = ?`, userID,
	).Scan(&msg, &setAt)
	if err != nil {
		return nil
	}
	return &AFKStatus{Message: msg, SetAt: time.Unix(setAt, 0)}
}

func SetAFK(userID, message string) {
	settingsDB.Exec(
		`INSERT INTO afk_status (user_id, message, set_at) VALUES (?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET message = excluded.message, set_at = excluded.set_at`,
		userID, message, time.Now().Unix(),
	)
}

func ClearAFK(userID string) {
	settingsDB.Exec(`DELETE FROM afk_status WHERE user_id = ?`, userID)
}

func GetFilters(scope, chatJID string) map[string]string {
	rows, err := settingsDB.Query(
		`SELECT keyword, response FROM filters WHERE scope = ? AND chat_jid = ?`, scope, chatJID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	m := map[string]string{}
	for rows.Next() {
		var k, v string
		if rows.Scan(&k, &v) == nil {
			m[k] = v
		}
	}
	return m
}

func SetFilter(scope, chatJID, keyword, response string) {
	settingsDB.Exec(
		`INSERT INTO filters (scope, chat_jid, keyword, response) VALUES (?, ?, ?, ?)
		 ON CONFLICT(scope, chat_jid, keyword) DO UPDATE SET response = excluded.response`,
		scope, chatJID, keyword, response,
	)
}

func DelFilter(scope, chatJID, keyword string) bool {
	res, err := settingsDB.Exec(
		`DELETE FROM filters WHERE scope = ? AND chat_jid = ? AND keyword = ?`, scope, chatJID, keyword,
	)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

func MatchFilter(scope, chatJID, text string) (response string, found bool) {
	rows, err := settingsDB.Query(
		`SELECT keyword, response FROM filters WHERE scope = ? AND chat_jid = ?`, scope, chatJID,
	)
	if err != nil {
		return "", false
	}
	defer rows.Close()
	lower := strings.ToLower(text)
	for rows.Next() {
		var k, v string
		if rows.Scan(&k, &v) == nil {
			if strings.Contains(lower, strings.ToLower(k)) {
				return v, true
			}
		}
	}
	return "", false
}

func GetAntistatusEnabled(chatJID string) bool {
	var enabled int
	settingsDB.QueryRow(`SELECT enabled FROM antistatus_settings WHERE chat_jid = ?`, chatJID).Scan(&enabled)
	return enabled == 1
}

func SetAntistatusEnabled(chatJID string, on bool) {
	v := 0
	if on {
		v = 1
	}
	settingsDB.Exec(
		`INSERT INTO antivv_settings (chat_jid, enabled) VALUES (?, ?)
		 ON CONFLICT(chat_jid) DO UPDATE SET enabled = excluded.enabled`,
		chatJID, v,
	)
}

func SaveMetaMessage(msgID, chatJID, responseText string) {
	settingsDB.Exec(
		`INSERT OR REPLACE INTO meta_messages (msg_id, chat_jid, response_text, created_at) VALUES (?, ?, ?, ?)`,
		msgID, chatJID, responseText, time.Now().Unix(),
	)
}

func GetMetaMessageText(msgID string) (responseText string, found bool) {
	err := settingsDB.QueryRow(
		`SELECT response_text FROM meta_messages WHERE msg_id = ?`, msgID,
	).Scan(&responseText)
	return responseText, err == nil
}

func UpdateMetaMessageText(msgID, responseText string) {
	settingsDB.Exec(
		`UPDATE meta_messages SET response_text = ? WHERE msg_id = ?`,
		responseText, msgID,
	)
}
