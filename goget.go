package main

import (
	"go/build"
	"fmt"
	"os"
	"path/filepath"
	"log"
	"github.com/kr/pretty"
	"regexp"
	"strings"
	"io/ioutil"
	"os/exec"
	"sync"
)

const (
	NET_PKG_PATTERN1 = `import\s+"(\w+\.\w+/.*)"` 
	NET_PKG_PATTERN2 = `(?s)import\s+\((.*?)\)`
	NET_PKG_PATTERN3 = `"(\w+\.\w+/.*)"`
)

var wg sync.WaitGroup
var debug bool 

func main() {
	log.SetFlags(0)
	log.SetPrefix("goget:")
	
	pkgs, err := findPackages()
	if err != nil {
		log.Fatalln(err)
	}
	for _, pkg := range pkgs {
		wg.Add(1)
		go goget(pkg)
	}
	wg.Wait()
}

//执行go get
func goget(pkg string) {
	defer wg.Done()
	log.Printf("Find package: %s", pkg)
	command := fmt.Sprintf("go get %s", pkg)
	log.Println(command)
    cmd := exec.Command("/bin/sh", "-c", command)
    var res []byte
    res, err := cmd.CombinedOutput()
    if err != nil {
    	log.Fatalln(err)
    }
    result := fmt.Sprintf("%s", res)
    if result != "" {
    	log.Println(result)
    }
}

//查找所有.go文件，找出import的网络包
func findPackages() ([]string, error) {
	dir, err := filepath.Abs(".")
	if err != nil {
		return nil, err
	}
	dp, err := build.ImportDir(dir, build.FindOnly)
	if err != nil {
		return nil, err 
	}
	if debug {
		pretty.Println(dp)
	}
	files, err := globAll(dp.Dir, "*.go")
	if err != nil {
		return nil, err 
	}
	var pkgs []string
	for _, file := range files {
		if content, err := ioutil.ReadFile(file); err == nil {
			pkgs = append(pkgs, findNetworkPkgs(string(content))...)
		}
	}
	return removeDuplicate(pkgs), nil
}

//根据文件内容匹配出网络包
func findNetworkPkgs(content string) []string {
	var pkgs []string 
	regex := regexp.MustCompile(NET_PKG_PATTERN1)
	res := regex.FindAllStringSubmatch(content, -1)
	for _, pkg := range res {
		pkgs = append(pkgs, pkg[1])
	}

	regex2 := regexp.MustCompile(NET_PKG_PATTERN2)
	res2 := regex2.FindStringSubmatch(content)
	if len(res2) > 1 {
		pkg_lines := strings.Split(res2[1], "\n\t")
		for _, pkg := range pkg_lines {
			regex_sub := regexp.MustCompile(NET_PKG_PATTERN3)
			res_sub := regex_sub.FindStringSubmatch(pkg)
			if len(res_sub) > 1 {
				pkgs = append(pkgs, res_sub[1])
			}
		}
	}
	return pkgs
}

//slice去重
func removeDuplicate(arr []string) []string {
	hash := make(map[string]bool)
	var arr_tmp []string
	for _, v := range arr {
		if exist, _ := hash[v]; !exist {
			hash[v] = true
			arr_tmp = append(arr_tmp, v)
		}
	}
	return arr_tmp
}

//递归找出一个目录下所有匹配的文件
func globAll(path string, patterns ...string) ([]string, error) {
	var files []string
	root, err := filepath.Glob(string(filepath.Join(path, "*")))
	if err != nil {
		return nil, err 
	}
	for _, file := range root {
		for _, pattern := range patterns {
			if matched, _ := filepath.Match(filepath.Join(path, pattern), file); matched {
				files = append(files, file)
				break
			}
		}
		if idDir(file) && !filepath.HasPrefix(file, ".") {
			sub_files, err := globAll(file, patterns...)
			if err != nil {
				return nil, err 
			}
			files = append(files, sub_files...)
		}
	}
	return files, nil 
}

//判断一个路径是否是目录
func idDir(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer f.Close()
	fs, err := f.Stat()
	if err != nil {
		fmt.Println(err)
		return false
	}
	switch mode := fs.Mode(); {
	case mode.IsDir():
		return true
	default:
		return false
	}
}
