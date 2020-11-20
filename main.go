package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
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
	Image  string   `json:"imageurl"`
}

func (rec *Recipe) addDirection(item string) {
	rec.Direct = append(rec.Direct, item)
}

func (rec *Recipe) addDirectionList(list []string) {
	rec.Direct = list
}

func (rec *Recipe) addIngredient(item string) {
	rec.Ingred = append(rec.Ingred, item)
}

func (rec *Recipe) addIngredientList(list []string) {
	rec.Ingred = list
}

func (rec *Recipe) addName(name string) {
	rec.Name = name
}

func (rec *Recipe) addLink(value string) {
	rec.Link = value
}

func (rec *Recipe) clear() {
	rec.Ingred = make([]string, 0)
	rec.Direct = make([]string, 0)
}

var (
	database *sql.DB
	rs       Recipes
)

const delimiter = "??><??"

func main() {
	// loadJSON(&rs)
	// ma := findURLLinks()
	// ma := map[int]string{8090: "https://www.food.com/recipe/the-naughty-things-i-do-for-chicken-tortilla-soup-173513", 5082: "https://www.food.com/recipe/burrito-bake-38947"}
	// imgData := getImageLinks(ma)
	// insertImageIntoDB(imgData)

	handler := http.NewServeMux()
	handler.HandleFunc("/", search)
	err := http.ListenAndServe(":8080", handler)
	checkErr(err, "")
	defer database.Close()
}

func insertImageIntoDB(imgData map[int]string) {
	database, err := sql.Open("sqlite3", "./recipes.db")
	checkErr(err, "Open DB err")
	defer database.Close()
	for k, v := range imgData {
		statement, err := database.Prepare("UPDATE recipes SET imageurl = \"" + v + "\" WHERE id = " + strconv.Itoa(k))
		checkErr(err, "Prepare issue")
		statement.Exec()
	}
}

func extractFoodImageLink() map[string]string {
	file, err := os.Open("/Users/brianmarx/Desktop/Cobo/DataSets/foodNamesImageLinks.txt")
	checkErr(err, "Open file err")
	s := bufio.NewScanner(file)
	imgLinkMap := make(map[string]string)
	for s.Scan() {
		start := strings.Split(s.Text(), "?:>")
		name := strings.TrimSpace(strings.TrimPrefix(start[0], "Key: "))
		imgLink := strings.TrimSpace(strings.TrimPrefix(start[1], " Value: "))
		if strings.Contains(imgLink, "data:image/gif;base64,") {
			imgLink = ""
		}
		imgLinkMap[name] = imgLink
	}
	// for k, v := range imgLinkMap {
	// 	fmt.Println("Link Map " + k + " Value " + v)
	// }

	return imgLinkMap
}

func getImageLinks(dataMap map[int]string) map[int]string {
	imageMap := make(map[int]string)
	database, err := sql.Open("sqlite3", "./recipes.db")
	defer database.Close()
	checkErr(err, "Open DB issue")
	foodData := extractFoodImageLink()
	for k, v := range dataMap {

		resp, err := http.Get(v)
		checkErr(err, "Response err")
		doc, err := goquery.NewDocumentFromReader(resp.Body)

		checkErr(err, "document err")

		switch getHost(v) {
		case "myrecipes.com":
			imageMap[k] = myRecipes(*doc)
		case "epicurious.com":
			imageMap[k] = epicurious(*doc)
		case "food.com":
			imageMap[k] = food(database, v, foodData)
		}

	}
	return imageMap
}

func food(database *sql.DB, url string, imgLinks map[string]string) string {
	str := ""

	rows, err := database.Query("SELECT id, name, url FROM recipes WHERE url = \"" + url + "\"")
	checkErr(err, "Query error")
	var id int
	var name string
	var urlString string
	keys := make([]string, 0, len(imgLinks))

	for key := range imgLinks {
		keys = append(keys, key)
	}

	for rows.Next() {
		err := rows.Scan(&id, &name, &urlString)
		fmt.Println("|" + name + "|")
		trim := strings.TrimSpace(name)
		checkErr(err, "Scan Error")
		for _, k := range keys {
			if strings.Contains(trim, k) && len(trim) == len(k) {
				up := imgLinks[k]
				str = up
			}
		}
	}
	return str
}

func getHost(val string) string {
	hosts := []string{"myrecipes.com", "epicurious.com", "food.com"}
	sw := ""
	for _, name := range hosts {
		if strings.Contains(val, name) {
			sw = name
		}
	}
	return sw
}

