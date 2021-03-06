package persist

import (
    "fmt"
    "log"
    "reflect"
    "strings"
    "errors"

    "database/sql"
    _ "github.com/mattn/go-sqlite3"
)

var tableSet map[string]reflect.Type
var db *sql.DB

type scanner func(*sql.Rows)

var goToSqliteKindMap map[reflect.Kind]string = map[reflect.Kind]string{
    reflect.Bool: "integer",
    reflect.Int: "integer",
    reflect.Int8: "integer",
    reflect.Int16: "integer",
    reflect.Int32: "integer",
    reflect.Int64: "integer",
    reflect.Uint: "integer",
    reflect.Uint8: "integer",
    reflect.Uint16: "integer",
    reflect.Uint32: "integer",
    reflect.Uint64: "integer",
    reflect.Float32: "real",
    reflect.Float64: "real",
}

// Initialize the persistence sqlite database.
func Init(dbPath string) error {
    tableSet = make(map[string]reflect.Type)
    var err error

    db, err = sql.Open("sqlite3", dbPath)

    return err
}

func getTypeName(i interface{}) string {
    typeName := fmt.Sprintf("%T", i)
    if typeName[0] == '*' {
        typeName = typeName[1:]
    }

    return typeName
}

func getTableName(i interface{}) string {
    return strings.ReplaceAll(getTypeName(i), ".", "_")
}

func getFieldMap(i interface{}) (fields map[string]interface{}) {
    fields = make(map[string]interface{})

    iType := reflect.TypeOf(i).Elem()
    iValue := reflect.ValueOf(i).Elem()
    for i := 0; i < iType.NumField(); i++ {
        fields[iType.Field(i).Name] = iValue.Field(i).Interface()
    }

    return fields
}

func getFieldArray(i interface{}) (fields []interface{}) {
    iValue := reflect.ValueOf(i).Elem()
    fields = make([]interface{}, iValue.NumField())
    for i := 0; i < len(fields); i++ {
        fields[i] = iValue.Field(i).Interface()
    }

    return fields
}

func getTableColumns(tableName string) map[string]interface{} {
    rows, err := db.Query(fmt.Sprintf("pragma table_info(%s)", tableName))
    if err != nil {
        log.Fatal(err)
    }

    defer rows.Close()

    columns := make(map[string]interface{})

    for rows.Next() {
        var cid string
        var name string
        var coltype string
        var notnull string
        var dflt_value string
        var pk string

        rows.Scan(&cid, &name, &coltype, &notnull, &dflt_value, &pk)
        columns[name] = coltype
    }

    return columns
}

func makeFieldDec(field reflect.StructField) string {
    sql := strings.Builder{}

    sql.WriteString(field.Name)

    if sqliteType, ok := goToSqliteKindMap[field.Type.Kind()]; ok {
        sql.WriteString(" " + sqliteType)
    } else {
        sql.WriteString(" string")
    }

    if field.Tag.Get("db-pk") == "true" {
        sql.WriteString(" primary key")
    }

    return sql.String()
}

func createTable(r interface{}) error {
    tableName := getTableName(r)

    sql := strings.Builder{}
    sql.WriteString("create table ")
    sql.WriteString(tableName)
    sql.WriteString(" (")

    rType := reflect.TypeOf(r).Elem()
    for i := 0; i < rType.NumField(); i++ {
        if i > 0 {
            sql.WriteString(", ")
        }

        field := rType.Field(i)
        sql.WriteString(makeFieldDec(field))
    }

    sql.WriteString(")")

    _, err := db.Exec(sql.String())

    return err
}

func verifyTable(r interface{}) (err error) {
    tableName := getTableName(r)

    if _, ok := tableSet[tableName]; ok {
        return nil
    }
    defer func () {tableSet[tableName] = reflect.TypeOf(r)}()

    log.Printf("verifying table %s", tableName)

    columns := getTableColumns(tableName)

    if len(columns) <= 0 {
        createTable(r)

        return nil
    }

    rType := reflect.TypeOf(r).Elem()
    sql := strings.Builder{}

    for i := 0; i < rType.NumField(); i++ {
        field := rType.Field(i)
        if _, ok := columns[field.Name]; !ok {
            sql.WriteString(fmt.Sprintf("alter table %s add column %s",
                    tableName, makeFieldDec(field)))
        }
    }

    if sql.Len() > 0 {
        _, err = db.Exec(sql.String())
    }

    return err
}

// Insert a new record into the store.
var insertSqlCache map[string]string = map[string]string{}

func Insert(r interface{}) (err error) {
    if err = verifyTable(r); err != nil {
        return err
    }

    tableName := getTableName(r)

    if _, ok := insertSqlCache[tableName]; !ok {
        sql := strings.Builder{}
        rType := reflect.TypeOf(r).Elem()

        sql.WriteString(fmt.Sprintf("insert into %s (", tableName))
        for i:= 0; i < rType.NumField(); i++ {
            if i > 0 {
                sql.WriteString(", ")
            }

            sql.WriteString(rType.Field(i).Name)
        }

        sql.WriteString(") values (?")
        sql.WriteString(strings.Repeat(", ?", rType.NumField()-1))
        sql.WriteString(")")

        insertSqlCache[tableName] = sql.String()
    }

    tx, err := db.Begin()
    if err != nil {
        return err
    }

    stmt, err := tx.Prepare(insertSqlCache[tableName])
    if err != nil {
        return err
    }
    defer stmt.Close()

    _, err = stmt.Exec(getFieldArray(r)...)
    if err != nil {
        tx.Rollback()
        return err
    }

    tx.Commit()

    return nil
}

// !!!FUTURE!!! wsm - the way the sql module is structured makes it
// *very* difficult to retrieve row data parametrically (see the
// https://pkg.go.dev/database/sql#Rows.Scan Scan() function). In
// the future it would be nice to be able to auto-populate the
// structure and return it, but sadly for now a callback it is.
func Find(r interface{}, populate scanner) (err error) {
    if err = verifyTable(r); err != nil {
        return err
    }

    tableName := getTableName(r)
    fields := getFieldMap(r)
    columns := strings.Builder{}
    where := strings.Builder{}
    whereValues := make([]interface{}, 0, len(fields))

    for field, value := range fields {
        if columns.Len() > 0 {
            columns.WriteString(", ")
        }
        columns.WriteString(field)

        if !reflect.ValueOf(value).IsZero() {
            if where.Len() > 0 {
                where.WriteString(" and ")
            }
            where.WriteString(fmt.Sprintf("%s = ?", field))
            whereValues = append(whereValues, value)
        }
    }

    sql := strings.Builder{}
    sql.WriteString(fmt.Sprintf("select %s from %s", columns.String(),
            tableName))

    if where.Len() > 0 {
        sql.WriteString(" where " + where.String())
    }

    stmt, err := db.Prepare(sql.String())
    if err != nil {
        return err
    }
    defer stmt.Close()

    rows, err := stmt.Query(whereValues...)
    if err != nil {
        return err
    }
    defer rows.Close()

    if !rows.Next() {
        return errors.New("record not found")
    }

    if err != nil {
        return err
    }

    populate(rows)

    return nil
}
