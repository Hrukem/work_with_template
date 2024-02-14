package main

import (
	"log"
	"os"
)

func main() {
	// ищем директории, в которых расположены файлы со сгенерироваными структурами
	var listDirectory []string
	// маршрут к проекту girvu-core-go
	pathEntities := "./../girvu-core-go/girvu/core/entities/"

	directory, err := os.ReadDir(pathEntities)
	if err != nil {
		log.Println("ошибка чтения директории", err)
	}
	for _, p := range directory {
		if p.IsDir() {
			listDirectory = append(listDirectory, p.Name())
		}
	}

	// получаем данные о структурах из AST
	var slsDataAST []AstData
	slsDataAST = Ast(listDirectory, pathEntities)

	// генерируем файл
	GenCrud(slsDataAST)
}
