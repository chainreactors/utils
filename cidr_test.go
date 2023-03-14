package ipcs

import (
	"testing"
)

func TestCIDR_Neext(t *testing.T) {
	c := NewCIDR("2001:0:53ab:0:0:0:0:0", 120)
	for i := 0; i < c.max; i++ {
		println(c.Next().String())
	}
}

func BenchmarkCIDR_Next100000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewCIDR("2001:0:53ab:0:0:0:0:0", 120).Next()
	}
}

func TestParseCIDR(t *testing.T) {
	println(ParseCIDR("127.0.0.1").String())
	println(ParseCIDR("::127.0.0.1").String())
	println(ParseCIDR("2001:0:53ab:0:0:0:0:0/120").String())
	println(ParseCIDR("2001:0:c38c:ffff:ffff:0000:0000:ffff").String())
	println(ParseCIDR("2001:0:c38c:ffff:ffff::").String())
	println(ParseCIDR("327.0.0.1"))
	println(ParseCIDR("2001:0:c38c:ffff:ffff:ffff:ffff:ffff1"))
}
