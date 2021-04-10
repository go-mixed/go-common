package utils

import (
	"container/list"
	"sync"
)

type ConcurrencyList struct {
	elements *list.List
	mu       *sync.Mutex
}

func NewConcurrencyList() *ConcurrencyList {
	return &ConcurrencyList{
		elements: list.New(),
		mu:       &sync.Mutex{},
	}
}

// 返回头部元素，不会在列表中删除
func (c *ConcurrencyList) HeadElement() *list.Element {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.elements.Front()
}

// 返回尾部元素，不会在列表中删除
func (c *ConcurrencyList) TailElement() *list.Element {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.elements.Back()
}

// 返回第几个元素，下标从0开始，如果是负数，则从尾部往头部数数
func (c *ConcurrencyList) AtElement(index int) *list.Element {
	c.mu.Lock()
	defer c.mu.Unlock()

	if index >= c.elements.Len() { // 超出范围
		return nil
	} else if index < 0 { // 负数从结尾开始
		index = c.elements.Len() + index
	}

	if index < 0 { // 超出范围
		return nil
	}

	i := 0
	for e := c.elements.Front(); e != nil; e = e.Next() {
		if i == index {
			return e
		}
	}

	return nil
}

// 返回头部的值，不会在列表中删除
func (c *ConcurrencyList) Head() interface{} {
	res := c.HeadElement()

	if res != nil {
		return res.Value
	}
	return nil
}

// 返回尾部的值，不会在列表中删除
func (c *ConcurrencyList) Tail() interface{} {
	res := c.TailElement()

	if res != nil {
		return res.Value
	}
	return nil
}

// 返回第几个值，下标从0开始，如果是负数，则从尾部往头部数数
func (c *ConcurrencyList) At(index int) interface{} {
	res := c.AtElement(index)

	if res != nil {
		return res.Value
	}
	return nil
}

// 添加一个值到列表尾部
func (c *ConcurrencyList) Push(value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.elements.PushBack(value)
}

// 添加一个值到列表头部
func (c *ConcurrencyList) PushHead(value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.elements.PushFront(value)
}

// 弹出头部的值，会在列表中删除该头部的元素
func (c *ConcurrencyList) Pop() interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	e := c.elements.Front()
	if e != nil {
		c.elements.Remove(e)
		return e.Value
	}

	return nil
}

// 弹出尾部的值，会在列表中删除该尾部的元素
func (c *ConcurrencyList) PopTail() interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	e := c.elements.Back()
	if e != nil {
		c.elements.Remove(e)
		return e.Value
	}

	return nil
}

// 移除第一个匹配的值，会从头部依次开始查找，只会删除第一个匹配的值
func (c *ConcurrencyList) Remove(value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for e := c.elements.Front(); e != nil; e = e.Next() {
		if value == e.Value {
			c.elements.Remove(e)
			return
		}
	}
}

// 移除一个元素
func (c *ConcurrencyList) RemoveElement(element *list.Element) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for e := c.elements.Front(); e != nil; e = e.Next() {
		if element == e {
			c.elements.Remove(e)
			return
		}
	}
}

// 返回列表的长度
func (c *ConcurrencyList) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.elements.Len()
}

// 清空列表
func (c *ConcurrencyList) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for {
		e := c.elements.Front()
		if e == nil {
			break
		}
		c.elements.Remove(e)
	}

}
