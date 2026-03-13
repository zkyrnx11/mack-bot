package db

func GetAnticallRows() map[string]string {
	rows, err := settingsDB.Query(`SELECT key, value FROM anticall_settings`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := map[string]string{}
	for rows.Next() {
		var k, v string
		if rows.Scan(&k, &v) == nil {
			result[k] = v
		}
	}
	return result
}

func SaveAnticallRows(pairs [][2]string) {
	if settingsDB == nil {
		return
	}
	upsert := `INSERT INTO anticall_settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`
	for _, row := range pairs {
		settingsDB.Exec(upsert, row[0], row[1])
	}
}
