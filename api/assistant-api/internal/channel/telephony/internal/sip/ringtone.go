package internal_sip_telephony

import (
	"embed"
	"sync"
)

var ringtoneCache sync.Map

const defaultRingtone = "ringtone_us"

//go:embed assets/*.ulaw
var ringtoneAssets embed.FS

func LoadRingtoneBytes(name string) []byte {
	if name == "" {
		name = defaultRingtone
	}
	if cached, ok := ringtoneCache.Load(name); ok {
		if b, ok := cached.([]byte); ok {
			return b
		}
	}
	data, err := ringtoneAssets.ReadFile("assets/" + name + ".ulaw")
	if err != nil {
		if name != defaultRingtone {
			return LoadRingtoneBytes(defaultRingtone)
		}
		return nil
	}
	ringtoneCache.Store(name, data)
	return data
}
