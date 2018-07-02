package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

type Page struct {
	Title    string
	Filename string
	Body     []byte
}

type RootPage struct {
	Title string
	Pages []*Page
}

const DataPath = "data/"

// write Page
func (p *Page) save() error {
	filename := DataPath + p.Filename + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

// read Page
func loadPage(filename string) (*Page, error) {

	fmt.Println("attempting to load: ", filename)
	var replacer = strings.NewReplacer("-", " ")
	title := replacer.Replace(filename)

	body, err := ioutil.ReadFile(DataPath + filename + ".txt")
	if err != nil {
		fmt.Println("error:", err)
		return nil, err
	}

	return &Page{Title: title, Filename: filename, Body: body}, nil
}

func formatTitle(t string) string {
	i := strings.LastIndex(t, ".")
	n := t[0:i]
	return strings.ToLower(n)
}

func getPages() ([]*Page, error) {
	dList, err := ioutil.ReadDir(DataPath)
	if err != nil {
		fmt.Println("error:", err)
		return nil, err
	}

	var pages []*Page
	for _, file := range dList {
		filename := formatTitle(file.Name())
		page, err := loadPage(filename)
		if err != nil {
			fmt.Println("error:", err)
			return nil, err
		}
		pages = append(pages, page)
	}

	return pages, nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates[tmpl].ExecuteTemplate(w, "layout", p)
	if err != nil {
		fmt.Println("error:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9-]+)$")

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("handling request:", r.URL.Path)

		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func saveHandler(w http.ResponseWriter, r *http.Request, filename string) {
	body := r.FormValue("body")

	var replacer = strings.NewReplacer("-", " ")
	title := replacer.Replace(filename)

	p := &Page{Title: title, Filename: filename, Body: []byte(body)}
	err := p.save()
	if err != nil {
		fmt.Println("error:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+filename, http.StatusFound)
}

func viewHandler(w http.ResponseWriter, r *http.Request, filename string) {
	p, err := loadPage(filename)
	if err != nil {
		fmt.Println("error:", err)
		http.NotFound(w, r)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, filename string) {
	p, err := loadPage(filename)
	if err != nil {
		var replacer = strings.NewReplacer("-", " ")
		var title = replacer.Replace(filename)
		p = &Page{Title: title, Filename: filename}
	}
	renderTemplate(w, "edit", p)
}

func newHandler(w http.ResponseWriter, r *http.Request) {
	err := templates["new"].ExecuteTemplate(w, "layout", nil)
	if err != nil {
		fmt.Println("error:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func createNewHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handling request:", r.URL)

	body := r.FormValue("body")
	fmt.Println("body:", body)

	// TODO: add validation
	title := r.FormValue("title")
	var replacer = strings.NewReplacer(" ", "-")
	filename := replacer.Replace(title)

	fmt.Println("title:", title)

	p := &Page{Title: title, Filename: filename, Body: []byte(body)}
	err := p.save()
	if err != nil {
		fmt.Println("error:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+filename, http.StatusFound)
}

func renderRootTemplate(w http.ResponseWriter, tmpl string, p *RootPage) {
	err := templates[tmpl].ExecuteTemplate(w, "layout", p)
	if err != nil {
		fmt.Println("error:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handling request:", r.URL)

	pages, err := getPages()
	if err != nil {
		fmt.Println("error:", err)
		http.Error(w, "Unable to read list", 500)
		return
	}

	c := &RootPage{Title: "Home", Pages: pages}
	renderRootTemplate(w, "root", c)
}

var templates = make(map[string]*template.Template)

func main() {

	templates["root"] = template.Must(template.ParseFiles("tmpl/root.html", "tmpl/layout.html"))
	templates["view"] = template.Must(template.ParseFiles("tmpl/view.html", "tmpl/layout.html"))
	templates["edit"] = template.Must(template.ParseFiles("tmpl/edit.html", "tmpl/layout.html"))
	templates["new"] = template.Must(template.ParseFiles("tmpl/new.html", "tmpl/layout.html"))

	fmt.Println("Serving ...")

	// register handlers
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	http.HandleFunc("/new/", newHandler)
	http.HandleFunc("/create/", createNewHandler)
	http.HandleFunc("/", rootHandler)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.ListenAndServe(":8080", nil)
}
