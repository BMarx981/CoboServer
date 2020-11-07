package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

//Recipes holds all of the data for our recipes
type Recipes struct {
	List []Recipe `json:"recipes"`
}

func (r *Recipes) clear() {
	r.List = nil
}

//Recipe struct contains all recipe data
type Recipe struct {
	Name   string   `json:"name"`
	Link   string   `json:"link"`
	Ingred []string `json:"ingredients"`
	Direct []string `json:"directions"`
}

func (rec *Recipe) addDirection(item string) {
	rec.Direct = append(rec.Direct, item)
}

func (rec *Recipe) addIngredient(item string) {
	rec.Ingred = append(rec.Ingred, item)
}

func (rec *Recipe) addName(name string) {
	rec.Name = name
}

func (rec *Recipe) addLink(value string) {
	rec.Link = value
}

var (
	database *sql.DB
	rs       Recipes
)

const delimiter = "??><??"

func main() {
	// loadJSON(str, &rs)

	handler := http.NewServeMux()
	handler.HandleFunc("/", search)
	err := http.ListenAndServe(":8080", handler)
	checkErr(err)
	defer database.Close()
}

func search(w http.ResponseWriter, r *http.Request) {
	var dbErr error
	database, dbErr = sql.Open("sqlite3", "./receipes.db")
	checkErr(dbErr)
	keys, good := r.URL.Query()["q"]
	if !good || len(keys[0]) < 1 {
		fmt.Println("Url is missing a search query")
		return
	}

	s := ""

	if len(keys) > 1 {
		for _, item := range keys {
			s += item + " OR "
		}
	} else {
		s = keys[0]
	}

	if keys != nil || len(keys) > 1 {
		query := "SELECT id, name, directions, ingredients, url FROM receipes WHERE name LIKE '%" + strings.ToLower(s) + "%';"
		var id int
		var name string
		var direct string
		var ing string
		var url string
		rows, err := database.Query(query, s)
		checkErr(err)
		if rows == nil {
			fmt.Println("Rows is nil", err)
			return
		}
		var r Recipe
		for rows.Next() {
			rows.Scan(&id, &name, &direct, &ing, &url)
			r.addName(name)
			r.addLink(url)
			directions := stringToArr(direct)
			for _, item := range directions {
				r.addDirection(item)
			}
			ingredients := stringToArr(ing)
			for _, item := range ingredients {
				r.addDirection(item)
			}
			rs.List = append(rs.List, r)
		}
		jData, err := json.Marshal(rs)
		checkErr(err)
		w.Header().Set("Content-Type", "application/json")
		w.Write(jData)
		rs.clear()
	}
}

func loadJSON(list []string, rs *Recipes) {
	database, err := sql.Open("sqlite3", "./receipes.db")
	checkErr(err)
	statement, err := database.Prepare("CREATE TABLE IF NOT EXISTS receipes (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, directions TEXT, ingredients TEXT, url TEXT)")
	checkErr(err)
	// statement.Exec()
	myr := "/Users/brianmarx/Desktop/Cobo/DataSets/MyRecipesDataSet-min.json"
	food := "/Users/brianmarx/Desktop/Cobo/DataSets/food9858dataSet-min.json"
	epi := "/Users/brianmarx/Desktop/Cobo/DataSets/epiDataSetDirectionsFixed-min.json"
	files := [3]string{myr, food, epi}
	for _, name := range files {

		f, err := os.Open(name)
		checkErr(err)
		defer f.Close()
		recs, err := ioutil.ReadAll(f)

		json.Unmarshal(recs, &rs)
		checkErr(err)
		for _, val := range rs.List {
			dir := arrToString(val.Direct)
			ins := arrToString(val.Ingred)
			name := val.Name
			link := val.Link

			statement, err = database.Prepare("INSERT INTO receipes (name, directions, ingredients, url) VALUES (?, ?, ?, ?)")
			checkErr(err)
			statement.Exec(name, dir, ins, link)
			checkErr(err)
			statement.Close()
			fmt.Println("Executed insert for: ", name)
		}
	}
}

func arrToString(list []string) string {
	str := ""
	for _, item := range list {
		str += item + delimiter
	}
	return str
}

func stringToArr(str string) []string {
	var list []string
	s := strings.Split(str, delimiter)
	for _, item := range s {
		list = append(list, item)
	}
	fmt.Println(list)
	return list
}

func checkErr(e error) {
	if e != nil {
		fmt.Println("Error with: ", e)
	}
}
