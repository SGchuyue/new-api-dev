package model

// 1. 新建 model/user_channel_ratio.go
type UserChannelRatio struct {
	Id        int     `json:"id" gorm:"primaryKey"`
	UserId    int     `json:"user_id" gorm:"index"`
	ChannelId int     `json:"channel_id" gorm:"index"`
	Ratio     float64 `json:"ratio" gorm:"default:1"`
}

func (r *UserChannelRatio) Create() error {
	return DB.Create(r).Error
}

func (r *UserChannelRatio) Update() error {
	return DB.Model(r).Updates(r).Error
}

func GetUserChannelRatio(userId, channelId int) (float64, bool) {
	var ratio UserChannelRatio
	err := DB.Where("user_id = ? AND channel_id = ?", userId, channelId).First(&ratio).Error
	if err != nil {
		return 0, false
	}
	return ratio.Ratio, true
}

func GetUserChannelRatios(userId int) ([]UserChannelRatio, error) {
	var ratios []UserChannelRatio
	err := DB.Where("user_id = ?", userId).Find(&ratios).Error
	return ratios, err
}

func GetAllUserChannelRatios(page, pageSize int) ([]UserChannelRatio, int64, error) {
	var ratios []UserChannelRatio
	var total int64
	DB.Model(&UserChannelRatio{}).Count(&total)
	err := DB.Offset((page - 1) * pageSize).Limit(pageSize).Find(&ratios).Error
	return ratios, total, err
}

func DeleteUserChannelRatio(id int) error {
	return DB.Where("id = ?", id).Delete(&UserChannelRatio{}).Error
}
