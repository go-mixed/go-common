package http_utils

import (
	"go-common/utils/core"
	"go-common/utils/list"
	"go-common/utils/text"
	"sort"
	"strings"
)

type Domains []string

type DomainSegment struct {
	ID    uint32     `json:"id"`
	Child DomainTree `json:"child"`
}

type DomainTree map[string]*DomainSegment

func AddDomain(domain string, id uint32, root DomainTree) {
	segments := strings.Split(domain, ".")
	var d = &DomainSegment{}
	var c = root
	var ok bool
	for i := len(segments) - 1; i >= 0; i-- {
		segment := strings.ToLower(segments[i])
		if d, ok = c[segment]; !ok {
			d = &DomainSegment{Child: DomainTree{}}
			c[segment] = d
		}
		c = d.Child
	}
	d.ID = id
}

func AddDomainList(domains map[uint32]string, root DomainTree) {
	for k, v := range domains {
		AddDomain(v, k, root)
	}
}

// 查找域名id
// fuzzy用于模糊查找匹配*， 如果用于域名查找id，用精确查找，如果用户请求host匹配，用模糊查找
func GetDomainId(domain string, root DomainTree, fuzzy bool) uint32 {
	segments := strings.Split(domain, ".")
	var id uint32
	var c = root
	var ok bool
	var d *DomainSegment
	for i := len(segments) - 1; i >= 0; i-- {
		segment := strings.ToLower(segments[i])
		if fuzzy {
			if d, ok = c["*"]; ok {
				if d.ID > 0 {
					id = d.ID
				}
			}
		}
		if d, ok = c[segment]; ok {
			if d.ID > 0 && i == 0 {
				id = d.ID
			}
			c = d.Child
		} else {
			break
		}
	}
	return id
}

// 删除域名
func DelDomain(domain string, root DomainTree) bool {
	segments := strings.Split(domain, ".")
	var c = root
	var ok bool
	var d *DomainSegment
	for i := len(segments) - 1; i >= 0; i-- {
		segment := strings.ToLower(segments[i])
		if d, ok = c[segment]; ok {
			if i == 0 && d.ID > 0 {
				d.ID = 0
				return true
			}
			c = d.Child
		} else {
			break
		}
	}
	return false
}

// 验证域名
func VerifyDomain(domain, patten string) bool {
	pattenTree := DomainTree{}
	AddDomain(patten, 1, pattenTree)
	id := GetDomainId(domain, pattenTree, true)
	return id > 0
}

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
func SortDomains(src interface{}, fn func(v interface{}) string) {
	_src := list_utils.ToInterfaces(src)
	sort.SliceStable(_src, func(i, j int) bool {
		d1 := strings.ToLower(fn(_src[i]))
		d2 := strings.ToLower(fn(_src[j]))
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

	_ = list_utils.InterfacesAs(_src, src)
}

func (d Domains) IsEmpty() bool {
	return len(d) == 0
}

func (d Domains) ToLower() Domains {
	for k, v := range d {
		d[k] = strings.ToLower(v)
	}
	return d
}

func (d Domains) Sort() Domains {
	_d := d[:]

	if d.IsEmpty() {
		return _d
	}
	// 按照域名的特有方式进行排序
	SortDomains(&_d, func(v interface{}) string {
		return v.(string)
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
