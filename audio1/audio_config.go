// SPDX-FileCopyrightText: 2018 - 2022 UnionTech Software Technology Co., Ltd.
//
// SPDX-License-Identifier: GPL-3.0-or-later

package audio

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	dbus "github.com/godbus/dbus/v5"
	soundthemeplayer "github.com/linuxdeepin/go-dbus-factory/system/org.deepin.dde.soundthemeplayer1"
	"github.com/linuxdeepin/go-lib/asound"
)

func (a *Audio) saveConfig() {
	logger.Debug("saveConfig")
	a.saverLocker.Lock()
	if a.isSaving {
		a.saverLocker.Unlock()
		return
	}

	a.isSaving = true
	a.saverLocker.Unlock()

	time.AfterFunc(time.Second*1, func() {
		a.doSaveConfig()

		a.saverLocker.Lock()
		a.isSaving = false
		a.saverLocker.Unlock()
	})
}

func (a *Audio) doSaveConfig() {
	var info = config{
		Profiles: make(map[string]string),
	}

	ctx := a.context()
	if ctx == nil {
		logger.Warning("failed to save config, ctx is nil")
		return
	}

	for _, card := range ctx.GetCardList() {
		info.Profiles[card.Name] = card.ActiveProfile.Name
	}

	for _, sinkInfo := range ctx.GetSinkList() {
		if a.getDefaultSinkName() != sinkInfo.Name {
			continue
		}

		info.Sink = sinkInfo.Name
		info.SinkPort = sinkInfo.ActivePort.Name
		info.SinkVolume = sinkInfo.Volume.Avg()
		break
	}

	for _, sourceInfo := range ctx.GetSourceList() {
		if a.getDefaultSourceName() != sourceInfo.Name {
			continue
		}

		info.Source = sourceInfo.Name
		info.SourcePort = sourceInfo.ActivePort.Name
		info.SourceVolume = sourceInfo.Volume.Avg()
		break
	}
	_, err := readConfig()
	if err != nil && !os.IsNotExist(err) {
		logger.Warning(err)
	}
	if len(info.SourcePort) != 0 {
		err = saveConfig(&info)
		if err != nil {
			logger.Warning("save config file failed:", info.string(), err)
		}
	}
	err = a.saveAudioState()
	if err != nil {
		logger.Warning(err)
	}

}

func (a *Audio) setReduceNoise(enable bool) error {
	logger.Debug("set reduce noise :", enable)
	var err error
	var out []byte
	if enable {
		sourceName := a.getDefaultSourceName()
		logger.Debugf("echoCancelEnable.sh --source_master=%s", sourceName)
		out, err = exec.Command("/usr/share/dde-daemon/audio/echoCancelEnable.sh", "--source_master="+sourceName).CombinedOutput()
		if err != nil {
			logger.Warningf("failed to enable reduce noise %v %s", err, out)
		}
	} else {
		out, err = exec.Command("pactl", "unload-module", "module-echo-cancel").CombinedOutput()
		if err != nil {
			logger.Warningf("failed to disable reduce noise %v %s", err, out)
		}
	}
	return err
}

func (a *Audio) saveAudioState() error {
	sysBus, err := dbus.SystemBus()
	if err != nil {
		return err
	}

	sink := a.getDefaultSink()
	if sink == nil {
		return fmt.Errorf("not found default sink")
	}
	sink.PropsMu.RLock()
	device := sink.props["alsa.device"]
	card := sink.props["alsa.card"]
	mute := sink.Mute
	volume := sink.Volume * 100.0
	sink.PropsMu.RUnlock()

	cardId, err := toALSACardId(card)
	if err != nil {
		return err
	}

	activePlayback := map[string]dbus.Variant{
		"card":   dbus.MakeVariant(cardId),
		"device": dbus.MakeVariant(device),
		"mute":   dbus.MakeVariant(mute),
		"volume": dbus.MakeVariant(volume),
	}

	player := soundthemeplayer.NewSoundThemePlayer(sysBus)
	err = player.SaveAudioState(0, activePlayback)
	return err
}

func toALSACardId(idx string) (cardId string, err error) {
	ctl, err := asound.CTLOpen("hw:"+idx, 0)
	if err != nil {
		return
	}
	defer ctl.Close()

	cardInfo, err := asound.NewCTLCardInfo()
	if err != nil {
		return
	}
	defer cardInfo.Free()

	err = ctl.CardInfo(cardInfo)
	if err != nil {
		return
	}

	cardId = cardInfo.GetID()
	return
}
