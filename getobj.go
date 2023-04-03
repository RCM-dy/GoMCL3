package main

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2/data/binding"
	"github.com/RCM-dy/GoMCL3/weblib"
	"github.com/tidwall/gjson"
)

func GetObj(indexes gjson.Result, outstr binding.String) (err error) {
	err = os.RemoveAll(info.ObjDir)
	if err != nil {
		return
	}
	err = os.RemoveAll(info.ObjBackupDir)
	if err != nil {
		return
	}
	map_to_resources := indexes.Get("map_to_resources")
	needbkp := false
	if map_to_resources.Exists() {
		if map_to_resources.IsBool() && map_to_resources.Bool() {
			needbkp = true
		}
	}
	obj := indexes.Get("objects")
	if !obj.IsObject() {
		return errors.New("type not need")
	}
	obj.ForEach(func(key, value gjson.Result) bool {
		objbkpPath := filepath.Join(info.ObjBackupDir, key.String())
		objbkpPath = ReplaceByMap(objbkpPath, map[string]string{
			"/": "\\",
		})
		hash := value.Get("hash").String()
		two := hash[:2]
		objdirs := filepath.Join(info.ObjDir, two)
		err = os.MkdirAll(objdirs, 0666)
		if err != nil {
			return false
		}
		objpath := filepath.Join(objdirs, hash)
		var rootURL string = "https://resources.download.minecraft.net/"
		if info.NeedReplaceURL() {
			isOneOfTheDefault := false
			for _, v := range defaultSources {
				if v == info.SourceName {
					isOneOfTheDefault = true
					break
				}
			}
			if isOneOfTheDefault {
				switch info.SourceName {
				case "mcbbs":
					rootURL = "https://download.mcbbs.net/assets/"
				case "bmclapi":
					rootURL = "https://bmclapi2.bangbang93.com/assets/"
				}
			}
		}
		uRL := rootURL + two + "/" + hash
		var b weblib.Bytes
		b, err = weblib.GetBytesFromString(uRL)
		if err != nil {
			return false
		}
		if Sha1Bytes(b) != hash {
			err = errors.New("hash not same")
			return false
		}
		err = os.WriteFile(objpath, b, 0666)
		if err != nil {
			return false
		}
		if needbkp {
			err = os.MkdirAll(filepath.Dir(objbkpPath), 0666)
			if err != nil {
				return false
			}
			err = os.WriteFile(objbkpPath, b, 0666)
			if err != nil {
				return false
			}
		}
		outstr.Set(hash)
		time.Sleep(800)
		return true
	})
	return
}
