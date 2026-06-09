# Step 8：Repository 层

> **目标**：创建数据库操作层（Repository），隔离 GORM 调用，为 Service 层提供接口。
> 参考 `steps-v1/01-基础设施先行.md` 的 1.4 节 + `steps-v1/05-基础业务FundService.md`。

---

## 8.1 为什么要加 Repository 层

直接在任何地方调用 `db.GORM.Where(...)` 会导致：
- 数据库操作散落在 handler、service 各处
- 测试时很难 mock（你需要真的 PostgreSQL）
- 换数据库或改查询时到处都要改

Repository 层把所有数据库操作集中在一起，上层通过接口调用。

## 8.2 UserRepository

```go
// internal/repository/user_repo.go
package repository

import (
    "context"
    "errors"
    "huahua-service/internal/model"
    "gorm.io/gorm"
)

type UserRepository interface {
    GetByID(ctx context.Context, id int64) (*model.User, error)
    GetByUsername(ctx context.Context, username string) (*model.User, error)
    GetByEmail(ctx context.Context, email string) (*model.User, error)
    GetByUID(ctx context.Context, uid string) (*model.User, error)
    Create(ctx context.Context, user *model.User) error
    Update(ctx context.Context, user *model.User) error
}

type userRepo struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
    return &userRepo{db: db}
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
    var user model.User
    err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, nil
    }
    return &user, err
}

func (r *userRepo) Create(ctx context.Context, user *model.User) error {
    return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepo) Update(ctx context.Context, user *model.User) error {
    return r.db.WithContext(ctx).Save(user).Error
}

// 其他方法类似......
```

## 8.3 FundBasicInfoRepository

```go
// internal/repository/fund_basic_info_repo.go
type FundBasicInfoRepository interface {
    GetByCode(ctx context.Context, code string) (*model.FundBasicInfo, error)
    FindAll(ctx context.Context) ([]model.FundBasicInfo, error)
}
```

## 8.4 FundNavRepository

```go
// internal/repository/fund_nav_repo.go
type FundNavRepository interface {
    GetByCode(ctx context.Context, code string, limit int) ([]model.FundNav, error)
    GetByDate(ctx context.Context, code string, date string) (*model.FundNav, error)
}
```

## 8.5 接口 + 实现的设计模式

```
UserRepository  (接口)    ← 业务层只依赖接口
       ↓
userRepo        (实现)    ← 实现里写真正的 GORM 代码
       ↓
MockUserRepo    (测试用)  ← 测试时替换成内存 map
```

**测试时的好处**：
```go
// 测试 auth service 时不需要真实数据库
userRepo := repository.NewMockUserRepository()
authService := auth.NewAuthService(nil, userRepo, jwtCfg)
```

---

## 本步总结

- ✅ UserRepository（用户 CRUD）
- ✅ FundBasicInfoRepository（基金信息）
- ✅ FundNavRepository（历史净值）
- ✅ 接口 + 实现模式，方便测试 mock
