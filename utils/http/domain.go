package http_utils

import (
	"go-common/utils/core"
	"go-common/utils/text"
	"sort"
	"strings"
)

type Domains []string

func DomainIndexOfWildCard(d string) int {
	if len(d) <= 0 {
		return -1
	}
	i := strings.Index(d, "*")
	j := strings.Index(d, "?")
	return core.If(i > j, i, j).(int)
}

func DomainHasWildCard(d string) bool {
	return DomainIndexOfWildCard(d) >= 0
}

// SortDomains 对域名进行排序
func SortDomains[T any](src []T, fn func(v T) string) {
	sort.SliceStable(src, func(i, j int) bool {
		d1 := strings.ToLower(fn(src[i]))
		d2 := strings.ToLower(fn(src[j]))
		l1 := len(d1)
		l2 := len(d2)

		minLen := core.If(l1 < l2, l1, l2).(int)

		// 倒着对比，谁先*, 谁拍后面
		for i := 1; i <= minLen; i++ {
			s1 := d1[l1-i]
			s2 := d2[l2-i]
			if s1 == s2 {
				continue
			} else if s1 == '*' || s1 == '?' { // 通配符靠后
				return false
			} else if s2 == '*' || s2 == '?' {
				return true
			} else {
				return s1 < s2
			}
		}

		// 能运行到这里说明s1[-minLen:] s2[-minLen:]完全相同
		// 此时需要判断s1[:minLen]/s2[:minLen], 即多余的部分是否有通配符，通配符排后
		if DomainHasWildCard(d1[:l1-minLen]) {
			return false
		} else if DomainHasWildCard(d2[:l2-minLen]) {
			return true
		}

		return l1 > l2 // 多余部分没有通配符, 此时看谁更长, 长的排到前面
	})
}

func (d Domains) IsEmpty() bool {
	return len(d) == 0
}

func (d Domains) ToLower() Domains {
	// copy to a new slice
	_d := append(Domains(nil), d...)
	return _d.AsLower()
}

func (d Domains) AsLower() Domains {
	// modify self
	for k, v := range d {
		d[k] = strings.ToLower(v)
	}
	return d
}

func (d Domains) Sort() Domains {
	// copy to a new slice
	_d := make(Domains, len(d))
	copy(_d, d)

	if d.IsEmpty() {
		return _d
	}
	// 按照域名的特有方式进行排序
	SortDomains(_d, func(v string) string {
		return v
	})
	return _d
}

func (d Domains) Match(domain string) (bool, string) {
	domain = strings.ToLower(domain)
	for _, _d := range d {
		if text_utils.WildcardMatch(_d, domain) {
			return true, _d
		}
	}
	return false, ""
}

// IsValidDomain validates if input string is a valid domain name.
func IsValidDomain(host string) bool {
	// See RFC 1035, RFC 3696.
	host = strings.TrimSpace(host)
	if len(host) == 0 || len(host) > 255 {
		return false
	}
	// host cannot start or end with "-"
	if host[len(host)-1:] == "-" || host[:1] == "-" {
		return false
	}
	// host cannot start or end with "_"
	if host[len(host)-1:] == "_" || host[:1] == "_" {
		return false
	}
	// host cannot start with a "."
	if host[:1] == "." {
		return false
	}
	// All non alphanumeric characters are invalid.
	if strings.ContainsAny(host, "`~!@#$%^&*()+={}[]|\\\"';:><?/") {
		return false
	}
	// No need to regexp match, since the list is non-exhaustive.
	// We let it valid and fail later.
	return true
}
