-- v0 -> v13: Latest schema for mack-bot sqlstore (no whatsmeow_ prefix)
-- -- Devices
CREATE TABLE device (
	jid TEXT PRIMARY KEY,
	lid TEXT,
	facebook_uuid TEXT,
	registration_id BIGINT NOT NULL CHECK ( registration_id >= 0 AND registration_id < 4294967296 ),
	noise_key         bytea NOT NULL CHECK ( length(noise_key) = 32 ),
	identity_key      bytea NOT NULL CHECK ( length(identity_key) = 32 ),
	signed_pre_key    bytea NOT NULL CHECK ( length(signed_pre_key) = 32 ),
	signed_pre_key_id INTEGER NOT NULL CHECK ( signed_pre_key_id >= 0 AND signed_pre_key_id < 16777216 ),
	signed_pre_key_sig bytea NOT NULL CHECK ( length(signed_pre_key_sig) = 64 ),
	adv_key             bytea NOT NULL,
	adv_details         bytea NOT NULL,
	adv_account_sig     bytea NOT NULL CHECK ( length(adv_account_sig) = 64 ),
	adv_account_sig_key bytea NOT NULL CHECK ( length(adv_account_sig_key) = 32 ),
	adv_device_sig      bytea NOT NULL CHECK ( length(adv_device_sig) = 64 ),
	platform      TEXT NOT NULL DEFAULT '',
	business_name TEXT NOT NULL DEFAULT '',
	push_name     TEXT NOT NULL DEFAULT '',
	lid_migration_ts BIGINT NOT NULL DEFAULT 0
);

-- Identity keys (Signal)
CREATE TABLE identity_keys (
	our_jid  TEXT,
	their_id TEXT,
	identity bytea NOT NULL CHECK ( length(identity) = 32 ),
	PRIMARY KEY (our_jid, their_id),
	FOREIGN KEY (our_jid) REFERENCES device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Pre-keys
CREATE TABLE pre_keys (
	jid      TEXT,
	key_id   INTEGER,
	key      bytea  NOT NULL,
	uploaded BOOLEAN NOT NULL DEFAULT false,
	PRIMARY KEY (jid, key_id),
	FOREIGN KEY (jid) REFERENCES device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Sessions
CREATE TABLE sessions (
	our_jid  TEXT,
	their_id TEXT,
	session  bytea,
	PRIMARY KEY (our_jid, their_id),
	FOREIGN KEY (our_jid) REFERENCES device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Sender keys (group messaging)
CREATE TABLE sender_keys (
	our_jid    TEXT,
	chat_id    TEXT,
	sender_id  TEXT,
	sender_key bytea NOT NULL,
	PRIMARY KEY (our_jid, chat_id, sender_id),
	FOREIGN KEY (our_jid) REFERENCES device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- App state sync keys
CREATE TABLE app_state_sync_keys (
	jid         TEXT,
	key_id      bytea,
	key_data    bytea NOT NULL,
	timestamp   BIGINT NOT NULL,
	fingerprint bytea NOT NULL,
	PRIMARY KEY (jid, key_id),
	FOREIGN KEY (jid) REFERENCES device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- App state version
CREATE TABLE app_state_version (
	jid  TEXT,
	name TEXT,
	version BIGINT NOT NULL,
	hash    bytea  NOT NULL,
	PRIMARY KEY (jid, name),
	FOREIGN KEY (jid) REFERENCES device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- App state mutation MACs
CREATE TABLE app_state_mutation_macs (
	jid       TEXT,
	name      TEXT,
	version   BIGINT,
	index_mac bytea     NOT NULL,
	value_mac bytea     NOT NULL,
	PRIMARY KEY (jid, name, version, index_mac),
	FOREIGN KEY (jid) REFERENCES device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Contacts
CREATE TABLE contacts (
	our_jid        TEXT,
	their_jid      TEXT,
	first_name     TEXT,
	full_name      TEXT,
	push_name      TEXT,
	business_name  TEXT,
	redacted_phone TEXT,
	PRIMARY KEY (our_jid, their_jid),
	FOREIGN KEY (our_jid) REFERENCES device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Chat settings
CREATE TABLE chat_settings (
	our_jid       TEXT,
	chat_jid      TEXT,
	muted_until   BIGINT NOT NULL DEFAULT 0,
	pinned        BOOLEAN NOT NULL DEFAULT false,
	archived      BOOLEAN NOT NULL DEFAULT false,
	PRIMARY KEY (our_jid, chat_jid),
	FOREIGN KEY (our_jid) REFERENCES device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Message secrets (MSKs)
CREATE TABLE message_secrets (
	our_jid    TEXT,
	chat_jid   TEXT,
	sender_jid TEXT,
	message_id TEXT,
	key        bytea NOT NULL,
	PRIMARY KEY (our_jid, chat_jid, sender_jid, message_id),
	FOREIGN KEY (our_jid) REFERENCES device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Privacy tokens
CREATE TABLE privacy_tokens (
	our_jid   TEXT,
	their_jid TEXT,
	token     bytea  NOT NULL,
	timestamp BIGINT NOT NULL,
	PRIMARY KEY (our_jid, their_jid),
	FOREIGN KEY (our_jid) REFERENCES device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- LID <-> phone number mapping
CREATE TABLE lid_map (
	lid TEXT PRIMARY KEY,
	pn  TEXT NOT NULL UNIQUE
);

-- Retry buffer
CREATE TABLE retry_buffer (
	our_jid    TEXT,
	chat_jid   TEXT,
	message_id TEXT,
	format     INTEGER,
	plaintext  bytea NOT NULL,
	timestamp  BIGINT NOT NULL,
	PRIMARY KEY (our_jid, chat_jid, message_id),
	FOREIGN KEY (our_jid) REFERENCES device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Event buffer (duplicate-message detection)
CREATE TABLE event_buffer (
	our_jid          TEXT,
	ciphertext_hash  bytea,
	plaintext        bytea  NOT NULL,
	server_timestamp BIGINT NOT NULL,
	insert_timestamp BIGINT NOT NULL,
	PRIMARY KEY (our_jid, ciphertext_hash),
	FOREIGN KEY (our_jid) REFERENCES device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);
