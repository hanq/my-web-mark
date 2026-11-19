package app

import (
	"fmt"

	"my-web-mark/internal/store"
)

// TagService 主题内标签服务（书签二次分类）。
type TagService struct {
	sh *shared
}

// TagInput 创建/更新标签入参。
type TagInput struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

func (in *TagInput) validate() error {
	if in.Name == "" {
		return fmt.Errorf("app: 标签名称不能为空")
	}
	return nil
}

// ListByTheme 返回某主题全部标签。
func (t *TagService) ListByTheme(themeID int64) ([]store.Tag, error) {
	return t.sh.store.ListTagsByTheme(themeID)
}

// ListAll 返回全部主题的标签（2.5D 全局筛选条用）。
func (t *TagService) ListAll() ([]store.Tag, error) {
	return t.sh.store.ListAllTags()
}

// Create 在指定主题下新建标签。
func (t *TagService) Create(themeID int64, in TagInput) (*store.Tag, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	tag := &store.Tag{ThemeID: themeID, Name: in.Name, Color: in.Color}
	if err := t.sh.store.CreateTag(tag); err != nil {
		return nil, err
	}
	return tag, nil
}

// Update 更新标签。
func (t *TagService) Update(id int64, in TagInput) error {
	if err := in.validate(); err != nil {
		return err
	}
	return t.sh.store.UpdateTag(&store.Tag{ID: id, Name: in.Name, Color: in.Color})
}

// Delete 删除标签（书签关联级联清除）。
func (t *TagService) Delete(id int64) error {
	return t.sh.store.DeleteTag(id)
}
