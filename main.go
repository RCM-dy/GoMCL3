package main

//go:generate goversioninfo

import (
	"bytes"
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/RCM-dy/GoMCL3/messageboxes"
	"github.com/RCM-dy/GoMCL3/theme"
	"github.com/RCM-dy/GoMCL3/weblib"
	"github.com/tidwall/gjson"
)

type VersionInfo struct {
	Id     string
	types  string
	Result gjson.Result
}

type UserInfo struct {
	UUID string
	Name string
}

type Info struct {
	SourceName  string
	Users       *UserInfo
	AccessToken string

	Workpath     string
	McDir        string
	VerDir       string
	AssetsDir    string
	IndexDir     string
	ObjDir       string
	ObjBackupDir string
	LibDir       string
}

func (i Info) NeedReplaceURL() bool {
	return i.SourceName != "mojang"
}

var info *Info = &Info{}

func init() {
	sourceNameb, err := os.ReadFile("SOURCE")
	if err != nil {
		if os.IsNotExist(err) {
			info.SourceName = "mojang"
		} else {
			messageboxes.ShowInfo("GoMCL3", err.Error())
			os.Exit(1)
		}
	} else {
		info.SourceName = string(sourceNameb)
	}
	w, err := os.Getwd()
	if err != nil {
		messageboxes.ShowInfo("GoMCL3", err.Error())
		os.Exit(1)
	}
	info.Workpath = w
	info.McDir = filepath.Join(w, ".minecraft")
	info.VerDir = filepath.Join(info.McDir, "versions")
	info.AssetsDir = filepath.Join(info.McDir, "assets")
	info.IndexDir = filepath.Join(info.AssetsDir, "indexes")
	info.ObjDir = filepath.Join(info.AssetsDir, "objects")
	info.ObjBackupDir = filepath.Join(info.AssetsDir, "virtual", "legacy")
	info.LibDir = filepath.Join(info.McDir, "libraries")
	err = MKdirs(
		info.McDir,
		info.VerDir,
		info.AssetsDir,
		info.IndexDir,
		info.ObjDir,
		info.ObjBackupDir,
		info.LibDir,
	)
	if err != nil {
		messageboxes.ShowInfo("GoMCL3", err.Error())
		os.Exit(1)
	}
}

var apps = app.NewWithID("com.github.minecraftlaun.GoMCL3")

