// Copyright 2016 NDP Systèmes. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package generate

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/types"
	"golang.org/x/tools/go/loader"
	"io/ioutil"
	"strings"
	"text/template"

	"github.com/inconshreveable/log15"
	"github.com/npiganeau/yep/yep/tools/logging"
)

const (
	CONFIG_PATH      string = "github.com/npiganeau/yep/config"
	MODELS_PATH      string = "github.com/npiganeau/yep/yep/models"
	POOL_PATH        string = "github.com/npiganeau/yep/pool"
	TEST_MODULE_PATH string = "github.com/npiganeau/yep/yep/tests/test_module"
)

var log log15.Logger

// CreateFileFromTemplate generates a new file from the given template and data
func CreateFileFromTemplate(fileName string, template *template.Template, data interface{}) {
	var srcBuffer bytes.Buffer
	template.Execute(&srcBuffer, data)
	srcData, err := format.Source(srcBuffer.Bytes())
	if err != nil {
		logging.LogAndPanic(log, "Error while formatting generated source file", "error", err, "fileName", fileName, "mData", fmt.Sprintf("%#v", data), "src", srcBuffer.String())
	}
	// Write to file
	err = ioutil.WriteFile(fileName, srcData, 0644)
	if err != nil {
		logging.LogAndPanic(log, "Error while saving generated source file", "error", err, "fileName", fileName)
	}
}

// moduleType describes a type of module
type PackageType int8

const (
	// The base package of a module
	BASE PackageType = iota
	// The defs package of a module
	DEFS
	// A sub package of a module (that is not defs)
	SUB
	// The yep/models package
	MODELS
)

// moduleInfo is a wrapper around loader.Package with additional data to
// describe a module.
type ModuleInfo struct {
	loader.PackageInfo
	ModType PackageType
}

// newModuleInfo returns a pointer to a new moduleInfo instance
func NewModuleInfo(pack *loader.PackageInfo, modType PackageType) *ModuleInfo {
	return &ModuleInfo{
		PackageInfo: *pack,
		ModType:     modType,
	}
}

// GetModulePackages returns a slice of PackageInfo for packages that are yep modules, that is:
// - A package that declares a "MODULE_NAME" constant
// - A package that is in a subdirectory of a package
// Also returns the 'yep/models' package since all models are initialized there
func GetModulePackages(program *loader.Program) []*ModuleInfo {
	modules := make(map[string]*ModuleInfo)

	// We add to the modulePaths all packages which define a MODULE_NAME constant
	// and we check for 'yep/models' package
	for _, pack := range program.AllPackages {
		obj := pack.Pkg.Scope().Lookup("MODULE_NAME")
		_, ok := obj.(*types.Const)
		if ok {
			modules[pack.Pkg.Path()] = NewModuleInfo(pack, BASE)
			continue
		}
		if pack.Pkg.Path() == MODELS_PATH {
			modules[pack.Pkg.Path()] = NewModuleInfo(pack, MODELS)
		}
	}

	// Now we add packages that live inside another module
	for _, pack := range program.AllPackages {
		for _, module := range modules {
			if strings.HasPrefix(pack.Pkg.Path(), module.Pkg.Path()) {
				typ := SUB
				if strings.HasSuffix(pack.String(), "defs") {
					typ = DEFS
				}
				modules[pack.Pkg.Path()] = NewModuleInfo(pack, typ)
			}
		}
	}

	// Finally, we build up our result slice from modules map
	modSlice := make([]*ModuleInfo, len(modules))
	var i int
	for _, mod := range modules {
		modSlice[i] = mod
		i++
	}
	return modSlice
}

// A MethodRef is a map key for a method in a model
type MethodRef struct {
	Model  string
	Method string
}

// TypeData holds a Type string and optional import path for this type.
type TypeData struct {
	Type       string
	ImportPath string
}

// DocAndParams is a holder for a function's doc string and parameters names
type MethodASTData struct {
	Doc        string
	Params     []string
	ReturnType TypeData
}

// GetMethodsDocAndParamsNames returns the doc string and parameters name of all
// methods of all YEP modules.
func GetMethodsASTData() map[MethodRef]MethodASTData {
	// Parse source code
	conf := loader.Config{
		AllowErrors: true,
		ParserMode:  parser.ParseComments,
	}
	conf.Import(CONFIG_PATH)
	conf.Import(TEST_MODULE_PATH)
	program, _ := conf.Load()
	modInfos := GetModulePackages(program)
	return GetMethodsASTDataForModules(modInfos)
}

