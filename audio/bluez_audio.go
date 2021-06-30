package audio

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"

	dbus "github.com/godbus/dbus"
	bluez "github.com/linuxdeepin/go-dbus-factory/org.bluez"
	"pkg.deepin.io/lib/pulse"
	"pkg.deepin.io/lib/xdg/basedir"
)

const (
	bluezModeA2dp    = "a2dp"
	bluezModeHeadset = "headset"
	bluezModeDefault = bluezModeA2dp
)

/* 蓝牙音频管理器 */
type BluezAudioManager struct {
	BluezAudioConfig map[string]string // cardName => bluezMode

	file string // 配置文件路径
}

/* 创建单例 */
func createBluezAudioManagerSingleton(path string) func() *BluezAudioManager {
	var m *BluezAudioManager = nil
	return func() *BluezAudioManager {
		if m == nil {
			m = NewBluezAudioManager(path)
		}

		return m
	}
}

// 获取单例
// 由于蓝牙模式管理需要在很多个对象中使用，放在Audio对象中需要添加额外参数传递到各个模块很不方便，因此在此创建一个全局的单例
var bluezAudioConfigFilePath = filepath.Join(basedir.GetUserConfigDir(), "deepin/dde-daemon/bluezAudio.json")
var GetBluezAudioManager = createBluezAudioManagerSingleton(bluezAudioConfigFilePath)

/* 创建蓝牙音频管理器 */
func NewBluezAudioManager(path string) *BluezAudioManager {
	return &BluezAudioManager{
		BluezAudioConfig: make(map[string]string),
		file:             path,
	}
}

/* 保存配置 */
func (m *BluezAudioManager) Save() {
	data, err := json.MarshalIndent(m.BluezAudioConfig, "", "  ")
	if err != nil {
		logger.Warning(err)
		return
	}

	err = ioutil.WriteFile(m.file, data, 644)
	if err != nil {
		logger.Warning(err)
		return
	}
}

/* 加载配置 */
func (m *BluezAudioManager) Load() {
	data, err := ioutil.ReadFile(m.file)
	if err != nil {
		logger.Warning(err)
		return
	}

	err = json.Unmarshal(data, m.BluezAudioConfig)
	if err != nil {
		logger.Warning(err)
		return
	}
}

/* 获取模式,这里应该使用 *pulse.Card.Name */
func (m *BluezAudioManager) GetMode(cardName string) string {
	mode, ok := m.BluezAudioConfig[cardName]
	if ok {
		return mode
	} else {
		return bluezModeDefault
	}
}

/* 设置模式，这里应该使用 *pulse.Card.Name */
func (m *BluezAudioManager) SetMode(cardName string, mode string) {
	m.BluezAudioConfig[cardName] = mode
	m.Save()
}

/* 判断设备是否是蓝牙设备，可以用声卡名，也可以用sink、端口等名称 */
func isBluezAudio(name string) bool {
	return strings.Contains(strings.ToLower(name), "bluez")
}

/* 判断蓝牙设备是否是音频设备，参数是bluez设备的DBus路径 */
func isBluezDeviceValid(bluezPath string) bool {
	systemBus, err := dbus.SystemBus()
	if err != nil {
		logger.Warning("[isDeviceValid] dbus connect failed:", err)
		return false
	}
	bluezDevice, err := bluez.NewDevice(systemBus, dbus.ObjectPath(bluezPath))
	if err != nil {
		logger.Warning("[isDeviceValid] new device failed:", err)
		return false
	}
	icon, err := bluezDevice.Icon().Get(0)
	if err != nil {
		logger.Warning("[isDeviceValid] get icon failed:", err)
		return false
	}
	if icon == "computer" {
		return false
	}
	return true
}

/* 设置蓝牙声卡模式 */
func (card *Card) SetBluezMode(mode string) {
	for _, profile := range card.Profiles {
		if strings.Contains(strings.ToLower(profile.Name), strings.ToLower(mode)) &&
			profile.Available != pulse.AvailableTypeNo {
			card.core.SetProfile(profile.Name)
			return
		}
	}
}

/* 自动设置蓝牙声卡的模式 */
func (card *Card) AutoSetBluezMode() {
	mode := GetBluezAudioManager().GetMode(card.core.Name)
	logger.Debugf("card %s auto set bluez mode %s", card.core.Name, mode)
	card.SetBluezMode(mode)
}

/* 获取蓝牙声卡的模式(a2dp/headset) */
func (card *Card) BluezMode() string {
	profileName := strings.ToLower(card.ActiveProfile.Name)
	if strings.Contains(strings.ToLower(profileName), bluezModeA2dp) {
		return bluezModeA2dp
	} else if strings.Contains(strings.ToLower(profileName), bluezModeHeadset) {
		return bluezModeHeadset
	} else {
		return ""
	}
}

/* 获取蓝牙声卡的可用模式 */
func (card *Card) BluezModeOpts() []string {
	opts := []string{}
	for _, profile := range card.Profiles {
		if profile.Available == 0 {
			logger.Debugf("%s %s is unavailable", card.core.Name, profile.Name)
			continue
		}

		if strings.Contains(profile.Description, "HFP") {
			logger.Debugf("%s %s is a HFP profile", card.core.Name, profile.Name)
			continue
		}

		if strings.Contains(strings.ToLower(profile.Name), "a2dp") {
			opts = append(opts, "a2dp")
		}

		if strings.Contains(strings.ToLower(profile.Name), "headset") {
			opts = append(opts, "headset")
		}
	}
	return opts
}
