package db

import "time"

// ── Meta AI cache ────────────────────────────────────────────────────────────

func SaveMetaPending(key, chatJID, senderJID string, msgID string) {
	settingsDB.Exec(
		`INSERT OR REPLACE INTO meta_pending (key, chat_jid, msg_id, sender_jid, created_at) VALUES (?, ?, ?, ?, ?)`,
		key, chatJID, msgID, senderJID, time.Now().Unix(),
	)
}

type MetaPending struct {
	ChatJID   string
	MsgID     string
	SenderJID string
}

func GetMetaPending(key string) (*MetaPending, bool) {
	var r MetaPending
	err := settingsDB.QueryRow(
		`SELECT chat_jid, msg_id, sender_jid FROM meta_pending WHERE key = ?`, key,
	).Scan(&r.ChatJID, &r.MsgID, &r.SenderJID)
	if err != nil {
		return nil, false
	}
	return &r, true
}

func SaveMetaSentID(responseID, msgID string) {
	settingsDB.Exec(
		`INSERT OR REPLACE INTO meta_sent_ids (response_id, msg_id, created_at) VALUES (?, ?, ?)`,
		responseID, msgID, time.Now().Unix(),
	)
}

func GetMetaSentID(responseID string) (string, bool) {
	var msgID string
	err := settingsDB.QueryRow(
		`SELECT msg_id FROM meta_sent_ids WHERE response_id = ?`, responseID,
	).Scan(&msgID)
	return msgID, err == nil
}

func SaveMetaLastResponse(responseID, text string) {
	settingsDB.Exec(
		`INSERT OR REPLACE INTO meta_last_response (response_id, text, created_at) VALUES (?, ?, ?)`,
		responseID, text, time.Now().Unix(),
	)
}

func GetMetaLastResponse(responseID string) (string, bool) {
	var text string
	err := settingsDB.QueryRow(
		`SELECT text FROM meta_last_response WHERE response_id = ?`, responseID,
	).Scan(&text)
	return text, err == nil
}

func PruneMetaCache(cutoff int64) {
	settingsDB.Exec(`DELETE FROM meta_pending WHERE created_at < ?`, cutoff)
	settingsDB.Exec(`DELETE FROM meta_sent_ids WHERE created_at < ?`, cutoff)
	settingsDB.Exec(`DELETE FROM meta_last_response WHERE created_at < ?`, cutoff)
}

// ── Spam tracking ────────────────────────────────────────────────────────────

func RecordSpamEvent(chatJID, userJID, msgID string) {
	settingsDB.Exec(
		`INSERT INTO spam_events (chat_jid, user_jid, msg_id, ts) VALUES (?, ?, ?, ?)`,
		chatJID, userJID, msgID, time.Now().Unix(),
	)
}

func CountRecentSpam(chatJID, userJID string, windowSecs int64) int {
	cutoff := time.Now().Unix() - windowSecs
	var n int
	settingsDB.QueryRow(
		`SELECT COUNT(*) FROM spam_events WHERE chat_jid = ? AND user_jid = ? AND ts > ?`,
		chatJID, userJID, cutoff,
	).Scan(&n)
	return n
}

func GetRecentSpamMsgIDs(chatJID, userJID string, windowSecs int64) []string {
	cutoff := time.Now().Unix() - windowSecs
	rows, err := settingsDB.Query(
		`SELECT msg_id FROM spam_events WHERE chat_jid = ? AND user_jid = ? AND ts > ?`,
		chatJID, userJID, cutoff,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func PruneSpamEvents(cutoffUnix int64) {
	settingsDB.Exec(`DELETE FROM spam_events WHERE ts < ?`, cutoffUnix)
}

// ── DM Spam tracking ────────────────────────────────────────────────────────

func RecordDMSpamEvent(userJID string) {
	settingsDB.Exec(
		`INSERT INTO dm_spam_events (user_jid, ts) VALUES (?, ?)`,
		userJID, time.Now().Unix(),
	)
}

func CountRecentDMSpam(userJID string, windowSecs int64) int {
	cutoff := time.Now().Unix() - windowSecs
	var n int
	settingsDB.QueryRow(
		`SELECT COUNT(*) FROM dm_spam_events WHERE user_jid = ? AND ts > ?`,
		userJID, cutoff,
	).Scan(&n)
	return n
}

func IsDMSpamWarned(userJID string) bool {
	var n int
	settingsDB.QueryRow(
		`SELECT COUNT(*) FROM dm_spam_warned WHERE user_jid = ?`, userJID,
	).Scan(&n)
	return n > 0
}

func SetDMSpamWarned(userJID string, warned bool) {
	if warned {
		settingsDB.Exec(
			`INSERT OR IGNORE INTO dm_spam_warned (user_jid, ts) VALUES (?, ?)`,
			userJID, time.Now().Unix(),
		)
	} else {
		settingsDB.Exec(`DELETE FROM dm_spam_warned WHERE user_jid = ?`, userJID)
	}
}

func PruneDMSpam(cutoffUnix int64) {
	settingsDB.Exec(`DELETE FROM dm_spam_events WHERE ts < ?`, cutoffUnix)
	settingsDB.Exec(`DELETE FROM dm_spam_warned WHERE ts < ?`, cutoffUnix)
}

// ── AFK cooldown ─────────────────────────────────────────────────────────────

func SetAFKCooldown(userJID string, until time.Time) {
	settingsDB.Exec(
		`INSERT OR REPLACE INTO afk_cooldown (user_jid, until_ts) VALUES (?, ?)`,
		userJID, until.Unix(),
	)
}

func IsAFKCooldownActive(userJID string) bool {
	var untilTS int64
	err := settingsDB.QueryRow(
		`SELECT until_ts FROM afk_cooldown WHERE user_jid = ?`, userJID,
	).Scan(&untilTS)
	if err != nil {
		return false
	}
	return time.Now().Unix() < untilTS
}

func PruneAFKCooldown() {
	settingsDB.Exec(`DELETE FROM afk_cooldown WHERE until_ts < ?`, time.Now().Unix())
}

// ── Anticall warned ──────────────────────────────────────────────────────────

func IsAnticallWarned(phone string) bool {
	var n int
	settingsDB.QueryRow(
		`SELECT COUNT(*) FROM anticall_warned WHERE phone = ?`, phone,
	).Scan(&n)
	return n > 0
}

func SetAnticallWarned(phone string, warned bool) {
	if warned {
		settingsDB.Exec(`INSERT OR IGNORE INTO anticall_warned (phone) VALUES (?)`, phone)
	} else {
		settingsDB.Exec(`DELETE FROM anticall_warned WHERE phone = ?`, phone)
	}
}
