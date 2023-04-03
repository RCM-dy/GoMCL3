package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tidwall/gjson"
)

func ReplaceByMap(s string, m map[string]string) string {
	for k, v := range m {
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}
func Sha1Bytes(datas []byte) string {
	s := sha1.New()
	s.Write(datas)
	return hex.EncodeToString(s.Sum(nil))
}
func runCMD(commands string) (output string, err error) {
	cmdpath := filepath.Join(os.Getenv("windir"), "System32", "cmd.exe")
	cmd := exec.Command(cmdpath, "/c", commands)
	out := bytes.NewBuffer(nil)
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Run()
	var o []byte
	o, err = io.ReadAll(out)
	if err != nil {
		return
	}
	output = string(o)
	return
}
func GetJavas() (m map[string]string, err error) {
	m = make(map[string]string)
	var c string
	c, err = runCMD("where java")
	if err != nil {
		return
	}
	var o []byte
	for _, v := range strings.Split(c, "\n") {
		if v == "" {
			continue
		}
		v = strings.ReplaceAll(v, "\r", "")
		cmd := exec.Command(v, "-version")
		out := bytes.NewBuffer(nil)
		cmd.Stdout = out
		cmd.Stderr = out
		cmd.Run()
		o, err = io.ReadAll(out)
		if err != nil {
			return
		}
		output := string(o)
		outs := []string{}
		for _, v1 := range strings.Split(output, "\n") {
			if v1 == "" {
				continue
			}
			outs = append(outs, strings.ReplaceAll(v1, "\r", ""))
		}
		ver := strings.Split(outs[0], " ")[2]
		ver = strings.ReplaceAll(ver, "\"", "")
		vers := strings.Split(ver, ".")[0]
		two := strings.Split(ver, ".")[1]
		if vers == "1" && two == "8" {
			vers = two
		}
		vs, ok := m[vers]
		if ok {
			if len(vs) < len(v) {
				continue
			}
		}
		m[vers] = v
	}
	return
}

func GetOsBit() string {
	return "x" + fmt.Sprintf("%d", 32<<(^uint(0)>>63))
}

func JointArgsOld(verinfo VersionInfo, cp string) (string, error) {
	args := verinfo.Result.Get("minecraftArguments")
	jvmargs := "-XX:HeapDumpPath=MojangTricksIntelDriversForPerformance_javaw.exe_minecraft.exe.heapdump -Djava.library.path=${natives_directory} -cp \"" + cp + "\""
	javaV := verinfo.Result.Get("javaVersion").Get("majorVersion").String()
	allJava, err := GetJavas()
	if err != nil {
		return "", err
	}
	jp, ok := allJava[javaV]
	if !ok {
		return "", errors.New("no java version")
	}
	jp += "\""
	jp = "\"" + jp
	allargs := jp + " " + jvmargs + " " + verinfo.Result.Get("mainClass").String() + " " + args.String()
	return ReplaceByMap(allargs, map[string]string{
		"${version_name}":      verinfo.Id,
		"${version_type}":      verinfo.types,
		"${assets_root}":       "\"" + info.AssetsDir + "\"",
		"${assets_index_name}": verinfo.Result.Get("assets").String(),
		"${game_directory}":    "\"" + info.McDir + "\"",
		"${natives_directory}": "\"" + filepath.Join(info.VerDir, verinfo.Id, "natives") + "\"",
	}), nil
}

func JointArgs(verinfo VersionInfo, cp string) (string, error) {
	result := verinfo.Result
	args := result.Get("arguments")
	if !args.Exists() {
		if result.Get("minecraftArguments").Exists() {
			return JointArgsOld(verinfo, cp)
		}
		return "", errors.New("args not found")
	}
	gameargs := args.Get("game")
	jvmargs := args.Get("jvm")
	strgameargs := ""
	gameargs.ForEach(func(key, value gjson.Result) bool {
		if value.Type.String() == "String" {
			strgameargs += value.String() + " "
		} else if value.IsObject() {
			tn := false
			value.ForEach(func(key, value gjson.Result) bool {
				if key.String() == "rules" {
					var notsame bool = false
					value.ForEach(func(key, value gjson.Result) bool {
						if key.String() == "action" {
							if value.String() != "allow" {
								notsame = true
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
					if notsame {
						tn = true
						return false
					}
				}
				return true
			})
			if tn {
				return true
			}
			if key.String() == "value" {
				if value.IsArray() {
					value.ForEach(func(key, value gjson.Result) bool {
						strgameargs += value.String() + " "
						return true
					})
				} else if value.Type.String() == "String" {
					strgameargs += value.String() + " "
				}
			}
		}
		return true
	})
	strgameargs = strings.TrimRight(strgameargs, " ")
	strjvmargs := ""
	jvmargs.ForEach(func(key, value gjson.Result) bool {
		if value.Type.String() == "String" {
			strjvmargs += value.String() + " "
		} else if value.IsObject() {
			tn := false
			value.ForEach(func(key, value gjson.Result) bool {
				if key.String() == "rules" {
					var notsame bool = false
					value.ForEach(func(key, value gjson.Result) bool {
						if key.String() == "action" {
							if value.String() != "allow" {
								notsame = true
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
					if notsame {
						tn = true
						return false
					}
				}
				return true
			})
			if tn {
				return true
			}
			if key.String() == "value" {
				if value.IsArray() {
					value.ForEach(func(key, value gjson.Result) bool {
						strjvmargs += value.String() + " "
						return true
					})
				} else if value.Type.String() == "String" {
					strjvmargs += value.String() + " "
				}
			}
		}
		return true
	})
	strjvmargs = strings.TrimRight(strjvmargs, " ")
	javaV := result.Get("javaVersion").Get("majorVersion").String()
	allJava, err := GetJavas()
	if err != nil {
		return "", err
	}
	jp, ok := allJava[javaV]
	if !ok {
		return "", errors.New("no java version")
	}
	jp += "\""
	jp = "\"" + jp
	return ReplaceByMap(jp+" "+strjvmargs+" "+result.Get("mainClass").String()+" "+strgameargs, map[string]string{
		"${classpath}":         "\"" + cp + "\"",
		"${version_type}":      verinfo.types,
		"${version_name}":      verinfo.Id,
		"${assets_root}":       "\"" + info.AssetsDir + "\"",
		"${assets_index_name}": result.Get("assets").String(),
		"${game_directory}":    "\"" + info.McDir + "\"",
		"${natives_directory}": "\"" + filepath.Join(info.VerDir, verinfo.Id, "natives") + "\"",
	}), nil
}
func GetDllBit(p string) (string, error) {
	cmd := exec.Command(".\\dumpbin\\dumpbin.exe", "/headers", p)
	out := bytes.NewBuffer(nil)
	cmd.Stdout = out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	o, err := io.ReadAll(out)
	if err != nil {
		return "", err
	}
	return strings.Split(strings.Split(string(o), " machine (")[1], ")")[0], nil
}
func MKdirs(dirs ...string) error {
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0666)
		if err != nil {
			return err
		}
	}
	return nil
}
