package utils

import "testing"

func TestClassifyISP(t *testing.T) {
	cases := map[string]string{
		"China Unicom Guangdong":       "中国联通",
		"中国联通":                         "中国联通",
		"Chinanet":                     "中国电信",
		"中国电信":                         "中国电信",
		"China Mobile Communications":  "中国移动",
		"中国移动":                         "中国移动",
		"China Education and Research": "中国网络",
		"Google LLC":                   "其他",
	}
	for in, want := range cases {
		if got := classifyISP(in); got != want {
			t.Errorf("classifyISP(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildLocation(t *testing.T) {
	cases := []struct {
		name              string
		isp, region, city string
		country           string
		want              string
	}{
		{"省+市", "中国移动", "湖北省", "武汉市", "", "中国移动/湖北省/武汉市"},
		{"直辖市去重", "中国联通", "北京市", "北京市", "", "中国联通/北京市"},
		{"只有省", "中国电信", "四川省", "", "", "中国电信/四川省"},
		{"只有市", "其他", "", "洛杉矶", "", "其他/洛杉矶"},
		{"只有国家", "其他", "", "", "美国", "其他/美国"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := buildLocation(c.isp, c.region, c.city, c.country); got != c.want {
				t.Errorf("buildLocation(%q,%q,%q,%q) = %q, want %q",
					c.isp, c.region, c.city, c.country, got, c.want)
			}
		})
	}
}

func TestGetIPLocationShortCircuitsLocalIPs(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "localhost", "192.168.1.5", "10.0.0.1"} {
		city, err := GetIPLocation(ip)
		if err != nil {
			t.Fatalf("GetIPLocation(%q) error: %v", ip, err)
		}
		if city != "本地网络" {
			t.Errorf("GetIPLocation(%q) = %q, want 本地网络", ip, city)
		}
	}
}
