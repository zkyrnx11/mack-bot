package upgrades_test

import (
	"testing"

	"github.com/zkyrnx11/mack/src/store/sqlstore/upgrades"
)

func TestTable_Registered(t *testing.T) {
	if len(upgrades.Table) == 0 {
		t.Error("upgrades.Table has no registered upgrades; expected at least one")
	}
}
