package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"
	"unicode"
)

type DataTempl struct {
	SchemaName   string
	StructName   string
	TableName    string
	Insert       string
	Update       string
	ColumnName   []string
	ParamsCreate string
	ParamsUpdate string
}

func GenCrud(slsAstData []AstData) {
	var d DataTempl
	var sB strings.Builder

	// создаём объекты
	for _, s := range slsAstData {
		// создаём папку
		err := os.MkdirAll("./internal/repository/"+s.SchemaName, 0777)
		if err != nil {
			log.Println("ошибка создания директории:", err)
		}

		// создаём файл
		pathOutFile := "./internal/repository/" + s.SchemaName + "/" + s.TableName + ".gen.go"
		wrt, err := os.Create(pathOutFile)
		if err != nil {
			log.Println("ошибка создания файла:", err)
		}

		// создаём шапку в файле
		d.SchemaName = s.SchemaName
		t := template.Must(template.New("createTitle").Parse(tempTitle))
		if err := t.Execute(wrt, d); err != nil {
			log.Println("ошибка генерации1:", err)
		}

		/* ________________создаём объекты в файле __________________________________*/

		// удаляем элемент "ID" из слайса
		var ColumnNameWithoutId []string
		for i, v := range s.ColumnName {
			if v == "id" {
				ColumnNameWithoutId = append(s.ColumnName[:i], s.ColumnName[i+1:]...)
			}
		}
		var namesColumn string
		for _, v := range ColumnNameWithoutId {
			namesColumn += v + ","
		}
		namesColumn = strings.TrimSuffix(namesColumn, ",")

		// конструируем запрос на Create
		sB.WriteString("INSERT INTO ")
		sB.WriteString(s.SchemaName + "." + s.TableName)
		sB.WriteString(" (" + namesColumn + ") VALUES (")
		for i := range ColumnNameWithoutId {
			sB.WriteString("$" + strconv.Itoa(i+1) + ",")
		}
		str := strings.TrimSuffix(sB.String(), ",")
		d.Insert = str + ") RETURNING id"
		sB.Reset()

		// конструируем параметры для запроса на Create
		var fieldsNameWithoutId []string
		// удаляем элемент "ID" из слайса
		for i, v := range s.FieldName {
			if v == "ID" {
				fieldsNameWithoutId = append(s.FieldName[:i], s.FieldName[i+1:]...)
			}
		}
		params := ""
		for _, v := range fieldsNameWithoutId {
			params += "input." + v + ","
		}
		d.ParamsCreate = params

		// конструируем запрос на Update
		sB.WriteString("UPDATE ")
		sB.WriteString(s.SchemaName + "." + s.TableName + " SET ")
		for i, v := range ColumnNameWithoutId {
			sB.WriteString(v + "=$" + strconv.Itoa(i+1) + ",")
		}
		str = strings.TrimSuffix(sB.String(), ",")
		d.Update = str + " WHERE id=$" + strconv.Itoa(len(ColumnNameWithoutId)+1)

		// конструируем параметры для запроса на Update
		d.ParamsUpdate = d.ParamsCreate + "input.ID"

		// конструируем имя структуры (PascalCase)
		var structName string
		slsStr := strings.Split(s.TableName, "_")
		for _, v := range slsStr {
			runes := []rune(v)
			runes[0] = unicode.ToUpper(runes[0])
			structName += string(runes)
		}

		d.StructName = structName
		d.TableName = s.TableName
		// d.SchemaName = s.SchemaName

		sB.Reset()

		t = template.Must(template.New("createFunctions").Parse(temp))
		/*		if err := t.Execute(os.Stdout, d); err != nil {
					log.Println("ошибка генерации:", err)
				}
		*/
		if err := t.Execute(wrt, d); err != nil {
			log.Println("ошибка генерации2:", err)
		}
	}
}

var tempTitle = `// Этот файл сгенерирован. Не вносить изменения вручную!! При повторной генерации они будут удалены.
package {{.SchemaName}}

import (
	"10.10.11.220/girvu/girvu-core-go.git/girvu/vpo/entities/{{.SchemaName}}"
	"context"
	"errors"
	"girvu-lk/internal/config"
	"girvu-lk/internal/dbwrap"
	"girvu-lk/internal/report"
	"girvu-lk/internal/repository/common"
	"github.com/jackc/pgx/v5"
)
`

