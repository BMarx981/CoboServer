package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

//Recipes holds all of the data for our recipes
type Recipes struct {
	List []Recipe `json:"recipes"`
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
	str := make([]string, 0)
	loadJSON(str, &rs)

	// handler := http.NewServeMux()
	// handler.HandleFunc("/", search)
	// err := http.ListenAndServe(":8080", handler)
	// checkErr(err)
}

func search(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query()["search"]
	s := ""
	for _, item := range key {
		s += item + " AND "
	}
	if key == nil || len(key[0]) < 1 {
		rows, err := database.Query("SELECT name, directions, ingredients, link FROM receipes WHERE name " + s)
		checkErr(err)
		var name string
		var direct string
		var ing string
		var link string
		for rows.Next() {
			rows.Scan(&name, &direct, &ing, &link)
		}
	}
}

func loadJSON(list []string, rs *Recipes) {
	database, err := sql.Open("sqlite3", "./receipes.db")
	checkErr(err)
	statement, err := database.Prepare("CREATE TABLE IF NOT EXISTS receipes (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, directions TEXT, ingredients TEXT, url TEXT)")
	checkErr(err)
	statement.Exec()
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
			result, err := statement.Exec(name, dir, ins, link)
			checkErr(err)
			tni, err := result.LastInsertId()
			checkErr(err)
			fmt.Println("Row inserted", tni)
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

func checkErr(e error) {
	if e != nil {
		fmt.Println("Error with: ", e)
	}
}
