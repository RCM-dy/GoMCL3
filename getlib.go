package main

import (
	_ "embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2/data/binding"
	"github.com/RCM-dy/GoMCL3/weblib"
	"github.com/RCM-dy/GoMCL3/zipfile"
	"github.com/tidwall/gjson"
)

//go:embed dumpbin.zip
var dumpbin []byte

func ReadDir(dirname string) ([]fs.FileInfo, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	list, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name() < list[j].Name() })
	return list, nil
}
func GetAllFile(pathname string, s []string) ([]string, error) {
	rd, err := ReadDir(pathname)
	if err != nil {
		return s, err
	}
	for _, fi := range rd {
		if fi.IsDir() {
			fullDir := filepath.Join(pathname, fi.Name())
			s, err = GetAllFile(fullDir, s)
			if err != nil {
				return s, err
			}
		} else {
			fullName := filepath.Join(pathname, fi.Name())
			s = append(s, fullName)
		}
	}
	return s, nil
}
func GetLib(verinfo VersionInfo, outstr binding.String) (cp string, err error) {
	libdir := info.LibDir
	libs := verinfo.Result.Get("libraries")
	clients := verinfo.Result.Get("downloads").Get("client")
	clientURL := clients.Get("url").String()
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
				clientURL = ReplaceByMap(clientURL, map[string]string{
					"https://launchermeta.mojang.com": "https://download.mcbbs.net",
					"https://launcher.mojang.com":     "https://download.mcbbs.net",
					"https://piston-meta.mojang.com":  "https://download.mcbbs.net",
				})
			case "bmclapi":
				clientURL = ReplaceByMap(clientURL, map[string]string{
					"https://launchermeta.mojang.com": "https://bmclapi2.bangbang93.com",
					"https://launcher.mojang.com":     "https://bmclapi2.bangbang93.com",
					"https://piston-meta.mojang.com":  "https://bmclapi2.bangbang93.com",
				})
			}
		}
	}
	clientSHA1 := clients.Get("sha1").String()
	var clientB weblib.Bytes
	clientB, err = weblib.GetBytesFromString(clientURL)
	if err != nil {
		return
	}
	if Sha1Bytes(clientB) != clientSHA1 {
		err = errors.New("hash not same")
		return
	}
	err = os.WriteFile(filepath.Join(info.VerDir, verinfo.Id, verinfo.Id+".jar"), clientB, 0666)
	if err != nil {
		return
	}
	var terr error = nil
	nativesLists := []int{}
	libA := libs.Array()
	for key, value := range libA {
		if !value.IsObject() {
			terr = errors.New("type not need")
			break
		}
		downloads := value.Get("downloads")
		if !downloads.Exists() {
			terr = errors.New("downloads key not found")
			break
		}
		rules := value.Get("rules")
		if rules.Exists() {
			ns := false
			rules.ForEach(func(key, value gjson.Result) bool {
				var notsame bool = false
				allow := true
				value.ForEach(func(key, value gjson.Result) bool {
					if key.String() == "action" {
						if value.String() != "allow" {
							allow = false
							return false
						}
					}
					if key.String() == "features" {
						value.ForEach(func(key, value gjson.Result) bool {
							if key.String() == "is_demo_user" {
								if value.IsBool() {
									if value.Bool() {
										notsame = true
										return false
									}
								}
							}
							if key.String() == "has_custom_resolution" {
								if value.IsBool() {
									if value.Bool() {
										notsame = true
										return false
									}
								}
							}
							return true
						})
					}
					if key.String() == "os" {
						value.ForEach(func(key, value gjson.Result) bool {
							if key.String() == "name" {
								if value.String() != "windows" {
									notsame = true
									return false
								}
								return true
							}
							if key.String() == "arch" {
								if value.String() != GetOsBit() {
									notsame = true
									return false
								}
							}
							return true
						})
					}
					return true
				})
				if !allow {
					return true
				}
				if notsame {
					ns = true
					return false
				}
				return true
			})
			if ns {
				continue
			}
		}
		artifact := downloads.Get("artifact")
		if !artifact.Exists() {
			if downloads.Get("classifiers").Exists() {
				nativesLists = append(nativesLists, key)
			}
			continue
		}
		if downloads.Get("classifiers").Exists() {
			nativesLists = append(nativesLists, key)
		}
		libURL := artifact.Get("url").String()
		if libURL == "" {
			continue
		}
		libpath := filepath.Join(libdir, artifact.Get("path").String())
		libpath = strings.ReplaceAll(libpath, "/", "\\")
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
					libURL = ReplaceByMap(libURL, map[string]string{
						"https://launchermeta.mojang.com": "https://download.mcbbs.net",
						"https://launcher.mojang.com":     "https://download.mcbbs.net",
						"https://piston-meta.mojang.com":  "https://download.mcbbs.net",
						"https://libraries.minecraft.net": "https://download.mcbbs.net/maven",
					})
				case "bmclapi":
					libURL = ReplaceByMap(libURL, map[string]string{
						"https://launchermeta.mojang.com": "https://bmclapi2.bangbang93.com",
						"https://launcher.mojang.com":     "https://bmclapi2.bangbang93.com",
						"https://piston-meta.mojang.com":  "https://bmclapi2.bangbang93.com",
						"https://libraries.minecraft.net": "https://bmclapi2.bangbang93.com/maven",
					})
				}
			}
		}
		terr = os.MkdirAll(filepath.Dir(libpath), 0666)
		if terr != nil {
			break
		}
		var libB weblib.Bytes
		libB, terr = weblib.GetBytesFromString(libURL)
		if terr != nil {
			break
		}
		terr = os.WriteFile(libpath, libB, 0666)
		if terr != nil {
			break
		}
		cp += libpath + ";"
		outstr.Set(libpath)
		time.Sleep(800)
	}
	if terr != nil {
		err = terr
		return
	}
	cp += filepath.Join(info.VerDir, verinfo.Id, verinfo.Id+".jar")
	nativedir := filepath.Join(info.VerDir, verinfo.Id, "natives")
	err = os.MkdirAll(nativedir, 0666)
	if err != nil {
		return
	}
	if len(nativesLists) < 1 {
		return
	}
	for _, v := range nativesLists {
		result := libA[v]
		downloads := result.Get("downloads")
		classifiers := downloads.Get("classifiers")
		if !classifiers.Exists() {
			continue
		}
		winkeyResult := result.Get("natives").Get("windows")
		if !winkeyResult.Exists() {
			continue
		}
		winkey := winkeyResult.String()
		native_win := classifiers.Get(winkey)
		libURL := native_win.Get("url").String()
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
					libURL = ReplaceByMap(libURL, map[string]string{
						"https://launchermeta.mojang.com": "https://download.mcbbs.net",
						"https://launcher.mojang.com":     "https://download.mcbbs.net",
						"https://piston-meta.mojang.com":  "https://download.mcbbs.net",
						"https://libraries.minecraft.net": "https://download.mcbbs.net/maven",
					})
				case "bmclapi":
					libURL = ReplaceByMap(libURL, map[string]string{
						"https://launchermeta.mojang.com": "https://bmclapi2.bangbang93.com",
						"https://launcher.mojang.com":     "https://bmclapi2.bangbang93.com",
						"https://piston-meta.mojang.com":  "https://bmclapi2.bangbang93.com",
						"https://libraries.minecraft.net": "https://bmclapi2.bangbang93.com/maven",
					})
				}
			}
		}
		var nativeB weblib.Bytes
		nativeB, err = weblib.GetBytesFromString(libURL)
		if err != nil {
			return
		}
		err = os.MkdirAll(".\\tmp", 0666)
		if err != nil {
			return
		}
		err = os.WriteFile(".\\tmp\\tmp.jar", nativeB, 0666)
		if err != nil {
			return
		}
		err = zipfile.Unzip(".\\tmp\\tmp.jar", nativedir)
		if err != nil {
			return
		}
		time.Sleep(800)
	}
	var all []string
	all, err = GetAllFile(nativedir, all)
	if err != nil {
		return
	}
	err = os.MkdirAll(".\\dumpbin", 0666)
	if err != nil {
		return
	}
	err = os.WriteFile("dumpbin.zip", dumpbin, 0666)
	if err != nil {
		return
	}
	err = zipfile.Unzip("dumpbin.zip", ".\\dumpbin")
	if err != nil {
		return
	}
	for _, v := range all {
		if filepath.Ext(v) != ".dll" {
			err = os.Remove(v)
			if err != nil {
				return
			}
			continue
		}
		var bit string
		bit, err = GetDllBit(v)
		if err != nil {
			return
		}
		if bit != GetOsBit() {
			err = os.Remove(v)
			if err != nil {
				return
			}
		}
	}
	return
}
