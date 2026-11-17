package app

import (
	"fmt"

	"my-web-mark/internal/crypto"
	"my-web-mark/internal/store"
)

// VaultService 管理员密码（金库）服务。
type VaultService struct {
	sh *shared
}

// VaultStatus 前端展示金库状态。
type VaultStatus struct {
	SetupRequired bool `json:"setupRequired"` // 尚未设置管理员密码
	Unlocked      bool `json:"unlocked"`      // 当前是否解锁
}

// Status 返回金库当前状态。
func (v *VaultService) Status() VaultStatus {
	has, err := v.sh.store.HasVaultMeta()
	return VaultStatus{
		SetupRequired: err == nil && !has,
		Unlocked:      v.isUnlocked(),
	}
}

func (v *VaultService) isUnlocked() bool {
	v.sh.mu.RLock()
	defer v.sh.mu.RUnlock()
	return v.sh.unlocked
}

// Setup 首次设置管理员密码。密码强度由前端校验，此处仅要求非空且 >= 6 位。
func (v *VaultService) Setup(password string) error {
	if len(password) < 6 {
		return fmt.Errorf("app: 管理员密码至少 6 位")
	}
	if has, _ := v.sh.store.HasVaultMeta(); has {
		return ErrVaultAlreadySetup
	}
	salt, err := crypto.NewSalt()
	if err != nil {
		return err
	}
	params := crypto.DefaultKDFParams()
	hash := crypto.HashPassword(password, salt, params)
	paramsJSON, err := crypto.MarshalParams(params)
	if err != nil {
		return err
	}
	if err := v.sh.store.SaveVaultMeta(&store.VaultMeta{
		Salt: salt, PasswordHash: hash, KDFParams: paramsJSON,
	}); err != nil {
		return err
	}
	// 设置完成后立即解锁，方便继续录入
	_, key := crypto.DeriveKey(password, salt, params)
	v.sh.mu.Lock()
	v.sh.aesKey = key
	v.sh.unlocked = true
	v.sh.mu.Unlock()
	return nil
}

// UnlockInput 解锁入参；空密码直接校验失败。
type UnlockInput struct {
	Password string `json:"password"`
}

// Unlock 用管理员密码解锁金库；成功后内存持有派生密钥。
func (v *VaultService) Unlock(in UnlockInput) error {
	if v.isUnlocked() {
		return ErrVaultAlreadyUnlocked
	}
	meta, err := v.sh.store.GetVaultMeta()
	if err == store.ErrNotFound {
		return ErrVaultSetupRequired
	}
	if err != nil {
		return err
	}
	params := crypto.UnmarshalParams(meta.KDFParams)
	if !crypto.VerifyPassword(in.Password, meta.Salt, meta.PasswordHash, params) {
		return fmt.Errorf("app: 管理员密码错误")
	}
	_, key := crypto.DeriveKey(in.Password, meta.Salt, params)
	v.sh.mu.Lock()
	v.sh.aesKey = key
	v.sh.unlocked = true
	v.sh.mu.Unlock()
	return nil
}

// Lock 锁定金库并清零内存密钥。
func (v *VaultService) Lock() {
	v.sh.mu.Lock()
	defer v.sh.mu.Unlock()
	if v.sh.aesKey != nil {
		crypto.WipeKey(v.sh.aesKey)
	}
	v.sh.aesKey = nil
	v.sh.unlocked = false
}

// ChangePasswordInput 修改管理员密码入参。
type ChangePasswordInput struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// ChangePassword 修改管理员密码，并用新派生密钥重加密全部账号密码。
// 整体失败时旧数据保持不变（重加密在事务中回滚）。
func (v *VaultService) ChangePassword(in ChangePasswordInput) error {
	if len(in.NewPassword) < 6 {
		return fmt.Errorf("app: 新密码至少 6 位")
	}
	meta, err := v.sh.store.GetVaultMeta()
	if err == store.ErrNotFound {
		return ErrVaultSetupRequired
	}
	if err != nil {
		return err
	}
	oldParams := crypto.UnmarshalParams(meta.KDFParams)
	if !crypto.VerifyPassword(in.OldPassword, meta.Salt, meta.PasswordHash, oldParams) {
		return fmt.Errorf("app: 原管理员密码错误")
	}

	// 校验当前金库必须已解锁（重加密需要旧密钥）
	if !v.isUnlocked() {
		return ErrVaultLocked
	}
	v.sh.mu.RLock()
	oldKey := make([]byte, len(v.sh.aesKey))
	copy(oldKey, v.sh.aesKey)
	v.sh.mu.RUnlock()

	newSalt, err := crypto.NewSalt()
	if err != nil {
		return err
	}
	newParams := crypto.DefaultKDFParams()
	newHash := crypto.HashPassword(in.NewPassword, newSalt, newParams)
	_, newKey := crypto.DeriveKey(in.NewPassword, newSalt, newParams)
	newParamsJSON, _ := crypto.MarshalParams(newParams)

	// 事务内用新密钥重加密所有账号
	err = v.sh.store.RekeyAccounts(func(_ int64, cipher []byte) ([]byte, error) {
		plain, err := crypto.Decrypt(oldKey, cipher)
		if err != nil {
			return nil, err
		}
		return crypto.Encrypt(newKey, plain)
	})
	if err != nil {
		return fmt.Errorf("app: 重加密账号数据失败: %w", err)
	}

	// 保存新凭据
	if err := v.sh.store.SaveVaultMeta(&store.VaultMeta{
		Salt: newSalt, PasswordHash: newHash, KDFParams: newParamsJSON,
	}); err != nil {
		return err
	}

	// 切换内存密钥
	crypto.WipeKey(oldKey)
	v.sh.mu.Lock()
	v.sh.aesKey = newKey
	v.sh.mu.Unlock()
	return nil
}
