package parsers

import (
	"encoding/json"
	"github.com/chainreactors/utils/fileutils"
	"testing"
)

func TestResultsData_ToCsv(t *testing.T) {
	content := fileutils.LoadCommonArg("2.dat1")
	var results GOGOData
	err := json.Unmarshal(content, &results)
	if err != nil {
		println(err.Error())
		return
	}
	println(results.ToZombie())
}