func epicurious(doc goquery.Document) string {
	str := ""
	doc.Find("picture[class=photo-wrap]").Find("img").Each(func(i int, nodes *goquery.Selection) {
		t, b := nodes.Attr("data-src")
		if b {
			str = t
		}
	})
	fmt.Println(str)
	return str
}

func myRecipes(doc goquery.Document) string {
	str := ""
	doc.Find(".image-container").Find(".component").Each(func(i int, nodes *goquery.Selection) {
		t, b := nodes.Attr("data-src")
		if b {
			str = t
		}
	})
	return str
}

func findURLLinks() map[int]string {
	dataMap := make(map[int]string)
	database, err := sql.Open("sqlite3", "./recipes.db")
	defer database.Close()
	checkErr(err, "Open DB issue")
	rows, err := database.Query("SELECT id, name, url FROM recipes")
	checkErr(err, "Query issue")
	for rows.Next() {
		var id int
		var name string
		var url string
		err := rows.Scan(&id, &name, &url)
		dataMap[id] = url
		checkErr(err, "Scan err")
		// fmt.Println(" name and url: " + strconv.Itoa(id) + " " + name + " " + url)
	}
	return dataMap
}

func search(w http.ResponseWriter, r *http.Request) {
	var dbErr error
	database, dbErr = sql.Open("sqlite3", "./recipes.db")
	checkErr(dbErr, "")
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
		query := "SELECT id, name, directions, ingredients, url, image FROM recipes WHERE name LIKE '%" + strings.ToLower(s) + "%' LIMIT 10;"
		var id int
		var name string
		var direct string
		var ing string
		var url string
		var image string
		rows, err := database.Query(query, s)
		checkErr(err, "")
		if rows == nil {
			fmt.Println("Rows is nil", err)
			return
		}
		var r Recipe
		for rows.Next() {
			rows.Scan(&id, &name, &direct, &ing, &url, &image)
			r.addName(name)
			r.addLink(url)
			r.Image = image
			directions := stringToArr(direct)
			for _, item := range directions {
				r.addDirection(item)
			}
			ingredients := stringToArr(ing)
			for _, item := range ingredients {
				r.addIngredient(item)
			}
			rs.List = append(rs.List, r)
			r.clear()
		}
		jData, err := json.Marshal(rs)
		checkErr(err, "")
		w.Header().Set("Content-Type", "application/json")
		w.Write(jData)
		rs.clear()
	}
}

