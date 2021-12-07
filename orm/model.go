package orm

import (
	"gorm.io/gorm"
	"strings"
)

// Where filed: value 快速输入Where条件
// "name": "abc"  => name = 'abc'
// "name like ?": "%abc%  => name like '%abc%'
// "name in ?": []string{"abc", "cdf"}  => name IN ('abc', 'cdf')
type Where map[string]interface{}

type QuickOrm struct {
	DB           *gorm.DB
	defaultWhere Where
}

func NewQuickOrm(db *gorm.DB) *QuickOrm {
	return &QuickOrm{db, Where{}}
}

func (o *QuickOrm) AddDefaultWhere(field string, value interface{}) *QuickOrm {
	o.defaultWhere[field] = value
	return o
}

func (o *QuickOrm) buildWhere(db *gorm.DB, kv Where) *gorm.DB {
	for k, v := range kv {
		if strings.Contains(k, "?") { // 包含 ? 则说明k 是完整的表达式
			db = db.Where(k, v)
		} else { // 不然一律按照=处理
			db = db.Where(k+" = ?", v)
		}
	}
	return db
}

func (o *QuickOrm) BuildWhere(db *gorm.DB, kv Where) *gorm.DB {
	db = o.buildWhere(db, kv)
	db = o.buildWhere(db, o.defaultWhere)
	return db
}

func (o *QuickOrm) GetModel(kv Where, out interface{}, preloads ...string) (int64, error) {
	db := o.DB
	for _, preload := range preloads {
		db = db.Preload(preload)
	}
	db = o.BuildWhere(db, kv)
	result := db.First(out)
	return result.RowsAffected, result.Error
}

func (o *QuickOrm) GetModels(kv Where, out interface{}, preloads ...string) (int64, error) {
	db := o.DB
	for _, preload := range preloads {
		db = db.Preload(preload)
	}
	db = o.BuildWhere(db, kv)
	result := db.Find(out)
	return result.RowsAffected, result.Error
}

func (o *QuickOrm) GetCount(model interface{}, kv Where) (int64, error) {
	db := o.DB
	var c int64 = 0
	db = db.Model(model)
	db = o.BuildWhere(db, kv)
	err := db.Count(&c).Error
	return c, err
}

func (o *QuickOrm) DeleteModels(models interface{}, kv Where) (int64, error) {
	db := o.DB
	db = o.BuildWhere(db, kv)
	result := db.Delete(models)
	return result.RowsAffected, result.Error
}