// GetMethodsASTDataForModules returns the MethodASTData for all methods in given modules.
func GetMethodsASTDataForModules(modInfos []*ModuleInfo) map[MethodRef]MethodASTData {
	res := make(map[MethodRef]MethodASTData)
	// Parse all modules for comments and params names
	// In the same loop, we both :
	// - Get method ast data for all functions
	// - Get the list of methods by parsing 'CreateMethod'
	meths := make(map[MethodRef]ast.Node)
	funcs := make(map[ast.Node]MethodASTData)
	for _, modInfo := range modInfos {
		for _, file := range modInfo.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.FuncDecl:
					funcs[n] = MethodASTData{
						Doc:        extractDocString(node),
						Params:     extractParams(node.Type),
						ReturnType: extractReturnType(node.Type, modInfo),
					}
				case *ast.FuncLit:
					funcs[n] = MethodASTData{
						Doc:        "",
						Params:     extractParams(node.Type),
						ReturnType: extractReturnType(node.Type, modInfo),
					}
				case *ast.CallExpr:
					var fNameNode *ast.Ident
					switch nf := node.Fun.(type) {
					case *ast.SelectorExpr:
						fNameNode = nf.Sel
					case *ast.Ident:
						fNameNode = nf
					default:
						return true
					}
					if fNameNode.Name != "CreateMethod" {
						return true
					}
					modelName := ""
					if mn, ok := node.Args[0].(*ast.BasicLit); ok {
						modelName = strings.Trim(mn.Value, `"`)
					}
					methodName := ""
					if mn, ok := node.Args[1].(*ast.BasicLit); ok {
						methodName = strings.Trim(mn.Value, `"`)
					}

					var funcDecl ast.Node
					switch fd := node.Args[2].(type) {
					case *ast.Ident:
						funcDecl = fd.Obj.Decl.(*ast.FuncDecl)
					case *ast.FuncLit:
						funcDecl = fd
					}
					meths[MethodRef{Model: modelName, Method: methodName}] = funcDecl
				}
				return true
			})
		}
	}
	// Now we extract the doc and params from funcs only for methods
	for ref, meth := range meths {
		res[ref] = funcs[meth]
	}
	return res
}

// extractParams extracts the parameters name of the given FuncType
func extractParams(ft *ast.FuncType) []string {
	var params []string
	for i, pl := range ft.Params.List {
		if i == 0 {
			// pass the first argument (rs)
			continue
		}
		for _, nn := range pl.Names {
			params = append(params, nn.Name)
		}
	}
	return params
}

// extractReturnType returns the return type of the first returned value
// of the given FuncType as a string and an import path if needed.
func extractReturnType(ft *ast.FuncType, modInfo *ModuleInfo) TypeData {
	var returnType, importPath string
	if ft.Results != nil && len(ft.Results.List) > 0 {
		returnType = types.TypeString(modInfo.TypeOf(ft.Results.List[0].Type), (*types.Package).Name)
		importPath = computeExportPath(modInfo.TypeOf(ft.Results.List[0].Type))
	}
	if importPath == POOL_PATH {
		returnType = strings.Replace(returnType, "pool.", "", 1)
		importPath = ""
	}

	importPathTokens := strings.Split(importPath, ".")
	if len(importPathTokens) > 0 {
		importPath = strings.Join(importPathTokens[:len(importPathTokens)-1], ".")
	}

	return TypeData{
		Type:       returnType,
		ImportPath: importPath,
	}
}

// computeExportPath returns the import path of the given type
func computeExportPath(typ types.Type) string {
	var res string
	switch typ := typ.(type) {
	case *types.Struct, *types.Named:
		res = types.TypeString(typ, (*types.Package).Path)
	case *types.Pointer:
		res = computeExportPath(typ.Elem())
	case *types.Slice:
		res = computeExportPath(typ.Elem())
	}
	return res
}

// extractDocString returns the documentation string for the given func decl.
func extractDocString(fd *ast.FuncDecl) string {
	var docString string
	if fd.Doc != nil {
		for _, d := range fd.Doc.List {
			docString = fmt.Sprintf("%s\n%s", docString, d.Text)
		}
	}
	return docString
}

func init() {
	log = logging.GetLogger("tools/generate")
}