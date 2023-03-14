package ipcs

import (
	"testing"
)

func TestParseIP(t *testing.T) {
	println(ParseIP("127.0.0.1").String())
	println(ParseIP("::127.0.0.1").String())
	println(ParseIP("2001:0:53ab:0:0:0:0:0").String())
	println(ParseIP("2001:0:c38c:ffff:ffff:0000:0000:ffff").String())
	println(ParseIP("2001:0:c38c:ffff:ffff::").String())
	println(ParseIP("327.0.0.1"))
	println(ParseIP("2001:0:c38c:ffff:ffff:ffff:ffff:ffff1"))
}