func loadJSON(rs *Recipes) {
	database, err := sql.Open("sqlite3", "./recipes.db")
	checkErr(err, "table issue")
	statement, err := database.Prepare("CREATE TABLE IF NOT EXISTS recipes (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, directions TEXT, ingredients TEXT, url TEXT)")
	checkErr(err, "create table issue")
	statement.Exec()
	myr := "/Users/brianmarx/Desktop/Cobo/DataSets/MyRecipesDataSet-min.json"
	food := "/Users/brianmarx/Desktop/Cobo/DataSets/food9858dataSet-min.json"
	epi := "/Users/brianmarx/Desktop/Cobo/DataSets/epiDataSetDirectionsFixed-min.json"
	files := [3]string{myr, food, epi}
	for _, name := range files {

		f, err := os.Open(name)
		checkErr(err, "")
		defer f.Close()
		recs, err := ioutil.ReadAll(f)

		json.Unmarshal(recs, &rs)
		checkErr(err, "")
		for _, val := range rs.List {
			dir := arrToString(val.Direct)
			ins := arrToString(val.Ingred)
			name := val.Name
			link := "https://" + val.Link

			statement, err = database.Prepare("INSERT INTO recipes (name, ingredients, directions, url) VALUES (?, ?, ?, ?);")
			checkErr(err, "Insert error")
			statement.Exec(name, ins, dir, link)
			checkErr(err, "")
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
		if item == "" {
			continue
		} else {
			list = append(list, item)
		}
	}
	return list
}

func checkErr(e error, s string) {
	if e != nil {
		fmt.Println("Error with: "+s, e)
	}
}

// func runOnce() {
// 	database, err := sql.Open("sqlite3", "./recipes.db")
// 	checkErr(err, "Open DB err")
// 	defer database.Close()
// 	statement, err := database.Prepare("ALTER TABLE recipes ADD COLUMN imageurl TEXT")
// 	checkErr(err, "alter table issue")
// 	statement.Exec()
// 	checkErr(err, "execute alter table issue")
// }

// var html = `<div class="recipe-layout__media">
// 			<div class="recipe-media">
// 				<!---->
// 				<div class="recipe-hero">
// 					<div class="recipe-hero__item">
// 						<div class="recipe-image theme-gradient">
// 							<div class="inner-wrapper">
// 								<img src="https://img.sndimg.com:443/food/image/upload/c_thumb,q_80,w_571,h_321/v1/img/recipes/17/35/13/picieHeQV.jpg" class="recipe-image__img" data-src="https://img.sndimg.com:443/food/image/upload/c_thumb,q_80,w_571,h_321/v1/img/recipes/17/35/13/picieHeQV.jpg" lazy="loaded">
// 							</div>
// 						</div>
// 					</div>
// 					<div style="display:none;">
// 						<!---->
// 						<!---->
// 					</div>
// 				</div>
// 				<ul class="photo-gallery">
// 					<li class="photo-gallery__item">
// 						<div class="photo-gallery__thumbnail active theme-border">
// 							<div>
// 								<div class="recipe-image theme-gradient">
// 									<div class="inner-wrapper">
// 										<img src="https://img.sndimg.com:443/food/image/upload/c_thumb,q_80,w_111,h_62/v1/img/recipes/17/35/13/picieHeQV.jpg" class="recipe-image__img" data-src="https://img.sndimg.com:443/food/image/upload/c_thumb,q_80,w_111,h_62/v1/img/recipes/17/35/13/picieHeQV.jpg" lazy="loaded">
// 									</div>
// 								</div>
// 							</div>
// 						</div>
// 						<!---->
// 					</li>
// 					<li class="photo-gallery__item">
// 						<div class="photo-gallery__thumbnail">
// 							<div>
// 								<div class="recipe-image theme-gradient">
// 									<div class="inner-wrapper">
// 										<img src="https://img.sndimg.com:443/food/image/upload/c_thumb,q_80,w_111,h_62/v1/img/recipes/17/35/13/picrVbjU5.jpg" class="recipe-image__img" data-src="https://img.sndimg.com:443/food/image/upload/c_thumb,q_80,w_111,h_62/v1/img/recipes/17/35/13/picrVbjU5.jpg" lazy="loaded">
// 									</div>
// 								</div>
// 							</div>
// 						</div>
// 						<!---->
// 					</li>
// 					<li class="photo-gallery__item">
// 						<div class="photo-gallery__thumbnail">
// 							<div>
// 								<div class="recipe-image theme-gradient">
// 									<div class="inner-wrapper">
// 										<img src="https://img.sndimg.com:443/food/image/upload/c_thumb,q_80,w_111,h_62/v1/img/recipes/17/35/13/pichr6JgO.jpg" class="recipe-image__img" data-src="https://img.sndimg.com:443/food/image/upload/c_thumb,q_80,w_111,h_62/v1/img/recipes/17/35/13/pichr6JgO.jpg" lazy="loaded">
// 									</div>
// 								</div>
// 							</div>
// 						</div>
// 						<!---->
// 					</li>
// 					<li class="photo-gallery__item">
// 						<div class="photo-gallery__thumbnail">
// 							<div>
// 								<div class="recipe-image theme-gradient">
// 									<div class="inner-wrapper">
// 										<img src="https://img.sndimg.com:443/food/image/upload/c_thumb,q_80,w_111,h_62/v1/img/recipes/17/35/13/picFm8kHd.jpg" class="recipe-image__img" data-src="https://img.sndimg.com:443/food/image/upload/c_thumb,q_80,w_111,h_62/v1/img/recipes/17/35/13/picFm8kHd.jpg" lazy="loaded">
// 									</div>
// 								</div>
// 							</div>
// 						</div>
// 						<!---->
// 					</li>
// 					<li class="photo-gallery__item">
// 						<div class="photo-gallery__thumbnail">
// 							<div>
// 								<div class="recipe-image theme-gradient">
// 									<div class="inner-wrapper">
// 										<img src="https://img.sndimg.com:443/food/image/upload/c_thumb,q_80,w_111,h_62/v1/img/recipes/17/35/13/picNczEkf.jpg" class="recipe-image__img" data-src="https://img.sndimg.com:443/food/image/upload/c_thumb,q_80,w_111,h_62/v1/img/recipes/17/35/13/picNczEkf.jpg" lazy="loaded">
// 									</div>
// 								</div>
// 							</div>
// 						</div>
// 						<!---->
// 					</li>
// 				</ul>
// 				<!---->
// 				<!---->
// 			</div>
// 		</div>`
