package slice

import "strings"

type String struct {
	Values []string
}

func NewSliceString(s []string) *String {
	n := QuickSortString(s)
	return &String{n}
}

// QuickSortString 快速排序
func QuickSortString(arr []string) []string {
	if len(arr) <= 1 {
		return arr
	}
	splitdata := arr[0]           //第一个数据
	low := make([]string, 0, 0)   //比我小的数据
	hight := make([]string, 0, 0) //比我大的数据
	mid := make([]string, 0, 0)   //与我一样大的数据
	mid = append(mid, splitdata)  //加入一个
	for i := 1; i < len(arr); i++ {
		if strings.Compare(arr[i], splitdata) < 0 {
			low = append(low, arr[i])
		} else if strings.Compare(arr[i], splitdata) > 0 {
			hight = append(hight, arr[i])
		} else {
			mid = append(mid, arr[i])
		}
	}
	low, hight = QuickSortString(low), QuickSortString(hight)
	myarr := append(append(low, mid...), hight...)
	return myarr
}

func (r *String) GetIndex(s string) int {
	var index = -1
	var arr = r.Values
	var step = 1
	for len(arr) > 0 {
		mid := len(arr) / 2
		if strings.Compare(s, arr[mid]) == 0 {
			index += mid + 1
			break
		} else if strings.Compare(s, arr[mid]) < 0 {
			arr = arr[:mid]
		} else {
			index += mid + 1
			arr = arr[mid+1:]
		}
		step += 1
	}
	return index
}

func (r *String) Contains(k string) bool {
	index := r.GetIndex(k)
	if index >= 0 && index < len(r.Values) && r.Values[index] == k {
		return true
	}
	return false
}
