package ip

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

type Ranger struct {
	// 采用map查找的方式，map和slice相比，slice在少量的时候速度较快，map采用的hash，在大量数据的时候较快
	// 原理是一个ip在某一个掩码段转化的只有一个ip值，通过从掩码从小到大查找，并转换唯一的ip值，可以快速索引定位
	Data map[int]map[string]bool
	// 查找已存在的掩码范围，避免不必要的cidr转换开销，并且排序
	Bits []int
}

func NewIpRanger() *Ranger {
	return &Ranger{make(map[int]map[string]bool), []int{}}
}

func ParseIP(ip string) (*net.IPNet, error) {
	if strings.Index(ip, "/") < 0 {
		ip = fmt.Sprintf("%s/32", ip)
	}
	_, _net, err := net.ParseCIDR(ip)
	return _net, err
}

func (r *Ranger) getIpByMask(ip string, mask int) string {
	ip = fmt.Sprintf("%s/%d", ip, mask)
	_, _net, _ := net.ParseCIDR(ip)
	return _net.IP.String()
}

func (r *Ranger) AddIP(n *net.IPNet) {
	bit, _ := n.Mask.Size()
	if _, ok := r.Data[bit]; !ok {
		r.Data[bit] = make(map[string]bool)
		r.addBit(bit)
	}
	r.Data[bit][n.IP.String()] = true
}

func (r *Ranger) RemoveIP(n *net.IPNet) error {
	bit, _ := n.Mask.Size()
	if d, ok := r.Data[bit]; ok {
		if _, ok := d[n.IP.String()]; ok {
			delete(r.Data[bit], n.IP.String())
			if len(r.Data[bit]) == 0 {
				delete(r.Data, bit)
				r.delIndex(bit)
			}
		} else {
			return errors.New("ip not found")
		}
	} else {
		return errors.New("ip not found")
	}
	return nil
}

func (r *Ranger) getIndex(bit int) int {
	var index = -1
	var arr = r.Bits
	for len(arr) > 0 {
		mid := len(arr) / 2
		if bit == arr[mid] {
			index += mid + 1
			break
		} else if bit < arr[mid] {
			arr = arr[:mid]
		} else {
			index += mid + 1
			arr = arr[mid+1:]
		}
	}
	return index
}

// 采用二分查找法添加
func (r *Ranger) addBit(bit int) int {
	index := r.getIndex(bit)
	if index < 0 {
		r.Bits = append([]int{bit}, r.Bits...)
	} else {
		if r.Bits[index] == bit {
			return index
		} else if index == (len(r.Bits) - 1) {
			r.Bits = append(r.Bits, bit)
			return index + 1
		} else {
			last := append([]int{}, r.Bits[index+1:]...)
			r.Bits = append(r.Bits[0:index+1], bit)
			r.Bits = append(r.Bits, last...)
		}
	}
	return index
}

// 采用二分查找法添加
func (r *Ranger) delIndex(bit int) int {
	index := r.getIndex(bit)
	if r.Bits[index] == bit {
		if index == 0 {
			r.Bits = r.Bits[1:]
		} else if index == (len(r.Bits) - 1) {
			r.Bits = r.Bits[:len(r.Bits)-1]
		} else {
			r.Bits = append(r.Bits[:index], r.Bits[index+1:]...)
		}
	}
	return index
}

func (r *Ranger) Contains(n *net.IPNet) bool {
	var exist bool
	_mask, _ := n.Mask.Size()
	for _, i := range r.Bits {
		_ip := r.getIpByMask(n.IP.String(), i)
		if _, ok := r.Data[i][_ip]; ok {
			return true
		}
		if _mask < i {
			break
		}
	}
	return exist
}
