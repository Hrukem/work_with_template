package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"golang.org/x/exp/slices"
	"strings"
	"unicode"
)

type AstData struct {
	SchemaName string
	TableName  string
	FieldName  []string
	ColumnName []string
}

func getColumnName(s *ast.StructType) []string {
	var columnName []string
	for _, field := range s.Fields.List {
		if len(field.Names) == 0 {
			continue
		}
		sls := strings.Split(field.Tag.Value, `" `) // разбиваем теги на элементы разделённые пробелами с кавычкой
		for _, v := range sls {
			if strings.Contains(v, "db:") {
				dbTag := strings.Split(v, ":")                                         // ищем элемент в котором есть подстрока "db:" и разбиваем на 2 строки (разделитель ":")
				columnName = append(columnName, strings.ReplaceAll(dbTag[1], `"`, "")) // берём второй элемент (очистив его от кавычек)
				break
			}
		}
	}
	return columnName
}

func getFieldsName(s *ast.StructType) []string {
	var fieldsName []string
	for _, field := range s.Fields.List {
		if len(field.Names) == 0 {
			continue
		}
		fieldsName = append(fieldsName, field.Names[0].Name)
	}
	return fieldsName
}

func snakeCase(s string) string {
	var str strings.Builder
	var prev rune
	for i, r := range s {
		// check if we should insert a underscore
		if i > 0 && unicode.IsUpper(r) && unicode.IsLower(prev) {
			str.WriteRune('_')
		}
		// lower case all characters
		str.WriteRune(unicode.ToLower(r))
		prev = r
	}
	return str.String()
}

func Ast(listDirectory []string, pathEntities string) []AstData {
	var result []AstData

	for _, directName := range listDirectory {
		dir := pathEntities + directName

		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
		if err != nil {
			panic(err)
		}
		for _, pkg := range pkgs {
			for _, file := range pkg.Files {
				// fmt.Printf("working on file %v\n", fileName)
				ast.Inspect(file, func(n ast.Node) bool {
					/* вывод всей информации
					if err = ast.Print(fset, n); err != nil {
											fmt.Println("error ast:", err)
										}
					*/
					var data AstData
					// try to convert n to ast.TypeSpec
					if typeSpec, isTypeSpec := n.(*ast.TypeSpec); isTypeSpec {
						s, isStructType := typeSpec.Type.(*ast.StructType)
						// check if conversion was successful
						if !isStructType {
							return true
						}

						fieldName := getFieldsName(s)
						// отключение генерации объекта, если в именах полей нет "ID"
						if !slices.Contains(fieldName, "ID") {
							return true
						}
						data.FieldName = fieldName

						structName := snakeCase(typeSpec.Name.Name)

						data.SchemaName = directName
						data.TableName = structName
						data.ColumnName = getColumnName(s)

						result = append(result, data)
					}
					return true
				})
			}
		}
	}
	return result
}
