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
	"visualization"
	"visualization/fileparser"
)
type request struct {
	Project string `json:"project"`
	IsFunctionType bool `json:"isFunctionType"`
	SelectedFunction string `json:"selectedFunction"`
	SelectedStruct string `json:"selectedStruct"`
}

func getBaseDir(project string) string {
	path, err := os.Getwd()
	if err != nil {
		Log.Sugar().Errorf("can't get working dir, error:%s", err.Error())
		return "."
	}
	projectName := filepath.Base(project)
	baseDir := 	path+"/resource/"+projectName
	Log.Sugar().Infof("base dir:%s", baseDir)
	return baseDir
}

func main() {
	r := gin.Default()
	parsers := make(map[string]fileparser.Parser, 0)

	content, err := ioutil.ReadFile("/root/codeviewer/resource/projects.txt")
	if err != nil {
		Log.Sugar().Errorf("can't read projects.txt, error:%s", err.Error())
		os.Exit(-1)
	}

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


	r.GET("/load/", func(c *gin.Context) {
		projectRelation := make(map[string]map[string][]string, 0)
		for key, parser := range parsers {
			projectRelation[key] = parser.Relation()
		}
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
				"status":  "fail",
				"displayText": string(text),
			})
		}
		baseDir := getBaseDir(data.Project)
		if data.IsFunctionType {
			baseFunction := strings.ReplaceAll(data.SelectedFunction, "/", "_")
			path := fmt.Sprintf("%s/function_%s.png", baseDir, baseFunction)
			if ok, err := fileutil.IsFile(path); err != nil || !ok {
				parsers[data.Project].DrawFunction(data.SelectedFunction, 10)
			}
			textPath = fmt.Sprintf("%s/function_%s.txt", baseDir, baseFunction)

		} else {
			baseStruct := strings.ReplaceAll(data.SelectedStruct, "/", "_")
			path := fmt.Sprintf("%s/struct_%s.png", baseDir, baseStruct)

			textPath = fmt.Sprintf("%s/struct_%s.txt", baseDir, baseStruct)
			if ok, err := fileutil.IsFile(path); err != nil || !ok {
				structName := "\"" + data.SelectedStruct + "\""
				parsers[data.Project].DrawStruct(structName, 10)
			}
		}
		text, _ = fileutil.ReadContent(textPath)


		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"displayText": string(text),
		})
	})

	r.Run(":8081")
}