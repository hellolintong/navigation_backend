package main

import (
	"conf/fileutil"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"visualization"
	"visualization/fileparser"
)

type request struct {
	Project          string `json:"project"`
	IsFunctionType   bool   `json:"isFunctionType"`
	SelectedFunction string `json:"selectedFunction"`
	SelectedStruct   string `json:"selectedStruct"`
}

func getBaseDir(project string) string {
	path, err := os.Getwd()
	if err != nil {
		Log.Sugar().Errorf("can't get working dir, error:%s", err.Error())
		return "."
	}
	projectName := filepath.Base(project)
	baseDir := path + "/resource/" + projectName
	Log.Sugar().Infof("base dir:%s", baseDir)
	return baseDir
}

func main() {
	locker := sync.Mutex{}
	r := gin.Default()
	parsers := make(map[string]fileparser.Parser, 0)

	// 读取配置文件获取需要解析的项目目录
	content, err := ioutil.ReadFile("/root/codeviewer/resource/projects.txt")
	//content, err := ioutil.ReadFile("/Users/lintong/go/src/navigation/resource/projects.txt")
	if err != nil {
		Log.Sugar().Errorf("can't read projects.txt, error:%s", err.Error())
		os.Exit(-1)
	}

	// 解析目录
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		Log.Sugar().Infof("read line:%s", line)
		filename := filepath.Base(line)
		parser, err := visualization.NewParser(line)
		if err != nil {
			Log.Sugar().Errorf("can't new parser:%s, error:%s", line, err.Error())
			continue
		}
		parsers[filename] = parser
	}

	// 解析项目
	// map[project]map[module]map[struct][]function
	projectRelation := make(map[string]map[string]map[string][]string, 0)
	for key, parser := range parsers {
		projectRelation[key] = parser.Relation()
	}

	calleeCodeSnippets := make(map[string]map[string]string, 0)
	callerCodeSnippets := make(map[string]map[string]string, 0)
	structCodeSnippets := make(map[string]map[string]string, 0)

	r.GET("/load/", func(c *gin.Context) {
		j, err := json.Marshal(projectRelation)
		if err != nil {
			Log.Sugar().Errorf("can't marshal project relation, error:%s", err.Error())
			c.String(http.StatusInternalServerError, err.Error())
		}
		c.String(http.StatusOK, string(j))
	})

	r.POST("/draw/", func(c *gin.Context) {
		var data request
		var text []byte
		var textPath string
		err := c.ShouldBindJSON(&data)
		if err != nil {
			Log.Sugar().Error("can't bind json, error:%s", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":      "fail",
				"displayText": string(text),
			})
		}
		baseDir := getBaseDir(data.Project)
		calleeCodeSnippet := make(map[string]string, 0)
		callerCodeSnippet := make(map[string]string, 0)
		structCodeSnippet := make(map[string]string, 0)

		if data.IsFunctionType {
			baseFunction := strings.ReplaceAll(data.SelectedFunction, "/", "_")
			// 处理被调用函数的关系
			path := fmt.Sprintf("%s/function_callee_%s.png", baseDir, baseFunction)
			if ok, err := fileutil.IsFile(path); err != nil || !ok {
				if value, ok := parsers[data.Project]; ok {
					locker.Lock()
					value.DrawCalleeFunction(data.SelectedFunction, 10)
					locker.Unlock()
					textPath = fmt.Sprintf("%s/function_callee_%s.txt", baseDir, baseFunction)

				} else {
					c.JSON(http.StatusOK, gin.H{
						"status":            "fail",
						"displayText":       "",
						"calleeCodeSnippet": calleeCodeSnippet,
						"callerCodeSnippet": callerCodeSnippet,
						"structCodeSnippet": structCodeSnippet,
					})
					return
				}
			}

			// 处理调用该函数的关系
			path = fmt.Sprintf("%s/function_caller_%s.png", baseDir, baseFunction)
			if ok, err := fileutil.IsFile(path); err != nil || !ok {
				if value, ok := parsers[data.Project]; ok {
					locker.Lock()
					value.DrawCallerFunction(data.SelectedFunction, 10)
					locker.Unlock()
					textPath = fmt.Sprintf("%s/function_caller_%s.txt", baseDir, baseFunction)

				} else {
					c.JSON(http.StatusOK, gin.H{
						"status":            "fail",
						"displayText":       "",
						"calleeCodeSnippet": calleeCodeSnippet,
						"callerCodeSnippet": callerCodeSnippet,
						"structCodeSnippet": structCodeSnippet,
					})
					return
				}
			}

			locker.Lock()
			if snippet, ok := calleeCodeSnippets[data.SelectedFunction]; !ok {
				calleeCodeSnippet = parsers[data.Project].GetFunctionCalleeCodeSnippet(data.SelectedFunction)
				calleeCodeSnippets[data.SelectedFunction] = calleeCodeSnippet
			} else {
				calleeCodeSnippet = snippet
			}

			if snippet, ok := callerCodeSnippets[data.SelectedFunction]; !ok {
				callerCodeSnippet = parsers[data.Project].GetFunctionCallerCodeSnippet(data.SelectedFunction)
				callerCodeSnippets[data.SelectedFunction] = callerCodeSnippet
			} else {
				callerCodeSnippet = snippet
			}

			locker.Unlock()
		} else {
			baseStruct := strings.ReplaceAll(data.SelectedStruct, "/", "_")
			path := fmt.Sprintf("%s/struct_%s.png", baseDir, baseStruct)

			textPath = fmt.Sprintf("%s/struct_%s.txt", baseDir, baseStruct)
			if ok, err := fileutil.IsFile(path); err != nil || !ok {
				structName := "\"" + data.SelectedStruct + "\""
				if value, ok := parsers[data.Project]; ok {
					locker.Lock()
					value.DrawStruct(structName, 10)
					locker.Unlock()
				} else {
					c.JSON(http.StatusOK, gin.H{
						"status":            "fail",
						"displayText":       "",
						"calleeCodeSnippet": calleeCodeSnippet,
						"callerCodeSnippet": callerCodeSnippet,
						"structCodeSnippet": structCodeSnippet,
					})
					return
				}
			}

			locker.Lock()
			if snippet, ok := structCodeSnippets[data.SelectedStruct]; !ok {
				structCodeSnippet = parsers[data.Project].GetStructCodeSnippet("\"" + data.SelectedStruct + "\"")
				structCodeSnippets[data.SelectedStruct] = structCodeSnippet
			} else {
				structCodeSnippet = snippet
			}
			locker.Unlock()
		}
		text, _ = fileutil.ReadContent(textPath)

		c.JSON(http.StatusOK, gin.H{
			"status":            "success",
			"displayText":       string(text),
			"calleeCodeSnippet": calleeCodeSnippet,
			"callerCodeSnippet": callerCodeSnippet,
			"structCodeSnippet": structCodeSnippet,
		})
	})

	r.Run(":8081")
}
