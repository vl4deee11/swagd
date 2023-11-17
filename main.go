package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

const startFile = `swagger: '2.0'
info:
  title: %s
  version: 1.0.0
schemes:
  - http
host: localhost
basePath: /
consumes:
  - application/json
produces:
  - application/json
`

type SwagS struct {
	Path        *yaml.MapSlice `yaml:"paths"`
	Parameters  *yaml.MapSlice `yaml:"parameters"`
	Definitions *yaml.MapSlice `yaml:"definitions"`
}

func findKey(ms yaml.MapSlice, k string) interface{} {
	for i := range ms {
		if kk, ok := ms[i].Key.(string); ok && kk == k {
			return ms[i].Value
		}
	}
	return nil
}

func findKeyOk(ms yaml.MapSlice, k string) (interface{}, bool) {
	for i := range ms {
		if kk, ok := ms[i].Key.(string); ok && kk == k {
			return ms[i].Value, true
		}
	}
	return nil, false
}

func asOriginalOrder(orig yaml.MapSlice, target yaml.MapSlice) {
	mm := map[interface{}]int64{}
	for i := range orig {
		mm[orig[i].Key] = int64(i)
	}
	sort.Slice(target, func(i, j int) bool {
		return mm[target[i].Key] < mm[target[j].Key]
	})
}

func main() {
	// nowrap
	yaml.FutureLineWrap()

	var (
		outDir      string
		title       string
		origSwagger string
		pathIdx     int64
		auto        bool
	)
	flag.StringVar(&origSwagger, "in", "./spec/swagger/swagger.yml", "-in=./spec/swagger/swagger.yml")
	flag.StringVar(&outDir, "out-dir", "./spec/swagger", "-out-dir=./spec/swagger")
	flag.StringVar(&title, "title", "service", "-title=logistics-orders")
	flag.Int64Var(&pathIdx, "path-index", 1, "-path-index=1")
	flag.BoolVar(&auto, "auto-split", true, "-auto-split=true")
	flag.Parse()

	filename, _ := filepath.Abs(origSwagger)
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalln(err)
	}

	orig := yaml.MapSlice{}
	if err = yaml.Unmarshal(yamlFile, &orig); err != nil {
		log.Fatalln(err)
	}

	swgs := make(map[string]*yaml.MapSlice)
	paths := findKey(orig, "paths").(yaml.MapSlice)

	for i := range paths {
		kT, vT := paths[i].Key, paths[i].Value
		k := kT.(string)
		v := vT.(yaml.MapSlice)
		file := ""
		if auto {
			file = strings.Split(k, "/")[pathIdx]
		} else {
			log.Printf("\n\nHANDLER = %s\nWRITE NAME OF FILE FOR THIS HANDLER (OR 'exit' TO SAVE AND STOP)...\n", k)
			file = ""
			_, _ = fmt.Scanln(&file)
			if file == "exit" {
				break
			}
			log.Printf("WRITE TO = %s.yml\n", file)
		}
		if _, ok := swgs[file]; ok {
			fpaths := findKey(*swgs[file], "paths").(*yaml.MapSlice)
			*fpaths = append(*fpaths, yaml.MapItem{Key: k, Value: v})
			walkMAP(v, swgs[file], orig)
		} else {
			swgs[file] = &yaml.MapSlice{
				{
					Key:   "paths",
					Value: &yaml.MapSlice{},
				},
				{
					Key:   "definitions",
					Value: &yaml.MapSlice{},
				},
				{
					Key:   "parameters",
					Value: &yaml.MapSlice{},
				},
			}
			fpaths := findKey(*swgs[file], "paths").(*yaml.MapSlice)
			*fpaths = append(*fpaths, yaml.MapItem{Key: k, Value: v})
			walkMAP(v, swgs[file], orig)
		}
	}
	bb := new(bytes.Buffer)
	var f *os.File
	for fn, v := range swgs {
		bb.WriteString(fmt.Sprintf(startFile, title))
		enc := yaml.NewEncoder(bb)

		pathsP := findKey(*v, "paths").(*yaml.MapSlice)
		pathsOrig := findKey(orig, "paths").(yaml.MapSlice)
		asOriginalOrder(pathsOrig, *pathsP)

		definitionsP := findKey(*v, "definitions").(*yaml.MapSlice)
		definitionsOrig := findKey(orig, "definitions").(yaml.MapSlice)
		asOriginalOrder(definitionsOrig, *definitionsP)

		parametersP := findKey(*v, "parameters").(*yaml.MapSlice)
		parametersOrig := findKey(orig, "parameters").(yaml.MapSlice)
		asOriginalOrder(parametersOrig, *parametersP)

		err = enc.Encode(&SwagS{
			Path:        pathsP,
			Definitions: definitionsP,
			Parameters:  parametersP,
		})
		if err != nil {
			log.Fatalln(err)
		}

		f, err = os.Create(outDir + "/" + fn + ".yml")
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

func walkMAP(vmap yaml.MapSlice, mmap *yaml.MapSlice, orig yaml.MapSlice) {
	for _, v := range vmap {
		handleVal(v.Value, mmap, orig)
	}
}

func walkDefinitions(defn string, mmap *yaml.MapSlice, orig yaml.MapSlice) {
	def := findKey(findKey(orig, "definitions").(yaml.MapSlice), defn).(yaml.MapSlice)
	if _, ok := findKeyOk(*findKey(*mmap, "definitions").(*yaml.MapSlice), defn); ok {
		return
	}
	ms := findKey(*mmap, "definitions").(*yaml.MapSlice)
	*ms = append(*ms, yaml.MapItem{Key: defn, Value: def})
	for _, v := range def {
		handleVal(v.Value, mmap, orig)
	}
}

func walkParameters(parn string, mmap *yaml.MapSlice, orig yaml.MapSlice) {
	par := findKey(findKey(orig, "parameters").(yaml.MapSlice), parn).(yaml.MapSlice)
	if _, ok := findKeyOk(*findKey(*mmap, "parameters").(*yaml.MapSlice), parn); ok {
		return
	}
	ms := findKey(*mmap, "parameters").(*yaml.MapSlice)
	*ms = append(*ms, yaml.MapItem{Key: parn, Value: par})
	for _, v := range par {
		handleVal(v.Value, mmap, orig)
	}
}

func handleVal(v interface{}, mmap *yaml.MapSlice, orig yaml.MapSlice) {
	if vv, ok := v.(string); ok {
		if strings.HasPrefix(vv, "#/definitions") {
			log.Printf("DEFENITIONS = %s\n", vv)
			walkDefinitions(strings.Split(vv, "/")[2], mmap, orig)
		} else if strings.HasPrefix(vv, "#/parameters") {
			log.Printf("PARAMETERS = %s\n", vv)
			walkParameters(strings.Split(vv, "/")[2], mmap, orig)
		}
	} else if vv, ok := v.(yaml.MapSlice); ok {
		walkMAP(vv, mmap, orig)
	} else if vv, ok := v.([]interface{}); ok {
		for i := range vv {
			if lv, ok := vv[i].(yaml.MapSlice); ok {
				walkMAP(lv, mmap, orig)
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
