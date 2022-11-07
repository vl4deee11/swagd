package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const startFile = `swagger: '2.0'
info:
  title: NAME API
  version: 1.0.0
schemes:
  - http
basePath: /
`

func main() {
	var origSwagger string
	flag.StringVar(&origSwagger, "in", "", "-in=../../../swagger-doc/swagger.yml")
	flag.Parse()

	filename, _ := filepath.Abs(origSwagger)
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalln(err)
	}

	orig := make(map[string]interface{})
	if err = yaml.Unmarshal(yamlFile, &orig); err != nil {
		log.Fatalln(err)
	}

	swgs := make(map[string]interface{})
	paths := orig["paths"].(map[string]interface{})
	for k, v := range paths {
		log.Printf("HANDLER = %s\nWRITE NAME OF FILE FOR THIS HANDLER (OR 'exit' TO SAVE AND STOP)...\n", k)
		file := ""
		_, _ = fmt.Scanln(&file)
		if file == "exit" {
			break
		}
		log.Printf("WRITE TO = %s.yml\n", file)
		if _, ok := swgs[file]; ok {
			fpaths := swgs[file].(map[string]interface{})["paths"].(map[string]interface{})
			fpaths[k] = v
			walkMSI(v.(map[string]interface{}), swgs[file].(map[string]interface{}), orig)
		} else {
			swgs[file] = map[string]interface{}{
				"paths":       map[string]interface{}{},
				"definitions": map[string]interface{}{},
				"parameters":  map[string]interface{}{},
			}
			fpaths := swgs[file].(map[string]interface{})["paths"].(map[string]interface{})
			fpaths[k] = v
			walkMSI(v.(map[string]interface{}), swgs[file].(map[string]interface{}), orig)
		}
		delete(paths, k)
	}
	bb := new(bytes.Buffer)
	var f *os.File
	for fn, v := range swgs {
		bb.WriteString(startFile)
		enc := yaml.NewEncoder(bb)
		enc.SetIndent(2)
		err = enc.Encode(&v)
		if err != nil {
			log.Fatalln(err)
		}

		f, err = os.Create(fn + ".yml")
		if err != nil {
			log.Fatalln(err)
		}

		_, err = f.Write(bb.Bytes())
		if err != nil {
			log.Fatalln(err)
		}
		bb.Reset()

		delete(swgs, fn)
		f.Close()
	}
	log.Println("STOP WORK")
}

func walkMSI(vmap map[string]interface{}, mmap, orig map[string]interface{}) {
	for _, v := range vmap {
		handleVal(v, mmap, orig)
	}
}

func walkMII(vmap map[interface{}]interface{}, mmap, orig map[string]interface{}) {
	for _, v := range vmap {
		handleVal(v, mmap, orig)
	}
}

func walkDefinitions(defn string, mmap, orig map[string]interface{}) {
	def := orig["definitions"].(map[string]interface{})[defn].(map[string]interface{})
	if _, ok := mmap["definitions"].(map[string]interface{})[defn]; ok {
		return
	}
	mmap["definitions"].(map[string]interface{})[defn] = def
	for _, v := range def {
		handleVal(v, mmap, orig)
	}
}

func walkParameters(parn string, mmap, orig map[string]interface{}) {
	par := orig["parameters"].(map[string]interface{})[parn].(map[string]interface{})
	if _, ok := mmap["parameters"].(map[string]interface{})[parn]; ok {
		return
	}
	mmap["parameters"].(map[string]interface{})[parn] = par
	for _, v := range par {
		handleVal(v, mmap, orig)
	}
}

func handleVal(v interface{}, mmap, orig map[string]interface{}) {
	if vv, ok := v.(string); ok {
		if strings.HasPrefix(vv, "#/definitions") {
			log.Printf("DEFENITIONS = %s\n", vv)
			walkDefinitions(strings.Split(vv, "/")[2], mmap, orig)
		} else if strings.HasPrefix(vv, "#/parameters") {
			log.Printf("PARAMETERS = %s\n", vv)
			walkParameters(strings.Split(vv, "/")[2], mmap, orig)
		}
	} else if vv, ok := v.(map[string]interface{}); ok {
		walkMSI(vv, mmap, orig)
	} else if vv, ok := v.(map[interface{}]interface{}); ok {
		walkMII(vv, mmap, orig)
	} else if vv, ok := v.([]interface{}); ok {
		for i := range vv {
			if lv, ok := vv[i].(map[string]interface{}); ok {
				walkMSI(lv, mmap, orig)
			} else if lv, ok := vv[i].(map[interface{}]interface{}); ok {
				walkMII(lv, mmap, orig)
			} else if sv, ok := vv[i].(string); ok {
				if strings.HasPrefix(sv, "#/definitions") {
					log.Printf("DEFINITIONS = %s\n", sv)
					walkDefinitions(strings.Split(sv, "/")[2], mmap, orig)
				} else if strings.HasPrefix(sv, "#/parameters") {
					log.Printf("PARAMETERS = %s\n", sv)
					walkParameters(strings.Split(sv, "/")[2], mmap, orig)
				}
			}
		}
	}
}