func main() {
	apps.Settings().SetTheme(&theme.MyTheme{})
	w := apps.NewWindow("Golang MCL3")
	w.Resize(fyne.NewSize(775, 435))
	w.Show()
	w.SetOnClosed(func() {
		apps.Quit()
	})
	mainicon, err := fyne.LoadResourceFromURLString("https://cdn.img.z0z0r4.top/2023/03/31/6426b0e00511d.png")
	if err != nil {
		println(err.Error())
		return
	}
	w.SetIcon(mainicon)
	var version_manifest_v2_URL string = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"
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
				version_manifest_v2_URL = "https://download.mcbbs.net/mc/game/version_manifest_v2.json"
			case "bmclapi":
				version_manifest_v2_URL = "https://bmclapi2.bangbang93.com/mc/game/version_manifest_v2.json"
			}
		}
	}
	version_manifest_v2, err := weblib.GetBytesFromString(version_manifest_v2_URL)
	if err != nil {
		println(err.Error())
		return
	}
	versionids := gjson.GetBytes(version_manifest_v2, "versions.#.id|@ugly").String()
	versionids = strings.ReplaceAll(versionids, "[", "")
	versionids = strings.ReplaceAll(versionids, "]", "")
	versionids = strings.ReplaceAll(versionids, "\"", "")
	veridlists := strings.Split(versionids, ",")
	verchooseEntry := widget.NewSelectEntry(veridlists)
	outStr := binding.NewString()
	outlabel := widget.NewLabelWithData(outStr)
	w.SetContent(container.NewAppTabs(
		container.NewTabItem("下载", container.NewHBox(
			container.NewVBox(
				verchooseEntry,
				widget.NewButton("       确定       ", func() {
					needVer := verchooseEntry.Text
					messageboxes.ShowInfo("GoMCL3", "开始下载\n版本: "+needVer)
					verinfos := gjson.GetBytes(version_manifest_v2, "versions.#(id==\""+needVer+"\")")
					if !verinfos.Exists() {
						messageboxes.ShowInfo("GOMCL3", "下载失败\n原因: 版本没有找到")
						return
					}
					verURL := verinfos.Get("url").String()
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
								verURL = ReplaceByMap(verURL, map[string]string{
									"https://launchermeta.mojang.com": "https://download.mcbbs.net",
									"https://launcher.mojang.com":     "https://download.mcbbs.net",
									"https://piston-meta.mojang.com":  "https://download.mcbbs.net",
								})
							case "bmclapi":
								verURL = ReplaceByMap(verURL, map[string]string{
									"https://launchermeta.mojang.com": "https://bmclapi2.bangbang93.com",
									"https://launcher.mojang.com":     "https://bmclapi2.bangbang93.com",
									"https://piston-meta.mojang.com":  "https://bmclapi2.bangbang93.com",
								})
							}
						}
					}
					verjsonB, err := weblib.GetBytesFromString(verURL)
					if err != nil {
						messageboxes.ShowInfo("GOMCL3", "下载失败\n版本信息获取失败")
						outlabel.SetText("")
						return
					}
					if Sha1Bytes(verjsonB) != verinfos.Get("sha1").String() {
						messageboxes.ShowInfo("GOMCL3", "下载失败\n版本信息hash校验失败")
						outlabel.SetText("")
						return
					}
					verjsonDir := filepath.Join(info.VerDir, needVer)
					err = os.MkdirAll(verjsonDir, 0666)
					if err != nil {
						messageboxes.ShowInfo("GOMCL3", "下载失败\n文件夹创建失败")
						outlabel.SetText("")
						return
					}
					err = os.WriteFile(filepath.Join(verjsonDir, needVer+".json"), verjsonB, 0666)
					if err != nil {
						messageboxes.ShowInfo("GOMCL3", "下载失败\n原因: "+err.Error())
						outlabel.SetText("")
						return
					}
					verjson := string(verjsonB)
					verInfo := VersionInfo{Id: needVer, types: gjson.Get(verjson, "type").String(), Result: gjson.Parse(verjson)}
					_, _, err = GetIndex(verInfo)
					if err != nil {
						messageboxes.ShowInfo("GOMCL3", "下载失败\n原因: "+err.Error())
						outlabel.SetText("")
						return
					}
					outlabel.SetText("开始下载资源库文件")
					time.Sleep(5 * time.Second)
					cp, err := GetLib(verInfo, outStr)
					if err != nil {
						messageboxes.ShowInfo("GOMCL3", "下载失败\n原因: "+err.Error())
						outlabel.SetText("")
						return
					}
					args, err := JointArgs(verInfo, cp)
					if err != nil {
						messageboxes.ShowInfo("GOMCL3", "下载失败\n原因: "+err.Error())
						outlabel.SetText("")
						return
					}
					err = os.WriteFile(verInfo.Id+"-backup.args", []byte(args), 0666)
					if err != nil {
						messageboxes.ShowInfo("GOMCL3", "下载失败\n原因: "+err.Error())
						outlabel.SetText("")
						return
					}
					w.Resize(fyne.NewSize(500, 300))
					outlabel.SetText("")
					messageboxes.ShowInfo("GoMCL3", "下载完成")
				}),
				outlabel,
			),
		)),
		container.NewTabItem("启动", container.NewVBox(
			verchooseEntry,
			widget.NewButton("开始", func() {
				needVer := verchooseEntry.Text
				messageboxes.ShowInfo("GoMCL3", "开始启动\n版本: "+needVer)
				argspath := needVer + "-backup.args"
				argB, err := os.ReadFile(argspath)
				if err != nil {
					messageboxes.ShowInfo("GOMCL3", "启动失败\n原因: "+err.Error())
					return
				}
				verjsonpath := filepath.Join(info.VerDir, needVer, needVer+".json")
				verjson, err := os.ReadFile(verjsonpath)
				if err != nil {
					messageboxes.ShowInfo("GOMCL3", "启动失败\n原因: "+err.Error())
					return
				}
				indexB, err := os.ReadFile(filepath.Join(info.IndexDir, gjson.GetBytes(verjson, "assets").String()+".json"))
				if err != nil {
					messageboxes.ShowInfo("GOMCL3", "启动失败\n原因: "+err.Error())
					return
				}
				indexes := gjson.ParseBytes(indexB)
				err = GetObj(indexes, outStr)
				if err != nil {
					messageboxes.ShowInfo("GOMCL3", "启动失败\n原因: "+err.Error())
					return
				}
				args := string(argB)
				if info.Users == nil {
					messageboxes.ShowInfo("GoMCL3", "开始登录")
					var (
						un    string
						token string
						uuid  string
						err   error
					)
					un, uuid, token, err = MSLogin()
					if err != nil {
						panic(err)
					}
					info.Users = &UserInfo{UUID: uuid, Name: un}
					info.AccessToken = token
				}
				args = ReplaceByMap(args, map[string]string{
					"${auth_player_name}":  info.Users.Name,
					"${auth_uuid}":         info.Users.UUID,
					"${auth_access_token}": info.AccessToken,
					"${user_type}":         "msa",
					"-Dos.name=Windows 10": "-Dos.name=\"Windows 10\"",
				})
				err = os.WriteFile("start.bat", []byte(args), 0666)
				if err != nil {
					messageboxes.ShowInfo("GOMCL3", "启动失败\n原因: "+err.Error())
					return
				}
				outlabel.SetText("")
				cmd := exec.Command("cmd.exe", "/c", "start.bat")
				out := bytes.NewBuffer(nil)
				cmd.Stderr = out
				cmd.Stdout = out
				cmd.Start()
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
				cmd.Wait()
			}),
			outlabel,
		)),
		container.NewTabItem("更多", container.NewVBox(
			widget.NewButton("退出", func() {
				w.Close()
			}),
			widget.NewButton("获取下载源", func() {
				messageboxes.ShowInfo("GoMCL3", info.SourceName)
			}),
		)),
	))
	apps.Run()
}
