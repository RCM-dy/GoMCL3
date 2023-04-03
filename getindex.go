package main

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/RCM-dy/GoMCL3/messageboxes"
	"github.com/RCM-dy/GoMCL3/weblib"
	"github.com/tidwall/gjson"
)

func GetIndex(verinfo VersionInfo) (gjson.Result, string, error) {
	verResult := verinfo.Result
	assetIndex := verResult.Get("assetIndex")
	indexid := assetIndex.Get("id").String()
	indexURL := assetIndex.Get("url").String()
	indexSHA1 := assetIndex.Get("sha1").String()
	if indexid == "" || indexURL == "" || indexSHA1 == "" {
		return gjson.Result{}, "", errors.New("有一个或多个值为空")
	}
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
				indexURL = ReplaceByMap(indexURL, map[string]string{
					"https://launchermeta.mojang.com/": "https://download.mcbbs.net/",
					"https://launcher.mojang.com/":     "https://download.mcbbs.net/",
					"https://piston-meta.mojang.com/":  "https://download.mcbbs.net/",
				})
			case "bmclapi":
				indexURL = ReplaceByMap(indexURL, map[string]string{
					"https://launchermeta.mojang.com/": "https://bmclapi2.bangbang93.com/",
					"https://launcher.mojang.com/":     "https://bmclapi2.bangbang93.com/",
					"https://piston-meta.mojang.com/":  "https://bmclapi2.bangbang93.com/",
				})
			}
		}
	}
	indexB, err := weblib.GetBytesFromString(indexURL)
	if err != nil {
		messageboxes.ShowInfo("GoMCL3", "文件下载失败")
		return gjson.Result{}, "", err
	}
	if Sha1Bytes(indexB) != indexSHA1 {
		return gjson.Result{}, "", errors.New("hash not same")
	}
	return gjson.ParseBytes(indexB), indexid, os.WriteFile(filepath.Join(info.IndexDir, indexid+".json"), indexB, 0666)
}