var temp = `
type {{.StructName}} struct {
	common.DbData
	appConfig *config.AppConfig
}

func New{{.StructName}}(
	appConfig *config.AppConfig,
	db common.DbTx,
	report *report.Report,
	tx *common.Transaction,
) *{{.StructName}} {
	return &{{.StructName}}{
		appConfig: appConfig,
		DbData: common.DbData{Db: db, Report: report, QueryDeadline: appConfig.DB.QueryDeadline,
			QueryTimeWarning: appConfig.DB.QueryTimeWarning, Transaction: tx},
	}
}

func (o *{{.StructName}} ) getLogger(  ctx context.Context) (Logger common.ILogRequestDataProvider ) {
	RequestData := ctx.Value("RequestData")
	if RequestData == nil {
		panic("нету значения логгера")
	}
	RequestDataI, ok := RequestData.(common.ILogRequestDataProvider)
	if !ok {
		panic("значение в интерфейсе логгера не того типа ")
	}
	return RequestDataI
}

func (o *{{.StructName}}) Create(ctx context.Context, input {{.SchemaName}}.{{.StructName}}) (int64, error) {
	lrdp, ok := ctx.Value("RequestData").(common.ILogRequestDataProvider)
	if !ok {
		return 0, errors.New("Ошибка обработки контекста в функции {{.StructName}}.Create() ")
	}

	query := "{{.Insert}}"

	params := []any{ {{.ParamsCreate}} } 

	var id int64
	var obj dbwrap.RequestToDb[{{.SchemaName}}.{{.StructName}}]
	if err := obj.Scan(ctx, lrdp, o.DbRW(nil), query, params, nil, &id); err != nil {
		return 0, err
	}
	
	return id, nil
}

func (o *{{.StructName}}) Update(ctx context.Context, input {{.SchemaName}}.{{.StructName}}) error {
	lrdp, ok := ctx.Value("RequestData").(common.ILogRequestDataProvider)
	if !ok {
		return errors.New("Ошибка обработки контекста в функции {{.StructName}}.Update() ")
	}

	var obj dbwrap.RequestToDb[any]

	query := "{{.Update}}"
	params := []any { {{.ParamsUpdate}} }

	_, err := obj.ExecWrap(ctx, lrdp, o.DbRW(o.GetTx(ctx)), query, params)
	if err != nil {
		return err
	}

	return nil
}

func (o *{{.StructName}}) Find(ctx context.Context, ID int64) ({{.SchemaName}}.{{.StructName}}, error) {
	lrdp, ok := ctx.Value("RequestData").(common.ILogRequestDataProvider)
	if !ok {
		return {{.SchemaName}}.{{.StructName}}{}, errors.New("Ошибка обработки контекста в функции {{.StructName}}.Find() ")
	}

	var obj dbwrap.RequestToDb[{{.SchemaName}}.{{.StructName}}]

	query := "SELECT * FROM {{.SchemaName}}.{{.TableName}} WHERE id=$1"
	params := []any{ID}

	return obj.OneRow(ctx, lrdp, o.DbRW(nil), query, params, errors.New("Данные не найдены "), pgx.RowToStructByName[{{.SchemaName}}.{{.StructName}}])
}

func (o *{{.StructName}}) Delete(ctx context.Context, ID int64) error {
	lrdp, ok := ctx.Value("RequestData").(common.ILogRequestDataProvider)
	if !ok {
		return errors.New("Ошибка обработки контекста в функции {{.StructName}}.Delete() ")
	}

	var obj dbwrap.RequestToDb[any]

	query := "DELETE FROM {{.SchemaName}}.{{.TableName}} WHERE id=$1"
	params := []any{ID}

	_, err := obj.ExecWrap(ctx, lrdp, o.DbRW(o.GetTx(ctx)), query, params)
	if err != nil {
		return err
	}

	return nil
}
`
