package e2e

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/pubgo/sqlmock"
	"github.com/stretchr/testify/assert"
)

type User struct {
	ID        uint       `gorm:"primaryKey,autoincrement" json:"id,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	Name      string     `gorm:"size:255;not null" json:"name,omitempty" validate:"required"`
	Email     string     `gorm:"size:255;not null;unique" json:"email,omitempty" validate:"required,email"`
}

func (u User) TableName() string {
	return "users"
}

type TestTab struct {
	ID    uint64 `gorm:"column:id"`
	Name  string `gorm:"column:name"`
	CTime uint32 `gorm:"column:ctime"`
	MTime uint32 `gorm:"column:mtime"`
}

func (u TestTab) TableName() string {
	return "test_tabs"
}

func Test_Select(t *testing.T) {
	mock := New(t)

	mock.Find(&TestTab{ID: 1}).
		Return(&TestTab{
			ID:    1,
			Name:  "test",
			CTime: 1630250445,
			MTime: 1630250445,
		})

	var testTab *TestTab
	err := mock.DB().WithContext(context.Background()).Where("id = ?", 1).Find(&testTab).Error
	assert.Nil(t, err)
	assert.NotNil(t, testTab)
	assert.Equal(t, uint64(1), testTab.ID)
	assert.Equal(t, "test", testTab.Name)
	assert.Equal(t, uint32(1630250445), testTab.CTime)
}

func TestCreate(t *testing.T) {
	mock := New(t)

	var n = time.Now()
	u := []*User{
		{
			CreatedAt: &n,
			UpdatedAt: &n,
			Name:      "sheep",
			Email:     "example@gmail.com",
		},
		{
			CreatedAt: &n,
			UpdatedAt: &n,
			Name:      "sheep",
			Email:     "example@gmail.com",
		},
	}

	mock.ExpectBegin()
	mock.Create(u).
		Return(&User{
			ID:   2,
			Name: "sheep",
		})
	mock.ExpectCommit()

	err := mock.DB().Create(u).Error
	assert.NoError(t, err)
	assert.NotNil(t, u)
	assert.Equal(t, u[0].ID, uint(2))
	assert.Equal(t, u[0].Name, "sheep")
}

func TestDelete(t *testing.T) {
	mock := New(t)

	mock.ExpectBegin()
	mock.Delete(&User{Name: "sheep"}).
		ReturnResult(1, 1)
	mock.ExpectCommit()

	ret := mock.DB().Where("name = ?", "sheep").Delete(&User{})
	assert.NoError(t, ret.Error)
	assert.Equal(t, ret.RowsAffected, int64(1))
}

func TestUpdate(t *testing.T) {
	mock := New(t)

	mock.ExpectBegin()
	mock.Update(&User{}).
		WithArgs(sqlmock.Any(), "sheep", "sheep").
		ReturnResult(1, 1)
	opt := mock.DB().Where("name = ?", "sheep").Updates(&User{Name: "sheep"})
	assert.NoError(t, opt.Error)
	assert.Equal(t, opt.RowsAffected, int64(1))
	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.Update(&User{}).
		WithArgsChecker(func(args []driver.Value) error {
			assert.Equal(t, len(args), 3)
			assert.Equal(t, args[1], "sheep")
			assert.Equal(t, args[2], "sheep")
			return nil
		}).
		ReturnResult(1, 1)
	mock.ExpectCommit()

	opt = mock.DB().Where("name = ?", "sheep").Updates(&User{Name: "sheep"})
	assert.NoError(t, opt.Error)
	assert.Equal(t, opt.RowsAffected, int64(1))
}

func TestFindById(t *testing.T) {
	mock := New(t)

	var n = time.Now()
	mock.Find(&User{ID: 1}).
		Return(&User{
			ID:        1,
			Name:      "hello",
			Email:     "example@gmail.com",
			CreatedAt: &n,
			UpdatedAt: &n,
		})

	var user *User
	var err = mock.DB().Where("id = ?", 1).First(&user).Error
	assert.Nil(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, user.Name, "hello")

	mock.Find(&User{}).
		WithArgs(1, 2).
		Return(&User{
			ID:   2,
			Name: "hello1",
		})

	var user1 *User
	err = mock.DB().Select("id").Where("id in ?", []int{1, 2}).First(&user1).Error
	assert.Nil(t, err)
	assert.NotNil(t, user1)
	assert.Equal(t, user1.Name, "hello1")

	mock.Sql(`SELECT "ID" FROM "USERS" WHERE ID = $1`).
		WithArgs(3).
		Return([]*User{
			{
				ID:   2,
				Name: "hello2",
			},
			{
				ID:   3,
				Name: "hello3",
			},
		})

	var user2 []*User
	err = mock.DB().Select("id").Where("id = ?", 3).Find(&user2).Error
	assert.Nil(t, err)
	assert.Equal(t, len(user2), 2)
	assert.Equal(t, user2[1].Name, "hello3")
}
