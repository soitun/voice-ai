package internal_sip_telephony

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

var ringtoneCache sync.Map

const defaultRingtone = "ringtone_us"

func LoadRingtoneBytes(name string) []byte {
	if name == "" {
		name = defaultRingtone
	}
	if cached, ok := ringtoneCache.Load(name); ok {
		if b, ok := cached.([]byte); ok {
			return b
		}
	}
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil
	}
	assetsDir := filepath.Join(filepath.Dir(thisFile), "assets")
	fullPath := filepath.Join(assetsDir, name+".ulaw")

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if name != defaultRingtone {
			return LoadRingtoneBytes(defaultRingtone)
		}
		return nil
	}
	ringtoneCache.Store(name, data)
	return data
}
