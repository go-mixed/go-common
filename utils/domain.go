package utils

import (
	"go-common/utils/list"
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
	return If(i > j, i, j).(int)
}

func DomainHasWildCard(d string) bool {
	return DomainIndexOfWildCard(d) >= 0
}

// SortDomains 对域名进行排序
func SortDomains(src interface{}, fn func(v interface{}) string) {
	_src := list.ToInterfaces(src)
	sort.SliceStable(_src, func(i, j int) bool {
		d1 := strings.ToLower(fn(_src[i]))
		d2 := strings.ToLower(fn(_src[j]))
		l1 := len(d1)
		l2 := len(d2)

		minLen := If(l1 < l2, l1, l2).(int)

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

	_ = list.InterfacesAs(_src, src)
}

func (d Domains) IsEmpty() bool {
	return len(d) == 0
}

func (d Domains) Sort() Domains {
	_d := d[:]
	// 按照域名的特有方式进行排序
	SortDomains(&_d, func(v interface{}) string {
		return v.(string)
	})
	return _d
}

func (d Domains) Match(domain string) bool {
	for _, _d := range d {
		if WildcardMatch(_d, domain) {
			return true
		}
	}
	return false
}
