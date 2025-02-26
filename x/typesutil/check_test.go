package typesutil_test

import (
	goast "go/ast"
	"go/importer"
	goparser "go/parser"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/goplus/gop/ast"
	"github.com/goplus/gop/parser"
	"github.com/goplus/gop/token"
	"github.com/goplus/gop/x/typesutil"
	"github.com/goplus/mod/gopmod"
)

func init() {
	if os.Getenv("GOPROOT") == "" {
		dir, _ := os.Getwd()
		os.Setenv("GOPROOT", filepath.Clean(filepath.Join(dir, "./../..")))
	}
	typesutil.SetDebug(typesutil.DbgFlagDefault)
}

func loadFiles(fset *token.FileSet, file string, src interface{}, goxfile string, goxsrc interface{}, gofile string, gosrc interface{}) ([]*ast.File, []*goast.File, error) {
	var files []*ast.File
	var gofiles []*goast.File
	if file != "" {
		f, err := parser.ParseFile(fset, file, src, 0)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, f)
	}
	if goxfile != "" {
		f, err := parser.ParseFile(fset, goxfile, goxsrc, parser.ParseGoPlusClass)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, f)
	}
	if gofile != "" {
		f, err := goparser.ParseFile(fset, gofile, gosrc, 0)
		if err != nil {
			return nil, nil, err
		}
		gofiles = append(gofiles, f)
	}
	return files, gofiles, nil
}

func checkFiles(fset *token.FileSet, file string, src interface{}, goxfile string, goxsrc interface{}, gofile string, gosrc interface{}) (*typesutil.Info, *types.Info, error) {
	files, gofiles, err := loadFiles(fset, file, src, goxfile, goxsrc, gofile, gosrc)
	if err != nil {
		return nil, nil, err
	}
	return checkInfo(fset, files, gofiles)
}

func checkInfo(fset *token.FileSet, files []*ast.File, gofiles []*goast.File) (*typesutil.Info, *types.Info, error) {
	conf := &types.Config{}
	conf.Importer = importer.Default()
	conf.Error = func(err error) {
		log.Println(err)
	}
	chkOpts := &typesutil.Config{
		Types: types.NewPackage("main", "main"),
		Fset:  fset,
		Mod:   gopmod.Default,
	}
	info := &typesutil.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
	}
	ginfo := &types.Info{
		Types:      make(map[goast.Expr]types.TypeAndValue),
		Defs:       make(map[*goast.Ident]types.Object),
		Uses:       make(map[*goast.Ident]types.Object),
		Implicits:  make(map[goast.Node]types.Object),
		Selections: make(map[*goast.SelectorExpr]*types.Selection),
		Scopes:     make(map[goast.Node]*types.Scope),
	}
	check := typesutil.NewChecker(conf, chkOpts, ginfo, info)
	err := check.Files(gofiles, files)
	return info, ginfo, err
}

func TestCheckFiles(t *testing.T) {
	fset := token.NewFileSet()
	info, ginfo, err := checkFiles(fset, "main.gop", `
import "fmt"

type Point struct {
	x int
	y int
}
pt := &Point{}
pt.x = 100
pt.y = 200
fmt.Println(pt)

gopt := GoPoint{100,200}
gopt.Test()
gotest()
fmt.Println(GoValue)
fmt.Println(&Rect{100,200})
`, "Rect.gox", `
var (
	x int
	y int
)
`, "util.go", `package main
var GoValue string
type GoPoint struct {
	x int
	y int
}
func (p *GoPoint) Test() {
}
func gotest() {
}
`)
	if err != nil || info == nil || ginfo == nil {
		t.Fatalf("check failed: %v", err)
	}

	for def, obj := range info.Defs {
		o := info.ObjectOf(def)
		if o != obj {
			t.Fatal("bad obj", o)
		}
	}
	for use, obj := range info.Uses {
		o := info.ObjectOf(use)
		if o.String() != obj.String() {
			t.Fatal("bad obj", o)
		}
		typ := info.TypeOf(use)
		if typ.String() != obj.Type().String() {
			t.Fatal("bad typ", typ)
		}
	}

}

func TestCheckGoFiles(t *testing.T) {
	fset := token.NewFileSet()
	info, ginfo, err := checkFiles(fset, "", "", "", "", "main.go", `package main
type GoPoint struct {
	x int
	y int
}
func main() {
}
`)
	if err != nil || info == nil || ginfo == nil {
		t.Fatalf("check failed: %v", err)
	}
}

func TestCheckError(t *testing.T) {
	fset := token.NewFileSet()
	_, _, err := checkFiles(fset, "main.gop", `
type Point struct {
	x int
	y int
}
pt := &Point1{}
println(pt)
`, "", "", "", "")
	if err == nil {
		t.Fatal("no error")
	}
	_, _, err = checkFiles(fset, "main.gop", `
var i int = "hello"
`, "", "", "", "")
	if err == nil {
		t.Fatal("no error")
	}
}
