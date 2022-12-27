package orm

import (
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"strings"
)

// Where filed: value 快速输入Where条件
//
//	"name": "abc"  => name = 'abc'
//	"name like ?": "%abc%  => name like '%abc%'
//	"name in ?": []string{"abc", "cdf"}  => name IN ('abc', 'cdf')
type Where map[string]any
type KVs map[string]any

type QuickOrm struct {
	DB           *gorm.DB
	defaultWhere Where
}

func NewQuickOrm(db *gorm.DB) *QuickOrm {
	return &QuickOrm{db, Where{}}
}

func (o *QuickOrm) AddDefaultWhere(field string, value any) *QuickOrm {
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

// Load 对于已经读取的模型, 可以使用本函数再次加载关联关系
func (o *QuickOrm) Load(models any, loads ...string) (int64, error) {
	db := o.DB
	for _, load := range loads {
		db = db.Preload(load)
	}
	result := db.Find(models)
	return result.RowsAffected, result.Error
}

// GetModel 获取第一个Model, 如果没有找到, 不会返回错误, 但是第一个返回参数为0
//
//	var user = User{}
//	o.GetModel(Where{"id": 123}, &users) // SELECT * FROM `users` WHERE ID = 123
func (o *QuickOrm) GetModel(kv Where, out any, preloads ...string) (int64, error) {
	db := o.DB
	for _, preload := range preloads {
		db = db.Preload(preload)
	}
	db = o.BuildWhere(db, kv)
	result := db.First(out)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return result.RowsAffected, result.Error
}

// GetModels 获取符合要求的Models列表
//
//	var users []User
//	o.GetModel(Where{"id in ?": []int{1, 2}}, &users) // SELECT * FROM `users` WHERE `ID` IN (1, 2)
func (o *QuickOrm) GetModels(kv Where, out any, preloads ...string) (int64, error) {
	db := o.DB
	for _, preload := range preloads {
		db = db.Preload(preload)
	}
	db = o.BuildWhere(db, kv)
	result := db.Find(out)
	return result.RowsAffected, result.Error
}

func (o *QuickOrm) GetDB() *gorm.DB {
	db := o.DB
	db = o.buildWhere(db, o.defaultWhere)
	return db
}

// GetCount 获取数量
//
//	o.GetModel(&User{}, Where{"name like ?": "%abc%"}) // SELECT COUNT(*) FROM `users` WHERE `name` LIKE '%abc%'
func (o *QuickOrm) GetCount(model any, kv Where) (int64, error) {
	var c int64 = 0
	db := o.DB.Model(model)
	db = o.BuildWhere(db, kv)
	err := db.Count(&c).Error
	return c, err
}

// CreateModel 创建单个model, 可以传递需要被忽略的列名
func (o *QuickOrm) CreateModel(model any, omitColumns ...string) (any, error) {
	if err := o.DB.Omit(omitColumns...).Create(model).Error; err != nil {
		return nil, err
	}
	return model, nil
}

// CreateModels 创建单个model, 可以传递需要被忽略的列名
func (o *QuickOrm) CreateModels(models any, omitColumns ...string) (any, error) {
	if err := o.DB.Omit(omitColumns...).Create(models).Error; err != nil {
		return nil, err
	}
	return models, nil
}

// UpdateModel 修改单个Model。当model有ID值时, where条件会加上 ID = xxx
// 如果model为空对象，会报错：ErrMissingWhereClause
//
//	var user = &User{ID: 1, Name: "", Gender: "", Active": true}
//	o.UpdateModel(&user, KVs{"name": "abc", "gender": "female", "active": false}) // 更新name/gender/active字段
//	o.UpdateModel(&user, User{Name: "abc", Gender: "female", Active: false} // 更新name/gender字段(非零字段)。active字段不会被更新
func (o *QuickOrm) UpdateModel(model any, updateKvs any) (int64, error) {
	return o.UpdateModels(model, nil, updateKvs)
}

// UpdateModels 修改多个Models。当model有ID值时, where条件会加上 ID = xxx
// 默认阻止全局操作，即UPDATE xx SET xx = xx。如果model为空对象，且kv为空，会报错：ErrMissingWhereClause
//
//	var user = &User{ID: 1}
//	o.UpdateModels(user, nil, KVs{"name": "abc"}) // UPDATE `users` SET `name` = 'abc' WHERE `name` = 'abc' AND `id` = 1
//	o.UpdateModels(&User{}, Where{"name": "abc"}, KVs{"gender": "female"}) // UPDATE `users` SET `gender` = 'female' WHERE `name` = 'abc'
func (o *QuickOrm) UpdateModels(model any, kv Where, updateKvs any) (int64, error) {
	var result *gorm.DB
	db := o.DB.Model(model)
	db = o.BuildWhere(db, kv)
	if _, ok := updateKvs.(KVs); ok { // 将KVs类型强制转换为map[string]any
		result = db.Updates(map[string]any(updateKvs.(KVs)))
	} else {
		result = db.Updates(updateKvs)
	}
	return result.RowsAffected, result.Error
}

// DeleteModels 删除指定的models，或者按照条件删除多个。当model有ID值时, where条件会加上 ID = xxx
//
//	o.DeleteModels(&User{ID: 1}) // DELETE FROM `users` WHERE ID = 1
//	o.DeleteModels([]User{User{ID: 1}, User{ID: 2}}) // DELETE FROM `users` WHERE ID IN (1, 2)
//	o.DeleteModels(&User{}, Where{"name like": "%abc%"}) // DELETE FROM `users` WHERE `name` LIKE '%abc%'
//	o.DeleteModels([]*User{User{ID: 1}, User{ID: 2}}, Where{"name like": "%abc%"}) // DELETE FROM `users` WHERE `ID` IN (1, 2) AND `name` LIKE '%abc%'
func (o *QuickOrm) DeleteModels(models any, kv Where) (int64, error) {
	db := o.DB
	db = o.BuildWhere(db, kv)
	result := db.Delete(models)
	return result.RowsAffected, result.Error
}
